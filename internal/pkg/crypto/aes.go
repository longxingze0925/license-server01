package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

// GenerateAESKey 生成随机 AES 密钥
func GenerateAESKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

// GenerateNonce 生成随机 Nonce
func GenerateNonce(size int) (string, error) {
	nonce := make([]byte, size)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce), nil
}

// DeriveKey 使用 HKDF 派生密钥
// 用于从主密钥 + 设备信息派生出设备专用密钥
func DeriveKey(masterKey []byte, salt string, info string) ([]byte, error) {
	hash := sha256.New
	hkdfReader := hkdf.New(hash, masterKey, []byte(salt), []byte(info))

	derivedKey := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(hkdfReader, derivedKey); err != nil {
		return nil, err
	}
	return derivedKey, nil
}

// EncryptAESGCM 使用 AES-GCM 加密数据
func EncryptAESGCM(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// nonce 附加在密文前面
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptAESGCM 使用 AES-GCM 解密数据
func DecryptAESGCM(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// EncryptAESGCMBase64 加密并返回 Base64 编码
func EncryptAESGCMBase64(plaintext []byte, key []byte) (string, error) {
	ciphertext, err := EncryptAESGCM(plaintext, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptAESGCMBase64 从 Base64 解码并解密
func DecryptAESGCMBase64(ciphertextBase64 string, key []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return nil, err
	}
	return DecryptAESGCM(ciphertext, key)
}

// SHA256Hash 计算 SHA256 哈希
func SHA256Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// SHA256HashString 计算字符串的 SHA256 哈希
func SHA256HashString(s string) string {
	return SHA256Hash([]byte(s))
}
