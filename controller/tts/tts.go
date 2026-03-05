package tts

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nanjiek/GopherMind/common/code"
	"github.com/nanjiek/GopherMind/common/tts"
	"github.com/nanjiek/GopherMind/controller"
)

type (
	TTSRequest struct {
		Text string `json:"text,omitempty"`
	}
	TTSResponse struct {
		TaskID string `json:"task_id,omitempty"`
		controller.Response
	}
	QueryTTSResponse struct {
		TaskID     string `json:"task_id,omitempty"`
		TaskStatus string `json:"task_status,omitempty"`
		TaskResult string `json:"task_result,omitempty"`
		controller.Response
	}
)

type TTSServices struct {
	ttsService *tts.TTSService
}

func NewTTSServices() *TTSServices {
	return &TTSServices{ttsService: tts.NewTTSService()}
}

func CreateTTSTask(c *gin.Context) {
	ttsSvc := NewTTSServices()
	req := new(TTSRequest)
	res := new(TTSResponse)

	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	if req.Text == "" {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	// comment cleaned
	if err != nil {
		c.JSON(http.StatusOK, res.CodeOf(code.TTSFail))
		return
	}

	res.Success()
	res.TaskID = taskID
	c.JSON(http.StatusOK, res)
}

func QueryTTSTask(c *gin.Context) {
	ttsSvc := NewTTSServices()
	res := new(QueryTTSResponse)
	taskID := c.Query("task_id")
	if taskID == "" {
		c.JSON(http.StatusOK, res.CodeOf(code.CodeInvalidParams))
		return
	}

	qResp, err := ttsSvc.ttsService.QueryTTSFull(c, taskID)
	if err != nil {
		log.Println("鏌ヨ璇煶鍚堟垚浠诲姟澶辫触:", err.Error())
		c.JSON(http.StatusOK, res.CodeOf(code.TTSFail))
		return
	}

	if len(qResp.TasksInfo) == 0 {
		c.JSON(http.StatusOK, res.CodeOf(code.TTSFail))
		return
	}

	res.Success()
	res.TaskID = qResp.TasksInfo[0].TaskID

	// comment cleaned
	if qResp.TasksInfo[0].TaskResult != nil {
		res.TaskResult = qResp.TasksInfo[0].TaskResult.SpeechURL
	}
	res.TaskStatus = qResp.TasksInfo[0].TaskStatus
	c.JSON(http.StatusOK, res)
}
