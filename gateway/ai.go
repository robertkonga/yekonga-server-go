package gateway

import (
	"errors"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/gateway/ai"
)

// NewAIProvider creates the AI provider for defaultConfig.Provider (anthropic
// when empty, openai, google, deepseek, meta, microsoft, or any provider added
// via ai.RegisterFactory). It fails when the provider is unknown or has no API
// key.
func NewAIProvider(defaultConfig *config.AIGatewayConfig) (ai.Provider, error) {
	if defaultConfig == nil {
		return nil, errors.New("ai config is required")
	}

	return ai.New(ai.Config{
		Provider:        defaultConfig.Provider,
		APIKey:          defaultConfig.APIKey,
		BaseURL:         defaultConfig.BaseURL,
		Model:           defaultConfig.Model,
		System:          defaultConfig.System,
		MaxTokens:       defaultConfig.MaxTokens,
		Effort:          defaultConfig.Effort,
		DisableFallback: defaultConfig.DisableFallback,
	})
}
