package service

import (
	"OMEGA3-IOT/internal/eventbus"
	"OMEGA3-IOT/internal/logger"
	"OMEGA3-IOT/internal/repository"
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// PresenceService manages device online/offline lifecycle.
// It marks devices online on MQTT activity, offline on shutdown events or staleness.
type PresenceService struct {
	instanceRepo   repository.InstanceRepository
	eventBus       *eventbus.EventBus
	offlineTimeout time.Duration
	checkInterval  time.Duration
	stopCh         chan struct{}
	wg             sync.WaitGroup

	// in-memory cache of last-known online devices to avoid redundant DB writes
	onlineDevices sync.Map // map[string]bool
}

func NewPresenceService(
	instanceRepo repository.InstanceRepository,
	eventBus *eventbus.EventBus,
	offlineTimeoutSec int,
	checkIntervalSec int,
) *PresenceService {
	if offlineTimeoutSec <= 0 {
		offlineTimeoutSec = 300 // default 5 minutes
	}
	if checkIntervalSec <= 0 {
		checkIntervalSec = 60 // default 1 minute
	}

	return &PresenceService{
		instanceRepo:   instanceRepo,
		eventBus:       eventBus,
		offlineTimeout: time.Duration(offlineTimeoutSec) * time.Second,
		checkInterval:  time.Duration(checkIntervalSec) * time.Second,
		stopCh:         make(chan struct{}),
	}
}

// Start launches the background goroutine that periodically checks for stale devices.
func (ps *PresenceService) Start() {
	ps.wg.Add(1)
	go ps.run()
	log.Printf("[PresenceService] Started (offline timeout: %v, check interval: %v)", ps.offlineTimeout, ps.checkInterval)
}

// Stop gracefully shuts down the background checker.
func (ps *PresenceService) Stop() {
	close(ps.stopCh)
	ps.wg.Wait()
	log.Println("[PresenceService] Stopped")
}

// MarkOnline marks a device as online. Called when MQTT data arrives.
// Uses an in-memory cache to skip redundant DB writes for devices already known online.
func (ps *PresenceService) MarkOnline(deviceUUID string) {
	if _, loaded := ps.onlineDevices.LoadOrStore(deviceUUID, true); loaded {
		// Already known online — just update LastSeen via the normal property update path
		return
	}

	// First time seeing this device online (or it was previously offline)
	if err := ps.instanceRepo.UpdateOnlineStatus(deviceUUID, true, time.Now().Unix()); err != nil {
		log.Printf("[PresenceService] Failed to mark device %s online: %v", deviceUUID, err)
		ps.onlineDevices.Delete(deviceUUID)
		return
	}

	log.Printf("[PresenceService] Device %s is now ONLINE", deviceUUID)
	ps.emitStatusChange(deviceUUID, true)
}

// MarkOffline marks a device as offline and emits a status change event.
func (ps *PresenceService) MarkOffline(deviceUUID string) {
	if _, loaded := ps.onlineDevices.LoadAndDelete(deviceUUID); !loaded {
		return // already known offline, skip
	}

	if err := ps.instanceRepo.UpdateOnlineStatus(deviceUUID, false, time.Now().Unix()); err != nil {
		log.Printf("[PresenceService] Failed to mark device %s offline: %v", deviceUUID, err)
		return
	}

	log.Printf("[PresenceService] Device %s is now OFFLINE", deviceUUID)
	ps.emitStatusChange(deviceUUID, false)
}

// HandleShutdownEvent processes a device-initiated shutdown/offline event.
func (ps *PresenceService) HandleShutdownEvent(deviceUUID string) {
	ps.MarkOffline(deviceUUID)
}

func (ps *PresenceService) run() {
	defer ps.wg.Done()
	ticker := time.NewTicker(ps.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ps.stopCh:
			return
		case <-ticker.C:
			ps.checkStaleDevices()
		}
	}
}

// checkStaleDevices scans all devices that are marked online in the DB
// and marks those with stale LastSeen as offline.
func (ps *PresenceService) checkStaleDevices() {
	threshold := time.Now().Add(-ps.offlineTimeout).Unix()

	// Iterate over our in-memory cache of online devices
	ps.onlineDevices.Range(func(key, _ interface{}) bool {
		deviceUUID := key.(string)

		instance, err := ps.instanceRepo.FindByUUID(deviceUUID)
		if err != nil {
			// Device may have been deleted — remove from cache
			ps.onlineDevices.Delete(deviceUUID)
			return true
		}

		if instance.LastSeen < threshold {
			ps.MarkOffline(deviceUUID)
		}
		return true
	})
}

func (ps *PresenceService) emitStatusChange(deviceUUID string, online bool) {
	status := "offline"
	if online {
		status = "online"
	}

	message := fmt.Sprintf("Device %s is now %s", deviceUUID, status)
	event := logger.NewDeviceLogEvent(deviceUUID, logger.LogLevelInfo, message, logger.LogEventDeviceStatusChange)
	event.Metadata["status"] = status

	ps.eventBus.Publish(context.Background(), event)
}

// GetOnlineDevices returns a snapshot of currently online device UUIDs (from cache).
func (ps *PresenceService) GetOnlineDevices() []string {
	var devices []string
	ps.onlineDevices.Range(func(key, _ interface{}) bool {
		devices = append(devices, key.(string))
		return true
	})
	return devices
}
