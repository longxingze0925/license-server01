package service

import (
	"license-server/internal/model"
	"log"
	"time"
)

// SchedulerService 定时任务服务
type SchedulerService struct {
	emailService   *EmailService
	webhookService *WebhookService
}

// NewSchedulerService 创建定时任务服务
func NewSchedulerService() *SchedulerService {
	return &SchedulerService{
		emailService:   NewEmailService(),
		webhookService: NewWebhookService(),
	}
}

// Start 启动定时任务
func (s *SchedulerService) Start() {
	// 每天凌晨 2 点检查到期提醒
	go s.runDaily(2, 0, s.CheckExpirationReminders)

	// 每小时检查异常
	go s.runHourly(s.CheckAnomalies)

	// 每天凌晨 3 点清理过期数据
	go s.runDaily(3, 0, s.CleanupExpiredData)

	log.Println("定时任务服务已启动")
}

// runDaily 每天定时执行
func (s *SchedulerService) runDaily(hour, minute int, task func()) {
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
		if next.Before(now) {
			next = next.Add(24 * time.Hour)
		}
		time.Sleep(next.Sub(now))
		task()
	}
}

// runHourly 每小时执行
func (s *SchedulerService) runHourly(task func()) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		task()
	}
}

// CheckExpirationReminders 检查到期提醒
func (s *SchedulerService) CheckExpirationReminders() {
	log.Println("开始检查授权到期提醒...")

	// 提醒天数：7天、3天、1天
	reminderDays := []int{7, 3, 1}

	for _, days := range reminderDays {
		s.sendRemindersForDays(days)
	}

	log.Println("授权到期提醒检查完成")
}

// sendRemindersForDays 发送指定天数的提醒
func (s *SchedulerService) sendRemindersForDays(days int) {
	// 计算目标日期范围
	now := time.Now()
	targetStart := now.AddDate(0, 0, days)
	targetEnd := targetStart.Add(24 * time.Hour)

	// 查找即将到期的授权
	var licenses []model.License
	model.DB.Preload("Application").
		Where("status = ? AND expire_at >= ? AND expire_at < ?",
			model.LicenseStatusActive, targetStart, targetEnd).
		Find(&licenses)

	for _, license := range licenses {
		s.sendLicenseExpirationReminder(&license, days)
	}

	// 查找即将到期的订阅
	var subscriptions []model.Subscription
	model.DB.Preload("Customer").Preload("Application").
		Where("status = ? AND expire_at >= ? AND expire_at < ?",
			model.SubscriptionStatusActive, targetStart, targetEnd).
		Find(&subscriptions)

	for _, sub := range subscriptions {
		s.sendSubscriptionExpirationReminder(&sub, days)
	}
}

// sendLicenseExpirationReminder 发送授权到期提醒
func (s *SchedulerService) sendLicenseExpirationReminder(license *model.License, days int) {
	// 查找关联的设备和客户
	var devices []model.Device
	model.DB.Preload("Customer").Where("license_id = ?", license.ID).Find(&devices)

	appName := "未知应用"
	if license.Application != nil {
		appName = license.Application.Name
	}

	// 给每个设备的客户发送提醒
	sentEmails := make(map[string]bool)
	for _, device := range devices {
		if device.Customer != nil && device.Customer.Email != "" && !sentEmails[device.Customer.Email] {
			data := ExpirationReminderData{
				UserName:      device.Customer.Name,
				AppName:       appName,
				LicenseType:   string(license.Type),
				ExpireAt:      license.ExpireAt.Format("2006-01-02 15:04"),
				RemainingDays: days,
				RenewURL:      "#", // 可配置续费链接
			}
			if err := s.emailService.SendExpirationReminder(device.Customer.Email, data); err != nil {
				log.Printf("发送到期提醒失败: %v", err)
			} else {
				sentEmails[device.Customer.Email] = true
				log.Printf("已发送到期提醒: %s -> %s", license.LicenseKey[:8], device.Customer.Email)
			}
		}
	}

	// 触发 Webhook
	s.webhookService.SendWebhook(license.AppID, "license.expiring", map[string]interface{}{
		"license_id":     license.ID,
		"remaining_days": days,
		"expire_at":      license.ExpireAt,
	})
}

// sendSubscriptionExpirationReminder 发送订阅到期提醒
func (s *SchedulerService) sendSubscriptionExpirationReminder(sub *model.Subscription, days int) {
	if sub.Customer == nil || sub.Customer.Email == "" {
		return
	}

	appName := "未知应用"
	if sub.Application != nil {
		appName = sub.Application.Name
	}

	data := ExpirationReminderData{
		UserName:      sub.Customer.Name,
		AppName:       appName,
		LicenseType:   string(sub.PlanType),
		ExpireAt:      sub.ExpireAt.Format("2006-01-02 15:04"),
		RemainingDays: days,
		RenewURL:      "#",
	}

	if err := s.emailService.SendExpirationReminder(sub.Customer.Email, data); err != nil {
		log.Printf("发送订阅到期提醒失败: %v", err)
	} else {
		log.Printf("已发送订阅到期提醒: %s -> %s", sub.ID[:8], sub.Customer.Email)
	}
}

// CheckAnomalies 检查异常
func (s *SchedulerService) CheckAnomalies() {
	log.Println("开始检查异常行为...")

	s.checkFrequentActivations()
	s.checkMultipleIPLogins()
	s.checkSuspiciousDevices()

	log.Println("异常行为检查完成")
}

// checkFrequentActivations 检查频繁激活
func (s *SchedulerService) checkFrequentActivations() {
	// 查找过去1小时内激活次数超过10次的授权
	oneHourAgo := time.Now().Add(-1 * time.Hour)

	type Result struct {
		LicenseID string
		Count     int64
	}

	var results []Result
	model.DB.Model(&model.LicenseEvent{}).
		Select("license_id, count(*) as count").
		Where("event_type = ? AND created_at > ?", model.LicenseEventActivated, oneHourAgo).
		Group("license_id").
		Having("count(*) > ?", 10).
		Scan(&results)

	for _, r := range results {
		var license model.License
		if err := model.DB.First(&license, "id = ?", r.LicenseID).Error; err == nil {
			s.webhookService.TriggerAnomalyDetected(license.AppID, "frequent_activation", map[string]interface{}{
				"license_id":       r.LicenseID,
				"activation_count": r.Count,
				"time_window":      "1 hour",
			})
			log.Printf("检测到频繁激活异常: license=%s, count=%d", r.LicenseID[:8], r.Count)
		}
	}
}

// checkMultipleIPLogins 检查多IP登录
func (s *SchedulerService) checkMultipleIPLogins() {
	// 查找过去24小时内从超过5个不同IP登录的设备
	oneDayAgo := time.Now().Add(-24 * time.Hour)

	type Result struct {
		DeviceID string
		IPCount  int64
	}

	var results []Result
	model.DB.Model(&model.Heartbeat{}).
		Select("device_id, count(distinct ip_address) as ip_count").
		Where("created_at > ?", oneDayAgo).
		Group("device_id").
		Having("count(distinct ip_address) > ?", 5).
		Scan(&results)

	for _, r := range results {
		var device model.Device
		if err := model.DB.Preload("License").First(&device, "id = ?", r.DeviceID).Error; err == nil && device.License != nil {
			s.webhookService.TriggerAnomalyDetected(device.License.AppID, "multiple_ip_login", map[string]interface{}{
				"device_id":   r.DeviceID,
				"machine_id":  device.MachineID,
				"ip_count":    r.IPCount,
				"time_window": "24 hours",
			})
			log.Printf("检测到多IP登录异常: device=%s, ip_count=%d", r.DeviceID[:8], r.IPCount)
		}
	}
}

// checkSuspiciousDevices 检查可疑设备
func (s *SchedulerService) checkSuspiciousDevices() {
	// 查找同一机器码绑定了多个授权的情况
	type Result struct {
		MachineID    string
		LicenseCount int64
	}

	var results []Result
	model.DB.Model(&model.Device{}).
		Select("machine_id, count(distinct license_id) as license_count").
		Where("license_id IS NOT NULL AND license_id != ''").
		Group("machine_id").
		Having("count(distinct license_id) > ?", 3).
		Scan(&results)

	for _, r := range results {
		log.Printf("检测到可疑设备: machine_id=%s, license_count=%d", r.MachineID[:16], r.LicenseCount)
		// 可以添加更多处理逻辑，如自动加入黑名单等
	}
}

// CleanupExpiredData 清理过期数据
func (s *SchedulerService) CleanupExpiredData() {
	log.Println("开始清理过期数据...")

	// 清理30天前的心跳记录
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	result := model.DB.Where("created_at < ?", thirtyDaysAgo).Delete(&model.Heartbeat{})
	log.Printf("清理心跳记录: %d 条", result.RowsAffected)

	// 清理90天前的 Webhook 日志
	ninetyDaysAgo := time.Now().AddDate(0, 0, -90)
	result = model.DB.Where("created_at < ?", ninetyDaysAgo).Delete(&model.WebhookLog{})
	log.Printf("清理 Webhook 日志: %d 条", result.RowsAffected)

	log.Println("过期数据清理完成")
}
