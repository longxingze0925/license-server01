package model

import "time"

// ClientSession 客户端会话（用于刷新令牌与会话撤销）
type ClientSession struct {
	BaseModel
	TenantID         string     `gorm:"type:char(36);index;not null" json:"tenant_id"`
	AppID            string     `gorm:"type:char(36);index;not null" json:"app_id"`
	CustomerID       string     `gorm:"type:char(36);index" json:"customer_id"`
	DeviceID         string     `gorm:"type:char(36);index;not null" json:"device_id"`
	MachineID        string     `gorm:"type:varchar(128);index;not null" json:"machine_id"`
	AuthMode         string     `gorm:"type:varchar(20);not null" json:"auth_mode"` // license/subscription
	RefreshTokenHash string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"-"`
	UserAgent        string     `gorm:"type:varchar(255)" json:"user_agent"`
	ClientIP         string     `gorm:"type:varchar(45)" json:"client_ip"`
	LastUsedAt       *time.Time `json:"last_used_at"`
	ExpiresAt        time.Time  `gorm:"index;not null" json:"expires_at"`
	RevokedAt        *time.Time `gorm:"index" json:"revoked_at"`
}

func (ClientSession) TableName() string {
	return "client_sessions"
}

func (s *ClientSession) IsRevoked() bool {
	return s.RevokedAt != nil
}

func (s *ClientSession) IsExpired(now time.Time) bool {
	return !s.ExpiresAt.After(now)
}
