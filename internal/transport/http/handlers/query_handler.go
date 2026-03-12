package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"gophermind/internal/core/model"
	"gophermind/internal/core/service"
	httpcontracts "gophermind/pkg/contracts/http"
)

// QueryHandler 处理 /query。
type QueryHandler struct {
	svc    *service.QueryService
	logger *zap.Logger
}

// NewQueryHandler 构建 QueryHandler。
func NewQueryHandler(svc *service.QueryService, logger *zap.Logger) *QueryHandler {
	return &QueryHandler{svc: svc, logger: logger}
}

// Handle 执行同步问答。
func (h *QueryHandler) Handle(c *gin.Context) {
	var req httpcontracts.QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, httpcontracts.Err(40001, "invalid request body"))
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, httpcontracts.Err(40103, "missing user id"))
		return
	}

	out, err := h.svc.Query(c.Request.Context(), model.QueryInput{
		UserID:    userID,
		SessionID: req.SessionID,
		Question:  req.Question,
		ModelType: req.ModelType,
		UseRAG:    req.UseRAG,
	})
	if err != nil {
		if h.logger != nil {
			h.logger.Error("query failed", zap.Error(err))
		}
		c.JSON(http.StatusInternalServerError, httpcontracts.Err(50001, "query failed"))
		return
	}

	citations := make([]httpcontracts.CitationResponse, 0, len(out.Citations))
	for _, ct := range out.Citations {
		citations = append(citations, httpcontracts.CitationResponse{
			DocID:   ct.DocID,
			ChunkID: ct.ChunkID,
			Score:   ct.Score,
		})
	}

	c.JSON(http.StatusOK, httpcontracts.OK(httpcontracts.QueryData{
		SessionID: out.SessionID,
		Answer:    out.Answer,
		Citations: citations,
		Usage: httpcontracts.UsageResponse{
			Provider:     out.Usage.Provider,
			InputTokens:  out.Usage.InputTokens,
			OutputTokens: out.Usage.OutputTokens,
		},
		RequestID: out.RequestID,
	}))
}
