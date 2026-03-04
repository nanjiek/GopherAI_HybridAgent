package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type MainConfig struct {
	Port    int    `toml:"port"`
	AppName string `toml:"appName"`
	Host    string `toml:"host"`
}

type EmailConfig struct {
	Authcode string `toml:"authcode"`
	Email    string `toml:"email"`
}

type RedisConfig struct {
	RedisPort     int    `toml:"port"`
	RedisDb       int    `toml:"db"`
	RedisHost     string `toml:"host"`
	RedisPassword string `toml:"password"`
}

type MysqlConfig struct {
	MysqlPort         int    `toml:"port"`
	MysqlHost         string `toml:"host"`
	MysqlUser         string `toml:"user"`
	MysqlPassword     string `toml:"password"`
	MysqlDatabaseName string `toml:"databaseName"`
	MysqlCharset      string `toml:"charset"`
}

type JwtConfig struct {
	ExpireDuration int    `toml:"expire_duration"`
	Issuer         string `toml:"issuer"`
	Subject        string `toml:"subject"`
	Key            string `toml:"key"`
}

type Rabbitmq struct {
	RabbitmqPort     int    `toml:"port"`
	RabbitmqHost     string `toml:"host"`
	RabbitmqUsername string `toml:"username"`
	RabbitmqPassword string `toml:"password"`
	RabbitmqVhost    string `toml:"vhost"`
}

type RagModelConfig struct {
	RagEmbeddingModel string `toml:"embeddingModel"`
	RagChatModelName  string `toml:"chatModelName"`
	RagDocDir         string `toml:"docDir"`
	RagBaseUrl        string `toml:"baseUrl"`
	RagDimension      int    `toml:"dimension"`
}

type VoiceServiceConfig struct {
	VoiceServiceApiKey    string `toml:"voiceServiceApiKey"`
	VoiceServiceSecretKey string `toml:"voiceServiceSecretKey"`
}

type OpenAIConfig struct {
	APIKey    string `toml:"apiKey"`
	BaseURL   string `toml:"baseUrl"`
	ModelName string `toml:"modelName"`
}

type KimiConfig struct {
	APIKey    string `toml:"apiKey"`
	BaseURL   string `toml:"baseUrl"`
	ModelName string `toml:"modelName"`
}

type NewsConfig struct {
	Enable          bool   `toml:"enable"`
	APIKey          string `toml:"apiKey"`
	BaseURL         string `toml:"baseUrl"`
	DefaultLanguage string `toml:"defaultLanguage"`
	DefaultPageSize int    `toml:"defaultPageSize"`
	FastPollMinutes int    `toml:"fastPollMinutes"`
	NormalPollMins  int    `toml:"normalPollMinutes"`
	DeepPollMins    int    `toml:"deepPollMinutes"`
	MaxWorkers      int    `toml:"maxWorkers"`
}

type MCPConfig struct {
	BaseURL string `toml:"baseUrl"`
}

type ImageConfig struct {
	ModelPath string `toml:"modelPath"`
	LabelPath string `toml:"labelPath"`
	InputH    int    `toml:"inputH"`
	InputW    int    `toml:"inputW"`
}

type Config struct {
	EmailConfig        `toml:"emailConfig"`
	RedisConfig        `toml:"redisConfig"`
	MysqlConfig        `toml:"mysqlConfig"`
	JwtConfig          `toml:"jwtConfig"`
	MainConfig         `toml:"mainConfig"`
	Rabbitmq           `toml:"rabbitmqConfig"`
	RagModelConfig     `toml:"ragModelConfig"`
	VoiceServiceConfig `toml:"voiceServiceConfig"`
	OpenAIConfig       `toml:"openaiConfig"`
	KimiConfig         `toml:"kimiConfig"`
	NewsConfig         `toml:"newsConfig"`
	MCPConfig          `toml:"mcpConfig"`
	ImageConfig        `toml:"imageConfig"`
}

type RedisKeyConfig struct {
	CaptchaPrefix   string
	IndexName       string
	IndexNamePrefix string
}

var DefaultRedisKeyConfig = RedisKeyConfig{
	CaptchaPrefix:   "captcha:%s",
	IndexName:       "rag_docs:%s:idx",
	IndexNamePrefix: "rag_docs:%s:",
}

var config *Config

func InitConfig() error {
	if _, err := toml.DecodeFile("config/config.toml", config); err != nil {
		return err
	}
	if err := config.Validate(); err != nil {
		return err
	}
	return nil
}

func GetConfig() *Config {
	if config == nil {
		config = new(Config)
		if err := InitConfig(); err != nil {
			log.Fatal(err.Error())
		}
	}
	return config
}

func (c *Config) Validate() error {
	if c == nil {
		return errors.New("config is nil")
	}

	if c.MainConfig.Port <= 0 {
		return fmt.Errorf("mainConfig.port must be > 0")
	}
	if strings.TrimSpace(c.MainConfig.Host) == "" {
		return fmt.Errorf("mainConfig.host is required")
	}

	if strings.TrimSpace(c.MysqlHost) == "" || c.MysqlPort <= 0 || strings.TrimSpace(c.MysqlDatabaseName) == "" {
		return fmt.Errorf("mysqlConfig.host/port/databaseName are required")
	}

	if strings.TrimSpace(c.JwtConfig.Key) == "" {
		return fmt.Errorf("jwtConfig.key is required")
	}

	if strings.TrimSpace(c.RagModelConfig.RagBaseUrl) == "" || strings.TrimSpace(c.RagModelConfig.RagChatModelName) == "" {
		return fmt.Errorf("ragModelConfig.baseUrl/chatModelName are required")
	}
	if c.RagModelConfig.RagDimension <= 0 {
		return fmt.Errorf("ragModelConfig.dimension must be > 0")
	}

	if strings.TrimSpace(c.ImageConfig.ModelPath) == "" || strings.TrimSpace(c.ImageConfig.LabelPath) == "" {
		return fmt.Errorf("imageConfig.modelPath/labelPath are required")
	}
	if c.ImageConfig.InputH <= 0 || c.ImageConfig.InputW <= 0 {
		return fmt.Errorf("imageConfig.inputH/inputW must be > 0")
	}

	if strings.TrimSpace(c.MCPConfig.BaseURL) == "" {
		return fmt.Errorf("mcpConfig.baseUrl is required")
	}

	openAIKey := firstNonEmpty(os.Getenv("OPENAI_API_KEY"), c.OpenAIConfig.APIKey)
	kimiKey := firstNonEmpty(os.Getenv("KIMI_API_KEY"), c.KimiConfig.APIKey)
	if openAIKey == "" && kimiKey == "" {
		return fmt.Errorf("at least one provider key is required: OPENAI_API_KEY or KIMI_API_KEY")
	}

	if c.NewsConfig.DefaultPageSize <= 0 {
		c.NewsConfig.DefaultPageSize = 10
	}
	if c.NewsConfig.FastPollMinutes <= 0 {
		c.NewsConfig.FastPollMinutes = 5
	}
	if c.NewsConfig.NormalPollMins <= 0 {
		c.NewsConfig.NormalPollMins = 15
	}
	if c.NewsConfig.DeepPollMins <= 0 {
		c.NewsConfig.DeepPollMins = 60
	}
	if c.NewsConfig.MaxWorkers <= 0 {
		c.NewsConfig.MaxWorkers = 5
	}
	if strings.TrimSpace(c.NewsConfig.DefaultLanguage) == "" {
		c.NewsConfig.DefaultLanguage = "zh"
	}
	if c.NewsConfig.Enable {
		newsKey := firstNonEmpty(os.Getenv("NEWS_API_KEY"), c.NewsConfig.APIKey)
		if newsKey == "" {
			return fmt.Errorf("newsConfig.enable=true but NEWS_API_KEY/newsConfig.apiKey is empty")
		}
		if strings.TrimSpace(c.NewsConfig.BaseURL) == "" {
			return fmt.Errorf("newsConfig.enable=true but newsConfig.baseUrl is empty")
		}
	}

	return nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
