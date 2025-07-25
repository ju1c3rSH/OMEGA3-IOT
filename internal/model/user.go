package model

import (
	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID           uint   `gorm:"primaryKey;autoIncrement"`
	UserUUID     string `json:"user_uuid"`
	UserName     string `json:"user_name" example:"dev_001"  validate:"required,min=3,max=20"`
	Type         int    `json:"type" example:"door_sensor"`
	Online       bool   `json:"online" example:"true"`
	Description  string `json:"description,omitempty" example:"自我介绍"`
	LastSeen     int64  `json:"last_seen" example:"1751193600"`
	IP           string `json:"ip" example:"<UNK>"`
	PasswordHash string `json:"password_hash" example:"<UNK>" gorm:"not null"`
	CreatedAt    int64  `json:"created_at" example:"1751193600"`
	UpdatedAt    int64  `json:"updated_at" example:"1751193600"`
	Role         int    `json:"role" example:"1"`
	Status       int    `json:"status" example:"1"`
	//Status 0:OK ; 1:Blocked
	//Role 0:Admin ;1:User
	//privatekey?
}

type RegUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
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
