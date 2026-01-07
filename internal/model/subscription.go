package model

import "time"

// Subscription 订阅模型（账号密码模式）
type Subscription struct {
	BaseModel
	TenantID     string             `gorm:"type:char(36);index;not null" json:"tenant_id"`      // 所属租户
	CustomerID   string             `gorm:"type:char(36);index;not null" json:"customer_id"`    // 关联客户
	ClientUserID *string            `gorm:"type:char(36);index" json:"client_user_id"`          // 关联客户端用户（可选）
	AppID        string             `gorm:"type:varchar(36);not null;index" json:"app_id"`
	PlanType     PlanType           `gorm:"type:varchar(20);default:free" json:"plan_type"`
	MaxDevices   int                `gorm:"default:1" json:"max_devices"`
	Features     string             `gorm:"type:json" json:"features"`
	Status       SubscriptionStatus `gorm:"type:varchar(20);default:active" json:"status"`
	StartAt      *time.Time         `json:"start_at"`
	ExpireAt     *time.Time         `json:"expire_at"`
	CancelledAt  *time.Time         `json:"cancelled_at"`
	AutoRenew    bool               `gorm:"default:false" json:"auto_renew"`
	Notes        string             `gorm:"type:text" json:"notes"`
	// 关联
	Tenant      *Tenant      `gorm:"foreignKey:TenantID" json:"tenant,omitempty"`
	Customer    *Customer    `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Application *Application `gorm:"foreignKey:AppID" json:"application,omitempty"`
	Devices     []Device     `gorm:"foreignKey:SubscriptionID" json:"devices,omitempty"`
}

// PlanType 套餐类型
type PlanType string

const (
	PlanTypeFree       PlanType = "free"       // 免费版
	PlanTypeBasic      PlanType = "basic"      // 基础版
	PlanTypePro        PlanType = "pro"        // 专业版
	PlanTypeEnterprise PlanType = "enterprise" // 企业版
)

// SubscriptionStatus 订阅状态
type SubscriptionStatus string

const (
	SubscriptionStatusActive    SubscriptionStatus = "active"    // 有效
	SubscriptionStatusExpired   SubscriptionStatus = "expired"   // 已过期
	SubscriptionStatusCancelled SubscriptionStatus = "cancelled" // 已取消
	SubscriptionStatusSuspended SubscriptionStatus = "suspended" // 已暂停
)

func (Subscription) TableName() string {
	return "subscriptions"
}

// IsValid 检查订阅是否有效
func (s *Subscription) IsValid() bool {
	if s.Status != SubscriptionStatusActive {
		return false
	}
	if s.ExpireAt != nil && time.Now().After(*s.ExpireAt) {
		return false
	}
	return true
}

// RemainingDays 剩余天数
func (s *Subscription) RemainingDays() int {
	if s.ExpireAt == nil {
		return -1 // 永久
	}
	remaining := time.Until(*s.ExpireAt)
	if remaining < 0 {
		return 0
	}
	return int(remaining.Hours() / 24)
}
