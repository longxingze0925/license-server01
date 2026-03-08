package clientauth

import (
	"errors"
	"license-server/internal/config"
	cryptopkg "license-server/internal/pkg/crypto"
	"license-server/internal/pkg/utils"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AuthModeLicense      = "license"
	AuthModeSubscription = "subscription"
)

const (
	defaultClientAccessTokenExpireSeconds  = 15 * 60        // 15 分钟
	defaultClientRefreshTokenExpireSeconds = 30 * 24 * 3600 // 30 天
)

// AccessClaims 客户端访问令牌声明
type AccessClaims struct {
	TokenType  string `json:"typ"`
	SessionID  string `json:"sid"`
	TenantID   string `json:"tenant_id"`
	AppID      string `json:"app_id"`
	CustomerID string `json:"customer_id"`
	DeviceID   string `json:"device_id"`
	MachineID  string `json:"machine_id"`
	AuthMode   string `json:"auth_mode"`
	jwt.RegisteredClaims
}

func GenerateAccessToken(sessionID, tenantID, appID, customerID, deviceID, machineID, authMode string) (string, time.Time, error) {
	secret := getAccessTokenSecret()
	if secret == "" {
		return "", time.Time{}, errors.New("missing client access token secret")
	}

	now := time.Now()
	expiresAt := now.Add(getAccessTokenTTL())
	claims := AccessClaims{
		TokenType:  "client_access",
		SessionID:  sessionID,
		TenantID:   tenantID,
		AppID:      appID,
		CustomerID: customerID,
		DeviceID:   deviceID,
		MachineID:  machineID,
		AuthMode:   authMode,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now.Add(-5 * time.Second)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expiresAt, nil
}

func ParseAccessToken(tokenString string) (*AccessClaims, error) {
	secret := getAccessTokenSecret()
	if secret == "" {
		return nil, errors.New("missing client access token secret")
	}

	token, err := jwt.ParseWithClaims(tokenString, &AccessClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("invalid signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*AccessClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != "client_access" || claims.SessionID == "" || claims.AppID == "" || claims.DeviceID == "" || claims.MachineID == "" {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

func GenerateRefreshToken() (plainToken, tokenHash string, expiresAt time.Time) {
	plainToken = utils.GenerateRandomString(64)
	tokenHash = cryptopkg.SHA256HashString(plainToken)
	expiresAt = time.Now().Add(getRefreshTokenTTL())
	return
}

func GetAccessTokenExpireSeconds() int {
	cfg := config.Get()
	if cfg == nil || cfg.Security.ClientAccessTokenExpireSeconds <= 0 {
		return defaultClientAccessTokenExpireSeconds
	}
	return cfg.Security.ClientAccessTokenExpireSeconds
}

func GetRefreshTokenExpireSeconds() int {
	cfg := config.Get()
	if cfg == nil || cfg.Security.ClientRefreshTokenExpireSeconds <= 0 {
		return defaultClientRefreshTokenExpireSeconds
	}
	return cfg.Security.ClientRefreshTokenExpireSeconds
}

func getAccessTokenTTL() time.Duration {
	return time.Duration(GetAccessTokenExpireSeconds()) * time.Second
}

func getRefreshTokenTTL() time.Duration {
	return time.Duration(GetRefreshTokenExpireSeconds()) * time.Second
}

func getAccessTokenSecret() string {
	cfg := config.Get()
	if cfg == nil {
		return ""
	}

	if secret := strings.TrimSpace(cfg.Security.ClientAccessTokenSecret); secret != "" {
		return secret
	}

	return cfg.JWT.Secret
}
