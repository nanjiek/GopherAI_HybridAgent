package image

import (
	"github.com/nanjiek/GopherAI_HybridAgent/common/code"
	"github.com/nanjiek/GopherAI_HybridAgent/controller"
	"github.com/nanjiek/GopherAI_HybridAgent/service/image"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type (
	RecognizeImageResponse struct {
		ClassName string `json:"class_name,omitempty"` // AI回答
		controller.Response
	}
)

func RecognizeImage(c *gin.Context) {
	res := new(RecognizeImageResponse)
	file, err := c.FormFile("image")
	if err != nil {
		log.Println("FormFile fail ", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	className, err := image.RecognizeImage(file)
	if err != nil {
		log.Println("RecognizeImage fail ", err)
		c.JSON(http.StatusOK, res.CodeOf(code.CodeServerBusy))
		return
	}

	res.Success()
	res.ClassName = className
	c.JSON(http.StatusOK, res)
}
