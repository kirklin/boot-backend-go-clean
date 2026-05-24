package ai

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
)

// ProviderConfig holds provider-agnostic LLM connection settings.
// Field names use generic "AI" terminology to avoid leaking the
// underlying framework (Eino) into calling code.
type ProviderConfig struct {
	Model   string // e.g. "gpt-4o", "deepseek-chat"
	APIKey  string
	BaseURL string // optional – for OpenAI-compatible endpoints
}

// Provider holds the initialised AI components.
// It acts as the composition root for all AI infrastructure, similar to
// how database.Database works for the persistence layer.
//
// Downstream code (use-cases, controllers) should depend on
// model.ChatModel / model.ToolCallingChatModel interfaces – never on
// the concrete Provider type – to keep the domain free of framework deps.
type Provider struct {
	// ChatModel is the core Eino component.
	// Everything else (agents, chains, graphs) is built on top of it.
	ChatModel model.ToolCallingChatModel

	config *ProviderConfig
}

// NewProvider creates and initialises the Eino ChatModel backed by an
// OpenAI-compatible endpoint.
func NewProvider(ctx context.Context, cfg *ProviderConfig) (*Provider, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("ai: API key is required (set AI_API_KEY)")
	}
	if cfg.Model == "" {
		return nil, fmt.Errorf("ai: model name is required (set AI_MODEL)")
	}

	chatModelCfg := &einoopenai.ChatModelConfig{
		APIKey: cfg.APIKey,
		Model:  cfg.Model,
	}
	if cfg.BaseURL != "" {
		chatModelCfg.BaseURL = cfg.BaseURL
	}

	cm, err := einoopenai.NewChatModel(ctx, chatModelCfg)
	if err != nil {
		return nil, fmt.Errorf("ai: failed to create chat model: %w", err)
	}

	return &Provider{
		ChatModel: cm,
		config:    cfg,
	}, nil
}

// Close releases any resources held by the provider.
func (p *Provider) Close() error {
	// The current OpenAI ChatModel implementation is stateless (HTTP client),
	// so there is nothing to close. This method exists for forward compatibility
	// with providers that hold persistent connections (e.g. gRPC, WebSocket).
	return nil
}
