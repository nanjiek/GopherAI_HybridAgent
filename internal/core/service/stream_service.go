package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"gophermind/internal/core/model"
	"gophermind/internal/obs/metrics"
)

// StreamService 提供流式问答能力。
type StreamService struct {
	repo     SessionRepository
	sessions *SessionService
	router   ModelRouter
	rag      RAGClient
	cache    SessionCache
	logger   *zap.Logger
}

// NewStreamService 构建 StreamService。
func NewStreamService(repo SessionRepository, sessions *SessionService, router ModelRouter, rag RAGClient, cache SessionCache, logger *zap.Logger) *StreamService {
	return &StreamService{
		repo:     repo,
		sessions: sessions,
		router:   router,
		rag:      rag,
		cache:    cache,
		logger:   logger,
	}
}

// Stream 执行流式问答并逐 token 回调。
func (s *StreamService) Stream(ctx context.Context, in model.QueryInput, onToken func(string) error) (model.QueryOutput, error) {
	modelType := normalizedStreamModelType(in.ModelType)
	success := false
	defer func() {
		metrics.IncStreamRequest(success, modelType, in.UseRAG)
	}()

	requestID := uuid.NewString()
	start := time.Now()
	sessionID := in.SessionID
	if sessionID == "" {
		title := in.Question
		if len(title) > 64 {
			title = title[:64]
		}
		created, err := s.repo.CreateSessionWithFirstMessage(ctx, in.UserID, title, in.Question, requestID)
		if err != nil {
			return model.QueryOutput{}, err
		}
		sessionID = created.ID
	} else {
		if err := s.repo.AppendUserMessage(ctx, in.UserID, in.SessionID, in.Question, requestID); err != nil {
			return model.QueryOutput{}, err
		}
	}

	prompt, citations := s.buildPrompt(ctx, in.UserID, sessionID, in.Question, in.UseRAG)
	var streamErr error
	firstTokenObserved := false
	modelCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()

	answer, usage, err := s.router.GenerateStreamWithFallback(modelCtx, modelType, prompt, func(token string) error {
		if !firstTokenObserved {
			firstTokenObserved = true
			metrics.ObserveStreamFirstToken(time.Since(start))
		}
		if err := s.cache.AppendStreamChunk(ctx, requestID, token, 10*time.Minute); err != nil {
			s.logger.Warn("cache stream chunk failed", zap.Error(err))
		}
		if err := onToken(token); err != nil {
			streamErr = err
			return err
		}
		return nil
	})
	if err != nil {
		return model.QueryOutput{}, err
	}
	if streamErr != nil {
		return model.QueryOutput{}, streamErr
	}

	if err := s.repo.AppendAssistantMessage(ctx, in.UserID, sessionID, answer, requestID, usage.Provider, modelType); err != nil {
		return model.QueryOutput{}, err
	}
	_ = s.cache.SetSummary(ctx, in.UserID, sessionID, in.Question+"\n"+answer, 24*time.Hour)

	success = true
	return model.QueryOutput{
		RequestID: requestID,
		SessionID: sessionID,
		Answer:    answer,
		Citations: citations,
		Usage:     usage,
	}, nil
}

func (s *StreamService) buildPrompt(ctx context.Context, userID string, sessionID string, question string, useRAG bool) (string, []model.Citation) {
	summary, err := s.sessions.LoadSummary(ctx, userID, sessionID)
	if err != nil {
		s.logger.Warn("load summary failed", zap.Error(err))
	}
	base := "You are a helpful assistant.\n"
	if summary != "" {
		base += "Conversation summary:\n" + summary + "\n"
	}

	var docs []model.RAGDocument
	if useRAG {
		retrieved, err := s.rag.Retrieve(ctx, userID, question, 20)
		if err != nil {
			s.logger.Warn("rag retrieve failed", zap.Error(err))
		}
		if len(retrieved) > 0 {
			reranked, err := s.rag.Rerank(ctx, question, retrieved, 5)
			if err == nil {
				docs = reranked
			} else {
				docs = retrieved
			}
		}
		kg, _ := s.rag.KnowledgeGraphPlaceholder(ctx, question)
		if kg != "" {
			base += "Knowledge Graph Context:\n" + kg + "\n"
		}
	}

	citations := make([]model.Citation, 0, len(docs))
	if len(docs) > 0 {
		base += "Retrieved context:\n"
		for _, d := range docs {
			base += "- " + d.Content + "\n"
			citations = append(citations, model.Citation{
				DocID:   d.DocID,
				ChunkID: d.ChunkID,
				Score:   d.Score,
			})
		}
	}
	base += "\nUser question:\n" + question
	return base, citations
}

func normalizedStreamModelType(modelType string) string {
	modelType = strings.TrimSpace(strings.ToLower(modelType))
	if modelType == "" {
		return "auto"
	}
	return modelType
}
