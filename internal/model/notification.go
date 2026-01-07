package model

// Notification 通知模型
type Notification struct {
	BaseModel
	UserID  string `gorm:"type:varchar(36);index" json:"user_id"`
	OrgID   string `gorm:"type:varchar(36);index" json:"org_id"`
	Type    string `gorm:"type:varchar(50);not null" json:"type"`
	Title   string `gorm:"type:varchar(200);not null" json:"title"`
	Content string `gorm:"type:text" json:"content"`
	IsRead  bool   `gorm:"default:false" json:"is_read"`
}

func (Notification) TableName() string {
	return "notifications"
}

// Webhook 配置
type Webhook struct {
	BaseModel
	OrgID           string `gorm:"type:varchar(36);not null" json:"org_id"`
	URL             string `gorm:"type:varchar(500);not null" json:"url"`
	Secret          string `gorm:"type:varchar(100);not null" json:"-"`
	Events          string `gorm:"type:json;not null" json:"events"`
	Status          string `gorm:"type:varchar(20);default:active" json:"status"`
	LastTriggeredAt *string `json:"last_triggered_at"`
}

func (Webhook) TableName() string {
	return "webhooks"
}

// WebhookLog Webhook 日志
type WebhookLog struct {
	BaseModel
	WebhookID      string `gorm:"type:varchar(36);not null" json:"webhook_id"`
	EventType      string `gorm:"type:varchar(50);not null" json:"event_type"`
	Payload        string `gorm:"type:json;not null" json:"payload"`
	ResponseStatus int    `json:"response_status"`
	ResponseBody   string `gorm:"type:text" json:"response_body"`
	Success        bool   `json:"success"`
}

func (WebhookLog) TableName() string {
	return "webhook_logs"
}

// Setting 系统配置
type Setting struct {
	BaseModel
	Key         string `gorm:"type:varchar(100);uniqueIndex;not null" json:"key"`
	Value       string `gorm:"type:text" json:"value"`
	Description string `gorm:"type:varchar(255)" json:"description"`
}

func (Setting) TableName() string {
	return "settings"
}
