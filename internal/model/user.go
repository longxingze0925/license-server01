package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"license-server/internal/pkg/crypto"
)

// User 用户模型
type User struct {
	BaseModel
	Email            string     `gorm:"type:varchar(100);uniqueIndex;not null" json:"email"`
	Password         string     `gorm:"type:varchar(255);not null" json:"-"`
	Name             string     `gorm:"type:varchar(50);not null" json:"name"`
	Phone            string     `gorm:"type:varchar(20)" json:"phone"`
	Avatar           string     `gorm:"type:varchar(500)" json:"avatar"`
	Role             UserRole   `gorm:"type:varchar(20);default:user" json:"role"`
	EmailVerified    bool       `gorm:"default:false" json:"email_verified"`
	TwoFactorEnabled bool       `gorm:"default:false" json:"two_factor_enabled"`
	TwoFactorSecret  string     `gorm:"type:varchar(100)" json:"-"`
	Status           UserStatus `gorm:"type:varchar(20);default:active" json:"status"`
	LastLoginAt      *time.Time `json:"last_login_at"`
	// 关联
	Organizations []OrganizationUser `gorm:"foreignKey:UserID" json:"organizations,omitempty"`
}

type UserRole string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusPending  UserStatus = "pending"
)

func (User) TableName() string {
	return "users"
}

// SetPassword 设置密码（加密）
// 兼容预哈希和明文密码
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashedPassword)
	return nil
}

// SetPasswordWithPreHash 设置密码（支持客户端预哈希）
// password: 客户端传来的密码
// isPreHashed: 是否已预哈希
func (u *User) SetPasswordWithPreHash(password string, isPreHashed bool) error {
	hashedPassword, err := crypto.PreparePasswordForStorage(password, u.Email, isPreHashed)
	if err != nil {
		return err
	}
	u.Password = hashedPassword
	return nil
}

// CheckPassword 验证密码（兼容旧版本）
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// CheckPasswordWithPreHash 验证密码（支持客户端预哈希）
// password: 客户端传来的密码
// isPreHashed: 是否已预哈希
func (u *User) CheckPasswordWithPreHash(password string, isPreHashed bool) bool {
	return crypto.CheckPassword(password, u.Password, u.Email, isPreHashed)
}

// BeforeCreate 创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if err := u.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}
	return nil
}
