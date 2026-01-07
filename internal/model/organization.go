package model

import (
	"time"
)

// Organization 组织模型
type Organization struct {
	BaseModel
	Name        string     `gorm:"type:varchar(100);not null" json:"name"`
	OwnerID     string     `gorm:"type:varchar(36);not null" json:"owner_id"`
	Logo        string     `gorm:"type:varchar(500)" json:"logo"`
	Description string     `gorm:"type:text" json:"description"`
	Status      OrgStatus  `gorm:"type:varchar(20);default:active" json:"status"`
	// 关联
	Owner    *User              `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members  []OrganizationUser `gorm:"foreignKey:OrgID" json:"members,omitempty"`
	Licenses []License          `gorm:"foreignKey:OrgID" json:"licenses,omitempty"`
}

type OrgStatus string

const (
	OrgStatusActive   OrgStatus = "active"
	OrgStatusDisabled OrgStatus = "disabled"
)

func (Organization) TableName() string {
	return "organizations"
}

// OrganizationUser 组织成员关联表
type OrganizationUser struct {
	BaseModel
	OrgID     string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_org_user" json:"org_id"`
	UserID    string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_org_user" json:"user_id"`
	Role      OrgRole   `gorm:"type:varchar(20);default:member" json:"role"`
	Status    string    `gorm:"type:varchar(20);default:active" json:"status"`
	InvitedBy string    `gorm:"type:varchar(36)" json:"invited_by"`
	JoinedAt  time.Time `gorm:"autoCreateTime" json:"joined_at"`
	// 关联
	Organization *Organization `gorm:"foreignKey:OrgID" json:"organization,omitempty"`
	User         *User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type OrgRole string

const (
	OrgRoleOwner  OrgRole = "owner"
	OrgRoleAdmin  OrgRole = "admin"
	OrgRoleMember OrgRole = "member"
)

func (OrganizationUser) TableName() string {
	return "organization_users"
}

// Invitation 邀请表
type Invitation struct {
	BaseModel
	OrgID     string           `gorm:"type:varchar(36);not null" json:"org_id"`
	Email     string           `gorm:"type:varchar(100);not null" json:"email"`
	Role      OrgRole          `gorm:"type:varchar(20);default:member" json:"role"`
	Token     string           `gorm:"type:varchar(100);uniqueIndex;not null" json:"token"`
	InvitedBy string           `gorm:"type:varchar(36);not null" json:"invited_by"`
	Status    InvitationStatus `gorm:"type:varchar(20);default:pending" json:"status"`
	ExpireAt  time.Time        `gorm:"not null" json:"expire_at"`
	// 关联
	Organization *Organization `gorm:"foreignKey:OrgID" json:"organization,omitempty"`
}

type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusExpired  InvitationStatus = "expired"
)

func (Invitation) TableName() string {
	return "invitations"
}
