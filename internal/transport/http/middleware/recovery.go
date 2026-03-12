package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery 捕获 panic 并返回统一错误。
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if logger != nil {
			logger.Error("panic recovered", zap.Any("error", recovered))
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"code":    50000,
			"message": "internal server error",
		})
	})
}
