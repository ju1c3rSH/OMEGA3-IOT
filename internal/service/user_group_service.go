package service

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// UserGroupService handles group management business logic.
type UserGroupService struct {
	db                  *gorm.DB
	groupRepo           repository.UserGroupRepository
	memberRepo          repository.GroupMemberRepository
	policyRepo          repository.GroupPolicyRepository
	inviteRepo          repository.GroupInviteRepository
	deviceShareRepo     repository.GroupDeviceShareRepository
	instanceRepo        repository.InstanceRepository
	userRepo            repository.UserRepository
}

// NewUserGroupService creates a new UserGroupService.
func NewUserGroupService(
	db *gorm.DB,
	groupRepo repository.UserGroupRepository,
	memberRepo repository.GroupMemberRepository,
	policyRepo repository.GroupPolicyRepository,
	inviteRepo repository.GroupInviteRepository,
	deviceShareRepo repository.GroupDeviceShareRepository,
	instanceRepo repository.InstanceRepository,
	userRepo repository.UserRepository,
) *UserGroupService {
	return &UserGroupService{
		db:              db,
		groupRepo:       groupRepo,
		memberRepo:      memberRepo,
		policyRepo:      policyRepo,
		inviteRepo:      inviteRepo,
		deviceShareRepo: deviceShareRepo,
		instanceRepo:    instanceRepo,
		userRepo:        userRepo,
	}
}

// ==================== Group CRUD ====================

// CreateGroup creates a new group with the creator as owner and group_admin.
func (s *UserGroupService) CreateGroup(name, description, ownerUUID string, maxMembers int) (*model.UserGroup, error) {
	group, err := model.NewUserGroup(name, description, ownerUUID, maxMembers)
	if err != nil {
		return nil, err
	}

	// Create default policy
	policy := model.NewGroupPolicy(group.GroupUUID)

	// Create owner as group member
	ownerMember := model.NewGroupMember(group.GroupUUID, ownerUUID, model.GroupRoleOwner, "")

	err = s.db.Transaction(func(tx *gorm.DB) error {
		txGroupRepo := s.groupRepo.WithTx(tx)
		txMemberRepo := s.memberRepo.WithTx(tx)
		txPolicyRepo := s.policyRepo.WithTx(tx)

		if err := txGroupRepo.Create(group); err != nil {
			return fmt.Errorf("create group: %w", err)
		}
		if err := txPolicyRepo.Create(policy); err != nil {
			return fmt.Errorf("create group policy: %w", err)
		}
		if err := txMemberRepo.Create(ownerMember); err != nil {
			return fmt.Errorf("create owner member: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return group, nil
}

// GetGroup returns a group by UUID.
func (s *UserGroupService) GetGroup(groupUUID string) (*model.UserGroup, error) {
	group, err := s.groupRepo.FindByUUID(groupUUID)
	if err != nil {
		return nil, fmt.Errorf("group not found: %w", err)
	}
	if group.Status == model.GroupStatusDissolved {
		return nil, fmt.Errorf("group is dissolved")
	}
	return group, nil
}

// GetUserGroups returns all active groups a user is a member of.
func (s *UserGroupService) GetUserGroups(userUUID string) ([]model.UserGroup, error) {
	members, err := s.memberRepo.FindActiveByUserUUID(userUUID)
	if err != nil {
		return nil, err
	}

	var groups []model.UserGroup
	for _, m := range members {
		group, err := s.groupRepo.FindByUUID(m.GroupUUID)
		if err != nil {
			continue // skip if group not found
		}
		if group.Status == model.GroupStatusActive {
			groups = append(groups, *group)
		}
	}
	return groups, nil
}

// UpdateGroup updates group info (name, description, max_members). Caller must be group_admin.
func (s *UserGroupService) UpdateGroup(groupUUID, callerUUID string, req *model.UpdateGroupRequest) error {
	if err := s.requireGroupRole(groupUUID, callerUUID, model.GroupRoleAdmin); err != nil {
		return err
	}

	fields := map[string]interface{}{"updated_at": time.Now().Unix()}
	if req.Name != nil {
		fields["name"] = *req.Name
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.MaxMembers != nil {
		fields["max_members"] = *req.MaxMembers
	}

	return s.groupRepo.UpdateFields(groupUUID, fields)
}

// DissolveGroup dissolves a group. Only group_owner can do this.
func (s *UserGroupService) DissolveGroup(groupUUID, callerUUID string) error {
	group, err := s.groupRepo.FindByUUID(groupUUID)
	if err != nil {
		return fmt.Errorf("group not found: %w", err)
	}
	if group.OwnerUUID != callerUUID {
		return fmt.Errorf("permission denied")
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		txGroupRepo := s.groupRepo.WithTx(tx)
		txDeviceShareRepo := s.deviceShareRepo.WithTx(tx)

		// Revoke all device shares for this group
		if err := txDeviceShareRepo.RevokeAllByGroup(groupUUID); err != nil {
			return fmt.Errorf("revoke device shares: %w", err)
		}
		// Mark group as dissolved
		return txGroupRepo.UpdateFields(groupUUID, map[string]interface{}{
			"status":     model.GroupStatusDissolved,
			"updated_at": time.Now().Unix(),
		})
	})
}

// ==================== Member Management ====================

// GetMembers returns all active members of a group.
func (s *UserGroupService) GetMembers(groupUUID, callerUUID string) ([]model.GroupMember, error) {
	if err := s.requireMembership(groupUUID, callerUUID); err != nil {
		return nil, err
	}
	return s.memberRepo.FindActiveByGroupUUID(groupUUID)
}

// AddMember adds a user to a group. Handles approval logic.
func (s *UserGroupService) AddMember(groupUUID, userUUID, inviterUUID string) error {
	// Check if group exists and is active
	group, err := s.GetGroup(groupUUID)
	if err != nil {
		return err
	}

	// Check if user is already a member
	exists, err := s.memberRepo.ExistsActive(groupUUID, userUUID)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("user is already a member")
	}

	// Check member limit
	if group.MaxMembers > 0 {
		count, err := s.memberRepo.CountActiveByGroup(groupUUID)
		if err != nil {
			return err
		}
		if count >= int64(group.MaxMembers) {
			return fmt.Errorf("group member limit reached")
		}
	}

	// Check policy for approval requirement
	policy, err := s.policyRepo.FindByGroupUUID(groupUUID)
	if err != nil {
		return err
	}

	status := model.GroupMemberStatusActive
	if policy.RequireApproval {
		status = model.GroupMemberStatusPending
	}

	member := model.NewGroupMember(groupUUID, userUUID, model.GroupRoleMember, inviterUUID)
	member.Status = status

	return s.memberRepo.Create(member)
}

// ApproveMember approves a pending member. Caller must be group_admin.
func (s *UserGroupService) ApproveMember(groupUUID, targetUUID, callerUUID string) error {
	if err := s.requireGroupRole(groupUUID, callerUUID, model.GroupRoleAdmin); err != nil {
		return err
	}

	member, err := s.memberRepo.FindByGroupAndUser(groupUUID, targetUUID)
	if err != nil {
		return fmt.Errorf("member not found: %w", err)
	}
	if member.Status != model.GroupMemberStatusPending {
		return fmt.Errorf("member is not pending approval")
	}

	// Check member limit
	group, err := s.groupRepo.FindByUUID(groupUUID)
	if err != nil {
		return err
	}
	if group.MaxMembers > 0 {
		count, err := s.memberRepo.CountActiveByGroup(groupUUID)
		if err != nil {
			return err
		}
		if count >= int64(group.MaxMembers) {
			return fmt.Errorf("group member limit reached")
		}
	}

	return s.memberRepo.UpdateFields(member.ID, map[string]interface{}{
		"status": model.GroupMemberStatusActive,
	})
}

// RejectMember rejects a pending member. Caller must be group_admin.
func (s *UserGroupService) RejectMember(groupUUID, targetUUID, callerUUID string) error {
	if err := s.requireGroupRole(groupUUID, callerUUID, model.GroupRoleAdmin); err != nil {
		return err
	}

	member, err := s.memberRepo.FindByGroupAndUser(groupUUID, targetUUID)
	if err != nil {
		return fmt.Errorf("member not found: %w", err)
	}
	if member.Status != model.GroupMemberStatusPending {
		return fmt.Errorf("member is not pending approval")
	}

	return s.memberRepo.Delete(member.ID)
}

// RemoveMember kicks a member from a group. Caller must be group_admin.
func (s *UserGroupService) RemoveMember(groupUUID, targetUUID, callerUUID string) error {
	if err := s.requireGroupRole(groupUUID, callerUUID, model.GroupRoleAdmin); err != nil {
		return err
	}

	// Cannot kick the owner
	group, err := s.groupRepo.FindByUUID(groupUUID)
	if err != nil {
		return err
	}
	if group.OwnerUUID == targetUUID {
		return fmt.Errorf("cannot remove the group owner")
	}

	member, err := s.memberRepo.FindActiveByGroupAndUser(groupUUID, targetUUID)
	if err != nil {
		return fmt.Errorf("member not found: %w", err)
	}

	return s.memberRepo.UpdateFields(member.ID, map[string]interface{}{
		"status": model.GroupMemberStatusKicked,
	})
}

// LeaveGroup allows a member to leave a group. Auto-revokes device shares.
func (s *UserGroupService) LeaveGroup(groupUUID, userUUID string) error {
	group, err := s.groupRepo.FindByUUID(groupUUID)
	if err != nil {
		return fmt.Errorf("group not found: %w", err)
	}
	if group.OwnerUUID == userUUID {
		return fmt.Errorf("owner cannot leave the group")
	}

	member, err := s.memberRepo.FindActiveByGroupAndUser(groupUUID, userUUID)
	if err != nil {
		return fmt.Errorf("not a member of this group: %w", err)
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		txMemberRepo := s.memberRepo.WithTx(tx)
		txDeviceShareRepo := s.deviceShareRepo.WithTx(tx)

		// Mark member as left
		if err := txMemberRepo.UpdateFields(member.ID, map[string]interface{}{
			"status": model.GroupMemberStatusLeft,
		}); err != nil {
			return err
		}

		// Revoke all device shares from this user to this group
		shares, err := txDeviceShareRepo.FindActiveByGroup(groupUUID)
		if err != nil {
			return err
		}
		for _, share := range shares {
			if share.OwnerUUID == userUUID {
				if err := txDeviceShareRepo.Revoke(share.ID); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// UpdateMemberRole changes a member's role. Only group_owner can do this.
func (s *UserGroupService) UpdateMemberRole(groupUUID, targetUUID, callerUUID string, newRole int) error {
	group, err := s.groupRepo.FindByUUID(groupUUID)
	if err != nil {
		return fmt.Errorf("group not found: %w", err)
	}
	if group.OwnerUUID != callerUUID {
		return fmt.Errorf("permission denied")
	}
	if targetUUID == callerUUID {
		return fmt.Errorf("cannot change your own role")
	}
	if newRole < model.GroupRoleMember || newRole > model.GroupRoleAdmin {
		return fmt.Errorf("invalid role")
	}

	member, err := s.memberRepo.FindActiveByGroupAndUser(groupUUID, targetUUID)
	if err != nil {
		return fmt.Errorf("member not found: %w", err)
	}

	return s.memberRepo.UpdateFields(member.ID, map[string]interface{}{
		"role": newRole,
	})
}

// ==================== Device Management ====================

// GetGroupDevices returns devices visible to a member based on group policy.
func (s *UserGroupService) GetGroupDevices(groupUUID, callerUUID string) ([]model.GroupDeviceShare, error) {
	if err := s.requireMembership(groupUUID, callerUUID); err != nil {
		return nil, err
	}

	policy, err := s.policyRepo.FindByGroupUUID(groupUUID)
	if err != nil {
		return nil, err
	}

	role, _ := s.getMemberRole(groupUUID, callerUUID)

	switch policy.DeviceVisibility {
	case model.DeviceVisibilityAdminAll:
		// Admin sees all, members see shared only
		if role >= model.GroupRoleAdmin {
			return s.getAllGroupMemberDevices(groupUUID)
		}
		return s.deviceShareRepo.FindActiveByGroup(groupUUID)

	case model.DeviceVisibilitySharedOnly:
		return s.deviceShareRepo.FindActiveByGroup(groupUUID)

	case model.DeviceVisibilitySelective:
		return s.deviceShareRepo.FindActiveByGroup(groupUUID)

	default:
		return s.deviceShareRepo.FindActiveByGroup(groupUUID)
	}
}

// ShareDeviceToGroup shares a device to a group. Caller must own the device.
func (s *UserGroupService) ShareDeviceToGroup(groupUUID, instanceUUID, callerUUID, permission string) error {
	if err := s.requireMembership(groupUUID, callerUUID); err != nil {
		return err
	}

	// Verify device ownership
	instance, err := s.instanceRepo.FindByUUID(instanceUUID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}
	if instance.OwnerUUID != callerUUID {
		return fmt.Errorf("permission denied: you do not own this device")
	}

	// Check if already shared
	existing, err := s.deviceShareRepo.FindActiveByGroupAndDevice(groupUUID, instanceUUID)
	if err == nil && existing != nil {
		return fmt.Errorf("device is already shared to this group")
	}

	share := model.NewGroupDeviceShare(groupUUID, instanceUUID, callerUUID, permission, callerUUID)
	return s.deviceShareRepo.Create(share)
}

// RevokeGroupDeviceShare revokes a device share from a group.
func (s *UserGroupService) RevokeGroupDeviceShare(groupUUID, instanceUUID, callerUUID string) error {
	share, err := s.deviceShareRepo.FindActiveByGroupAndDevice(groupUUID, instanceUUID)
	if err != nil {
		return fmt.Errorf("share not found: %w", err)
	}

	// Only the sharer or group_admin can revoke
	if share.SharedBy != callerUUID {
		if err := s.requireGroupRole(groupUUID, callerUUID, model.GroupRoleAdmin); err != nil {
			return fmt.Errorf("permission denied")
		}
	}

	return s.deviceShareRepo.Revoke(share.ID)
}

// SendGroupDeviceAction sends an action to a device shared in a group.
func (s *UserGroupService) SendGroupDeviceAction(groupUUID, instanceUUID, callerUUID, command string, params map[string]interface{}) error {
	// Check membership
	if err := s.requireMembership(groupUUID, callerUUID); err != nil {
		return err
	}

	// Check policy for device access level
	policy, err := s.policyRepo.FindByGroupUUID(groupUUID)
	if err != nil {
		return err
	}

	role, _ := s.getMemberRole(groupUUID, callerUUID)

	// Check if user can access this device
	share, err := s.deviceShareRepo.FindActiveByGroupAndDevice(groupUUID, instanceUUID)
	if err != nil {
		return fmt.Errorf("device not shared to this group: %w", err)
	}

	// Owner of device can always send actions
	if share.OwnerUUID == callerUUID {
		return nil // let the existing device action handler process it
	}

	// Check admin access level
	if role >= model.GroupRoleAdmin && policy.AdminDeviceAccess == model.AdminDeviceAccessFull {
		// Check permission level
		if share.Permission == "write" || share.Permission == "read_write" {
			return nil // allowed
		}
	}

	return fmt.Errorf("permission denied: insufficient device access")
}

// ==================== Policy Management ====================

// GetPolicy returns the group policy.
func (s *UserGroupService) GetPolicy(groupUUID, callerUUID string) (*model.GroupPolicy, error) {
	if err := s.requireMembership(groupUUID, callerUUID); err != nil {
		return nil, err
	}
	return s.policyRepo.FindByGroupUUID(groupUUID)
}

// UpdatePolicy updates the group policy. Caller must be group_admin.
func (s *UserGroupService) UpdatePolicy(groupUUID, callerUUID string, req *model.UpdatePolicyRequest) error {
	if err := s.requireGroupRole(groupUUID, callerUUID, model.GroupRoleAdmin); err != nil {
		return err
	}

	fields := map[string]interface{}{"updated_at": time.Now().Unix()}
	if req.DeviceVisibility != nil {
		fields["device_visibility"] = *req.DeviceVisibility
	}
	if req.AdminDeviceAccess != nil {
		fields["admin_device_access"] = *req.AdminDeviceAccess
	}
	if req.AllowSearchInvite != nil {
		fields["allow_search_invite"] = *req.AllowSearchInvite
	}
	if req.AllowInviteLink != nil {
		fields["allow_invite_link"] = *req.AllowInviteLink
	}
	if req.AllowMemberInvite != nil {
		fields["allow_member_invite"] = *req.AllowMemberInvite
	}
	if req.RequireApproval != nil {
		fields["require_approval"] = *req.RequireApproval
	}
	if req.ApprovalMode != nil {
		fields["approval_mode"] = *req.ApprovalMode
	}

	return s.policyRepo.UpdateFields(groupUUID, fields)
}

// ==================== Helpers ====================

// CheckMembership checks if a user is an active member of a group (public wrapper).
func (s *UserGroupService) CheckMembership(groupUUID, userUUID string) error {
	return s.requireMembership(groupUUID, userUUID)
}

// requireMembership checks if a user is an active member of a group.
func (s *UserGroupService) requireMembership(groupUUID, userUUID string) error {
	exists, err := s.memberRepo.ExistsActive(groupUUID, userUUID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("permission denied: not a group member")
	}
	return nil
}

// requireGroupRole checks if a user has at least the specified role in a group.
func (s *UserGroupService) requireGroupRole(groupUUID, userUUID string, minRole int) error {
	member, err := s.memberRepo.FindActiveByGroupAndUser(groupUUID, userUUID)
	if err != nil {
		return fmt.Errorf("permission denied: not a group member")
	}
	if member.Role < minRole {
		return fmt.Errorf("permission denied: insufficient role")
	}
	return nil
}

// getMemberRole returns the role of a user in a group. Returns -1 if not a member.
func (s *UserGroupService) getMemberRole(groupUUID, userUUID string) (int, error) {
	member, err := s.memberRepo.FindActiveByGroupAndUser(groupUUID, userUUID)
	if err != nil {
		return -1, err
	}
	return member.Role, nil
}

// getAllGroupMemberDevices returns all devices owned by active members of a group.
func (s *UserGroupService) getAllGroupMemberDevices(groupUUID string) ([]model.GroupDeviceShare, error) {
	members, err := s.memberRepo.FindActiveByGroupUUID(groupUUID)
	if err != nil {
		return nil, err
	}

	var allShares []model.GroupDeviceShare
	for _, m := range members {
		devices, err := s.instanceRepo.FindByOwnerUUID(m.UserUUID)
		if err != nil {
			continue
		}
		for _, d := range devices {
			allShares = append(allShares, model.GroupDeviceShare{
				GroupUUID:    groupUUID,
				InstanceUUID: d.InstanceUUID,
				OwnerUUID:    d.OwnerUUID,
				Permission:   "read_write",
				Status:       model.GroupDeviceShareStatusActive,
			})
		}
	}
	return allShares, nil
}
