package handler

import (
	"license-server/internal/middleware"
	"license-server/internal/model"
	"license-server/internal/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

type CustomerHandler struct{}

func NewCustomerHandler() *CustomerHandler {
	return &CustomerHandler{}
}

// CreateCustomerRequest 创建客户请求
type CreateCustomerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password"` // 可选，订阅模式需要
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Company  string `json:"company"`
	Remark   string `json:"remark"`
	Metadata string `json:"metadata"` // JSON 格式
	OwnerID  string `json:"owner_id"` // 所属团队成员ID
}

// Create 创建客户
func (h *CustomerHandler) Create(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)

	var req CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	// 检查邮箱是否已存在（同一租户内）
	var existingCustomer model.Customer
	if err := model.DB.Where("email = ? AND tenant_id = ?", req.Email, tenantID).First(&existingCustomer).Error; err == nil {
		response.Error(c, 400, "该邮箱已存在")
		return
	}

	// 如果没有指定所属成员，默认为当前用户
	ownerID := req.OwnerID
	if ownerID == "" {
		ownerID = userID
	}

	// 验证所属成员是否存在
	if ownerID != "" {
		var owner model.TeamMember
		if err := model.DB.Where("id = ? AND tenant_id = ?", ownerID, tenantID).First(&owner).Error; err != nil {
			response.Error(c, 400, "所属成员不存在")
			return
		}
	}

	customer := model.Customer{
		TenantID: tenantID,
		OwnerID:  ownerID,
		Email:    req.Email,
		Name:     req.Name,
		Phone:    req.Phone,
		Company:  req.Company,
		Remark:   req.Remark,
		Metadata: req.Metadata,
		Status:   model.CustomerStatusActive,
	}

	// 如果提供了密码，则设置密码（使用预哈希逻辑，与SDK保持一致）
	if req.Password != "" {
		if err := customer.SetPasswordWithPreHash(req.Password, false); err != nil {
			response.ServerError(c, "密码加密失败")
			return
		}
	}

	if err := model.DB.Create(&customer).Error; err != nil {
		response.ServerError(c, "创建客户失败")
		return
	}

	response.Success(c, gin.H{
		"id":         customer.ID,
		"email":      customer.Email,
		"name":       customer.Name,
		"owner_id":   customer.OwnerID,
		"status":     customer.Status,
		"created_at": customer.CreatedAt,
	})
}

// List 获取客户列表
func (h *CustomerHandler) List(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	userID := middleware.GetUserID(c)
	userRole := middleware.GetUserRole(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	keyword := c.Query("keyword")
	ownerID := c.Query("owner_id") // 按所属成员筛选

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := model.DB.Model(&model.Customer{}).Where("tenant_id = ?", tenantID)

	// 权限控制：非管理员只能看到自己名下的客户
	isAdmin := userRole == string(model.RoleOwner) || userRole == string(model.RoleAdmin)
	if !isAdmin {
		query = query.Where("owner_id = ?", userID)
	} else if ownerID != "" {
		// 管理员可以按所属成员筛选
		query = query.Where("owner_id = ?", ownerID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		query = query.Where("email LIKE ? OR name LIKE ? OR company LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	query.Count(&total)

	var customers []model.Customer
	query.Preload("Owner").Offset((page - 1) * pageSize).Limit(pageSize).Order("created_at DESC").Find(&customers)

	result := make([]gin.H, 0, len(customers))
	for _, cust := range customers {
		item := gin.H{
			"id":            cust.ID,
			"email":         cust.Email,
			"name":          cust.Name,
			"phone":         cust.Phone,
			"company":       cust.Company,
			"status":        cust.Status,
			"owner_id":      cust.OwnerID,
			"has_password":  cust.HasPassword(),
			"last_login_at": cust.LastLoginAt,
			"created_at":    cust.CreatedAt,
		}
		// 添加所属成员信息
		if cust.Owner != nil {
			item["owner_name"] = cust.Owner.Name
			item["owner_email"] = cust.Owner.Email
		}
		result = append(result, item)
	}

	response.SuccessPage(c, result, total, page, pageSize)
}

// Get 获取客户详情
func (h *CustomerHandler) Get(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var customer model.Customer
	if err := model.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&customer).Error; err != nil {
		response.NotFound(c, "客户不存在")
		return
	}

	// 获取关联数据统计
	var licenseCount, subscriptionCount, deviceCount int64
	model.DB.Model(&model.License{}).Where("customer_id = ?", id).Count(&licenseCount)
	model.DB.Model(&model.Subscription{}).Where("customer_id = ?", id).Count(&subscriptionCount)
	model.DB.Model(&model.Device{}).Where("customer_id = ?", id).Count(&deviceCount)

	response.Success(c, gin.H{
		"id":            customer.ID,
		"email":         customer.Email,
		"name":          customer.Name,
		"phone":         customer.Phone,
		"company":       customer.Company,
		"status":        customer.Status,
		"metadata":      customer.Metadata,
		"remark":        customer.Remark,
		"has_password":  customer.HasPassword(),
		"last_login_at": customer.LastLoginAt,
		"last_login_ip": customer.LastLoginIP,
		"created_at":    customer.CreatedAt,
		"stats": gin.H{
			"licenses":      licenseCount,
			"subscriptions": subscriptionCount,
			"devices":       deviceCount,
		},
	})
}

// UpdateCustomerRequest 更新客户请求
type UpdateCustomerRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Company  string `json:"company"`
	Remark   string `json:"remark"`
	Metadata string `json:"metadata"`
	Status   string `json:"status"`
}

// Update 更新客户
func (h *CustomerHandler) Update(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var req UpdateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	var customer model.Customer
	if err := model.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&customer).Error; err != nil {
		response.NotFound(c, "客户不存在")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Company != "" {
		updates["company"] = req.Company
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}
	if req.Metadata != "" {
		updates["metadata"] = req.Metadata
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}

	if len(updates) > 0 {
		if err := model.DB.Model(&customer).Updates(updates).Error; err != nil {
			response.ServerError(c, "更新客户失败")
			return
		}
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除客户
func (h *CustomerHandler) Delete(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var customer model.Customer
	if err := model.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&customer).Error; err != nil {
		response.NotFound(c, "客户不存在")
		return
	}

	// 删除关联数据
	tx := model.DB.Begin()

	// 删除设备
	tx.Where("customer_id = ?", id).Delete(&model.Device{})

	// 删除订阅
	tx.Where("customer_id = ?", id).Delete(&model.Subscription{})

	// 更新授权码（解除关联）
	tx.Model(&model.License{}).Where("customer_id = ?", id).Update("customer_id", nil)

	// 删除客户
	if err := tx.Delete(&customer).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "删除客户失败")
		return
	}

	tx.Commit()
	response.SuccessWithMessage(c, "删除成功", nil)
}

// Disable 禁用客户
func (h *CustomerHandler) Disable(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var customer model.Customer
	if err := model.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&customer).Error; err != nil {
		response.NotFound(c, "客户不存在")
		return
	}

	customer.Status = model.CustomerStatusDisabled
	model.DB.Save(&customer)

	response.SuccessWithMessage(c, "客户已禁用", nil)
}

// Enable 启用客户
func (h *CustomerHandler) Enable(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var customer model.Customer
	if err := model.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&customer).Error; err != nil {
		response.NotFound(c, "客户不存在")
		return
	}

	customer.Status = model.CustomerStatusActive
	model.DB.Save(&customer)

	response.SuccessWithMessage(c, "客户已启用", nil)
}

// ResetPassword 重置客户密码
func (h *CustomerHandler) ResetPassword(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var req struct {
		Password string `json:"password" binding:"required,min=6"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	var customer model.Customer
	if err := model.DB.Where("id = ? AND tenant_id = ?", id, tenantID).First(&customer).Error; err != nil {
		response.NotFound(c, "客户不存在")
		return
	}

	if err := customer.SetPasswordWithPreHash(req.Password, false); err != nil {
		response.ServerError(c, "密码加密失败")
		return
	}

	model.DB.Save(&customer)
	response.SuccessWithMessage(c, "密码已重置", nil)
}

// GetLicenses 获取客户的授权码
func (h *CustomerHandler) GetLicenses(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var licenses []model.License
	model.DB.Preload("Application").Where("customer_id = ? AND tenant_id = ?", id, tenantID).Find(&licenses)

	var result []gin.H
	for _, l := range licenses {
		item := gin.H{
			"id":           l.ID,
			"license_key":  l.LicenseKey,
			"type":         l.Type,
			"status":       l.Status,
			"max_devices":  l.MaxDevices,
			"activated_at": l.ActivatedAt,
			"expire_at":    l.ExpireAt,
			"created_at":   l.CreatedAt,
		}
		if l.Application != nil {
			item["app_name"] = l.Application.Name
		}
		result = append(result, item)
	}

	response.Success(c, result)
}

// GetSubscriptions 获取客户的订阅
func (h *CustomerHandler) GetSubscriptions(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var subscriptions []model.Subscription
	model.DB.Preload("Application").Where("customer_id = ? AND tenant_id = ?", id, tenantID).Find(&subscriptions)

	var result []gin.H
	for _, s := range subscriptions {
		item := gin.H{
			"id":          s.ID,
			"plan_type":   s.PlanType,
			"status":      s.Status,
			"max_devices": s.MaxDevices,
			"start_at":    s.StartAt,
			"expire_at":   s.ExpireAt,
			"created_at":  s.CreatedAt,
		}
		if s.Application != nil {
			item["app_name"] = s.Application.Name
		}
		result = append(result, item)
	}

	response.Success(c, result)
}

// GetDevices 获取客户的设备
func (h *CustomerHandler) GetDevices(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	id := c.Param("id")

	var devices []model.Device
	model.DB.Where("customer_id = ? AND tenant_id = ?", id, tenantID).Find(&devices)

	var result []gin.H
	for _, d := range devices {
		result = append(result, gin.H{
			"id":               d.ID,
			"machine_id":       d.MachineID,
			"device_name":      d.DeviceName,
			"hostname":         d.Hostname,
			"os_type":          d.OSType,
			"os_version":       d.OSVersion,
			"app_version":      d.AppVersion,
			"ip_address":       d.IPAddress,
			"status":           d.Status,
			"last_heartbeat_at": d.LastHeartbeatAt,
			"created_at":       d.CreatedAt,
		})
	}

	response.Success(c, result)
}
