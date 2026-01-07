package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"license-server/internal/config"
	"license-server/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersMiddleware 安全响应头中间件
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 防止点击劫持
		c.Header("X-Frame-Options", "DENY")

		// 防止 MIME 类型嗅探
		c.Header("X-Content-Type-Options", "nosniff")

		// XSS 保护
		c.Header("X-XSS-Protection", "1; mode=block")

		// 引用策略
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		// 内容安全策略
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:;")

		// 权限策略
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// HTTPS 严格传输安全（仅在 HTTPS 时启用）
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}

		c.Next()
	}
}

// CSRFToken CSRF Token 结构
type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

// CSRFManager CSRF Token 管理器
type CSRFManager struct {
	tokens map[string]*CSRFToken
	mu     sync.RWMutex
	expiry time.Duration
}

var (
	csrfManager     *CSRFManager
	csrfManagerOnce sync.Once
)

// GetCSRFManager 获取 CSRF 管理器单例
func GetCSRFManager() *CSRFManager {
	csrfManagerOnce.Do(func() {
		cfg := config.Get()
		expiry := time.Duration(cfg.Security.CSRFTokenExpiry) * time.Minute
		if expiry == 0 {
			expiry = 60 * time.Minute
		}
		csrfManager = &CSRFManager{
			tokens: make(map[string]*CSRFToken),
			expiry: expiry,
		}
		go csrfManager.cleanup()
	})
	return csrfManager
}

// GenerateToken 生成 CSRF Token
func (m *CSRFManager) GenerateToken(sessionID string) string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 生成随机 Token
	bytes := make([]byte, 32)
	rand.Read(bytes)
	token := hex.EncodeToString(bytes)

	m.tokens[sessionID] = &CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(m.expiry),
	}

	return token
}

// ValidateToken 验证 CSRF Token
func (m *CSRFManager) ValidateToken(sessionID, token string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stored, exists := m.tokens[sessionID]
	if !exists {
		return false
	}

	if time.Now().After(stored.ExpiresAt) {
		return false
	}

	return stored.Token == token
}

// cleanup 清理过期 Token
func (m *CSRFManager) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, token := range m.tokens {
			if now.After(token.ExpiresAt) {
				delete(m.tokens, key)
			}
		}
		m.mu.Unlock()
	}
}

// CSRFMiddleware CSRF 保护中间件
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.Get()

		// 如果未启用 CSRF 保护，跳过
		if !cfg.Security.CSRFEnabled {
			c.Next()
			return
		}

		// 安全方法不需要验证
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		// 获取 session ID（使用 JWT token 或 cookie）
		sessionID := c.GetHeader("Authorization")
		if sessionID == "" {
			cookie, err := c.Cookie(cfg.Security.CSRFCookieName)
			if err == nil {
				sessionID = cookie
			}
		}

		if sessionID == "" {
			response.Error(c, 403, "缺少会话标识")
			c.Abort()
			return
		}

		// 获取 CSRF Token
		csrfToken := c.GetHeader("X-CSRF-Token")
		if csrfToken == "" {
			csrfToken = c.PostForm("_csrf")
		}

		if csrfToken == "" {
			response.Error(c, 403, "缺少 CSRF Token")
			c.Abort()
			return
		}

		// 验证 Token
		manager := GetCSRFManager()
		if !manager.ValidateToken(sessionID, csrfToken) {
			response.Error(c, 403, "无效的 CSRF Token")
			c.Abort()
			return
		}

		c.Next()
	}
}

// GenerateCSRFToken 生成 CSRF Token 的处理函数
func GenerateCSRFToken(c *gin.Context) {
	sessionID := c.GetHeader("Authorization")
	if sessionID == "" {
		response.BadRequest(c, "缺少会话标识")
		return
	}

	manager := GetCSRFManager()
	token := manager.GenerateToken(sessionID)

	response.Success(c, gin.H{
		"csrf_token": token,
	})
}
