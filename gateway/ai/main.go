// Package ai is a small provider-neutral API for chatting with an AI model.
//
// main.go holds the provider-agnostic surface: the request/response types, the
// Provider interface and the factory registry. Each provider lives in its own
// file and registers itself here: anthropic.go (Claude) and
// openai_compatible.go (OpenAI, Google, DeepSeek, Meta, Microsoft). Other
// providers can be added from outside the package with RegisterFactory.
//
//	provider, _ := ai.New(ai.Config{Provider: ai.ProviderAnthropic, APIKey: "..."})
//	res, err := provider.Chat(ctx, ai.ChatRequest{
//		System:   "You are a helpful assistant.",
//		Messages: []ai.Message{{Role: ai.RoleUser, Content: "Hello"}},
//	})
//	// res.Text holds the reply
package ai

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Provider names, as used in Config.Provider.
const (
	ProviderAnthropic = "anthropic" // Claude, through the Anthropic API
	ProviderOpenAI    = "openai"    // GPT models, through the OpenAI API
	ProviderGoogle    = "google"    // Gemini, through the Gemini API
	ProviderDeepSeek  = "deepseek"  // DeepSeek API
	ProviderMeta      = "meta"      // Muse models, through the Meta Model API
	ProviderMicrosoft = "microsoft" // Azure OpenAI; Config.BaseURL is the resource's endpoint and Model the deployment name
)

// Values of ChatResponse.StopReason shared by every provider; a provider's
// other reasons are passed through as it reports them.
const (
	StopEndTurn   = "end_turn"   // the reply is complete
	StopMaxTokens = "max_tokens" // the reply was cut short by the token limit
	StopRefusal   = "refusal"    // the model declined to answer
)

// Message roles.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
)

var (
	ErrUnknownProvider = errors.New("ai: unknown provider")
	ErrInvalidRequest  = errors.New("ai: invalid request")
	// ErrRefused is returned, together with the response, when the model
	// declines to answer. ChatResponse.RefusalCategory says why when known.
	ErrRefused = errors.New("ai: request refused by the model")
)

// ProviderError is returned when a provider's API rejects a call.
type ProviderError struct {
	Provider   string
	HTTPStatus int
	Type       string // provider's error type, e.g. "rate_limit_error"
	Message    string
	RequestID  string
	Err        error // the provider SDK's own error
}

func (e *ProviderError) Error() string {
	msg := e.Provider + ": "
	if e.HTTPStatus != 0 {
		msg += "http " + strconv.Itoa(e.HTTPStatus) + ": "
	}
	if e.Type != "" {
		msg += "[" + e.Type + "] "
	}
	return msg + e.Message
}

func (e *ProviderError) Unwrap() error { return e.Err }

// Retryable reports whether the same call may succeed later: the provider was
// rate limiting (429) or failing (5xx).
func (e *ProviderError) Retryable() bool {
	return e.HTTPStatus == 429 || e.HTTPStatus >= 500
}

// Message is one turn of a conversation.
type Message struct {
	Role    string `json:"role"` // RoleUser or RoleAssistant
	Content string `json:"content"`
}

// ChatRequest asks the model for the next reply in a conversation. Only
// Messages is required; the other fields override the provider's Config.
type ChatRequest struct {
	Model     string    `json:"model,omitempty"`
	System    string    `json:"system,omitempty"` // system prompt
	Messages  []Message `json:"messages"`         // the conversation so far, ending with a user message
	MaxTokens int       `json:"maxTokens,omitempty"`
	Effort    string    `json:"effort,omitempty"` // low, medium, high, xhigh or max
}

// Usage is what a call consumed.
type Usage struct {
	InputTokens  int64 `json:"inputTokens"`
	OutputTokens int64 `json:"outputTokens"`
}

// ChatResponse is the model's reply.
type ChatResponse struct {
	Provider        string      `json:"provider"`
	Model           string      `json:"model"`                     // model that produced the reply; differs from the requested one after a fallback
	Text            string      `json:"text"`                      // the reply
	StopReason      string      `json:"stopReason"`                // why the reply ended: StopEndTurn, StopMaxTokens, StopRefusal, or a reason of the provider's own
	RefusalCategory string      `json:"refusalCategory,omitempty"` // set on some refusals
	Usage           Usage       `json:"usage"`
	Raw             interface{} `json:"-"` // the provider's own response
}

// Config configures one provider.
type Config struct {
	Provider        string
	APIKey          string // empty = the provider's environment variable
	BaseURL         string // overrides the provider's endpoint; required for ProviderMicrosoft
	Model           string // default model; required (here or per request) by every provider but ProviderAnthropic
	System          string // default system prompt
	MaxTokens       int    // default max tokens per reply; 0 = the provider's default
	Effort          string // default effort
	DisableFallback bool   // ProviderAnthropic: never retry a refused request on another model
}

// Provider is implemented by every AI provider.
type Provider interface {
	Name() string
	// Chat returns the model's complete reply.
	Chat(ctx context.Context, request ChatRequest) (*ChatResponse, error)
	// ChatStream is Chat, also passing the reply to onText piece by piece as
	// it is generated.
	ChatStream(ctx context.Context, request ChatRequest, onText func(text string)) (*ChatResponse, error)
}

// Factory builds a provider from its config.
type Factory func(cfg Config) (Provider, error)

var (
	registryMu sync.RWMutex
	registry   = map[string]Factory{}
)

// RegisterFactory makes a provider available to New under name, replacing any
// provider already registered under it.
func RegisterFactory(name string, factory Factory) {
	registryMu.Lock()
	defer registryMu.Unlock()

	registry[normalize(name)] = factory
}

// Providers lists the registered provider names.
func Providers() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

// New creates the provider named by cfg.Provider (ProviderAnthropic when empty).
func New(cfg Config) (Provider, error) {
	name := normalize(cfg.Provider)
	if name == "" {
		name = ProviderAnthropic
	}

	registryMu.RLock()
	factory, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrUnknownProvider, cfg.Provider)
	}

	cfg.Provider = name

	return factory(cfg)
}

func normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// validate checks the parts of a request every provider needs.
func (r ChatRequest) validate() error {
	if len(r.Messages) == 0 {
		return fmt.Errorf("%w: messages are required", ErrInvalidRequest)
	}

	for i, m := range r.Messages {
		if m.Role != RoleUser && m.Role != RoleAssistant {
			return fmt.Errorf("%w: messages[%d]: unknown role %q", ErrInvalidRequest, i, m.Role)
		}
		if strings.TrimSpace(m.Content) == "" {
			return fmt.Errorf("%w: messages[%d]: content is empty", ErrInvalidRequest, i)
		}
	}

	return nil
}
