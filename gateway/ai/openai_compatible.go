package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// openAIDialect is what differs between the providers that speak the OpenAI
// Chat Completions format.
type openAIDialect struct {
	name        string
	baseURL     string // empty = Config.BaseURL is required
	keyEnv      string // environment variable holding the API key
	keyHeader   string // header carrying the key; empty = "Authorization: Bearer <key>"
	tokensParam string // request field limiting the reply's tokens
	streamUsage bool   // accepts stream_options.include_usage
	noMaxEffort bool   // has no "max" effort; "xhigh" is its highest
}

var openAIDialects = []openAIDialect{
	{name: ProviderOpenAI, baseURL: "https://api.openai.com/v1", keyEnv: "OPENAI_API_KEY", tokensParam: "max_completion_tokens", streamUsage: true},
	{name: ProviderGoogle, baseURL: "https://generativelanguage.googleapis.com/v1beta/openai", keyEnv: "GEMINI_API_KEY", tokensParam: "max_tokens", streamUsage: true, noMaxEffort: true},
	{name: ProviderDeepSeek, baseURL: "https://api.deepseek.com", keyEnv: "DEEPSEEK_API_KEY", tokensParam: "max_tokens", streamUsage: true},
	{name: ProviderMeta, baseURL: "https://api.meta.ai/v1", keyEnv: "MODEL_API_KEY", tokensParam: "max_completion_tokens"},
	// Azure OpenAI: the endpoint belongs to the customer's resource, e.g.
	// https://<resource>.openai.azure.com/openai/v1, and Model is the deployment name.
	{name: ProviderMicrosoft, keyEnv: "AZURE_OPENAI_API_KEY", keyHeader: "api-key", tokensParam: "max_completion_tokens", streamUsage: true, noMaxEffort: true},
}

// A reply that isn't streamed has no natural point to give up at, so it gets
// this long unless the caller's context sets its own deadline.
const openAIRequestTimeout = 10 * time.Minute

var openAIHTTPClient = &http.Client{}

func init() {
	for _, dialect := range openAIDialects {
		RegisterFactory(dialect.name, func(cfg Config) (Provider, error) {
			return newOpenAICompatible(dialect, cfg)
		})
	}
}

// OpenAICompatibleProvider talks to a provider through the OpenAI Chat
// Completions format: OpenAI, Google (Gemini), DeepSeek, Meta and Microsoft
// (Azure OpenAI).
type OpenAICompatibleProvider struct {
	dialect openAIDialect
	config  Config
	apiKey  string
	url     string
}

func newOpenAICompatible(dialect openAIDialect, cfg Config) (Provider, error) {
	baseURL := firstNonEmpty(cfg.BaseURL, dialect.baseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("ai: %s: baseURL is required", dialect.name)
	}

	apiKey := firstNonEmpty(cfg.APIKey, os.Getenv(dialect.keyEnv))
	if apiKey == "" {
		return nil, fmt.Errorf("ai: %s: apiKey is required (or set %s)", dialect.name, dialect.keyEnv)
	}

	return &OpenAICompatibleProvider{
		dialect: dialect,
		config:  cfg,
		apiKey:  apiKey,
		url:     strings.TrimRight(baseURL, "/") + "/chat/completions",
	}, nil
}

func (p *OpenAICompatibleProvider) Name() string { return p.dialect.name }

func (p *OpenAICompatibleProvider) Chat(ctx context.Context, request ChatRequest) (*ChatResponse, error) {
	return p.ChatStream(ctx, request, nil)
}

func (p *OpenAICompatibleProvider) ChatStream(ctx context.Context, request ChatRequest, onText func(text string)) (*ChatResponse, error) {
	stream := onText != nil

	body, model, err := p.body(request, stream)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if _, hasDeadline := ctx.Deadline(); !stream && !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, openAIRequestTimeout)
		defer cancel()
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, p.url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if p.dialect.keyHeader != "" {
		httpRequest.Header.Set(p.dialect.keyHeader, p.apiKey)
	} else {
		httpRequest.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	httpResponse, err := openAIHTTPClient.Do(httpRequest)
	if err != nil {
		return nil, err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode >= 400 {
		raw, _ := io.ReadAll(httpResponse.Body)
		return nil, p.apiError(httpResponse, raw)
	}

	var reply openAIReply
	if stream {
		err = p.readStream(httpResponse.Body, &reply, onText)
	} else {
		err = p.readJSON(httpResponse.Body, &reply)
	}
	if err != nil {
		return nil, err
	}

	response := &ChatResponse{
		Provider:   p.dialect.name,
		Model:      firstNonEmpty(reply.model, model),
		Text:       reply.text.String(),
		StopReason: openAIStopReason(reply.finishReason),
		Usage:      reply.usage,
		Raw:        reply.raw,
	}

	if reply.refusal.Len() > 0 || response.StopReason == StopRefusal {
		response.StopReason = StopRefusal
		return response, ErrRefused
	}

	return response, nil
}

// body builds the request body and returns it with the model it asks for.
func (p *OpenAICompatibleProvider) body(request ChatRequest, stream bool) ([]byte, string, error) {
	if err := request.validate(); err != nil {
		return nil, "", err
	}

	// Model names differ per provider and change often, so there is no default.
	model := firstNonEmpty(request.Model, p.config.Model)
	if model == "" {
		return nil, "", fmt.Errorf("%w: %s: model is required", ErrInvalidRequest, p.dialect.name)
	}

	messages := make([]map[string]string, 0, len(request.Messages)+1)
	if system := firstNonEmpty(request.System, p.config.System); system != "" {
		messages = append(messages, map[string]string{"role": "system", "content": system})
	}
	for _, m := range request.Messages {
		messages = append(messages, map[string]string{"role": m.Role, "content": m.Content})
	}

	body := map[string]interface{}{
		"model":    model,
		"messages": messages,
	}

	// Without a limit of our own the provider's default for the model applies.
	maxTokens := p.config.MaxTokens
	if request.MaxTokens > 0 {
		maxTokens = request.MaxTokens
	}
	if maxTokens > 0 {
		body[p.dialect.tokensParam] = maxTokens
	}

	if effort := strings.ToLower(firstNonEmpty(request.Effort, p.config.Effort)); effort != "" {
		if effort == "max" && p.dialect.noMaxEffort {
			effort = "xhigh"
		}
		body["reasoning_effort"] = effort
	}

	if stream {
		body["stream"] = true
		if p.dialect.streamUsage {
			body["stream_options"] = map[string]bool{"include_usage": true}
		}
	}

	raw, err := json.Marshal(body)

	return raw, model, err
}

// openAIReply is a reply gathered from either a JSON body or a stream.
type openAIReply struct {
	model        string
	text         strings.Builder
	refusal      strings.Builder
	finishReason string
	usage        Usage
	raw          interface{}
}

type openAIMessage struct {
	Content string `json:"content"`
	Refusal string `json:"refusal"`
}

// openAIChunk is a whole reply (Message) or one piece of a streamed one (Delta).
type openAIChunk struct {
	Model   string `json:"model"`
	Choices []struct {
		Message      openAIMessage `json:"message"`
		Delta        openAIMessage `json:"delta"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
	Error json.RawMessage `json:"error"`
}

func (r *openAIReply) add(chunk openAIChunk, onText func(text string)) {
	if chunk.Model != "" {
		r.model = chunk.Model
	}
	if chunk.Usage != nil {
		r.usage = Usage{InputTokens: chunk.Usage.PromptTokens, OutputTokens: chunk.Usage.CompletionTokens}
	}

	// Only one choice is ever requested.
	if len(chunk.Choices) == 0 {
		return
	}
	choice := chunk.Choices[0]

	if choice.FinishReason != "" {
		r.finishReason = choice.FinishReason
	}
	r.text.WriteString(choice.Message.Content)
	r.refusal.WriteString(choice.Message.Refusal)
	r.refusal.WriteString(choice.Delta.Refusal)

	if choice.Delta.Content != "" {
		r.text.WriteString(choice.Delta.Content)
		if onText != nil {
			onText(choice.Delta.Content)
		}
	}
}

func (p *OpenAICompatibleProvider) readJSON(body io.Reader, reply *openAIReply) error {
	raw, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	var chunk openAIChunk
	if err := json.Unmarshal(raw, &chunk); err != nil {
		return fmt.Errorf("ai: %s: unreadable response: %w", p.dialect.name, err)
	}

	reply.add(chunk, nil)
	reply.raw = json.RawMessage(raw)

	return nil
}

// readStream reads server-sent events until the stream ends or "[DONE]".
func (p *OpenAICompatibleProvider) readStream(body io.Reader, reply *openAIReply, onText func(text string)) error {
	reader := bufio.NewReader(body)
	event := ""

	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimRight(line, "\r\n")

		switch {
		case line == "":
			event = ""
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
			if data == "[DONE]" {
				return nil
			}

			var chunk openAIChunk
			if json.Unmarshal([]byte(data), &chunk) != nil {
				break // not a chunk, e.g. a keep-alive
			}
			// A failure after the stream opened arrives as an event, not a status.
			if event == "error" || len(chunk.Error) > 0 {
				return p.parseError([]byte(data))
			}

			reply.add(chunk, onText)
		}

		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

// openAIStopReason maps a finish_reason to the package's stop reasons.
func openAIStopReason(finishReason string) string {
	switch finishReason {
	case "stop":
		return StopEndTurn
	case "length":
		return StopMaxTokens
	case "content_filter":
		return StopRefusal
	}

	return finishReason
}

// openAIErrorBody is {"error": {...}}, or the bare inner object some
// providers send mid-stream.
type openAIErrorBody struct {
	Error *openAIErrorDetail `json:"error"`
	openAIErrorDetail
}

type openAIErrorDetail struct {
	Message string      `json:"message"`
	Type    string      `json:"type"`
	Status  string      `json:"status"` // Google's name for the type
	Code    interface{} `json:"code"`   // a string, or the HTTP status as a number
}

func (p *OpenAICompatibleProvider) parseError(raw []byte) *ProviderError {
	providerErr := &ProviderError{Provider: p.dialect.name, Message: strings.TrimSpace(string(raw))}

	var body openAIErrorBody
	if json.Unmarshal(raw, &body) != nil {
		// Google wraps the same object in an array.
		var list []openAIErrorBody
		if json.Unmarshal(raw, &list) != nil || len(list) == 0 {
			return providerErr
		}
		body = list[0]
	}

	detail := body.openAIErrorDetail
	if body.Error != nil {
		detail = *body.Error
	}

	if detail.Message != "" {
		providerErr.Message = detail.Message
	}
	providerErr.Type = firstNonEmpty(detail.Type, detail.Status)
	if code, ok := detail.Code.(string); ok && providerErr.Type == "" {
		providerErr.Type = code
	}

	return providerErr
}

func (p *OpenAICompatibleProvider) apiError(response *http.Response, raw []byte) error {
	providerErr := p.parseError(raw)
	providerErr.HTTPStatus = response.StatusCode
	providerErr.RequestID = firstNonEmpty(response.Header.Get("x-request-id"), response.Header.Get("apim-request-id"))
	if providerErr.Message == "" {
		providerErr.Message = response.Status
	}

	return providerErr
}
