package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// GenerateUUID 生成 UUID
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateRandomString 生成随机字符串
func GenerateRandomString(length int) string {
	bytes := make([]byte, length/2+1)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)[:length]
}

// GenerateLicenseKey 生成授权码
// 格式: XXXX-XXXX-XXXX-XXXX
func GenerateLicenseKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	hex := strings.ToUpper(hex.EncodeToString(bytes))
	return fmt.Sprintf("%s-%s-%s-%s", hex[0:4], hex[4:8], hex[8:12], hex[12:16])
}

// GenerateAppKey 生成应用 Key
func GenerateAppKey() string {
	return GenerateRandomString(32)
}

// GenerateAppSecret 生成应用 Secret
func GenerateAppSecret() string {
	return GenerateRandomString(64)
}

// GenerateInviteToken 生成邀请 Token
func GenerateInviteToken() string {
	return GenerateRandomString(32)
}

// MaskEmail 隐藏邮箱中间部分
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	name := parts[0]
	domain := parts[1]
	if len(name) <= 2 {
		return email
	}
	masked := name[0:1] + "***" + name[len(name)-1:]
	return masked + "@" + domain
}

// MaskLicenseKey 隐藏授权码中间部分
func MaskLicenseKey(key string) string {
	if len(key) < 8 {
		return key
	}
	return key[0:4] + "-****-****-" + key[len(key)-4:]
}
