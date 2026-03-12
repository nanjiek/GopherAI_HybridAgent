package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config 汇总服务运行配置，默认值与设计文档保持一致。
type Config struct {
	ServiceName string
	HTTP        HTTPConfig
	MySQL       MySQLConfig
	Redis       RedisConfig
	RabbitMQ    RabbitMQConfig
	Auth        AuthConfig
	Upload      UploadConfig
	Model       ModelConfig
	RAG         RAGConfig
}

type HTTPConfig struct {
	Address         string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

type MySQLConfig struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Addrs        []string
	Username     string
	Password     string
	PoolSize     int
	MinIdleConns int
}

type RabbitMQConfig struct {
	URL         string
	TaskQueue   string
	ResultQueue string
	RetryQueue1 string
	RetryQueue2 string
	RetryQueue3 string
	DLQQueue    string
	MaxRetry    int
	RetryDelay1 time.Duration
	RetryDelay2 time.Duration
	RetryDelay3 time.Duration
}

type AuthConfig struct {
	AccessSecret    string
	RefreshSecret   string
	AccessTTL       time.Duration
	RefreshTTL      time.Duration
	EnableDevBypass bool
}

type UploadConfig struct {
	Dir              string
	MaxFileSizeBytes int64
	AllowedExts      []string
}

type ModelConfig struct {
	OpenAIBaseURL string
	OpenAIAPIKey  string
	OpenAIModel   string
	OllamaBaseURL string
	OllamaModel   string
	BGEBaseURL    string
	BGEModel      string
}

type RAGConfig struct {
	PythonServiceURL string
}

// Load 从环境变量读取配置并提供保守默认值。
func Load() Config {
	return Config{
		ServiceName: getEnv("SERVICE_NAME", "gophermind-backend"),
		HTTP: HTTPConfig{
			Address:         getEnv("HTTP_ADDR", ":9090"),
			ReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second),
		},
		MySQL: MySQLConfig{
			DSN:             getEnv("MYSQL_DSN", "root:password@tcp(mysql:3306)/gophermind?charset=utf8mb4&parseTime=True&loc=Local"),
			MaxOpenConns:    getInt("MYSQL_MAX_OPEN_CONNS", 20),
			MaxIdleConns:    getInt("MYSQL_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getDuration("MYSQL_CONN_MAX_LIFETIME", 30*time.Minute),
		},
		Redis: RedisConfig{
			Addrs:        getStringSlice("REDIS_ADDRS", []string{"redis:6379"}),
			Username:     getEnv("REDIS_USERNAME", ""),
			Password:     getEnv("REDIS_PASSWORD", ""),
			PoolSize:     getInt("REDIS_POOL_SIZE", 200),
			MinIdleConns: getInt("REDIS_MIN_IDLE_CONNS", 10),
		},
		RabbitMQ: RabbitMQConfig{
			URL:         getEnv("RABBITMQ_URL", "amqp://guest:guest@rabbitmq:5672/"),
			TaskQueue:   getEnv("RABBITMQ_TASK_QUEUE", "task_queue"),
			ResultQueue: getEnv("RABBITMQ_RESULT_QUEUE", "result_queue"),
			RetryQueue1: getEnv("RABBITMQ_RETRY_QUEUE_1", "task_queue.retry.1"),
			RetryQueue2: getEnv("RABBITMQ_RETRY_QUEUE_2", "task_queue.retry.2"),
			RetryQueue3: getEnv("RABBITMQ_RETRY_QUEUE_3", "task_queue.retry.3"),
			DLQQueue:    getEnv("RABBITMQ_DLQ_QUEUE", "task_queue.dlq"),
			MaxRetry:    getInt("RABBITMQ_MAX_RETRY", 3),
			RetryDelay1: getDuration("RABBITMQ_RETRY_DELAY_1", 5*time.Second),
			RetryDelay2: getDuration("RABBITMQ_RETRY_DELAY_2", 30*time.Second),
			RetryDelay3: getDuration("RABBITMQ_RETRY_DELAY_3", 120*time.Second),
		},
		Auth: AuthConfig{
			AccessSecret:    getEnv("JWT_ACCESS_SECRET", getEnv("JWT_SECRET", "gophermind-dev-access-secret")),
			RefreshSecret:   getEnv("JWT_REFRESH_SECRET", getEnv("JWT_SECRET", "gophermind-dev-refresh-secret")),
			AccessTTL:       getDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:      getDuration("JWT_REFRESH_TTL", 168*time.Hour),
			EnableDevBypass: getBool("AUTH_DEV_BYPASS", false),
		},
		Upload: UploadConfig{
			Dir:              getEnv("UPLOAD_DIR", "./data/uploads"),
			MaxFileSizeBytes: getInt64("UPLOAD_MAX_FILE_SIZE_BYTES", 20*1024*1024),
			AllowedExts: getStringSlice("UPLOAD_ALLOWED_EXTS", []string{
				".txt", ".md", ".pdf", ".doc", ".docx", ".csv", ".json", ".png", ".jpg", ".jpeg", ".webp",
			}),
		},
		Model: ModelConfig{
			OpenAIBaseURL: getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			OpenAIAPIKey:  getEnv("OPENAI_API_KEY", ""),
			OpenAIModel:   getEnv("OPENAI_MODEL", "gpt-4.1-mini"),
			OllamaBaseURL: getEnv("OLLAMA_BASE_URL", "http://localhost:11434"),
			OllamaModel:   getEnv("OLLAMA_MODEL", "qwen2.5:7b"),
			BGEBaseURL:    getEnv("BGE_BASE_URL", "http://localhost:8001"),
			BGEModel:      getEnv("BGE_MODEL", "bge-reranker-v2-m3"),
		},
		RAG: RAGConfig{
			PythonServiceURL: getEnv("RAG_PYTHON_URL", "http://localhost:8000"),
		},
	}
}

func getEnv(key string, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

func getInt(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func getDuration(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return v
}

func getInt64(key string, fallback int64) int64 {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return fallback
	}
	return v
}

func getStringSlice(key string, fallback []string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	parts := strings.Split(raw, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	if len(res) == 0 {
		return fallback
	}
	return res
}

func getBool(key string, fallback bool) bool {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return v
}
