package token

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"gophermind/internal/config"
)

const (
	// TokenTypeAccess 表示短 TTL 访问令牌。
	TokenTypeAccess = "access"
	// TokenTypeRefresh 表示可轮换刷新令牌。
	TokenTypeRefresh = "refresh"
)

// Claims 定义 JWT 声明字段。
type Claims struct {
	UserID    string `json:"user_id"`
	Role      string `json:"role"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// TokenPair 表示登录/刷新返回的令牌对。
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	RefreshJTI       string
}

// Manager 提供 JWT 签发和校验能力。
type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

// NewManager 构建 JWT 管理器。
func NewManager(cfg config.AuthConfig) *Manager {
	return &Manager{
		accessSecret:  []byte(cfg.AccessSecret),
		refreshSecret: []byte(cfg.RefreshSecret),
		accessTTL:     cfg.AccessTTL,
		refreshTTL:    cfg.RefreshTTL,
	}
}

// GenerateTokenPair 签发 access+refresh。
func (m *Manager) GenerateTokenPair(userID string, role string) (TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(m.accessTTL)
	refreshExp := now.Add(m.refreshTTL)
	accessJTI := uuid.NewString()
	refreshJTI := uuid.NewString()

	accessClaims := Claims{
		UserID:    userID,
		Role:      role,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        accessJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
		},
	}
	refreshClaims := Claims{
		UserID:    userID,
		Role:      role,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ID:        refreshJTI,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
		},
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString(m.accessSecret)
	if err != nil {
		return TokenPair{}, err
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString(m.refreshSecret)
	if err != nil {
		return TokenPair{}, err
	}

	return TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		AccessExpiresAt:  accessExp,
		RefreshExpiresAt: refreshExp,
		RefreshJTI:       refreshJTI,
	}, nil
}

// ParseAccessToken 校验 access token。
func (m *Manager) ParseAccessToken(raw string) (*Claims, error) {
	return m.parse(raw, TokenTypeAccess, m.accessSecret)
}

// ParseRefreshToken 校验 refresh token。
func (m *Manager) ParseRefreshToken(raw string) (*Claims, error) {
	return m.parse(raw, TokenTypeRefresh, m.refreshSecret)
}

func (m *Manager) parse(raw string, expectedType string, secret []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	if claims.TokenType != expectedType {
		return nil, errors.New("invalid token type")
	}
	return claims, nil
}

// HashToken 用于持久化 refresh token 哈希，避免明文落库。
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
