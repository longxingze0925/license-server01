package model

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"license-server/internal/pkg/crypto"
)

// ClientUser 客户端用户模型（软件端用户，与后台管理用户分离）
type ClientUser struct {
	ID           string         `gorm:"type:char(36);primaryKey" json:"id"`
	Email        string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password     string         `gorm:"type:varchar(255);not null" json:"-"`
	Name         string         `gorm:"type:varchar(100)" json:"name"`
	Phone        string         `gorm:"type:varchar(20)" json:"phone"`
	Avatar       string         `gorm:"type:varchar(500)" json:"avatar"`
	Status       string         `gorm:"type:varchar(20);default:active" json:"status"` // active, disabled, pending
	Remark       string         `gorm:"type:text" json:"remark"`
	LastLoginAt  *time.Time     `json:"last_login_at"`
	LastLoginIP  string         `gorm:"type:varchar(50)" json:"last_login_ip"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Subscriptions []Subscription `gorm:"foreignKey:ClientUserID" json:"subscriptions,omitempty"`
	// 注意：设备通过 Subscription 间接关联，不直接关联
	SyncData      []ClientSyncData `gorm:"foreignKey:ClientUserID" json:"sync_data,omitempty"`
}

// BeforeCreate 创建前钩子
func (u *ClientUser) BeforeCreate(tx *gorm.DB) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	return nil
}

// SetPassword 设置密码（兼容旧版本）
func (u *ClientUser) SetPassword(password string) error {
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
func (u *ClientUser) SetPasswordWithPreHash(password string, isPreHashed bool) error {
	hashedPassword, err := crypto.PreparePasswordForStorage(password, u.Email, isPreHashed)
	if err != nil {
		return err
	}
	u.Password = hashedPassword
	return nil
}

// CheckPassword 验证密码（兼容旧版本）
func (u *ClientUser) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// CheckPasswordWithPreHash 验证密码（支持客户端预哈希）
// password: 客户端传来的密码
// isPreHashed: 是否已预哈希
func (u *ClientUser) CheckPasswordWithPreHash(password string, isPreHashed bool) bool {
	return crypto.CheckPassword(password, u.Password, u.Email, isPreHashed)
}

// ClientUserStatus 客户端用户状态
const (
	ClientUserStatusActive   = "active"
	ClientUserStatusDisabled = "disabled"
	ClientUserStatusPending  = "pending"
)
