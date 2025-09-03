package model

import (
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	UserUUID     string `json:"user_uuid" gorm:"uniqueIndex;not null;type:char(36)"`
	UserName     string `json:"user_name" gorm:"uniqueIndex;not null;size:50" validate:"required,min=3,max=20"`
	Type         int    `json:"type" example:"1"`
	Online       bool   `json:"online"`
	Description  string `json:"description,omitempty"`
	LastSeen     int64  `json:"last_seen"`
	IP           string `json:"ip" gorm:"size:45"`
	PasswordHash string `json:"password_hash" gorm:"not null"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
	Role         int    `json:"role" gorm:"index:idx_role_status"`
	Status       int    `json:"status" gorm:"index:idx_role_status"`
}
type RegUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type BindDeviceByRegCode struct {
	RegCode      string `json:"reg_code"`
	DeviceNick   string `json:"device_nick"`
	DeviceRemark string `json:"device_remark"`
}

type LoginUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (c *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	c.PasswordHash = string(hashedPassword)
	return nil

}

func (c *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(c.PasswordHash), []byte(password))
}

func (u *User) Validate() error {
	validate := validator.New()
	return validate.Struct(u)
}
