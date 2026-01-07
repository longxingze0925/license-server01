package handler

import (
	"license-server/internal/model"
	"license-server/internal/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ClientUserHandler 客户端用户管理
type ClientUserHandler struct{}

// NewClientUserHandler 创建客户端用户处理器
func NewClientUserHandler() *ClientUserHandler {
	return &ClientUserHandler{}
}

// List 获取客户端用户列表
func (h *ClientUserHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	offset := (page - 1) * pageSize

	var users []model.ClientUser
	var total int64

	query := model.DB.Model(&model.ClientUser{})

	// 搜索
	if search := c.Query("search"); search != "" {
		query = query.Where("email LIKE ? OR name LIKE ? OR phone LIKE ?",
			"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	// 状态筛选
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	query.Count(&total)
	query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users)

	var result []gin.H
	for _, user := range users {
		// 获取订阅数量
		var subscriptionCount int64
		model.DB.Model(&model.Subscription{}).Where("client_user_id = ?", user.ID).Count(&subscriptionCount)

		// 获取设备数量
		var deviceCount int64
		model.DB.Model(&model.Device{}).Where("client_user_id = ?", user.ID).Count(&deviceCount)

		result = append(result, gin.H{
			"id":                 user.ID,
			"email":              user.Email,
			"name":               user.Name,
			"phone":              user.Phone,
			"avatar":             user.Avatar,
			"status":             user.Status,
			"remark":             user.Remark,
			"last_login_at":      user.LastLoginAt,
			"last_login_ip":      user.LastLoginIP,
			"subscription_count": subscriptionCount,
			"device_count":       deviceCount,
			"created_at":         user.CreatedAt,
			"updated_at":         user.UpdatedAt,
		})
	}

	response.SuccessPage(c, result, total, page, pageSize)
}

// Get 获取客户端用户详情
func (h *ClientUserHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var user model.ClientUser
	if err := model.DB.Preload("Subscriptions").Preload("Subscriptions.Application").
		First(&user, "id = ?", id).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	// 获取设备列表
	var devices []model.Device
	model.DB.Where("client_user_id = ?", id).Find(&devices)

	response.Success(c, gin.H{
		"id":            user.ID,
		"email":         user.Email,
		"name":          user.Name,
		"phone":         user.Phone,
		"avatar":        user.Avatar,
		"status":        user.Status,
		"remark":        user.Remark,
		"last_login_at": user.LastLoginAt,
		"last_login_ip": user.LastLoginIP,
		"subscriptions": user.Subscriptions,
		"devices":       devices,
		"created_at":    user.CreatedAt,
		"updated_at":    user.UpdatedAt,
	})
}

// Create 创建客户端用户
func (h *ClientUserHandler) Create(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=6"`
		Name     string `json:"name"`
		Phone    string `json:"phone"`
		Remark   string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 检查邮箱是否已存在
	var existingUser model.ClientUser
	if err := model.DB.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		response.BadRequest(c, "邮箱已被注册")
		return
	}

	user := model.ClientUser{
		Email:  req.Email,
		Name:   req.Name,
		Phone:  req.Phone,
		Remark: req.Remark,
		Status: model.ClientUserStatusActive,
	}

	if err := user.SetPassword(req.Password); err != nil {
		response.ServerError(c, "密码加密失败")
		return
	}

	if err := model.DB.Create(&user).Error; err != nil {
		response.ServerError(c, "创建用户失败")
		return
	}

	response.Success(c, gin.H{
		"id":         user.ID,
		"email":      user.Email,
		"name":       user.Name,
		"phone":      user.Phone,
		"status":     user.Status,
		"created_at": user.CreatedAt,
	})
}

// Update 更新客户端用户
func (h *ClientUserHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var user model.ClientUser
	if err := model.DB.First(&user, "id = ?", id).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	var req struct {
		Name   string `json:"name"`
		Phone  string `json:"phone"`
		Avatar string `json:"avatar"`
		Status string `json:"status"`
		Remark string `json:"remark"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}

	if err := model.DB.Model(&user).Updates(updates).Error; err != nil {
		response.ServerError(c, "更新失败")
		return
	}

	response.Success(c, gin.H{"message": "更新成功"})
}

// Delete 删除客户端用户
func (h *ClientUserHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	var user model.ClientUser
	if err := model.DB.First(&user, "id = ?", id).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	// 检查是否有关联的订阅
	var subscriptionCount int64
	model.DB.Model(&model.Subscription{}).Where("client_user_id = ?", id).Count(&subscriptionCount)
	if subscriptionCount > 0 {
		response.BadRequest(c, "该用户存在关联的订阅，无法删除")
		return
	}

	if err := model.DB.Delete(&user).Error; err != nil {
		response.ServerError(c, "删除失败")
		return
	}

	response.Success(c, gin.H{"message": "删除成功"})
}

// ResetPassword 重置密码
func (h *ClientUserHandler) ResetPassword(c *gin.Context) {
	id := c.Param("id")

	var user model.ClientUser
	if err := model.DB.First(&user, "id = ?", id).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	var req struct {
		Password string `json:"password" binding:"required,min=6"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "密码长度至少6位")
		return
	}

	if err := user.SetPassword(req.Password); err != nil {
		response.ServerError(c, "密码加密失败")
		return
	}

	if err := model.DB.Model(&user).Update("password", user.Password).Error; err != nil {
		response.ServerError(c, "重置密码失败")
		return
	}

	response.Success(c, gin.H{"message": "密码已重置"})
}

// ToggleStatus 切换用户状态
func (h *ClientUserHandler) ToggleStatus(c *gin.Context) {
	id := c.Param("id")

	var user model.ClientUser
	if err := model.DB.First(&user, "id = ?", id).Error; err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	newStatus := model.ClientUserStatusActive
	if user.Status == model.ClientUserStatusActive {
		newStatus = model.ClientUserStatusDisabled
	}

	if err := model.DB.Model(&user).Update("status", newStatus).Error; err != nil {
		response.ServerError(c, "状态更新失败")
		return
	}

	response.Success(c, gin.H{
		"status":  newStatus,
		"message": "状态已更新",
	})
}

// GetSubscriptions 获取用户的订阅列表
func (h *ClientUserHandler) GetSubscriptions(c *gin.Context) {
	id := c.Param("id")

	var subscriptions []model.Subscription
	if err := model.DB.Preload("Application").
		Where("client_user_id = ?", id).
		Order("created_at DESC").
		Find(&subscriptions).Error; err != nil {
		response.ServerError(c, "获取订阅列表失败")
		return
	}

	response.Success(c, subscriptions)
}

// GetDevices 获取用户的设备列表
func (h *ClientUserHandler) GetDevices(c *gin.Context) {
	id := c.Param("id")

	var devices []model.Device
	if err := model.DB.Where("client_user_id = ?", id).
		Order("last_heartbeat_at DESC").
		Find(&devices).Error; err != nil {
		response.ServerError(c, "获取设备列表失败")
		return
	}

	response.Success(c, devices)
}
