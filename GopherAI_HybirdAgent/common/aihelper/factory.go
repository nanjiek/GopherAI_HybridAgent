package aihelper

import (
	"context"
	"fmt"
	"sync"
)

type ModelCreator func(ctx context.Context, config map[string]interface{}) (AIModel, error)

type AIModelFactory struct {
	creators map[string]ModelCreator
}

var (
	globalFactory *AIModelFactory
	factoryOnce   sync.Once
)

func GetGlobalFactory() *AIModelFactory {
	factoryOnce.Do(func() {
		globalFactory = &AIModelFactory{
			creators: make(map[string]ModelCreator),
		}
		globalFactory.registerCreators()
	})
	return globalFactory
}

func (f *AIModelFactory) registerCreators() {
	f.creators["1"] = func(ctx context.Context, cfg map[string]interface{}) (AIModel, error) {
		return NewOpenAIModel(ctx)
	}

	f.creators["2"] = func(ctx context.Context, cfg map[string]interface{}) (AIModel, error) {
		username, ok := cfg["username"].(string)
		if !ok {
			return nil, fmt.Errorf("RAG model requires username")
		}
		return NewAliRAGModel(ctx, username)
	}

	f.creators["3"] = func(ctx context.Context, cfg map[string]interface{}) (AIModel, error) {
		username, ok := cfg["username"].(string)
		if !ok {
			return nil, fmt.Errorf("MCP model requires username")
		}
		return NewMCPModel(ctx, username)
	}

	f.creators["4"] = func(ctx context.Context, cfg map[string]interface{}) (AIModel, error) {
		baseURL, _ := cfg["baseURL"].(string)
		modelName, ok := cfg["modelName"].(string)
		if !ok {
			return nil, fmt.Errorf("Ollama model requires modelName")
		}
		return NewOllamaModel(ctx, baseURL, modelName)
	}

	f.creators["5"] = func(ctx context.Context, cfg map[string]interface{}) (AIModel, error) {
		return NewKimiModel(ctx)
	}
}

func (f *AIModelFactory) CreateAIModel(ctx context.Context, modelType string, cfg map[string]interface{}) (AIModel, error) {
	creator, ok := f.creators[modelType]
	if !ok {
		return nil, fmt.Errorf("unsupported model type: %s", modelType)
	}
	return creator(ctx, cfg)
}

func (f *AIModelFactory) CreateAIHelper(ctx context.Context, modelType string, sessionID string, cfg map[string]interface{}) (*AIHelper, error) {
	model, err := f.CreateAIModel(ctx, modelType, cfg)
	if err != nil {
		return nil, err
	}
	return NewAIHelper(model, sessionID), nil
}

func (f *AIModelFactory) RegisterModel(modelType string, creator ModelCreator) {
	f.creators[modelType] = creator
}
