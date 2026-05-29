package service

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// AdminService handles all admin management business logic.
type AdminService struct {
	db            *gorm.DB
	userRepo      repository.UserRepository
	adminUserRepo repository.AdminUserRepository
	adminDevRepo  repository.AdminDeviceRepository
	instanceRepo  repository.InstanceRepository
	groupRepo     repository.UserGroupRepository
	memberRepo    repository.GroupMemberRepository
	adminLogRepo  repository.AdminLogRepository
}

// NewAdminService creates a new AdminService.
func NewAdminService(
	db *gorm.DB,
	userRepo repository.UserRepository,
	adminUserRepo repository.AdminUserRepository,
	adminDevRepo repository.AdminDeviceRepository,
	instanceRepo repository.InstanceRepository,
	groupRepo repository.UserGroupRepository,
	memberRepo repository.GroupMemberRepository,
	adminLogRepo repository.AdminLogRepository,
) *AdminService {
	return &AdminService{
		db:            db,
		userRepo:      userRepo,
		adminUserRepo: adminUserRepo,
		adminDevRepo:  adminDevRepo,
		instanceRepo:  instanceRepo,
		groupRepo:     groupRepo,
		memberRepo:    memberRepo,
		adminLogRepo:  adminLogRepo,
	}
}

// ==================== Admin Login ====================

// AdminLogin authenticates an admin user and returns user info if successful.
func (s *AdminService) AdminLogin(username, password string) (*model.User, error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if user.CheckPassword(password) != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	role := model.Role(user.Role)
	if !role.IsAdmin() {
		return nil, fmt.Errorf("account is not an admin")
	}

	if user.Status != 0 {
		return nil, fmt.Errorf("account is disabled")
	}

	// Update last seen
	_ = s.userRepo.UpdateFields(user.UserUUID, map[string]interface{}{
		"last_seen": time.Now().Unix(),
	})

	return user, nil
}

// ==================== Admin Management (super_admin only) ====================

// GetAdmins returns all admin-level users.
func (s *AdminService) GetAdmins() ([]model.User, error) {
	admins, err := s.adminUserRepo.FindAdmins()
	if err != nil {
		return nil, err
	}
	for i := range admins {
		admins[i].PasswordHash = ""
	}
	return admins, nil
}

// PromoteUser sets a user's role to the given admin role.
func (s *AdminService) PromoteUser(targetUUID string, newRole model.Role, adminUUID, ip string) error {
	if !newRole.IsAdmin() || newRole == model.RoleSuperAdmin {
		return fmt.Errorf("invalid target role: must be moderator or admin")
	}

	user, err := s.userRepo.FindByUUID(targetUUID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if user.UserUUID == adminUUID {
		return fmt.Errorf("cannot change your own role")
	}

	oldRole := model.Role(user.Role)
	if err := s.userRepo.UpdateFields(targetUUID, map[string]interface{}{"role": int(newRole)}); err != nil {
		return err
	}

	s.logAction(adminUUID, "admin.promote", "user", targetUUID,
		fmt.Sprintf(`{"old_role":"%s","new_role":"%s"}`, oldRole, newRole), ip)
	return nil
}

// DemoteAdmin reverts an admin to normal user.
func (s *AdminService) DemoteAdmin(targetUUID, adminUUID, ip string) error {
	user, err := s.userRepo.FindByUUID(targetUUID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if user.UserUUID == adminUUID {
		return fmt.Errorf("cannot demote yourself")
	}
	if model.Role(user.Role) == model.RoleSuperAdmin {
		return fmt.Errorf("cannot demote super_admin")
	}

	oldRole := model.Role(user.Role)
	if err := s.userRepo.UpdateFields(targetUUID, map[string]interface{}{"role": int(model.RoleNormal)}); err != nil {
		return err
	}

	s.logAction(adminUUID, "admin.demote", "user", targetUUID,
		fmt.Sprintf(`{"old_role":"%s"}`, oldRole), ip)
	return nil
}

// UpdateAdminRole changes an admin's role level.
func (s *AdminService) UpdateAdminRole(targetUUID string, newRole model.Role, adminUUID, ip string) error {
	if newRole == model.RoleSuperAdmin {
		return fmt.Errorf("cannot assign super_admin via API")
	}

	user, err := s.userRepo.FindByUUID(targetUUID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if model.Role(user.Role) == model.RoleSuperAdmin {
		return fmt.Errorf("cannot modify super_admin role")
	}

	oldRole := model.Role(user.Role)
	if err := s.userRepo.UpdateFields(targetUUID, map[string]interface{}{"role": int(newRole)}); err != nil {
		return err
	}

	s.logAction(adminUUID, "admin.update_role", "user", targetUUID,
		fmt.Sprintf(`{"old_role":"%s","new_role":"%s"}`, oldRole, newRole), ip)
	return nil
}

// ==================== User Management ====================

// ListUsers returns a paginated list of users with filters.
func (s *AdminService) ListUsers(filter repository.UserListFilter, page, pageSize int) ([]repository.UserListItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.adminUserRepo.ListUsers(filter, page, pageSize)
}

// GetUser returns a user by UUID (sanitized for admin view).
func (s *AdminService) GetUser(userUUID string) (*model.User, error) {
	user, err := s.userRepo.FindByUUID(userUUID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	user.PasswordHash = ""
	return user, nil
}

// EditUser updates user info (nickname, description).
func (s *AdminService) EditUser(targetUUID string, nickname, description *string, adminUUID, ip string) error {
	fields := map[string]interface{}{"updated_at": time.Now().Unix()}
	if nickname != nil {
		fields["nickname"] = *nickname
	}
	if description != nil {
		fields["description"] = *description
	}
	if err := s.userRepo.UpdateFields(targetUUID, fields); err != nil {
		return err
	}
	s.logAction(adminUUID, "user.edit", "user", targetUUID, "", ip)
	return nil
}

// UpdateUserStatus enables or disables a user.
func (s *AdminService) UpdateUserStatus(targetUUID string, status int, adminUUID, ip string) error {
	user, err := s.userRepo.FindByUUID(targetUUID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if model.Role(user.Role).IsAdmin() {
		return fmt.Errorf("cannot change admin status via this endpoint")
	}
	if err := s.userRepo.UpdateFields(targetUUID, map[string]interface{}{"status": status}); err != nil {
		return err
	}
	s.logAction(adminUUID, "user.status", "user", targetUUID,
		fmt.Sprintf(`{"new_status":%d}`, status), ip)
	return nil
}

// DeleteUser deletes a user (super_admin only). Revokes all device shares.
func (s *AdminService) DeleteUser(targetUUID, adminUUID, ip string) error {
	user, err := s.userRepo.FindByUUID(targetUUID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if model.Role(user.Role).IsAdmin() {
		return fmt.Errorf("cannot delete admin users")
	}

	// Revoke device shares and delete in transaction
	return s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().Unix()
		// Revoke all shares where user is recipient
		if err := tx.Model(&model.DeviceShare{}).
			Where("shared_with_uuid = ? AND status = ?", targetUUID, "active").
			Updates(map[string]interface{}{"status": "revoked", "updated_at": now}).Error; err != nil {
			return err
		}
		// Delete user
		if err := tx.Delete(&model.User{}, user.ID).Error; err != nil {
			return err
		}
		s.logAction(adminUUID, "user.delete", "user", targetUUID,
			fmt.Sprintf(`{"username":"%s"}`, user.UserName), ip)
		return nil
	})
}

// ResetUserPassword resets a user's password.
func (s *AdminService) ResetUserPassword(targetUUID, newPassword, adminUUID, ip string) error {
	user, err := s.userRepo.FindByUUID(targetUUID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}
	if err := user.SetPassword(newPassword); err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	if err := s.userRepo.Update(user); err != nil {
		return err
	}
	s.logAction(adminUUID, "user.reset_password", "user", targetUUID, "", ip)
	return nil
}

// ==================== Device Management ====================

// ListDevices returns a paginated list of devices with filters.
func (s *AdminService) ListDevices(filter repository.DeviceListFilter, page, pageSize int) ([]repository.DeviceListItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.adminDevRepo.ListDevices(filter, page, pageSize)
}

// GetDevice returns a device by UUID.
func (s *AdminService) GetDevice(instanceUUID string) (*model.Instance, error) {
	device, err := s.instanceRepo.FindByUUID(instanceUUID)
	if err != nil {
		return nil, fmt.Errorf("device not found: %w", err)
	}
	return device, nil
}

// EditDevice updates device info.
func (s *AdminService) EditDevice(instanceUUID string, name, description, remark, sn *string, adminUUID, ip string) error {
	fields := map[string]interface{}{}
	if name != nil {
		fields["name"] = *name
	}
	if description != nil {
		fields["description"] = *description
	}
	if remark != nil {
		fields["remark"] = *remark
	}
	if sn != nil {
		fields["sn"] = *sn
	}
	if len(fields) == 0 {
		return nil
	}
	if err := s.instanceRepo.UpdateFields(instanceUUID, fields); err != nil {
		return err
	}
	s.logAction(adminUUID, "device.edit", "device", instanceUUID, "", ip)
	return nil
}

// DeleteDevice deletes a device and revokes all its shares.
func (s *AdminService) DeleteDevice(instanceUUID, adminUUID, ip string) error {
	device, err := s.instanceRepo.FindByUUID(instanceUUID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().Unix()
		// Revoke device shares
		if err := tx.Model(&model.DeviceShare{}).
			Where("instance_uuid = ? AND status = ?", instanceUUID, "active").
			Updates(map[string]interface{}{"status": "revoked", "updated_at": now}).Error; err != nil {
			return err
		}
		// Revoke group device shares
		if err := tx.Model(&model.GroupDeviceShare{}).
			Where("instance_uuid = ? AND status = ?", instanceUUID, model.GroupDeviceShareStatusActive).
			Updates(map[string]interface{}{"status": model.GroupDeviceShareStatusRevoked, "revoked_at": now}).Error; err != nil {
			return err
		}
		// Delete device
		if err := s.instanceRepo.DeleteByUUID(instanceUUID); err != nil {
			return err
		}
		s.logAction(adminUUID, "device.delete", "device", instanceUUID,
			fmt.Sprintf(`{"name":"%s"}`, device.Name), ip)
		return nil
	})
}

// TransferDevice transfers a device to a new owner.
func (s *AdminService) TransferDevice(instanceUUID, newOwnerUUID string, keepOriginalAccess bool, adminUUID, ip string) error {
	device, err := s.instanceRepo.FindByUUID(instanceUUID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}
	if _, err := s.userRepo.FindByUUID(newOwnerUUID); err != nil {
		return fmt.Errorf("new owner not found: %w", err)
	}

	oldOwner := device.OwnerUUID
	if oldOwner == newOwnerUUID {
		return fmt.Errorf("device already belongs to this user")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		// Transfer ownership
		if err := s.instanceRepo.WithTx(tx).UpdateFields(instanceUUID, map[string]interface{}{
			"owner_uuid": newOwnerUUID,
		}); err != nil {
			return err
		}

		// Optionally keep original owner with read access
		if keepOriginalAccess && oldOwner != "" {
			share := &model.DeviceShare{
				InstanceUUID:   instanceUUID,
				SharedWithUUID: oldOwner,
				SharedByUUID:   adminUUID,
				Permission:     "read",
				Status:         "active",
				CreatedAt:      time.Now().Unix(),
			}
			if err := tx.Create(share).Error; err != nil {
				return err
			}
		}

		s.logAction(adminUUID, "device.transfer", "device", instanceUUID,
			fmt.Sprintf(`{"old_owner":"%s","new_owner":"%s","keep_original":%t}`, oldOwner, newOwnerUUID, keepOriginalAccess), ip)
		return nil
	})
}

// ==================== Group Management ====================

// AdminGroupListItem is a projection for admin group list.
type AdminGroupListItem struct {
	GroupUUID   string `json:"group_uuid"`
	Name        string `json:"name"`
	OwnerUUID   string `json:"owner_uuid"`
	OwnerName   string `json:"owner_name"`
	MemberCount int64  `json:"member_count"`
	Status      int    `json:"status"`
	CreatedAt   int64  `json:"created_at"`
}

// ListGroups returns all groups with pagination.
func (s *AdminService) ListGroups(page, pageSize int) ([]AdminGroupListItem, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	if err := s.db.Model(&model.UserGroup{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var groups []model.UserGroup
	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Limit(pageSize).Offset(offset).Find(&groups).Error; err != nil {
		return nil, 0, err
	}

	result := make([]AdminGroupListItem, 0, len(groups))
	for _, g := range groups {
		item := AdminGroupListItem{
			GroupUUID: g.GroupUUID,
			Name:      g.Name,
			OwnerUUID: g.OwnerUUID,
			Status:    g.Status,
			CreatedAt: g.CreatedAt,
		}
		if owner, err := s.userRepo.FindByUUID(g.OwnerUUID); err == nil {
			item.OwnerName = owner.UserName
		}
		if count, err := s.memberRepo.CountActiveByGroup(g.GroupUUID); err == nil {
			item.MemberCount = count
		}
		result = append(result, item)
	}

	return result, total, nil
}

// GetGroup returns a group by UUID.
func (s *AdminService) GetGroup(groupUUID string) (*model.UserGroup, error) {
	group, err := s.groupRepo.FindByUUID(groupUUID)
	if err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}
	return group, nil
}

// GetGroupMembers returns members of a group.
func (s *AdminService) GetGroupMembers(groupUUID string) ([]model.GroupMember, error) {
	return s.memberRepo.FindActiveByGroupUUID(groupUUID)
}

// DissolveGroup dissolves a group and revokes all its device shares.
func (s *AdminService) DissolveGroup(groupUUID, adminUUID, ip string) error {
	group, err := s.groupRepo.FindByUUID(groupUUID)
	if err != nil {
		return fmt.Errorf("group not found: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now().Unix()
		// Revoke group device shares
		if err := tx.Model(&model.GroupDeviceShare{}).
			Where("group_uuid = ? AND status = ?", groupUUID, model.GroupDeviceShareStatusActive).
			Updates(map[string]interface{}{"status": model.GroupDeviceShareStatusRevoked, "revoked_at": now}).Error; err != nil {
			return err
		}
		// Mark group dissolved
		if err := s.groupRepo.WithTx(tx).UpdateFields(groupUUID, map[string]interface{}{
			"status": model.GroupStatusDissolved, "updated_at": now,
		}); err != nil {
			return err
		}

		s.logAction(adminUUID, "group.dissolve", "group", groupUUID,
			fmt.Sprintf(`{"name":"%s"}`, group.Name), ip)
		return nil
	})
}

// RemoveGroupMember removes a member from a group.
func (s *AdminService) RemoveGroupMember(groupUUID, targetUUID, adminUUID, ip string) error {
	member, err := s.memberRepo.FindActiveByGroupAndUser(groupUUID, targetUUID)
	if err != nil {
		return fmt.Errorf("member not found: %w", err)
	}
	if err := s.memberRepo.UpdateFields(member.ID, map[string]interface{}{
		"status": model.GroupMemberStatusKicked,
	}); err != nil {
		return err
	}
	s.logAction(adminUUID, "group.remove_member", "group", groupUUID,
		fmt.Sprintf(`{"removed_user":"%s"}`, targetUUID), ip)
	return nil
}

// ==================== System Statistics ====================

// SystemStats holds platform-wide statistics.
type SystemStats struct {
	TotalUsers       int64 `json:"total_users"`
	ActiveUsersToday int64 `json:"active_users_today"`
	TotalDevices     int64 `json:"total_devices"`
	OnlineDevices    int64 `json:"online_devices"`
	TotalGroups      int64 `json:"total_groups"`
	TotalAdmins      int64 `json:"total_admins"`
}

// GetSystemStats returns platform-wide statistics.
func (s *AdminService) GetSystemStats() (*SystemStats, error) {
	stats := &SystemStats{}
	var err error

	if stats.TotalUsers, err = s.adminUserRepo.CountAll(); err != nil {
		return nil, err
	}
	if stats.ActiveUsersToday, err = s.adminUserRepo.CountActiveToday(); err != nil {
		return nil, err
	}
	if stats.TotalDevices, err = s.adminDevRepo.CountAll(); err != nil {
		return nil, err
	}
	if stats.OnlineDevices, err = s.adminDevRepo.CountOnline(); err != nil {
		return nil, err
	}

	if err := s.db.Model(&model.UserGroup{}).Where("status = ?", model.GroupStatusActive).Count(&stats.TotalGroups).Error; err != nil {
		return nil, err
	}

	adminCount, _ := s.adminUserRepo.CountByRole(int(model.RoleAdmin))
	superAdminCount, _ := s.adminUserRepo.CountByRole(int(model.RoleSuperAdmin))
	stats.TotalAdmins = adminCount + superAdminCount

	return stats, nil
}

// ==================== Admin Logs ====================

// ListAdminLogs returns admin operation logs with pagination.
func (s *AdminService) ListAdminLogs(page, pageSize int) ([]model.AdminLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	total, err := s.adminLogRepo.Count()
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	logs, err := s.adminLogRepo.FindAll(pageSize, offset)
	return logs, total, err
}

// ==================== Bootstrap ====================

// BootstrapAdmin ensures the configured bootstrap user has super_admin role.
func (s *AdminService) BootstrapAdmin(username string) error {
	if username == "" {
		return nil
	}
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return nil // user doesn't exist yet, will be created via registration
	}
	if model.Role(user.Role) == model.RoleSuperAdmin {
		return nil // already super_admin
	}
	return s.userRepo.UpdateFields(user.UserUUID, map[string]interface{}{
		"role": int(model.RoleSuperAdmin),
	})
}

// ==================== Internal Helpers ====================

func (s *AdminService) logAction(adminUUID, action, targetType, targetUUID, detail, ip string) {
	entry := model.NewAdminLog(adminUUID, action, targetType, targetUUID, detail, ip)
	_ = s.adminLogRepo.Create(entry) // fire-and-forget
}
