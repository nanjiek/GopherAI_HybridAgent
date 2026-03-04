package router

import (
	"github.com/nanjiek/GopherAI_HybridAgent/controller/file"

	"github.com/gin-gonic/gin"
)

func FileRouter(r *gin.RouterGroup) {
	r.POST("/upload", file.UploadRagFile)
}
