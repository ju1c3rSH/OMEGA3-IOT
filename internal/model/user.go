package model

import (
	"github.com/go-playground/validator/v10"
)

type User struct {
	ID       uint   `gorm:"primaryKey;autoIncrement"`
	UserUUID string `json:"user_uuid" gorm:"uniqueIndex;not null;type:char(36)"`
	UserName string `json:"user_name" gorm:"uniqueIndex;not null;size:50" validate:"required,min=3,max=20"`
	Nickname string `json:"nickname" gorm:"size:50"`
	Avatar   string `json:"avatar" gorm:"size:255"`
	Type     int    `json:"type" example:"1"`
	Online   bool   `json:"online"`
	Description string `json:"description,omitempty"`
	LastSeen int64  `json:"last_seen"`
	IP       string `json:"ip" gorm:"size:45"`
	// PasswordHash 存储 DH 承诺值的十六进制编码（A = g^SHA256(password) mod p）
	// json:"-" 防止序列化泄漏到 API 响应中
	PasswordHash string `json:"-" gorm:"not null"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
	Role         int    `json:"role" gorm:"index:idx_role_status"`
	Status       int    `json:"status" gorm:"index:idx_role_status"`
}

type UserExtra struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	UserUUID     string `json:"user_uuid" gorm:"uniqueIndex;not null;type:char(36)"`
	ServiceToken string `json:"service_token"`
}
// RegUser 注册请求（客户端发送 DH 承诺值）
type RegUser struct {
	Username   string `json:"username" binding:"required"`
	Commitment string `json:"commitment" binding:"required"` // DH 承诺值 hex 编码
}

type BindDeviceByRegCode struct {
	RegCode      string `json:"reg_code"`
	DeviceNick   string `json:"device_nick"`
	DeviceRemark string `json:"device_remark"`
}

// LoginUser 登录请求（客户端发送 DH 证明值）
type LoginUser struct {
	Username string `json:"username" binding:"required"`
	Proof    string `json:"proof" binding:"required"` // DH 证明值 hex 编码
}

// ChallengeRequest 挑战请求（获取 Nonce）
type ChallengeRequest struct {
	Username string `json:"username" binding:"required"`
}

// ChallengeResponse 挑战响应（返回 Nonce）
type ChallengeResponse struct {
	Nonce string `json:"nonce"` // 随机数 hex 编码
	P     string `json:"p"`    // DH 素数 hex 编码
	G     string `json:"g"`    // DH 生成元 hex 编码
}

type UpdateProfileRequest struct {
	Nickname    *string `json:"nickname,omitempty"`
	Description *string `json:"description,omitempty"`
}

func (u *User) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}
