package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	NewsSourceTypeAPI  = "api"
	NewsSourceTypeRSS  = "rss"
	NewsSourceTypeSite = "site"

	NewsSourceApprovalPending  = "pending"
	NewsSourceApprovalApproved = "approved"
	NewsSourceApprovalRejected = "rejected"

	NewsSourceHealthUnknown  = "unknown"
	NewsSourceHealthHealthy  = "healthy"
	NewsSourceHealthDegraded = "degraded"
	NewsSourceHealthDown     = "down"
)

type NewsSource struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Name           string         `gorm:"type:varchar(128);uniqueIndex;not null" json:"name"`
	SourceType     string         `gorm:"type:varchar(16);index;not null" json:"source_type"`
	BaseURL        string         `gorm:"type:varchar(1024);not null" json:"base_url"`
	Region         string         `gorm:"type:varchar(32);index" json:"region"`
	Language       string         `gorm:"type:varchar(16);index" json:"language"`
	PollProfile    string         `gorm:"type:varchar(16);default:'normal'" json:"poll_profile"` // fast/normal/deep
	PollInterval   int            `gorm:"default:15" json:"poll_interval"`
	Enabled        bool           `gorm:"default:true;index" json:"enabled"`
	ApprovalStatus string         `gorm:"type:varchar(16);default:'pending';index" json:"approval_status"`
	HealthStatus   string         `gorm:"type:varchar(16);default:'unknown';index" json:"health_status"`
	LastCheckedAt  *time.Time     `json:"last_checked_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

type NewsArticle struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	SourceID    uint           `gorm:"index;not null" json:"source_id"`
	Title       string         `gorm:"type:varchar(512);index;not null" json:"title"`
	URL         string         `gorm:"type:varchar(1024);uniqueIndex;not null" json:"url"`
	Author      string         `gorm:"type:varchar(128)" json:"author,omitempty"`
	Description string         `gorm:"type:text" json:"description,omitempty"`
	Content     string         `gorm:"type:longtext" json:"content,omitempty"`
	Language    string         `gorm:"type:varchar(16);index" json:"language"`
	Region      string         `gorm:"type:varchar(32);index" json:"region"`
	PublishedAt time.Time      `gorm:"index" json:"published_at"`
	Hash        string         `gorm:"type:varchar(64);index" json:"hash"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

type NewsEvent struct {
	ID                   uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	Title                string         `gorm:"type:varchar(512);index;not null" json:"title"`
	Summary              string         `gorm:"type:text" json:"summary,omitempty"`
	SourceDiversityScore float64        `gorm:"default:0" json:"source_diversity_score"`
	RegionDiversityScore float64        `gorm:"default:0" json:"region_diversity_score"`
	StanceBalanceScore   float64        `gorm:"default:0" json:"stance_balance_score"`
	LastAggregatedAt     *time.Time     `json:"last_aggregated_at,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`
}

type NewsEventArticle struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	EventID   uint      `gorm:"index;not null" json:"event_id"`
	ArticleID uint      `gorm:"index;not null" json:"article_id"`
	CreatedAt time.Time `json:"created_at"`
}

type NewsEventView struct {
	ID          uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	EventID     uint           `gorm:"index;not null" json:"event_id"`
	ArticleID   uint           `gorm:"index;not null" json:"article_id"`
	Stance      string         `gorm:"type:varchar(16);index;not null" json:"stance"` // support/oppose/neutral/uncertain
	Claim       string         `gorm:"type:text" json:"claim"`
	EvidenceURL string         `gorm:"type:varchar(1024)" json:"evidence_url,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
