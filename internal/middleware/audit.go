package middleware

import (
	"bytes"
	"io"
	"license-server/internal/model"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// AuditMiddleware 审计日志中间件
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 跳过不需要记录的路径
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/client/") ||
			strings.HasPrefix(path, "/health") ||
			strings.Contains(path, "/statistics/") {
			c.Next()
			return
		}

		// 只记录写操作
		method := c.Request.Method
		if method == "GET" {
			c.Next()
			return
		}

		startTime := time.Now()

		// 读取请求体
		var requestBody string
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestBody = string(bodyBytes)
			// 重新设置请求体供后续使用
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

			// 脱敏处理密码字段
			if strings.Contains(requestBody, "password") {
				requestBody = maskSensitiveData(requestBody)
			}
		}

		// 处理请求
		c.Next()

		// 记录日志
		duration := time.Since(startTime).Milliseconds()

		tenantID, _ := c.Get("tenant_id")
		userID, _ := c.Get("user_id")
		userEmail, _ := c.Get("email")

		action, resource, resourceID := parseActionFromPath(method, path)

		log := model.AuditLog{
			TenantID:     toString(tenantID),
			UserID:       toString(userID),
			UserEmail:    toString(userEmail),
			Action:       action,
			Resource:     resource,
			ResourceID:   resourceID,
			Description:  generateDescription(action, resource),
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			RequestBody:  truncateString(requestBody, 2000),
			ResponseCode: c.Writer.Status(),
			Duration:     duration,
		}

		// 异步写入日志
		go func() {
			model.DB.Create(&log)
		}()
	}
}

// parseActionFromPath 从路径解析操作类型
func parseActionFromPath(method, path string) (action, resource, resourceID string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")

	// 解析资源类型
	for _, part := range parts {
		switch part {
		case "apps":
			resource = model.ResourceApp
		case "licenses":
			resource = model.ResourceLicense
		case "subscriptions":
			resource = model.ResourceSubscription
		case "devices":
			resource = model.ResourceDevice
		case "customers":
			resource = model.ResourceCustomer
		case "team", "members":
			resource = model.ResourceTeamMember
		case "tenant":
			resource = model.ResourceTenant
		case "scripts":
			resource = model.ResourceScript
		case "releases":
			resource = model.ResourceRelease
		case "auth":
			resource = model.ResourceUser
		}
	}

	// 解析操作类型
	switch method {
	case "POST":
		if strings.Contains(path, "/login") {
			action = model.ActionLogin
		} else if strings.Contains(path, "/revoke") {
			action = model.ActionRevoke
		} else if strings.Contains(path, "/reset") {
			action = model.ActionReset
		} else {
			action = model.ActionCreate
		}
	case "PUT":
		action = model.ActionUpdate
	case "DELETE":
		action = model.ActionDelete
	default:
		action = method
	}

	// 尝试提取资源ID
	for i, part := range parts {
		if len(part) == 36 && strings.Count(part, "-") == 4 {
			resourceID = part
			break
		}
		// 检查是否是资源类型后面的ID
		if i > 0 && isResourceType(parts[i-1]) && len(part) > 0 {
			resourceID = part
		}
	}

	return
}

func isResourceType(s string) bool {
	types := []string{"apps", "licenses", "subscriptions", "devices", "customers", "members", "scripts", "releases"}
	for _, t := range types {
		if s == t {
			return true
		}
	}
	return false
}

func generateDescription(action, resource string) string {
	actionMap := map[string]string{
		model.ActionCreate: "创建",
		model.ActionUpdate: "更新",
		model.ActionDelete: "删除",
		model.ActionLogin:  "登录",
		model.ActionRevoke: "吊销",
		model.ActionReset:  "重置",
	}
	resourceMap := map[string]string{
		model.ResourceUser:         "用户",
		model.ResourceTeamMember:   "团队成员",
		model.ResourceCustomer:     "客户",
		model.ResourceTenant:       "租户",
		model.ResourceApp:          "应用",
		model.ResourceLicense:      "授权",
		model.ResourceSubscription: "订阅",
		model.ResourceDevice:       "设备",
		model.ResourceScript:       "脚本",
		model.ResourceRelease:      "版本",
	}

	a := actionMap[action]
	if a == "" {
		a = action
	}
	r := resourceMap[resource]
	if r == "" {
		r = resource
	}

	return a + r
}

func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func maskSensitiveData(data string) string {
	// 简单的密码脱敏
	data = strings.ReplaceAll(data, `"password"`, `"password":"***"`)
	data = strings.ReplaceAll(data, `"old_password"`, `"old_password":"***"`)
	data = strings.ReplaceAll(data, `"new_password"`, `"new_password":"***"`)
	return data
}
