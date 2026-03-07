package license

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const fileSignatureAlgorithm = "RSA-SHA256"

func buildFileSignaturePayload(fileHash string, fileSize int64) []byte {
	normalizedHash := strings.ToLower(strings.TrimSpace(fileHash))
	return []byte(normalizedHash + ":" + strconv.FormatInt(fileSize, 10))
}

func (c *Client) verifyDownloadedFileSignature(fileHash string, fileSize int64, fileSignature string, signatureAlg string) error {
	if fileSignature == "" {
		if c.requireSignature {
			return ErrSignatureMissing
		}
		return nil
	}

	if signatureAlg != "" && !strings.EqualFold(signatureAlg, fileSignatureAlgorithm) {
		return fmt.Errorf("不支持的签名算法: %s", signatureAlg)
	}

	if c.publicKeyPEM == "" {
		if c.requireSignature {
			return errors.New("未配置公钥，无法验证文件签名")
		}
		return nil
	}

	if err := c.verifySignature(buildFileSignaturePayload(fileHash, fileSize), fileSignature); err != nil {
		return fmt.Errorf("%w: %v", ErrSignatureVerification, err)
	}
	return nil
}

func hashFileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()

	hasher := sha256.New()
	size, err := io.Copy(hasher, f)
	if err != nil {
		return "", 0, err
	}

	return hex.EncodeToString(hasher.Sum(nil)), size, nil
}
