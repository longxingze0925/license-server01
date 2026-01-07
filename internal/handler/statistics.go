package handler

import (
	"license-server/internal/middleware"
	"license-server/internal/model"
	"license-server/internal/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
)

type StatisticsHandler struct{}

func NewStatisticsHandler() *StatisticsHandler {
	return &StatisticsHandler{}
}

// Dashboard 仪表盘数据
func (h *StatisticsHandler) Dashboard(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	role, _ := c.Get("role")

	// 只读用户只显示部分统计卡片，但数据不按用户过滤
	isViewer := role == "viewer"

	// 客户统计
	var totalCustomers int64
	model.DB.Model(&model.Customer{}).Where("tenant_id = ?", tenantID).Count(&totalCustomers)

	// 应用统计
	var totalApps int64
	model.DB.Model(&model.Application{}).Where("tenant_id = ?", tenantID).Count(&totalApps)

	// 授权统计
	var totalLicenses int64
	model.DB.Model(&model.License{}).Where("tenant_id = ?", tenantID).Count(&totalLicenses)

	var activeLicenses int64
	model.DB.Model(&model.License{}).Where("tenant_id = ? AND status = ?", tenantID, model.LicenseStatusActive).Count(&activeLicenses)

	var pendingLicenses int64
	model.DB.Model(&model.License{}).Where("tenant_id = ? AND status = ?", tenantID, model.LicenseStatusPending).Count(&pendingLicenses)

	var expiredLicenses int64
	model.DB.Model(&model.License{}).Where("tenant_id = ? AND status = ?", tenantID, model.LicenseStatusExpired).Count(&expiredLicenses)

	// 订阅统计
	var totalSubscriptions int64
	model.DB.Model(&model.Subscription{}).Where("tenant_id = ?", tenantID).Count(&totalSubscriptions)

	var activeSubscriptions int64
	model.DB.Model(&model.Subscription{}).Where("tenant_id = ? AND status = ?", tenantID, model.SubscriptionStatusActive).Count(&activeSubscriptions)

	// 设备统计
	var totalDevices int64
	model.DB.Model(&model.Device{}).Where("tenant_id = ?", tenantID).Count(&totalDevices)

	var activeDevices int64
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	model.DB.Model(&model.Device{}).Where("tenant_id = ? AND last_active_at > ?", tenantID, oneDayAgo).Count(&activeDevices)

	// 今日新增
	today := time.Now().Truncate(24 * time.Hour)

	var todayCustomers int64
	model.DB.Model(&model.Customer{}).Where("tenant_id = ? AND created_at >= ?", tenantID, today).Count(&todayCustomers)

	var todayLicenses int64
	model.DB.Model(&model.License{}).Where("tenant_id = ? AND created_at >= ?", tenantID, today).Count(&todayLicenses)

	result := gin.H{
		"licenses": gin.H{
			"total":     totalLicenses,
			"active":    activeLicenses,
			"pending":   pendingLicenses,
			"expired":   expiredLicenses,
			"today_new": todayLicenses,
		},
		"subscriptions": gin.H{
			"total":  totalSubscriptions,
			"active": activeSubscriptions,
		},
		"devices": gin.H{
			"total":  totalDevices,
			"active": activeDevices,
		},
	}

	// 非只读用户显示更多统计
	if !isViewer {
		result["customers"] = gin.H{
			"total":     totalCustomers,
			"today_new": todayCustomers,
		}
		result["applications"] = gin.H{
			"total": totalApps,
		}
	}

	response.Success(c, result)
}

// AppStatistics 应用统计
func (h *StatisticsHandler) AppStatistics(c *gin.Context) {
	appID := c.Param("app_id")

	var app model.Application
	if err := model.DB.First(&app, "id = ?", appID).Error; err != nil {
		response.NotFound(c, "应用不存在")
		return
	}

	// 授权统计
	var totalLicenses int64
	model.DB.Model(&model.License{}).Where("app_id = ?", appID).Count(&totalLicenses)

	var activeLicenses int64
	model.DB.Model(&model.License{}).Where("app_id = ? AND status = ?", appID, model.LicenseStatusActive).Count(&activeLicenses)

	var pendingLicenses int64
	model.DB.Model(&model.License{}).Where("app_id = ? AND status = ?", appID, model.LicenseStatusPending).Count(&pendingLicenses)

	var expiredLicenses int64
	model.DB.Model(&model.License{}).Where("app_id = ? AND status = ?", appID, model.LicenseStatusExpired).Count(&expiredLicenses)

	// 设备统计
	var totalDevices int64
	model.DB.Model(&model.Device{}).
		Joins("JOIN licenses ON devices.license_id = licenses.id").
		Where("licenses.app_id = ?", appID).Count(&totalDevices)

	var activeDevices int64
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	model.DB.Model(&model.Device{}).
		Joins("JOIN licenses ON devices.license_id = licenses.id").
		Where("licenses.app_id = ? AND devices.last_active_at > ?", appID, oneDayAgo).Count(&activeDevices)

	// 脚本统计
	var totalScripts int64
	model.DB.Model(&model.Script{}).Where("app_id = ?", appID).Count(&totalScripts)

	// 版本统计
	var totalReleases int64
	model.DB.Model(&model.AppRelease{}).Where("app_id = ?", appID).Count(&totalReleases)

	var latestRelease model.AppRelease
	model.DB.Where("app_id = ? AND status = ?", appID, model.ReleaseStatusPublished).
		Order("version_code DESC").First(&latestRelease)

	response.Success(c, gin.H{
		"app": gin.H{
			"id":   app.ID,
			"name": app.Name,
		},
		"licenses": gin.H{
			"total":   totalLicenses,
			"active":  activeLicenses,
			"pending": pendingLicenses,
			"expired": expiredLicenses,
		},
		"devices": gin.H{
			"total":  totalDevices,
			"active": activeDevices,
		},
		"scripts": gin.H{
			"total": totalScripts,
		},
		"releases": gin.H{
			"total":          totalReleases,
			"latest_version": latestRelease.Version,
		},
	})
}

// LicenseTrend 授权趋势（最近30天）
func (h *StatisticsHandler) LicenseTrend(c *gin.Context) {
	appID := c.Query("app_id")

	type DayCount struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	var results []DayCount

	query := model.DB.Model(&model.License{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
		Group("DATE(created_at)").
		Order("date ASC")

	if appID != "" {
		query = query.Where("app_id = ?", appID)
	}

	query.Scan(&results)

	response.Success(c, results)
}

// DeviceTrend 设备趋势（最近30天）
func (h *StatisticsHandler) DeviceTrend(c *gin.Context) {
	appID := c.Query("app_id")

	type DayCount struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}

	var results []DayCount

	query := model.DB.Model(&model.Device{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", time.Now().AddDate(0, 0, -30)).
		Group("DATE(created_at)").
		Order("date ASC")

	if appID != "" {
		query = query.Joins("JOIN licenses ON devices.license_id = licenses.id").
			Where("licenses.app_id = ?", appID)
	}

	query.Scan(&results)

	response.Success(c, results)
}

// HeartbeatTrend 心跳趋势（最近24小时）
func (h *StatisticsHandler) HeartbeatTrend(c *gin.Context) {
	appID := c.Query("app_id")

	type HourCount struct {
		Hour  string `json:"hour"`
		Count int64  `json:"count"`
	}

	var results []HourCount

	query := model.DB.Model(&model.Heartbeat{}).
		Select("DATE_FORMAT(created_at, '%Y-%m-%d %H:00') as hour, COUNT(*) as count").
		Where("created_at >= ?", time.Now().Add(-24*time.Hour)).
		Group("hour").
		Order("hour ASC")

	if appID != "" {
		query = query.Joins("JOIN licenses ON heartbeats.license_id = licenses.id").
			Where("licenses.app_id = ?", appID)
	}

	query.Scan(&results)

	response.Success(c, results)
}

// LicenseTypeDistribution 授权类型分布
func (h *StatisticsHandler) LicenseTypeDistribution(c *gin.Context) {
	appID := c.Query("app_id")

	type TypeCount struct {
		Type  string `json:"type"`
		Count int64  `json:"count"`
	}

	var results []TypeCount

	query := model.DB.Model(&model.License{}).
		Select("type, COUNT(*) as count").
		Group("type")

	if appID != "" {
		query = query.Where("app_id = ?", appID)
	}

	query.Scan(&results)

	response.Success(c, results)
}

// DeviceOSDistribution 设备系统分布
func (h *StatisticsHandler) DeviceOSDistribution(c *gin.Context) {
	appID := c.Query("app_id")

	type OSCount struct {
		OSType string `json:"os_type"`
		Count  int64  `json:"count"`
	}

	var results []OSCount

	query := model.DB.Model(&model.Device{}).
		Select("os_type, COUNT(*) as count").
		Where("os_type IS NOT NULL AND os_type != ''").
		Group("os_type")

	if appID != "" {
		query = query.Joins("JOIN licenses ON devices.license_id = licenses.id").
			Where("licenses.app_id = ?", appID)
	}

	query.Scan(&results)

	response.Success(c, results)
}
