package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"gophermind/internal/obs/metrics"
)

// HTTPMetrics 记录 HTTP 请求业务指标。
func HTTPMetrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}
		metrics.ObserveHTTPRequest(
			c.Request.Method,
			path,
			strconv.Itoa(c.Writer.Status()),
			time.Since(start),
		)
	}
}
