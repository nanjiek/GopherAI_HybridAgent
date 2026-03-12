package service

import (
	"context"
	"time"

	"gophermind/internal/core/model"
	"gophermind/pkg/contracts/events"
)

// SessionRepository 定义 MySQL 存储能力。
type SessionRepository interface {
	CreateSessionWithFirstMessage(ctx context.Context, userID string, title string, question string, requestID string) (model.Session, error)
	AppendUserMessage(ctx context.Context, userID string, sessionID string, question string, requestID string) error
	AppendAssistantMessage(ctx context.Context, userID string, sessionID string, answer string, requestID string, provider string, modelName string) error
	GetSession(ctx context.Context, userID string, sessionID string) (model.Session, error)
	ListMessages(ctx context.Context, userID string, sessionID string) ([]model.Message, error)
}

// SessionCache 定义 Redis 与退化缓存能力。
type SessionCache interface {
	GetSummary(ctx context.Context, userID string, sessionID string) (string, bool, error)
	SetSummary(ctx context.Context, userID string, sessionID string, summary string, ttl time.Duration) error
	AppendStreamChunk(ctx context.Context, requestID string, chunk string, ttl time.Duration) error
	GetStreamChunks(ctx context.Context, requestID string) ([]string, error)
	IsIdempotent(ctx context.Context, consumer string, messageID string) (bool, error)
	MarkIdempotent(ctx context.Context, consumer string, messageID string, ttl time.Duration) error
	MarkDegraded(err error)
	IsDegraded() bool
}

// ModelProvider 是具体模型提供方实现接口。
type ModelProvider interface {
	Name() string
	Generate(ctx context.Context, prompt string) (string, model.Usage, error)
	GenerateStream(ctx context.Context, prompt string, onToken func(string) error) (string, model.Usage, error)
}

// ModelRouter 负责优先级路由与回退。
type ModelRouter interface {
	Get(modelType string) (ModelProvider, error)
	GenerateWithFallback(ctx context.Context, modelType string, prompt string) (string, model.Usage, error)
	GenerateStreamWithFallback(ctx context.Context, modelType string, prompt string, onToken func(string) error) (string, model.Usage, error)
}

// RAGClient 定义 Python RAG 服务接口。
type RAGClient interface {
	Embed(ctx context.Context, text string) ([]float64, error)
	Retrieve(ctx context.Context, userID string, query string, topK int) ([]model.RAGDocument, error)
	Rerank(ctx context.Context, query string, docs []model.RAGDocument, topN int) ([]model.RAGDocument, error)
	KnowledgeGraphPlaceholder(ctx context.Context, query string) (string, error)
}

// QueueProducer 定义 RabbitMQ 生产行为。
type QueueProducer interface {
	PublishTask(ctx context.Context, message events.TaskMessage) error
	PublishResult(ctx context.Context, message events.ResultMessage) error
	PublishRetryTask(ctx context.Context, message events.TaskMessage) error
	PublishDLQTask(ctx context.Context, message events.TaskMessage) error
}

// AuthRepository 定义认证数据访问能力。
type AuthRepository interface {
	CreateUser(ctx context.Context, username string, passwordHash string, role string) (model.AuthUser, error)
	GetUserByUsername(ctx context.Context, username string) (model.AuthUser, error)
	SaveRefreshToken(ctx context.Context, userID uint64, tokenJTI string, tokenHash string, deviceID string, expiresAt time.Time) error
	GetActiveRefreshToken(ctx context.Context, tokenJTI string, tokenHash string) (model.RefreshTokenRecord, error)
	RevokeRefreshToken(ctx context.Context, tokenJTI string) error
	RevokeAllRefreshTokens(ctx context.Context, userID uint64) error
	IsNotFound(err error) bool
}

// ConsumerInboxRepository 定义 MQ 幂等落库能力。
type ConsumerInboxRepository interface {
	BeginProcess(ctx context.Context, consumer string, messageID string, retryCount int) (bool, error)
	MarkSucceeded(ctx context.Context, consumer string, messageID string) error
	MarkFailed(ctx context.Context, consumer string, messageID string, retryCount int, lastErr string) error
	MarkDead(ctx context.Context, consumer string, messageID string, retryCount int, lastErr string) error
}
