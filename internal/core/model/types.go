package model

import "time"

// Session 表示用户会话。
type Session struct {
	ID            string
	UserID        string
	Title         string
	ModelPref     string
	LastMessageAt time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Message 表示会话消息。
type Message struct {
	ID        int64
	SessionID string
	UserID    string
	Role      string
	Content   string
	RequestID string
	Provider  string
	ModelName string
	CreatedAt time.Time
}

// Citation 对应 RAG 引用片段。
type Citation struct {
	DocID   string
	ChunkID string
	Score   float64
}

// Usage 描述 token 使用。
type Usage struct {
	Provider     string
	InputTokens  int
	OutputTokens int
}

// QueryInput 是查询入口参数。
type QueryInput struct {
	UserID    string
	SessionID string
	Question  string
	ModelType string
	UseRAG    bool
}

// QueryOutput 是查询统一返回结构。
type QueryOutput struct {
	RequestID string
	SessionID string
	Answer    string
	Citations []Citation
	Usage     Usage
}

// RAGDocument 表示检索到的文档块。
type RAGDocument struct {
	DocID    string
	ChunkID  string
	Content  string
	Score    float64
	Metadata map[string]string
}

// AuthUser 是认证域用户模型。
type AuthUser struct {
	ID           uint64
	Username     string
	PasswordHash string
	Role         string
}

// RefreshTokenRecord 是刷新令牌落库模型。
type RefreshTokenRecord struct {
	UserID    uint64
	TokenJTI  string
	TokenHash string
	DeviceID  string
	ExpiresAt time.Time
}
