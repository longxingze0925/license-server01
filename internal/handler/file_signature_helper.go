package handler

import (
	"fmt"
	cryptopkg "license-server/internal/pkg/crypto"
	"strconv"
	"strings"
)

const fileSignatureAlgorithm = "RSA-SHA256"

func buildFileSignaturePayload(fileHash string, fileSize int64) []byte {
	normalizedHash := strings.ToLower(strings.TrimSpace(fileHash))
	return []byte(normalizedHash + ":" + strconv.FormatInt(fileSize, 10))
}

func signFileSignature(privateKey string, fileHash string, fileSize int64) (string, error) {
	payload := buildFileSignaturePayload(fileHash, fileSize)
	signature, err := cryptopkg.Sign(privateKey, payload)
	if err != nil {
		return "", fmt.Errorf("文件签名失败: %w", err)
	}
	return signature, nil
}
