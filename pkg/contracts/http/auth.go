package httpcontracts

import "time"

// RegisterRequest 注册请求。
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=64"`
	Password string `json:"password" binding:"required,min=8,max=128"`
}

// LoginRequest 登录请求。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	DeviceID string `json:"device_id,omitempty"`
}

// RefreshRequest 刷新请求。
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	DeviceID     string `json:"device_id,omitempty"`
}

// LogoutRequest 注销请求。
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// TokenData 返回 token 对。
type TokenData struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}
