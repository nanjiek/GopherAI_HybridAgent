package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gophermind/internal/core/service"
	httpcontracts "gophermind/pkg/contracts/http"
)

// SessionHandler 处理 /session/:id。
type SessionHandler struct {
	svc    *service.SessionService
	logger *zap.Logger
}

// NewSessionHandler 构建 SessionHandler。
func NewSessionHandler(svc *service.SessionService, logger *zap.Logger) *SessionHandler {
	return &SessionHandler{svc: svc, logger: logger}
}

// GetSession 返回会话和历史消息。
func (h *SessionHandler) GetSession(c *gin.Context) {
	sessionID := c.Param("id")
	userID := c.GetString("user_id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40002, "missing session id"))
		return
	}

	sess, msgs, err := h.svc.GetSessionWithMessages(c.Request.Context(), userID, sessionID)
	if err != nil {
		if h.logger != nil {
			h.logger.Error("get session failed", zap.Error(err))
		}
		c.JSON(http.StatusNotFound, httpcontracts.Err(40401, "session not found"))
		return
	}

	items := make([]httpcontracts.SessionMessageResponse, 0, len(msgs))
	for _, m := range msgs {
		items = append(items, httpcontracts.SessionMessageResponse{
			Role:      m.Role,
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, httpcontracts.OK(httpcontracts.SessionData{
		SessionID: sess.ID,
		Title:     sess.Title,
		Messages:  items,
	}))
}
