package model

import "time"

type PublishTaskStatus string

const (
	PublishTaskStatusPending PublishTaskStatus = "pending"
	PublishTaskStatusRunning PublishTaskStatus = "running"
	PublishTaskStatusSuccess PublishTaskStatus = "success"
	PublishTaskStatusFailed  PublishTaskStatus = "failed"
)

type PublishTaskResource string

const (
	PublishTaskResourceHotUpdate PublishTaskResource = "hotupdate"
	PublishTaskResourceRelease   PublishTaskResource = "release"
)

type PublishTaskAction string

const (
	PublishTaskActionPublish   PublishTaskAction = "publish"
	PublishTaskActionDeprecate PublishTaskAction = "deprecate"
	PublishTaskActionRollback  PublishTaskAction = "rollback"
)

// PublishTask 发布动作异步任务
type PublishTask struct {
	BaseModel
	TenantID       string              `gorm:"type:char(36);index;not null" json:"tenant_id"`
	AppID          string              `gorm:"type:varchar(36);index;not null" json:"app_id"`
	Resource       PublishTaskResource `gorm:"type:varchar(20);index;not null" json:"resource"`
	ResourceID     string              `gorm:"type:varchar(36);index;not null" json:"resource_id"`
	Action         PublishTaskAction   `gorm:"type:varchar(20);index;not null" json:"action"`
	Status         PublishTaskStatus   `gorm:"type:varchar(20);index;not null;default:pending" json:"status"`
	Progress       int                 `gorm:"default:0" json:"progress"`
	Message        string              `gorm:"type:varchar(255)" json:"message"`
	ErrorMessage   string              `gorm:"type:text" json:"error_message"`
	ResultJSON     string              `gorm:"type:text" json:"result_json"`
	RequestedBy    string              `gorm:"type:varchar(36);index" json:"requested_by"`
	RequestedEmail string              `gorm:"type:varchar(100)" json:"requested_email"`
	StartedAt      *time.Time          `json:"started_at"`
	FinishedAt     *time.Time          `json:"finished_at"`
}

func (PublishTask) TableName() string {
	return "publish_tasks"
}
