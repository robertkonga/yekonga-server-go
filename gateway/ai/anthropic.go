package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/v2"
	"github.com/anthropics/anthropic-sdk-go/v2/option"
)

const (
	AnthropicDefaultModel     = "claude-opus-5-5"
	AnthropicDefaultMaxTokens = 16000
)

// anthropicFallbackModels can have a refused request retried on another model
// by the API itself (the `fallbacks` parameter).
var anthropicFallbackModels = map[string]bool{
	"claude-fable-5-2":  true,
	"claude-fable-5-1":  true,
	"claude-opus-5-5":   true,
	"claude-opus-5":     true,
	"claude-sonnet-5-5": true,
}

func init() {
	RegisterFactory(ProviderAnthropic, newAnthropic)
}

// AnthropicProvider talks to Claude through the Anthropic API.
type AnthropicProvider struct {
	config Config
	client anthropic.Client
}

func newAnthropic(cfg Config) (Provider, error) {
	// With no key the SDK falls back to the environment (ANTHROPIC_API_KEY).
	opts := []option.RequestOption{}
	if strings.TrimSpace(cfg.APIKey) != "" {
		opts = append(opts, option.WithAPIKey(strings.TrimSpace(cfg.APIKey)))
	}
	if strings.TrimSpace(cfg.BaseURL) != "" {
		opts = append(opts, option.WithBaseURL(strings.TrimSpace(cfg.BaseURL)))
	}

	if cfg.Model == "" {
		cfg.Model = AnthropicDefaultModel
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = AnthropicDefaultMaxTokens
	}

	return &AnthropicProvider{config: cfg, client: anthropic.NewClient(opts...)}, nil
}

func (p *AnthropicProvider) Name() string { return ProviderAnthropic }

func (p *AnthropicProvider) Chat(ctx context.Context, request ChatRequest) (*ChatResponse, error) {
	return p.ChatStream(ctx, request, nil)
}

// ChatStream always streams from the API, so long replies don't hit HTTP
// timeouts; Chat is the same call without a listener.
func (p *AnthropicProvider) ChatStream(ctx context.Context, request ChatRequest, onText func(text string)) (*ChatResponse, error) {
	params, err := p.params(request)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	stream := p.client.Beta.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	message := anthropic.BetaMessage{}
	for stream.Next() {
		event := stream.Current()
		if err := message.Accumulate(event); err != nil {
			return nil, err
		}

		if onText == nil {
			continue
		}
		if delta, ok := event.AsAny().(anthropic.BetaRawContentBlockDeltaEvent); ok {
			if text, ok := delta.Delta.AsAny().(anthropic.BetaTextDelta); ok && text.Text != "" {
				onText(text.Text)
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, p.wrapError(err)
	}

	response := &ChatResponse{
		Provider:   ProviderAnthropic,
		Model:      string(message.Model),
		StopReason: string(message.StopReason),
		Usage: Usage{
			InputTokens:  message.Usage.InputTokens,
			OutputTokens: message.Usage.OutputTokens,
		},
		Raw: message,
	}

	// The reply's text blocks sit among thinking blocks, which are skipped.
	var text strings.Builder
	for _, block := range message.Content {
		if b, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			text.WriteString(b.Text)
		}
	}
	response.Text = text.String()

	if response.StopReason == StopRefusal {
		response.RefusalCategory = string(message.StopDetails.Category)
		if response.RefusalCategory != "" {
			return response, fmt.Errorf("%w (%s)", ErrRefused, response.RefusalCategory)
		}
		return response, ErrRefused
	}

	return response, nil
}

func (p *AnthropicProvider) params(request ChatRequest) (anthropic.BetaMessageNewParams, error) {
	params := anthropic.BetaMessageNewParams{}

	if err := request.validate(); err != nil {
		return params, err
	}

	model := firstNonEmpty(request.Model, p.config.Model)
	params.Model = anthropic.Model(model)

	params.MaxTokens = int64(p.config.MaxTokens)
	if request.MaxTokens > 0 {
		params.MaxTokens = int64(request.MaxTokens)
	}

	if system := firstNonEmpty(request.System, p.config.System); system != "" {
		params.System = []anthropic.BetaTextBlockParam{{Text: system}}
	}

	if effort := firstNonEmpty(request.Effort, p.config.Effort); effort != "" {
		params.OutputConfig = anthropic.BetaOutputConfigParam{
			Effort: anthropic.BetaOutputConfigEffort(strings.ToLower(effort)),
		}
	}

	for _, m := range request.Messages {
		block := anthropic.NewBetaTextBlock(m.Content)
		if m.Role == RoleAssistant {
			params.Messages = append(params.Messages, anthropic.BetaMessageParam{
				OfAssistant: &anthropic.BetaAssistantMessageParam{
					Content: []anthropic.BetaContentBlockParamUnion{block},
				},
			})
			continue
		}
		params.Messages = append(params.Messages, anthropic.NewBetaUserMessage(block))
	}

	// These models' safety classifiers can decline a request; have the API
	// retry a declined one on the model Anthropic recommends instead.
	if !p.config.DisableFallback && anthropicFallbackModels[model] {
		params.Fallbacks = anthropic.BetaFallbacksParamOfDefault()
		params.Betas = []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01}
	}

	return params, nil
}

func (p *AnthropicProvider) wrapError(err error) error {
	var apierr *anthropic.Error
	if errors.As(err, &apierr) {
		return &ProviderError{
			Provider:   ProviderAnthropic,
			HTTPStatus: apierr.StatusCode,
			Type:       string(apierr.Type()),
			Message:    firstNonEmpty(apierr.Message, apierr.Error()),
			RequestID:  apierr.RequestID,
			Err:        err,
		}
	}

	return err
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}

	return ""
}
