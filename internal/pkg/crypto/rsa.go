package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
)

// GenerateRSAKeyPair 生成 RSA 密钥对
func GenerateRSAKeyPair(bits int) (publicKey, privateKey string, err error) {
	// 生成私钥
	privKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return "", "", err
	}

	// 编码私钥为 PEM 格式
	privKeyBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	// 编码公钥为 PEM 格式
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		return "", "", err
	}
	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pubKeyPEM), string(privKeyPEM), nil
}

// ParsePrivateKey 解析 PEM 格式的私钥
func ParsePrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}

	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// 尝试 PKCS8 格式
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, err
		}
		var ok bool
		privKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("not an RSA private key")
		}
	}

	return privKey, nil
}

// ParsePublicKey 解析 PEM 格式的公钥
func ParsePublicKey(publicKeyPEM string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(publicKeyPEM))
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("not an RSA public key")
	}

	return rsaPubKey, nil
}

// Sign 使用私钥签名数据
func Sign(privateKeyPEM string, data []byte) (string, error) {
	privKey, err := ParsePrivateKey(privateKeyPEM)
	if err != nil {
		return "", err
	}

	hashed := sha256.Sum256(data)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// Verify 使用公钥验证签名
func Verify(publicKeyPEM string, data []byte, signatureBase64 string) error {
	pubKey, err := ParsePublicKey(publicKeyPEM)
	if err != nil {
		return err
	}

	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return err
	}

	hashed := sha256.Sum256(data)
	return rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hashed[:], signature)
}

// Encrypt 使用公钥加密数据
func Encrypt(publicKeyPEM string, data []byte) (string, error) {
	pubKey, err := ParsePublicKey(publicKeyPEM)
	if err != nil {
		return "", err
	}

	encrypted, err := rsa.EncryptPKCS1v15(rand.Reader, pubKey, data)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(encrypted), nil
}

// Decrypt 使用私钥解密数据
func Decrypt(privateKeyPEM string, encryptedBase64 string) ([]byte, error) {
	privKey, err := ParsePrivateKey(privateKeyPEM)
	if err != nil {
		return nil, err
	}

	encrypted, err := base64.StdEncoding.DecodeString(encryptedBase64)
	if err != nil {
		return nil, err
	}

	return rsa.DecryptPKCS1v15(rand.Reader, privKey, encrypted)
}
