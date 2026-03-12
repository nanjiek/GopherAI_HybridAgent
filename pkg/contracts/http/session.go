package httpcontracts

import "time"

// SessionMessageResponse 对应会话历史项。
type SessionMessageResponse struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// SessionData 对应 GET /session/:id 的 data 字段。
type SessionData struct {
	SessionID string                   `json:"session_id"`
	Title     string                   `json:"title"`
	Messages  []SessionMessageResponse `json:"messages"`
}
