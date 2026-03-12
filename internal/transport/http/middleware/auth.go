package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gophermind/internal/config"
	"gophermind/internal/security/token"
)

// Auth 提供 access token 鉴权，支持显式开关的开发旁路。
func Auth(cfg config.AuthConfig, tokenM *token.Manager, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.EnableDevBypass {
			userID := c.GetHeader("X-User-ID")
			if userID != "" {
				c.Set("user_id", userID)
				c.Set("role", "dev")
				c.Next()
				return
			}
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    40101,
				"message": "missing bearer token",
			})
			return
		}

		raw := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := tokenM.ParseAccessToken(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    40102,
				"message": "invalid access token",
			})
			return
		}
		if claims.UserID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code":    40104,
				"message": "missing user id in token",
			})
			return
		}

		if logger != nil {
			logger.Debug("auth passed",
				zap.String("user_id", claims.UserID),
				zap.String("role", claims.Role),
				zap.String("jti", claims.ID),
			)
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("token_jti", claims.ID)
		c.Next()
	}
}

// RequireRole 限制最小角色权限。
func RequireRole(allowedRoles ...string) gin.HandlerFunc {
	roleSet := make(map[string]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		roleSet[r] = struct{}{}
	}
	return func(c *gin.Context) {
		role := c.GetString("role")
		if _, ok := roleSet[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    40301,
				"message": "insufficient role",
			})
			return
		}
		c.Next()
	}
}
