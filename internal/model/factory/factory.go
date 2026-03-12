package factory

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"gophermind/internal/core/model"
	"gophermind/internal/core/service"
)

// ModelFactory 提供多模型路由与回退策略。
type ModelFactory struct {
	providers map[string]service.ModelProvider
	logger    *zap.Logger
}

// NewModelFactory 注册 OpenAI/Ollama/BGE 提供方。
func NewModelFactory(openai service.ModelProvider, ollama service.ModelProvider, bge service.ModelProvider, logger *zap.Logger) *ModelFactory {
	return &ModelFactory{
		providers: map[string]service.ModelProvider{
			"openai": openai,
			"ollama": ollama,
			"bge":    bge,
			"auto":   openai,
		},
		logger: logger,
	}
}

// Get 返回指定 model type 对应 provider。
func (f *ModelFactory) Get(modelType string) (service.ModelProvider, error) {
	if modelType == "" {
		modelType = "auto"
	}
	p, ok := f.providers[modelType]
	if !ok {
		return nil, fmt.Errorf("unknown model type: %s", modelType)
	}
	return p, nil
}

// GenerateWithFallback 执行 openai -> ollama 回退策略。
func (f *ModelFactory) GenerateWithFallback(ctx context.Context, modelType string, prompt string) (string, model.Usage, error) {
	first, err := f.Get(modelType)
	if err != nil {
		return "", model.Usage{}, err
	}
	answer, usage, err := first.Generate(ctx, prompt)
	if err == nil {
		return answer, usage, nil
	}
	if first.Name() != "openai" {
		return "", model.Usage{}, err
	}
	f.logger.Warn("primary model failed, fallback to ollama", zap.Error(err))
	second, getErr := f.Get("ollama")
	if getErr != nil {
		return "", model.Usage{}, err
	}
	return second.Generate(ctx, prompt)
}

// GenerateStreamWithFallback 执行流式回退策略。
func (f *ModelFactory) GenerateStreamWithFallback(ctx context.Context, modelType string, prompt string, onToken func(string) error) (string, model.Usage, error) {
	first, err := f.Get(modelType)
	if err != nil {
		return "", model.Usage{}, err
	}
	answer, usage, err := first.GenerateStream(ctx, prompt, onToken)
	if err == nil {
		return answer, usage, nil
	}
	if first.Name() != "openai" {
		return "", model.Usage{}, err
	}
	f.logger.Warn("primary stream model failed, fallback to ollama", zap.Error(err))
	second, getErr := f.Get("ollama")
	if getErr != nil {
		return "", model.Usage{}, err
	}
	return second.GenerateStream(ctx, prompt, onToken)
}
