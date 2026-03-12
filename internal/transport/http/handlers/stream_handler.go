package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"gophermind/internal/core/model"
	"gophermind/internal/core/service"
	httpcontracts "gophermind/pkg/contracts/http"
)

// StreamHandler 处理 /stream/:session，并进行 SSE/WebSocket 协商。
type StreamHandler struct {
	svc      *service.StreamService
	logger   *zap.Logger
	upgrader websocket.Upgrader
}

// NewStreamHandler 构建 StreamHandler。
func NewStreamHandler(svc *service.StreamService, logger *zap.Logger) *StreamHandler {
	return &StreamHandler{
		svc:    svc,
		logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
	}
}

// Handle 按协议处理流式输出。
func (h *StreamHandler) Handle(c *gin.Context) {
	if websocket.IsWebSocketUpgrade(c.Request) {
		h.handleWS(c)
		return
	}
	h.handleSSE(c)
}

func (h *StreamHandler) handleSSE(c *gin.Context) {
	userID := c.GetString("user_id")
	sessionID := c.Param("session")
	question := c.Query("q")
	modelType := c.DefaultQuery("model_type", "auto")
	useRAG := parseBool(c.DefaultQuery("use_rag", "false"))

	if strings.TrimSpace(question) == "" {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40003, "missing query parameter q"))
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Flush()

	emit := func(event string, payload interface{}) error {
		b, _ := json.Marshal(payload)
		_, err := c.Writer.WriteString(fmt.Sprintf("event: %s\ndata: %s\n\n", event, string(b)))
		if err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}

	out, err := h.svc.Stream(c.Request.Context(), model.QueryInput{
		UserID:    userID,
		SessionID: sessionID,
		Question:  question,
		ModelType: modelType,
		UseRAG:    useRAG,
	}, func(token string) error {
		return emit("token", gin.H{"delta": token})
	})
	if err != nil {
		_ = emit("error", gin.H{"message": err.Error()})
		return
	}

	_ = emit("done", gin.H{
		"request_id": out.RequestID,
		"session_id": out.SessionID,
		"usage": gin.H{
			"provider":      out.Usage.Provider,
			"input_tokens":  out.Usage.InputTokens,
			"output_tokens": out.Usage.OutputTokens,
		},
	})
}

func (h *StreamHandler) handleWS(c *gin.Context) {
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	userID := c.GetString("user_id")
	sessionID := c.Param("session")
	question := c.Query("q")
	modelType := c.DefaultQuery("model_type", "auto")
	useRAG := parseBool(c.DefaultQuery("use_rag", "false"))
	if strings.TrimSpace(question) == "" {
		_ = conn.WriteJSON(gin.H{"type": "error", "message": "missing query parameter q"})
		return
	}

	seq := 0
	out, err := h.svc.Stream(c.Request.Context(), model.QueryInput{
		UserID:    userID,
		SessionID: sessionID,
		Question:  question,
		ModelType: modelType,
		UseRAG:    useRAG,
	}, func(token string) error {
		seq++
		return conn.WriteJSON(gin.H{
			"type":  "token",
			"delta": token,
			"seq":   seq,
		})
	})
	if err != nil {
		_ = conn.WriteJSON(gin.H{"type": "error", "message": err.Error()})
		return
	}
	_ = conn.WriteJSON(gin.H{
		"type":       "done",
		"request_id": out.RequestID,
		"session_id": out.SessionID,
		"usage": gin.H{
			"provider":      out.Usage.Provider,
			"input_tokens":  out.Usage.InputTokens,
			"output_tokens": out.Usage.OutputTokens,
		},
	})
}

func parseBool(v string) bool {
	ok, _ := strconv.ParseBool(v)
	return ok
}
