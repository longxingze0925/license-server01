package middleware

import (
	"strings"
	"time"

	"license-server/internal/model"
	"license-server/internal/pkg/clientauth"
	"license-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

const (
	clientCtxSessionID  = "client_session_id"
	clientCtxTenantID   = "client_tenant_id"
	clientCtxAppID      = "client_app_id"
	clientCtxCustomerID = "client_customer_id"
	clientCtxDeviceID   = "client_device_id"
	clientCtxMachineID  = "client_machine_id"
	clientCtxAuthMode   = "client_auth_mode"
	clientCtxSession    = "client_session"
)

// ClientAuthMiddleware 客户端访问令牌认证中间件
func ClientAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			response.Unauthorized(c, "缺少认证信息")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(c, "认证格式错误")
			c.Abort()
			return
		}

		claims, err := clientauth.ParseAccessToken(parts[1])
		if err != nil {
			response.Unauthorized(c, "无效的客户端令牌")
			c.Abort()
			return
		}

		var session model.ClientSession
		if err := model.DB.First(&session, "id = ?", claims.SessionID).Error; err != nil {
			response.Unauthorized(c, "会话不存在或已失效")
			c.Abort()
			return
		}

		now := time.Now()
		if session.IsRevoked() || session.IsExpired(now) {
			response.Unauthorized(c, "会话已过期，请重新登录")
			c.Abort()
			return
		}

		if session.TenantID != claims.TenantID ||
			session.AppID != claims.AppID ||
			session.DeviceID != claims.DeviceID ||
			session.MachineID != claims.MachineID {
			response.Unauthorized(c, "会话与设备不匹配")
			c.Abort()
			return
		}
		if claims.CustomerID != "" && session.CustomerID != claims.CustomerID {
			response.Unauthorized(c, "会话与用户不匹配")
			c.Abort()
			return
		}
		if claims.AuthMode != "" && session.AuthMode != claims.AuthMode {
			response.Unauthorized(c, "会话模式无效")
			c.Abort()
			return
		}

		c.Set(clientCtxSessionID, session.ID)
		c.Set(clientCtxTenantID, session.TenantID)
		c.Set(clientCtxAppID, session.AppID)
		c.Set(clientCtxCustomerID, session.CustomerID)
		c.Set(clientCtxDeviceID, session.DeviceID)
		c.Set(clientCtxMachineID, session.MachineID)
		c.Set(clientCtxAuthMode, session.AuthMode)
		c.Set(clientCtxSession, &session)

		c.Next()
	}
}

func GetClientSessionID(c *gin.Context) string {
	v, _ := c.Get(clientCtxSessionID)
	id, _ := v.(string)
	return id
}

func GetClientTenantID(c *gin.Context) string {
	v, _ := c.Get(clientCtxTenantID)
	id, _ := v.(string)
	return id
}

func GetClientAppID(c *gin.Context) string {
	v, _ := c.Get(clientCtxAppID)
	id, _ := v.(string)
	return id
}

func GetClientCustomerID(c *gin.Context) string {
	v, _ := c.Get(clientCtxCustomerID)
	id, _ := v.(string)
	return id
}

func GetClientDeviceID(c *gin.Context) string {
	v, _ := c.Get(clientCtxDeviceID)
	id, _ := v.(string)
	return id
}

func GetClientMachineID(c *gin.Context) string {
	v, _ := c.Get(clientCtxMachineID)
	id, _ := v.(string)
	return id
}

func GetClientAuthMode(c *gin.Context) string {
	v, _ := c.Get(clientCtxAuthMode)
	id, _ := v.(string)
	return id
}

func GetClientSession(c *gin.Context) *model.ClientSession {
	v, ok := c.Get(clientCtxSession)
	if !ok {
		return nil
	}
	session, _ := v.(*model.ClientSession)
	return session
}
