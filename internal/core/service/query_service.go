package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"gophermind/internal/core/model"
	"gophermind/internal/obs/metrics"
	"gophermind/pkg/contracts/events"
)

// QueryService 提供同步问答流程。
type QueryService struct {
	repo     SessionRepository
	sessions *SessionService
	router   ModelRouter
	rag      RAGClient
	queue    QueueProducer
	cache    SessionCache
	logger   *zap.Logger
}

// NewQueryService 构建 QueryService。
func NewQueryService(repo SessionRepository, sessions *SessionService, router ModelRouter, rag RAGClient, queue QueueProducer, cache SessionCache, logger *zap.Logger) *QueryService {
	return &QueryService{
		repo:     repo,
		sessions: sessions,
		router:   router,
		rag:      rag,
		queue:    queue,
		cache:    cache,
		logger:   logger,
	}
}

// Query 执行同步问答：写用户消息 -> RAG -> 模型生成 -> 持久化回复。
func (s *QueryService) Query(ctx context.Context, in model.QueryInput) (model.QueryOutput, error) {
	modelType := normalizedModelType(in.ModelType)
	success := false
	start := time.Now()
	defer func() {
		metrics.ObserveQueryLatency(time.Since(start))
		metrics.IncQueryRequest(success, modelType, in.UseRAG)
	}()

	requestID := uuid.NewString()
	jobID := uuid.NewString()

	sessionID, err := s.ensureSessionAndUserMessage(ctx, in, requestID)
	if err != nil {
		return model.QueryOutput{}, err
	}

	traceID := requestID
	_ = s.queue.PublishTask(ctx, events.TaskMessage{
		EventType:      "query.task",
		Version:        "v1",
		JobID:          jobID,
		IdempotencyKey: requestID,
		UserID:         in.UserID,
		SessionID:      sessionID,
		RequestID:      requestID,
		ModelType:      modelType,
		Question:       in.Question,
		UseRAG:         in.UseRAG,
		TraceID:        traceID,
		CreatedAt:      time.Now(),
	})

	prompt, citations := s.buildPrompt(ctx, in.UserID, sessionID, in.Question, in.UseRAG)

	modelCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	answer, usage, err := s.router.GenerateWithFallback(modelCtx, modelType, prompt)
	if err != nil {
		_ = s.queue.PublishResult(ctx, events.ResultMessage{
			EventType:      "query.result",
			Version:        "v1",
			JobID:          jobID,
			IdempotencyKey: requestID,
			RequestID:      requestID,
			SessionID:      sessionID,
			UserID:         in.UserID,
			Status:         "failed",
			Error:          err.Error(),
			TraceID:        traceID,
			CreatedAt:      time.Now(),
		})
		return model.QueryOutput{}, err
	}

	if err := s.repo.AppendAssistantMessage(ctx, in.UserID, sessionID, answer, requestID, usage.Provider, modelType); err != nil {
		return model.QueryOutput{}, err
	}

	_ = s.cache.SetSummary(ctx, in.UserID, sessionID, in.Question+"\n"+answer, 24*time.Hour)

	_ = s.queue.PublishResult(ctx, events.ResultMessage{
		EventType:      "query.result",
		Version:        "v1",
		JobID:          jobID,
		IdempotencyKey: requestID,
		RequestID:      requestID,
		SessionID:      sessionID,
		UserID:         in.UserID,
		Status:         "ok",
		Answer:         answer,
		Provider:       usage.Provider,
		TraceID:        traceID,
		CreatedAt:      time.Now(),
	})

	success = true
	return model.QueryOutput{
		RequestID: requestID,
		SessionID: sessionID,
		Answer:    answer,
		Citations: citations,
		Usage:     usage,
	}, nil
}

func (s *QueryService) ensureSessionAndUserMessage(ctx context.Context, in model.QueryInput, requestID string) (string, error) {
	title := in.Question
	if len(title) > 64 {
		title = title[:64]
	}

	if in.SessionID == "" {
		created, err := s.repo.CreateSessionWithFirstMessage(ctx, in.UserID, title, in.Question, requestID)
		if err != nil {
			return "", err
		}
		return created.ID, nil
	}

	if err := s.repo.AppendUserMessage(ctx, in.UserID, in.SessionID, in.Question, requestID); err != nil {
		return "", err
	}
	return in.SessionID, nil
}

func (s *QueryService) buildPrompt(ctx context.Context, userID string, sessionID string, question string, useRAG bool) (string, []model.Citation) {
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

func normalizedModelType(modelType string) string {
	modelType = strings.TrimSpace(strings.ToLower(modelType))
	if modelType == "" {
		return "auto"
	}
	return modelType
}
