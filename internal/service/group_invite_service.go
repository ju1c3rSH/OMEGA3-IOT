package service

import (
	"OMEGA3-IOT/internal/model"
	"OMEGA3-IOT/internal/repository"
	"fmt"
	"time"
)

// GroupInviteService handles group invitation business logic.
type GroupInviteService struct {
	groupService *UserGroupService
	inviteRepo   repository.GroupInviteRepository
	memberRepo   repository.GroupMemberRepository
	policyRepo   repository.GroupPolicyRepository
	groupRepo    repository.UserGroupRepository
	userRepo     repository.UserRepository
}

// NewGroupInviteService creates a new GroupInviteService.
func NewGroupInviteService(
	groupService *UserGroupService,
	inviteRepo repository.GroupInviteRepository,
	memberRepo repository.GroupMemberRepository,
	policyRepo repository.GroupPolicyRepository,
	groupRepo repository.UserGroupRepository,
	userRepo repository.UserRepository,
) *GroupInviteService {
	return &GroupInviteService{
		groupService: groupService,
		inviteRepo:   inviteRepo,
		memberRepo:   memberRepo,
		policyRepo:   policyRepo,
		groupRepo:    groupRepo,
		userRepo:     userRepo,
	}
}

// SearchInviteRequest is the request for searching and inviting a user.
type SearchInviteRequest struct {
	InviteeUUID string `json:"invitee_uuid" binding:"required"`
	ExpiresAt   int64  `json:"expires_at"`
}

// CreateLinkInviteRequest is the request for creating an invite link.
type CreateLinkInviteRequest struct {
	ExpiresAt int64 `json:"expires_at"`
}

// CreateSearchInvite invites a specific user by UUID.
func (s *GroupInviteService) CreateSearchInvite(groupUUID, inviterUUID string, req *SearchInviteRequest) (*model.GroupInvite, error) {
	// Check policy
	policy, err := s.policyRepo.FindByGroupUUID(groupUUID)
	if err != nil {
		return nil, fmt.Errorf("group policy not found: %w", err)
	}

	// Check inviter role
	inviterMember, err := s.memberRepo.FindActiveByGroupAndUser(groupUUID, inviterUUID)
	if err != nil {
		return nil, fmt.Errorf("permission denied: not a group member")
	}

	if !policy.AllowSearchInvite {
		return nil, fmt.Errorf("search invite is not allowed for this group")
	}
	if !policy.AllowMemberInvite && inviterMember.Role < model.GroupRoleAdmin {
		return nil, fmt.Errorf("permission denied: members cannot invite")
	}

	// Check if invitee exists
	_, err = s.userRepo.FindByUUID(req.InviteeUUID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check if already a member
	exists, err := s.memberRepo.ExistsActive(groupUUID, req.InviteeUUID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("user is already a member")
	}

	// Check for existing pending invite
	_, err = s.inviteRepo.FindPendingByGroupAndInvitee(groupUUID, req.InviteeUUID)
	if err == nil {
		return nil, fmt.Errorf("user already has a pending invite")
	}

	invite, err := model.NewSearchInvite(groupUUID, inviterUUID, req.InviteeUUID, req.ExpiresAt)
	if err != nil {
		return nil, err
	}

	if err := s.inviteRepo.Create(invite); err != nil {
		return nil, err
	}
	return invite, nil
}

// CreateLinkInvite creates an invite link/code for a group.
func (s *GroupInviteService) CreateLinkInvite(groupUUID, inviterUUID string, req *CreateLinkInviteRequest) (*model.GroupInvite, error) {
	// Check policy
	policy, err := s.policyRepo.FindByGroupUUID(groupUUID)
	if err != nil {
		return nil, fmt.Errorf("group policy not found: %w", err)
	}

	// Check inviter role
	inviterMember, err := s.memberRepo.FindActiveByGroupAndUser(groupUUID, inviterUUID)
	if err != nil {
		return nil, fmt.Errorf("permission denied: not a group member")
	}

	if !policy.AllowInviteLink {
		return nil, fmt.Errorf("invite link is not allowed for this group")
	}
	if !policy.AllowMemberInvite && inviterMember.Role < model.GroupRoleAdmin {
		return nil, fmt.Errorf("permission denied: members cannot create invite links")
	}

	invite, err := model.NewLinkInvite(groupUUID, inviterUUID, req.ExpiresAt)
	if err != nil {
		return nil, err
	}

	if err := s.inviteRepo.Create(invite); err != nil {
		return nil, err
	}
	return invite, nil
}

// AcceptInvite accepts an invite by code. If approval is required, creates a pending member.
func (s *GroupInviteService) AcceptInvite(inviteCode, userUUID string) (*model.GroupMember, error) {
	invite, err := s.inviteRepo.FindByCode(inviteCode)
	if err != nil {
		return nil, fmt.Errorf("invite not found: %w", err)
	}

	// Validate invite
	if invite.Status != model.InviteStatusPending {
		return nil, fmt.Errorf("invite is no longer valid")
	}
	if invite.ExpiresAt < time.Now().Unix() {
		// Mark as expired
		_ = s.inviteRepo.UpdateFields(invite.ID, map[string]interface{}{"status": model.InviteStatusExpired})
		return nil, fmt.Errorf("invite has expired")
	}

	// For search invites, check if the invitee matches
	if invite.Type == model.InviteTypeSearch && invite.InviteeUUID != userUUID {
		return nil, fmt.Errorf("this invite is not for you")
	}

	// Add member via group service (handles approval logic)
	err = s.groupService.AddMember(invite.GroupUUID, userUUID, invite.InviterUUID)
	if err != nil {
		return nil, err
	}

	// Mark invite as accepted
	_ = s.inviteRepo.UpdateFields(invite.ID, map[string]interface{}{"status": model.InviteStatusAccepted})

	// Return the member record
	member, err := s.memberRepo.FindByGroupAndUser(invite.GroupUUID, userUUID)
	if err != nil {
		return nil, fmt.Errorf("member created but could not retrieve: %w", err)
	}
	return member, nil
}

// GetPendingInvites returns pending invites for a group.
func (s *GroupInviteService) GetPendingInvites(groupUUID, callerUUID string) ([]model.GroupInvite, error) {
	// Only admin can see all invites
	if err := s.groupService.requireGroupRole(groupUUID, callerUUID, model.GroupRoleAdmin); err != nil {
		return nil, err
	}
	return s.inviteRepo.FindByGroupUUID(groupUUID)
}
