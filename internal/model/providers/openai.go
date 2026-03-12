package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"gophermind/internal/config"
	"gophermind/internal/core/model"
)

// OpenAIProvider 对接 OpenAI 兼容接口。
type OpenAIProvider struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
	logger     *zap.Logger

	mu          sync.Mutex
	failures    int
	lastFailure time.Time
}

// NewOpenAIProvider 构建 OpenAIProvider。
func NewOpenAIProvider(cfg config.ModelConfig, logger *zap.Logger) *OpenAIProvider {
	return &OpenAIProvider{
		baseURL: strings.TrimRight(cfg.OpenAIBaseURL, "/"),
		apiKey:  cfg.OpenAIAPIKey,
		model:   cfg.OpenAIModel,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
		logger: logger,
	}
}

func (p *OpenAIProvider) Name() string { return "openai" }

// Generate 调用模型生成完整文本。
func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, model.Usage, error) {
	if p.isCircuitOpen() {
		return "", model.Usage{}, errCircuitOpen
	}

	// 未配置 API Key 时，返回可测试的本地降级结果。
	if p.apiKey == "" {
		answer := "[openai-mock] " + prompt
		return answer, model.Usage{Provider: p.Name(), InputTokens: len(prompt) / 4, OutputTokens: len(answer) / 4}, nil
	}

	var answer string
	err := withRetry(ctx, 2, 500*time.Millisecond, func(callCtx context.Context) error {
		reqBody := map[string]any{
			"model": p.model,
			"messages": []map[string]string{
				{"role": "user", "content": prompt},
			},
			"temperature": 0.2,
		}
		buf := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
			return err
		}

		req, err := http.NewRequestWithContext(callCtx, http.MethodPost, p.baseURL+"/chat/completions", buf)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := p.httpClient.Do(req)
		if err != nil {
			p.markFailure()
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 300 {
			p.markFailure()
			return fmt.Errorf("openai status %d", resp.StatusCode)
		}

		var parsed struct {
			Choices []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			} `json:"choices"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
			p.markFailure()
			return err
		}
		if len(parsed.Choices) == 0 {
			p.markFailure()
			return fmt.Errorf("openai empty choices")
		}
		answer = parsed.Choices[0].Message.Content
		return nil
	})
	if err != nil {
		return "", model.Usage{}, err
	}
	p.resetFailure()
	return answer, model.Usage{Provider: p.Name(), InputTokens: len(prompt) / 4, OutputTokens: len(answer) / 4}, nil
}

// GenerateStream 用假流式输出保持接口一致性。
func (p *OpenAIProvider) GenerateStream(ctx context.Context, prompt string, onToken func(string) error) (string, model.Usage, error) {
	answer, usage, err := p.Generate(ctx, prompt)
	if err != nil {
		return "", model.Usage{}, err
	}
	for _, token := range splitToTokens(answer) {
		if err := onToken(token); err != nil {
			return "", model.Usage{}, err
		}
	}
	return answer, usage, nil
}

func (p *OpenAIProvider) isCircuitOpen() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.failures >= 5 && time.Since(p.lastFailure) < 60*time.Second
}

func (p *OpenAIProvider) markFailure() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures++
	p.lastFailure = time.Now()
}

func (p *OpenAIProvider) resetFailure() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.failures = 0
}
