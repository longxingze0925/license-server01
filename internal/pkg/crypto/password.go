package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// PasswordSaltVersion 密码盐版本，用于客户端预哈希
	PasswordSaltVersion = "license_salt_v1"
)

// ClientHashPassword 客户端预哈希密码（与 SDK 保持一致）
// 用于服务端验证或生成与客户端相同的哈希
func ClientHashPassword(password, email string) string {
	salted := fmt.Sprintf("%s:%s:%s", password, strings.ToLower(email), PasswordSaltVersion)
	hash := sha256.Sum256([]byte(salted))
	return hex.EncodeToString(hash[:])
}

// HashPassword 服务端最终密码哈希（bcrypt）
// 将客户端预哈希后的密码再次使用 bcrypt 哈希存储
func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CheckPassword 验证密码
// password: 客户端传来的密码（可能是预哈希的，也可能是明文）
// hashedPassword: 数据库中存储的 bcrypt 哈希
// email: 用户邮箱
// isPreHashed: 客户端是否已预哈希
func CheckPassword(password, hashedPassword, email string, isPreHashed bool) bool {
	var passwordToCheck string

	if isPreHashed {
		// 客户端已预哈希，直接使用
		passwordToCheck = password
	} else {
		// 客户端未预哈希（兼容旧版本 SDK），服务端进行预哈希
		passwordToCheck = ClientHashPassword(password, email)
	}

	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(passwordToCheck))
	return err == nil
}

// PreparePasswordForStorage 准备密码用于存储
// 将客户端传来的密码处理后返回 bcrypt 哈希
// password: 客户端传来的密码
// email: 用户邮箱
// isPreHashed: 客户端是否已预哈希
func PreparePasswordForStorage(password, email string, isPreHashed bool) (string, error) {
	var passwordToHash string

	if isPreHashed {
		// 客户端已预哈希，直接使用
		passwordToHash = password
	} else {
		// 客户端未预哈希（兼容旧版本 SDK），服务端进行预哈希
		passwordToHash = ClientHashPassword(password, email)
	}

	return HashPassword(passwordToHash)
}

// IsPreHashedPassword 检查密码是否是预哈希格式
// 预哈希密码是 64 字符的十六进制字符串（SHA256）
func IsPreHashedPassword(password string) bool {
	if len(password) != 64 {
		return false
	}
	for _, c := range password {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
