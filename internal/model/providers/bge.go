package providers

import (
	"context"
	"sort"
	"strings"
	"time"

	"go.uber.org/zap"

	"gophermind/internal/config"
	"gophermind/internal/core/model"
)

// BGEProvider 代表 reranker provider，实现统一工厂注册。
type BGEProvider struct {
	baseURL   string
	modelName string
	logger    *zap.Logger
}

// NewBGEProvider 构建 BGEProvider。
func NewBGEProvider(cfg config.ModelConfig, logger *zap.Logger) *BGEProvider {
	return &BGEProvider{
		baseURL:   cfg.BGEBaseURL,
		modelName: cfg.BGEModel,
		logger:    logger,
	}
}

func (p *BGEProvider) Name() string { return "bge" }

// Generate 保留统一接口实现，不作为主要生成模型使用。
func (p *BGEProvider) Generate(_ context.Context, prompt string) (string, model.Usage, error) {
	answer := "[bge-rerank-provider] " + prompt
	return answer, model.Usage{Provider: p.Name(), InputTokens: len(prompt) / 4, OutputTokens: len(answer) / 4}, nil
}

// GenerateStream 保留统一接口实现。
func (p *BGEProvider) GenerateStream(ctx context.Context, prompt string, onToken func(string) error) (string, model.Usage, error) {
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

// Rerank 提供简单重排钩子：按 query 命中词频排序。
func (p *BGEProvider) Rerank(_ context.Context, query string, docs []model.RAGDocument, topN int) ([]model.RAGDocument, error) {
	query = strings.ToLower(query)
	type scored struct {
		doc   model.RAGDocument
		score int
	}
	all := make([]scored, 0, len(docs))
	for _, d := range docs {
		cnt := strings.Count(strings.ToLower(d.Content), query)
		all = append(all, scored{doc: d, score: cnt})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].score == all[j].score {
			return all[i].doc.Score > all[j].doc.Score
		}
		return all[i].score > all[j].score
	})
	if topN <= 0 || topN > len(all) {
		topN = len(all)
	}
	out := make([]model.RAGDocument, 0, topN)
	for i := 0; i < topN; i++ {
		d := all[i].doc
		d.Score += float64(all[i].score) * 0.01
		out = append(out, d)
	}
	_ = time.Now()
	return out, nil
}
