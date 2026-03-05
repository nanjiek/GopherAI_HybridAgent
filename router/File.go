package router

import (
	"github.com/nanjiek/GopherMind/controller/file"
	"github.com/gin-gonic/gin"
)

func FileRouter(r *gin.RouterGroup) {
	r.POST("/upload", file.UploadRagFile)
}
