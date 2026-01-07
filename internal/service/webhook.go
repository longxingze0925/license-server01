package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"license-server/internal/model"
	"net/http"
	"time"
)

// WebhookService Webhook 服务
type WebhookService struct{}

// NewWebhookService 创建 Webhook 服务
func NewWebhookService() *WebhookService {
	return &WebhookService{}
}

// WebhookEvent 事件类型
type WebhookEvent string

const (
	EventLicenseCreated   WebhookEvent = "license.created"
	EventLicenseActivated WebhookEvent = "license.activated"
	EventLicenseExpired   WebhookEvent = "license.expired"
	EventLicenseRevoked   WebhookEvent = "license.revoked"
	EventLicenseRenewed   WebhookEvent = "license.renewed"
	EventDeviceBound      WebhookEvent = "device.bound"
	EventDeviceUnbound    WebhookEvent = "device.unbound"
	EventUserRegistered   WebhookEvent = "user.registered"
	EventUserLogin        WebhookEvent = "user.login"
	EventAnomalyDetected  WebhookEvent = "anomaly.detected"
)

// WebhookPayload Webhook 负载
type WebhookPayload struct {
	Event     WebhookEvent           `json:"event"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// SendWebhook 发送 Webhook
func (s *WebhookService) SendWebhook(appID string, event WebhookEvent, data map[string]interface{}) error {
	// 查找该应用的所有活跃 Webhook
	var webhooks []model.Webhook
	model.DB.Where("app_id = ? AND status = ?", appID, "active").Find(&webhooks)

	if len(webhooks) == 0 {
		return nil
	}

	payload := WebhookPayload{
		Event:     event,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// 异步发送所有 Webhook
	for _, webhook := range webhooks {
		go s.sendSingleWebhook(webhook, payloadBytes)
	}

	return nil
}

// sendSingleWebhook 发送单个 Webhook
func (s *WebhookService) sendSingleWebhook(webhook model.Webhook, payload []byte) {
	// 生成签名
	signature := s.generateSignature(webhook.Secret, payload)

	// 创建请求
	req, err := http.NewRequest("POST", webhook.URL, bytes.NewBuffer(payload))
	if err != nil {
		s.logWebhookResult(webhook.ID, false, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Webhook-Signature", signature)
	req.Header.Set("X-Webhook-Timestamp", time.Now().Format(time.RFC3339))

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logWebhookResult(webhook.ID, false, err.Error())
		return
	}
	defer resp.Body.Close()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	s.logWebhookResult(webhook.ID, success, resp.Status)
}

// generateSignature 生成 HMAC 签名
func (s *WebhookService) generateSignature(secret string, payload []byte) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

// logWebhookResult 记录 Webhook 结果
func (s *WebhookService) logWebhookResult(webhookID string, success bool, response string) {
	log := model.WebhookLog{
		WebhookID:    webhookID,
		Success:      success,
		ResponseBody: response,
	}
	model.DB.Create(&log)

	// 更新 Webhook 最后触发时间
	model.DB.Model(&model.Webhook{}).Where("id = ?", webhookID).Update("last_triggered_at", time.Now())
}

// TriggerLicenseCreated 触发授权创建事件
func (s *WebhookService) TriggerLicenseCreated(license *model.License) {
	s.SendWebhook(license.AppID, EventLicenseCreated, map[string]interface{}{
		"license_id":   license.ID,
		"license_key":  license.LicenseKey,
		"type":         license.Type,
		"max_devices":  license.MaxDevices,
		"duration_days": license.DurationDays,
	})
}

// TriggerLicenseActivated 触发授权激活事件
func (s *WebhookService) TriggerLicenseActivated(license *model.License, device *model.Device) {
	s.SendWebhook(license.AppID, EventLicenseActivated, map[string]interface{}{
		"license_id":  license.ID,
		"device_id":   device.ID,
		"machine_id":  device.MachineID,
		"device_name": device.DeviceName,
		"ip_address":  device.IPAddress,
	})
}

// TriggerLicenseExpired 触发授权过期事件
func (s *WebhookService) TriggerLicenseExpired(license *model.License) {
	s.SendWebhook(license.AppID, EventLicenseExpired, map[string]interface{}{
		"license_id":  license.ID,
		"license_key": license.LicenseKey,
		"expired_at":  license.ExpireAt,
	})
}

// TriggerAnomalyDetected 触发异常检测事件
func (s *WebhookService) TriggerAnomalyDetected(appID string, anomalyType string, details map[string]interface{}) {
	data := map[string]interface{}{
		"anomaly_type": anomalyType,
		"details":      details,
	}
	s.SendWebhook(appID, EventAnomalyDetected, data)
}
