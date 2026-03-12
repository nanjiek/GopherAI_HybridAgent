package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gophermind/internal/core/service"
	"gophermind/internal/obs/metrics"
	httpcontracts "gophermind/pkg/contracts/http"
)

// AuthHandler 负责认证相关接口。
type AuthHandler struct {
	svc    *service.AuthService
	logger *zap.Logger
}

// NewAuthHandler 构建 AuthHandler。
func NewAuthHandler(svc *service.AuthService, logger *zap.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, logger: logger}
}

// Register 注册账号。
func (h *AuthHandler) Register(c *gin.Context) {
	if h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, httpcontracts.Err(50301, "auth service unavailable"))
		return
	}
	var req httpcontracts.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40011, "invalid register request"))
		return
	}
	if err := h.svc.Register(c.Request.Context(), req.Username, req.Password); err != nil {
		if h.logger != nil {
			h.logger.Warn("register failed", zap.Error(err))
		}
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40012, "register failed"))
		return
	}
	c.JSON(http.StatusOK, httpcontracts.OK(gin.H{"registered": true}))
}

// Login 登录并返回 token 对。
func (h *AuthHandler) Login(c *gin.Context) {
	if h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, httpcontracts.Err(50301, "auth service unavailable"))
		return
	}
	var req httpcontracts.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40013, "invalid login request"))
		return
	}
	pair, err := h.svc.Login(c.Request.Context(), req.Username, req.Password, req.DeviceID)
	if err != nil {
		metrics.IncAuthLogin(false)
		if h.logger != nil {
			h.logger.Warn("login failed", zap.Error(err))
		}
		c.JSON(http.StatusUnauthorized, httpcontracts.Err(40111, "invalid credentials"))
		return
	}
	metrics.IncAuthLogin(true)
	c.JSON(http.StatusOK, httpcontracts.OK(httpcontracts.TokenData{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
	}))
}

// Refresh 刷新 token 对（refresh rotation）。
func (h *AuthHandler) Refresh(c *gin.Context) {
	if h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, httpcontracts.Err(50301, "auth service unavailable"))
		return
	}
	var req httpcontracts.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40014, "invalid refresh request"))
		return
	}
	pair, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken, req.DeviceID)
	if err != nil {
		metrics.IncAuthRefresh(false)
		if h.logger != nil {
			h.logger.Warn("refresh failed", zap.Error(err))
		}
		c.JSON(http.StatusUnauthorized, httpcontracts.Err(40112, "invalid refresh token"))
		return
	}
	metrics.IncAuthRefresh(true)
	c.JSON(http.StatusOK, httpcontracts.OK(httpcontracts.TokenData{
		AccessToken:      pair.AccessToken,
		RefreshToken:     pair.RefreshToken,
		AccessExpiresAt:  pair.AccessExpiresAt,
		RefreshExpiresAt: pair.RefreshExpiresAt,
	}))
}

// Logout 吊销 refresh token。
func (h *AuthHandler) Logout(c *gin.Context) {
	if h.svc == nil {
		c.JSON(http.StatusServiceUnavailable, httpcontracts.Err(50301, "auth service unavailable"))
		return
	}
	var req httpcontracts.LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40015, "invalid logout request"))
		return
	}
	if err := h.svc.Logout(c.Request.Context(), req.RefreshToken); err != nil {
		c.JSON(http.StatusUnauthorized, httpcontracts.Err(40113, "invalid refresh token"))
		return
	}
	c.JSON(http.StatusOK, httpcontracts.OK(gin.H{"logged_out": true}))
}
