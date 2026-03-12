package mysql

import "time"

// SessionModel 对应 sessions 表。
type SessionModel struct {
	ID            string         `gorm:"type:char(36);primaryKey"`
	UserID        string         `gorm:"size:64;index:idx_sessions_user_updated,priority:1;not null"`
	Title         string         `gorm:"size:255;not null"`
	ModelPref     string         `gorm:"size:32"`
	LastMessageAt *time.Time     `gorm:"index:idx_sessions_user_updated,priority:2,sort:desc"`
	CreatedAt     time.Time      `gorm:"autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	Messages      []MessageModel `gorm:"foreignKey:SessionID;references:ID"`
}

// MessageModel 对应 messages 表。
type MessageModel struct {
	ID        int64     `gorm:"primaryKey;autoIncrement"`
	SessionID string    `gorm:"type:char(36);index:idx_messages_session_time,priority:1;not null"`
	UserID    string    `gorm:"size:64;not null"`
	Role      string    `gorm:"size:16;not null"`
	Content   string    `gorm:"type:mediumtext;not null"`
	RequestID string    `gorm:"size:36;index"`
	Provider  string    `gorm:"size:64"`
	ModelName string    `gorm:"size:64"`
	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_messages_session_time,priority:2"`
}

// UserModel 对应 users 表，用于认证。
type UserModel struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement"`
	Username     string    `gorm:"size:64;uniqueIndex;not null"`
	PasswordHash string    `gorm:"size:255;not null"`
	Role         string    `gorm:"size:32;not null;default:user"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

// RefreshTokenModel 对应 refresh_tokens 表，保存刷新令牌哈希。
type RefreshTokenModel struct {
	ID          uint64     `gorm:"primaryKey;autoIncrement"`
	UserID      uint64     `gorm:"index;not null"`
	TokenJTI    string     `gorm:"size:64;uniqueIndex;not null"`
	TokenHash   string     `gorm:"size:128;not null"`
	DeviceID    string     `gorm:"size:128"`
	ExpiresAt   time.Time  `gorm:"index;not null"`
	RevokedAt   *time.Time `gorm:"index"`
	CreatedAt   time.Time  `gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime"`
}

// ConsumerInboxModel 对应 consumer_inbox 表，保证消息消费幂等落库。
type ConsumerInboxModel struct {
	ID         uint64     `gorm:"primaryKey;autoIncrement"`
	Consumer   string     `gorm:"size:64;not null;uniqueIndex:uk_consumer_message,priority:1"`
	MessageID  string     `gorm:"size:128;not null;uniqueIndex:uk_consumer_message,priority:2"`
	Status     string     `gorm:"size:32;not null;index"`
	RetryCount int        `gorm:"not null;default:0"`
	LastError  string     `gorm:"size:1024"`
	ProcessedAt *time.Time `gorm:"index"`
	CreatedAt  time.Time  `gorm:"autoCreateTime"`
	UpdatedAt  time.Time  `gorm:"autoUpdateTime"`
}
