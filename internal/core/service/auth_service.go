package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"gophermind/internal/security/token"
)

var (
	// ErrInvalidCredentials 表示用户名或密码错误。
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidRefreshToken 表示刷新令牌无效。
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
)

// AuthService 提供登录、刷新、注销能力。
type AuthService struct {
	repo   AuthRepository
	tokenM *token.Manager
}

// NewAuthService 构建 AuthService。
func NewAuthService(repo AuthRepository, tokenM *token.Manager) *AuthService {
	return &AuthService{
		repo:   repo,
		tokenM: tokenM,
	}
}

// Register 创建用户账号。
func (s *AuthService) Register(ctx context.Context, username string, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return ErrInvalidCredentials
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.repo.CreateUser(ctx, username, string(hash), "user")
	return err
}

// Login 登录并返回 access/refresh。
func (s *AuthService) Login(ctx context.Context, username string, password string, deviceID string) (token.TokenPair, error) {
	user, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if s.repo.IsNotFound(err) {
			return token.TokenPair{}, ErrInvalidCredentials
		}
		return token.TokenPair{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return token.TokenPair{}, ErrInvalidCredentials
	}

	pair, err := s.tokenM.GenerateTokenPair(user.Username, user.Role)
	if err != nil {
		return token.TokenPair{}, err
	}
	if err := s.repo.SaveRefreshToken(ctx, user.ID, pair.RefreshJTI, token.HashToken(pair.RefreshToken), deviceID, pair.RefreshExpiresAt); err != nil {
		return token.TokenPair{}, err
	}
	return pair, nil
}

// Refresh 刷新令牌，采用 refresh rotation。
func (s *AuthService) Refresh(ctx context.Context, refreshToken string, deviceID string) (token.TokenPair, error) {
	claims, err := s.tokenM.ParseRefreshToken(refreshToken)
	if err != nil {
		return token.TokenPair{}, ErrInvalidRefreshToken
	}
	hash := token.HashToken(refreshToken)
	record, err := s.repo.GetActiveRefreshToken(ctx, claims.ID, hash)
	if err != nil {
		if s.repo.IsNotFound(err) {
			return token.TokenPair{}, ErrInvalidRefreshToken
		}
		return token.TokenPair{}, err
	}
	if err := s.repo.RevokeRefreshToken(ctx, claims.ID); err != nil {
		return token.TokenPair{}, err
	}

	pair, err := s.tokenM.GenerateTokenPair(claims.UserID, claims.Role)
	if err != nil {
		return token.TokenPair{}, err
	}
	if err := s.repo.SaveRefreshToken(ctx, record.UserID, pair.RefreshJTI, token.HashToken(pair.RefreshToken), deviceID, pair.RefreshExpiresAt); err != nil {
		return token.TokenPair{}, err
	}
	return pair, nil
}

// Logout 吊销单个 refresh token。
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	claims, err := s.tokenM.ParseRefreshToken(refreshToken)
	if err != nil {
		return ErrInvalidRefreshToken
	}
	return s.repo.RevokeRefreshToken(ctx, claims.ID)
}
