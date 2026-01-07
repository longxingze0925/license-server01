package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tenant 租户/团队 - 资源隔离的顶层单位
type Tenant struct {
	ID        string         `gorm:"type:char(36);primaryKey" json:"id"`
	Name      string         `gorm:"type:varchar(100);not null" json:"name"`
	Slug      string         `gorm:"type:varchar(50);uniqueIndex" json:"slug"` // URL友好标识
	Logo      string         `gorm:"type:varchar(500)" json:"logo"`
	Email     string         `gorm:"type:varchar(100)" json:"email"` // 租户联系邮箱
	Phone     string         `gorm:"type:varchar(20)" json:"phone"`
	Website   string         `gorm:"type:varchar(255)" json:"website"`
	Address   string         `gorm:"type:varchar(500)" json:"address"`
	Status    TenantStatus   `gorm:"type:varchar(20);default:active" json:"status"`
	Plan      TenantPlan     `gorm:"type:varchar(20);default:free" json:"plan"` // 租户套餐
	Settings  string         `gorm:"type:text" json:"settings"`                 // JSON配置
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 配额限制
	MaxApplications int `gorm:"default:5" json:"max_applications"`   // 最大应用数
	MaxTeamMembers  int `gorm:"default:5" json:"max_team_members"`   // 最大团队成员数
	MaxCustomers    int `gorm:"default:1000" json:"max_customers"`   // 最大客户数
	MaxLicenses     int `gorm:"default:10000" json:"max_licenses"`   // 最大授权码数

	// 关联
	TeamMembers  []TeamMember  `gorm:"foreignKey:TenantID" json:"team_members,omitempty"`
	Applications []Application `gorm:"foreignKey:TenantID" json:"applications,omitempty"`
	Customers    []Customer    `gorm:"foreignKey:TenantID" json:"customers,omitempty"`
}

// TenantStatus 租户状态
type TenantStatus string

const (
	TenantStatusActive    TenantStatus = "active"    // 正常
	TenantStatusSuspended TenantStatus = "suspended" // 已暂停
	TenantStatusDeleted   TenantStatus = "deleted"   // 已删除
)

// TenantPlan 租户套餐
type TenantPlan string

const (
	TenantPlanFree       TenantPlan = "free"       // 免费版
	TenantPlanPro        TenantPlan = "pro"        // 专业版
	TenantPlanEnterprise TenantPlan = "enterprise" // 企业版
)

func (Tenant) TableName() string {
	return "tenants"
}

// BeforeCreate 创建前钩子
func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

// GetPlanLimits 获取套餐限制
func (t *Tenant) GetPlanLimits() map[string]int {
	switch t.Plan {
	case TenantPlanFree:
		return map[string]int{
			"max_applications": 2,
			"max_team_members": 2,
			"max_customers":    100,
			"max_licenses":     500,
		}
	case TenantPlanPro:
		return map[string]int{
			"max_applications": 10,
			"max_team_members": 10,
			"max_customers":    5000,
			"max_licenses":     50000,
		}
	case TenantPlanEnterprise:
		return map[string]int{
			"max_applications": -1, // 无限制
			"max_team_members": -1,
			"max_customers":    -1,
			"max_licenses":     -1,
		}
	default:
		return map[string]int{
			"max_applications": 2,
			"max_team_members": 2,
			"max_customers":    100,
			"max_licenses":     500,
		}
	}
}
