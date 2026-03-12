package rag

// EmbedRequest 定义 embedding 请求。
type EmbedRequest struct {
	Text string `json:"text"`
}

// EmbedResponse 定义 embedding 响应。
type EmbedResponse struct {
	Vector []float64 `json:"vector"`
}

// RetrieveRequest 定义检索请求。
type RetrieveRequest struct {
	UserID string `json:"user_id"`
	Query  string `json:"query"`
	TopK   int    `json:"top_k"`
}

// RetrieveDoc 定义检索文档块。
type RetrieveDoc struct {
	DocID    string            `json:"doc_id"`
	ChunkID  string            `json:"chunk_id"`
	Content  string            `json:"content"`
	Score    float64           `json:"score"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// RetrieveResponse 定义检索响应。
type RetrieveResponse struct {
	Documents []RetrieveDoc `json:"documents"`
}

// RerankRequest 定义重排请求。
type RerankRequest struct {
	Query string        `json:"query"`
	Docs  []RetrieveDoc `json:"docs"`
	TopN  int           `json:"top_n"`
}

// RerankResponse 定义重排响应。
type RerankResponse struct {
	Documents []RetrieveDoc `json:"documents"`
}

// KGRequest 定义知识图谱占位请求。
type KGRequest struct {
	Query string `json:"query"`
}

// KGResponse 定义知识图谱占位响应。
type KGResponse struct {
	Context string `json:"context"`
}
