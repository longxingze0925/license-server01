package handler

import (
	"license-server/internal/middleware"
	"license-server/internal/model"
	"license-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type TenantHandler struct{}

func NewTenantHandler() *TenantHandler {
	return &TenantHandler{}
}

// Get 获取当前租户信息
func (h *TenantHandler) Get(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var tenant model.Tenant
	if err := model.DB.First(&tenant, "id = ?", tenantID).Error; err != nil {
		response.NotFound(c, "租户不存在")
		return
	}

	// 获取统计信息
	var appCount, memberCount, customerCount, licenseCount int64
	model.DB.Model(&model.Application{}).Where("tenant_id = ?", tenantID).Count(&appCount)
	model.DB.Model(&model.TeamMember{}).Where("tenant_id = ?", tenantID).Count(&memberCount)
	model.DB.Model(&model.Customer{}).Where("tenant_id = ?", tenantID).Count(&customerCount)
	model.DB.Model(&model.License{}).Where("tenant_id = ?", tenantID).Count(&licenseCount)

	// 获取套餐限制
	limits := tenant.GetPlanLimits()

	response.Success(c, gin.H{
		"id":          tenant.ID,
		"name":        tenant.Name,
		"slug":        tenant.Slug,
		"logo":        tenant.Logo,
		"email":       tenant.Email,
		"phone":       tenant.Phone,
		"website":     tenant.Website,
		"address":     tenant.Address,
		"status":      tenant.Status,
		"plan":        tenant.Plan,
		"created_at":  tenant.CreatedAt,
		"usage": gin.H{
			"applications": appCount,
			"team_members": memberCount,
			"customers":    customerCount,
			"licenses":     licenseCount,
		},
		"limits": limits,
	})
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	Name    string `json:"name"`
	Logo    string `json:"logo"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Website string `json:"website"`
	Address string `json:"address"`
}

// Update 更新租户信息（需要 Owner 或 Admin 权限）
func (h *TenantHandler) Update(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)

	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	var tenant model.Tenant
	if err := model.DB.First(&tenant, "id = ?", tenantID).Error; err != nil {
		response.NotFound(c, "租户不存在")
		return
	}

	updates := map[string]interface{}{}
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Logo != "" {
		updates["logo"] = req.Logo
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	if req.Website != "" {
		updates["website"] = req.Website
	}
	if req.Address != "" {
		updates["address"] = req.Address
	}

	if len(updates) > 0 {
		if err := model.DB.Model(&tenant).Updates(updates).Error; err != nil {
			response.ServerError(c, "更新租户失败")
			return
		}
	}

	response.SuccessWithMessage(c, "更新成功", nil)
}

// Delete 删除租户（仅 Owner 可操作）
func (h *TenantHandler) Delete(c *gin.Context) {
	tenantID := middleware.GetTenantID(c)
	role := middleware.GetUserRole(c)

	// 只有 Owner 可以删除租户
	if role != string(model.RoleOwner) {
		response.Forbidden(c, "只有所有者可以删除租户")
		return
	}

	var tenant model.Tenant
	if err := model.DB.First(&tenant, "id = ?", tenantID).Error; err != nil {
		response.NotFound(c, "租户不存在")
		return
	}

	// 开始事务删除所有关联数据
	tx := model.DB.Begin()

	// 删除设备
	if err := tx.Where("tenant_id = ?", tenantID).Delete(&model.Device{}).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "删除设备失败")
		return
	}

	// 删除授权码
	if err := tx.Where("tenant_id = ?", tenantID).Delete(&model.License{}).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "删除授权码失败")
		return
	}

	// 删除订阅
	if err := tx.Where("tenant_id = ?", tenantID).Delete(&model.Subscription{}).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "删除订阅失败")
		return
	}

	// 删除客户
	if err := tx.Where("tenant_id = ?", tenantID).Delete(&model.Customer{}).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "删除客户失败")
		return
	}

	// 删除应用
	if err := tx.Where("tenant_id = ?", tenantID).Delete(&model.Application{}).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "删除应用失败")
		return
	}

	// 删除团队成员
	if err := tx.Where("tenant_id = ?", tenantID).Delete(&model.TeamMember{}).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "删除团队成员失败")
		return
	}

	// 删除团队邀请
	if err := tx.Where("tenant_id = ?", tenantID).Delete(&model.TeamInvitation{}).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "删除邀请失败")
		return
	}

	// 删除租户
	if err := tx.Delete(&tenant).Error; err != nil {
		tx.Rollback()
		response.ServerError(c, "删除租户失败")
		return
	}

	tx.Commit()
	response.SuccessWithMessage(c, "租户已删除", nil)
}
