package model

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

// InviteType constants
const (
	InviteTypeSearch = 0 // search by username/UUID
	InviteTypeLink   = 1 // invite link/code
)

// InviteStatus constants
const (
	InviteStatusPending  = 0
	InviteStatusAccepted = 1
	InviteStatusRejected = 2
	InviteStatusExpired  = 3
)

const inviteCodeCharset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
const inviteCodeLength = 16

// GroupInvite represents an invitation to join a group.
type GroupInvite struct {
	ID          uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	GroupUUID   string `json:"group_uuid" gorm:"type:varchar(36);not null;index"`
	InviteCode  string `json:"invite_code" gorm:"type:varchar(32);uniqueIndex"`
	InviterUUID string `json:"inviter_uuid" gorm:"type:varchar(36);not null"`
	InviteeUUID string `json:"invitee_uuid,omitempty" gorm:"type:varchar(36);index"` // empty for link invites
	Type        int    `json:"type" gorm:"default:0"`
	Status      int    `json:"status" gorm:"default:0;index"`
	ExpiresAt   int64  `json:"expires_at" gorm:"index"`
	CreatedAt   int64  `json:"created_at"`
}

// GenerateInviteCode generates a random invite code.
func GenerateInviteCode() (string, error) {
	b := make([]byte, inviteCodeLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(inviteCodeCharset))))
		if err != nil {
			return "", err
		}
		b[i] = inviteCodeCharset[n.Int64()]
	}
	return string(b), nil
}

// NewSearchInvite creates a search-based invite (targeted to a specific user).
func NewSearchInvite(groupUUID, inviterUUID, inviteeUUID string, expiresAt int64) (*GroupInvite, error) {
	code, err := GenerateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("generate invite code: %w", err)
	}
	if expiresAt == 0 {
		expiresAt = time.Now().Add(7 * 24 * time.Hour).Unix() // default 7 days
	}
	return &GroupInvite{
		GroupUUID:   groupUUID,
		InviteCode:  code,
		InviterUUID: inviterUUID,
		InviteeUUID: inviteeUUID,
		Type:        InviteTypeSearch,
		Status:      InviteStatusPending,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now().Unix(),
	}, nil
}

// NewLinkInvite creates a link-based invite (open to anyone with the code).
func NewLinkInvite(groupUUID, inviterUUID string, expiresAt int64) (*GroupInvite, error) {
	code, err := GenerateInviteCode()
	if err != nil {
		return nil, fmt.Errorf("generate invite code: %w", err)
	}
	if expiresAt == 0 {
		expiresAt = time.Now().Add(7 * 24 * time.Hour).Unix() // default 7 days
	}
	return &GroupInvite{
		GroupUUID:   groupUUID,
		InviteCode:  code,
		InviterUUID: inviterUUID,
		InviteeUUID: "",
		Type:        InviteTypeLink,
		Status:      InviteStatusPending,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now().Unix(),
	}, nil
}

