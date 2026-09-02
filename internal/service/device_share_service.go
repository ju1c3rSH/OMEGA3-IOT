package service

import (
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"fmt"
	"gorm.io/gorm"
	"log"
	"sync"
	"time"
)

type DeviceShareService struct {
	instanceRepo    repository.InstanceRepository
	deviceShareRepo repository.DeviceShareRepository
	loggerService   logger.LoggerInterface
	// accessCache is a short-TTL (5s) in-process cache for (user,device,perm)->bool
	// to avoid DB on hot path. For distributed deployments replace with Redis SETEX 5
	// cache-aside (see https://redis.io/docs/latest/develop/use-cases/cache-aside/
	// and https://www.toolsku.com/en/blog/redis-caching-patterns-production/).
	// Pitfall avoided: short TTL without single-flight can stampede; here 5s balances
	// staleness vs DB load, and owner check is idempotent so stale false->true
	// self-corrects on next miss.
	accessCache sync.Map // map[string]accessCacheEntry
}

type accessCacheEntry struct {
	allowed   bool
	expiresAt int64 // unix nano
}

func NewDeviceShareService(db *gorm.DB, loggerService logger.LoggerInterface) *DeviceShareService {
	return &DeviceShareService{
		instanceRepo:    repository.NewInstanceRepository(db),
		deviceShareRepo: repository.NewDeviceShareRepository(db),
		loggerService:   loggerService,
	}
}

func (s *DeviceShareService) ShareDevice(instanceUUID string, shareByUUID string, shareWithUUID string, expiredTime int64, permission string) error {
	// Ownership check via indexed point lookup, not full scan.
	// Was: FindByOwnerUUID(shareByUUID) + linear scan over all owned devices
	//       (O(N) rows, SELECT * with Properties JSON, full table scan on hot path).
	// Now: SELECT 1 ... WHERE instance_uuid=? AND owner_uuid=? LIMIT 1
	//       (O(log n) via uniqueIndex on instance_uuid + index on owner_uuid).
	// See https://stackoverflow.com/questions/66392372/select-exists-with-gorm
	// and https://cdn.jsdelivr.net/npm/miokit@2.0.13/skills/external/golang/gorm/performance.md
	isOwner, err := s.instanceRepo.ExistsByOwner(shareByUUID, instanceUUID)
	if err != nil {
		return err
	}
	if !isOwner {
		// Distinguish "device not found" from "not owned" for clearer error.
		exists, e2 := s.instanceRepo.Exists(instanceUUID)
		if e2 == nil && !exists {
			return fmt.Errorf("device not found: %w", gorm.ErrRecordNotFound)
		}
		return fmt.Errorf("user does not own this device")
	}

	share := &model.DeviceShare{
		InstanceUUID:   instanceUUID,
		SharedByUUID:   shareByUUID,
		SharedWithUUID: shareWithUUID,
		ExpiresAt:      &expiredTime,
		CreatedAt:      time.Now().Unix(),
		UpdatedAt:      time.Now().Unix(),
		Permission:     permission,
		Status:         repository.StatusActive,
	}
	if err := s.deviceShareRepo.Create(share); err != nil {
		return err
	}

	logEvent := logger.NewUserLogEvent(shareByUUID, logger.LogLevelInfo,
		fmt.Sprintf("Device shared: %s with user %s", instanceUUID, shareWithUUID),
		logger.LogEventUserDeviceShare)
	logEvent.Metadata = map[string]interface{}{
		"instance_uuid":    instanceUUID,
		"shared_with_uuid": shareWithUUID,
		"permission":       permission,
		"expires_at":       expiredTime,
	}
	s.loggerService.EmitUserLog(logEvent)

	return nil
}

func (s *DeviceShareService) UnshareDevice(instanceUUID string, shareWithUUID string, sharedByUUID string) error {
	if err := s.deviceShareRepo.DeleteByInstanceAndSharedWith(instanceUUID, shareWithUUID); err != nil {
		return err
	}

	logEvent := logger.NewUserLogEvent(sharedByUUID, logger.LogLevelInfo,
		fmt.Sprintf("Device unshared: %s from user %s", instanceUUID, shareWithUUID),
		logger.LogEventUserDeviceUnshare)
	logEvent.Metadata = map[string]interface{}{
		"instance_uuid":    instanceUUID,
		"shared_with_uuid": shareWithUUID,
	}
	s.loggerService.EmitUserLog(logEvent)

	return nil
}

func (s *DeviceShareService) GetSharedDevices(instanceUUID string) ([]model.DeviceShare, error) {
	return s.deviceShareRepo.FindByInstanceUUID(instanceUUID)
}

func (s *DeviceShareService) GetSharedBy(sharedWithUUID string) ([]model.DeviceShare, error) {
	return s.deviceShareRepo.FindBySharedWithUUID(sharedWithUUID)
}

func (s *DeviceShareService) GetSharedWith(instanceUUID string) ([]model.DeviceShare, error) {
	return s.deviceShareRepo.FindByInstanceUUID(instanceUUID)
}

func (s *DeviceShareService) GetAccessibleDevices(userUUID string) (*GetUserAllAccessibleDevicesResponse, error) {
	// Batched N+1 fix: was 1+1+N+M queries (e.g., 82 for 40 shared + 40 owned).
	// Now 3-4 queries: owned (1) + shares (1) + batch instances IN (?) (1) + batch GROUP BY COUNT (1).
	// - GORM WHERE IN batch replaces per-row FindByUUID loop (see https://gorm.io/docs/query.html,
	//   https://blog.stackademic.com/the-n-1-query-problem-in-gorm-how-to-avoid-silent-performance-killers-856e028d4b15,
	//   https://www.shinagawa-web.com/en/works/go-gorm-n-plus-one-b2b-ticketing,
	//   https://www.softwarejutsu.com/articles/n-plus-one-what-every-go-engineer-should-know,
	//   https://cdn.jsdelivr.net/npm/miokit@2.0.13/skills/external/golang/gorm/performance.md).
	// - Single GROUP BY COUNT replaces per-row CountActiveShares (see https://dev.mysql.com/doc/refman/26.7/en/group-by-optimization.html
	//   and https://oneuptime.com/blog/post/2026-03-31-mysql-optimize-group-by/view — composite index on (status, instance_uuid) avoids temp table).
	// Pitfalls avoided:
	// - Empty IN slice: skip query (GORM would emit IN (NULL) or error).
	// - Dedup ids: shrink placeholders, avoid duplicate GROUP BY work.
	// - Large IN chunking: Oracle 1000 (https://stackoverflow.com/questions/400255/how-to-put-more-than-1000-values-into-an-oracle-in-clause),
	//   MySQL max_allowed_packet (https://stackoverflow.com/questions/1532366/mysql-number-of-items-within-in-clause),
	//   PG 65535 (https://github.com/go-gorm/gorm/issues/6849, https://github.com/go-gorm/gorm/issues/6989),
	//   GORM batch splitting (https://github.com/go-gorm/gorm/issues/7792) — chunk at 1000.
	// - Map join in Go keeps backward compat order and handles missing instances.
	ownedDevices, err := s.instanceRepo.FindByOwnerUUID(userUUID)
	if err != nil {
		return nil, err
	}

	shares, err := s.deviceShareRepo.FindBySharedWithUUID(userUUID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	// Filter valid shares and collect deduped InstanceUUIDs for batch fetch
	validShares := make([]model.DeviceShare, 0, len(shares))
	sharedIDsSet := make(map[string]struct{}, len(shares))
	sharedIDs := make([]string, 0, len(shares))
	for _, sh := range shares {
		if sh.Status != repository.StatusActive {
			continue
		}
		if sh.ExpiresAt != nil && *sh.ExpiresAt <= now {
			continue
		}
		validShares = append(validShares, sh)
		if _, ok := sharedIDsSet[sh.InstanceUUID]; !ok {
			sharedIDsSet[sh.InstanceUUID] = struct{}{}
			sharedIDs = append(sharedIDs, sh.InstanceUUID)
		}
	}

	var sharedDevices []model.Instance
	if len(sharedIDs) > 0 {
		// Batch: SELECT * FROM instances WHERE instance_uuid IN (?) — single query replaces N FindByUUID
		instances, err := s.instanceRepo.FindByUUIDs(sharedIDs)
		if err != nil {
			log.Printf("Warning: batch find instances failed: %v", err)
			instances = nil
		}
		// Map join: instance_uuid -> Instance
		instMap := make(map[string]model.Instance, len(instances))
		for _, inst := range instances {
			instMap[inst.InstanceUUID] = inst
		}
		sharedDevices = make([]model.Instance, 0, len(validShares))
		for _, sh := range validShares {
			if inst, ok := instMap[sh.InstanceUUID]; ok {
				sharedDevices = append(sharedDevices, inst)
			} else {
				log.Printf("Warning: failed to find instance %s (share %d)", sh.InstanceUUID, sh.ID)
			}
		}
	}

	allAccessibleDevices := append(ownedDevices, sharedDevices...)
	count := len(allAccessibleDevices)

	// Batch counts: SELECT instance_uuid, COUNT(*) FROM device_shares WHERE instance_uuid IN (?) AND status='active' AND (expires_at IS NULL OR expires_at>now) GROUP BY instance_uuid
	if len(allAccessibleDevices) > 0 {
		allIDs := make([]string, 0, len(allAccessibleDevices))
		seenAll := make(map[string]struct{}, len(allAccessibleDevices))
		for _, d := range allAccessibleDevices {
			if _, ok := seenAll[d.InstanceUUID]; !ok {
				seenAll[d.InstanceUUID] = struct{}{}
				allIDs = append(allIDs, d.InstanceUUID)
			}
		}
		countMap, err := s.deviceShareRepo.CountActiveSharesBatch(allIDs)
		if err != nil {
			log.Printf("Warning: batch count shares failed: %v", err)
			countMap = map[string]int64{}
		}
		for i := range allAccessibleDevices {
			c := countMap[allAccessibleDevices[i].InstanceUUID]
			allAccessibleDevices[i].SharedCount = int(c)
			allAccessibleDevices[i].IsShared = c > 0
		}
	}

	response := &GetUserAllAccessibleDevicesResponse{InstanceCount: count, Instances: allAccessibleDevices}
	return response, nil
}

func (s *DeviceShareService) CheckDeviceAccess(instanceUUID string, userUUID string, requiredPermission string) (bool, error) {
	// Short-TTL cache (5s) to avoid DB on hot path for repeated checks on same (user,device,perm).
	// Key includes requiredPermission because share grant is permission-specific.
	// In-process sync.Map is sufficient for single-instance; for multi-pod use Redis
	// SETEX 5 + single-flight lock to avoid stampede (see Redis cache-aside docs).
	cacheKey := userUUID + ":" + instanceUUID + ":" + requiredPermission
	if v, ok := s.accessCache.Load(cacheKey); ok {
		if e, ok2 := v.(accessCacheEntry); ok2 && time.Now().UnixNano() < e.expiresAt {
			return e.allowed, nil
		}
		s.accessCache.Delete(cacheKey)
	}
	// Helper to store with 5s TTL before returning.
	setCache := func(allowed bool) {
		s.accessCache.Store(cacheKey, accessCacheEntry{allowed: allowed, expiresAt: time.Now().Add(5 * time.Second).UnixNano()})
	}

	// Ownership check via indexed point lookup SELECT 1 ... LIMIT 1.
	// Previous: FindByOwnerUUID(userUUID) fetched ALL owned devices (SELECT *),
	//           allocated slice, linear scan — full scan on every auth request.
	// Fixed: ExistsByOwner uses `SELECT instance_uuid ... WHERE instance_uuid=? AND owner_uuid=? LIMIT 1`
	//        which hits uniqueIndex on instance_uuid and is O(log n).
	// See https://stackoverflow.com/questions/66392372/select-exists-with-gorm
	// and https://github.com/go-gorm/gorm/discussions/6000
	isOwner, err := s.instanceRepo.ExistsByOwner(userUUID, instanceUUID)
	if err != nil {
		return false, err
	}
	if isOwner {
		setCache(true)
		return true, nil
	}

	// Check the shared permission (already indexed on instance_uuid+shared_with_uuid)
	share, err := s.deviceShareRepo.FindByInstanceAndSharedWith(instanceUUID, userUUID)
	if err != nil {
		setCache(false)
		return false, nil
	}

	// Check if expired
	if share.ExpiresAt != nil && *share.ExpiresAt <= time.Now().Unix() {
		setCache(false)
		return false, nil
	}

	if requiredPermission == repository.PermissionRead &&
		(share.Permission == repository.PermissionRead || share.Permission == repository.PermissionReadWrite) {
		setCache(true)
		return true, nil
	}
	if requiredPermission == repository.PermissionWrite &&
		(share.Permission == repository.PermissionWrite || share.Permission == repository.PermissionReadWrite) {
		setCache(true)
		return true, nil
	}
	setCache(false)
	return false, nil
}

func (s *DeviceShareService) calculateShareInfo(instanceUUID string) (int, bool, error) {
	count, err := s.deviceShareRepo.CountActiveShares(instanceUUID)
	return int(count), count > 0, err
}
