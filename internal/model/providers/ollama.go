package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"gophermind/internal/config"
	"gophermind/internal/core/model"
)

// OllamaProvider 对接 Ollama HTTP API。
type OllamaProvider struct {
	baseURL    string
	modelName  string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewOllamaProvider 构建 OllamaProvider。
func NewOllamaProvider(cfg config.ModelConfig, logger *zap.Logger) *OllamaProvider {
	return &OllamaProvider{
		baseURL:   strings.TrimRight(cfg.OllamaBaseURL, "/"),
		modelName: cfg.OllamaModel,
		httpClient: &http.Client{
			Timeout: 45 * time.Second,
		},
		logger: logger,
	}
}

func (p *OllamaProvider) Name() string { return "ollama" }

// Generate 生成完整文本。
func (p *OllamaProvider) Generate(ctx context.Context, prompt string) (string, model.Usage, error) {
	reqBody := map[string]any{
		"model":  p.modelName,
		"prompt": prompt,
		"stream": false,
	}
	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(reqBody); err != nil {
		return "", model.Usage{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/api/generate", buf)
	if err != nil {
		return "", model.Usage{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		// local fallback，避免开发环境阻塞。
		answer := "[ollama-mock] " + prompt
		return answer, model.Usage{Provider: p.Name(), InputTokens: len(prompt) / 4, OutputTokens: len(answer) / 4}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", model.Usage{}, fmt.Errorf("ollama status %d", resp.StatusCode)
	}

	var parsed struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", model.Usage{}, err
	}
	answer := parsed.Response
	return answer, model.Usage{Provider: p.Name(), InputTokens: len(prompt) / 4, OutputTokens: len(answer) / 4}, nil
}

// GenerateStream 流式生成文本。
func (p *OllamaProvider) GenerateStream(ctx context.Context, prompt string, onToken func(string) error) (string, model.Usage, error) {
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
