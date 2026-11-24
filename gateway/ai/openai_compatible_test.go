package ai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeOpenAI answers every request with status and body, and records the last
// request it received.
type fakeOpenAI struct {
	*httptest.Server
	path   string
	body   map[string]interface{}
	header http.Header
}

func newFakeOpenAI(t *testing.T, status int, contentType, body string) *fakeOpenAI {
	t.Helper()

	api := &fakeOpenAI{}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		api.body = map[string]interface{}{}
		json.Unmarshal(raw, &api.body)
		api.header = r.Header.Clone()
		api.path = r.URL.Path

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("x-request-id", "req_1")
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(api.Close)

	return api
}

func (api *fakeOpenAI) provider(t *testing.T, cfg Config) Provider {
	t.Helper()

	if cfg.APIKey == "" {
		cfg.APIKey = "test-key"
	}
	cfg.BaseURL = api.URL + "/v1/"

	provider, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return provider
}

const openAIReplyJSON = `{"id":"1","model":"served-model","choices":[{"index":0,"message":{"role":"assistant","content":"Hello, world","refusal":null},"finish_reason":"stop"}],"usage":{"prompt_tokens":12,"completion_tokens":7,"total_tokens":19}}`

func sse(chunks ...string) string {
	var b strings.Builder
	for _, chunk := range chunks {
		b.WriteString("data: " + chunk + "\n\n")
	}
	return b.String()
}

func TestOpenAICompatibleChat(t *testing.T) {
	api := newFakeOpenAI(t, http.StatusOK, "application/json", openAIReplyJSON)

	response, err := api.provider(t, Config{Provider: ProviderOpenAI, Model: "some-model", System: "Be brief."}).Chat(context.Background(), ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hi"},
			{Role: RoleAssistant, Content: "Hey"},
			{Role: RoleUser, Content: "Say hello"},
		},
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if response.Text != "Hello, world" || response.Provider != ProviderOpenAI || response.Model != "served-model" || response.StopReason != StopEndTurn {
		t.Errorf("response = %+v", response)
	}
	if response.Usage.InputTokens != 12 || response.Usage.OutputTokens != 7 {
		t.Errorf("usage = %+v", response.Usage)
	}

	if api.path != "/v1/chat/completions" {
		t.Errorf("path = %q", api.path)
	}
	if api.body["model"] != "some-model" {
		t.Errorf("model = %v", api.body["model"])
	}
	for _, field := range []string{"stream", "stream_options", "max_tokens", "max_completion_tokens", "reasoning_effort"} {
		if _, ok := api.body[field]; ok {
			t.Errorf("%s sent although not asked for: %v", field, api.body[field])
		}
	}

	messages, _ := api.body["messages"].([]interface{})
	if len(messages) != 4 {
		t.Fatalf("messages = %v", api.body["messages"])
	}
	for i, want := range []string{"system", "user", "assistant", "user"} {
		if got := messages[i].(map[string]interface{})["role"]; got != want {
			t.Errorf("messages[%d].role = %v, want %s", i, got, want)
		}
	}
	if got := messages[0].(map[string]interface{})["content"]; got != "Be brief." {
		t.Errorf("system prompt = %v", got)
	}
}

func TestOpenAICompatibleStream(t *testing.T) {
	api := newFakeOpenAI(t, http.StatusOK, "text/event-stream", sse(
		`{"model":"served-model","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}],"usage":null}`,
		`{"model":"served-model","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
		`{"model":"served-model","choices":[{"index":0,"delta":{"reasoning_content":"thinking"},"finish_reason":null}]}`,
		`{"model":"served-model","choices":[{"index":0,"delta":{"content":", world"},"finish_reason":null}]}`,
		`{"model":"served-model","choices":[{"index":0,"delta":{},"finish_reason":"length"}]}`,
		`{"model":"served-model","choices":[],"usage":{"prompt_tokens":12,"completion_tokens":7}}`,
		`[DONE]`,
		`{"choices":[{"index":0,"delta":{"content":"after the end"}}]}`,
	))

	var streamed []string
	response, err := api.provider(t, Config{Provider: ProviderOpenAI, Model: "some-model"}).ChatStream(context.Background(),
		ChatRequest{Messages: userMessage("Hi")}, func(text string) { streamed = append(streamed, text) })
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}

	if response.Text != "Hello, world" || response.StopReason != StopMaxTokens || response.Model != "served-model" {
		t.Errorf("response = %+v", response)
	}
	if got := strings.Join(streamed, "|"); got != "Hello|, world" {
		t.Errorf("streamed = %q", got)
	}
	if response.Usage.InputTokens != 12 || response.Usage.OutputTokens != 7 {
		t.Errorf("usage = %+v", response.Usage)
	}
	if api.body["stream"] != true {
		t.Errorf("stream = %v", api.body["stream"])
	}
}

// Each provider gets its own key header, token-limit field and effort levels.
func TestOpenAICompatibleDialects(t *testing.T) {
	cases := []struct {
		provider    string
		keyHeader   string
		keyValue    string
		tokensParam string
		maxEffort   string
		streamUsage bool
	}{
		{ProviderOpenAI, "Authorization", "Bearer test-key", "max_completion_tokens", "max", true},
		{ProviderGoogle, "Authorization", "Bearer test-key", "max_tokens", "xhigh", true},
		{ProviderDeepSeek, "Authorization", "Bearer test-key", "max_tokens", "max", true},
		{ProviderMeta, "Authorization", "Bearer test-key", "max_completion_tokens", "max", false},
		{ProviderMicrosoft, "Api-Key", "test-key", "max_completion_tokens", "xhigh", true},
	}

	for _, c := range cases {
		t.Run(c.provider, func(t *testing.T) {
			api := newFakeOpenAI(t, http.StatusOK, "text/event-stream", sse(
				`{"choices":[{"index":0,"delta":{"content":"ok"},"finish_reason":"stop"}]}`, `[DONE]`))

			provider := api.provider(t, Config{Provider: c.provider, Model: "config-model", MaxTokens: 500, Effort: "low"})
			if provider.Name() != c.provider {
				t.Errorf("Name = %q", provider.Name())
			}

			response, err := provider.ChatStream(context.Background(), ChatRequest{
				Model:     "request-model",
				MaxTokens: 100,
				Effort:    "MAX",
				Messages:  userMessage("Hi"),
			}, func(string) {})
			if err != nil {
				t.Fatalf("ChatStream: %v", err)
			}
			// The stream named no model, so the requested one is reported.
			if response.Model != "request-model" || response.Provider != c.provider {
				t.Errorf("response = %+v", response)
			}

			if got := api.header.Get(c.keyHeader); got != c.keyValue {
				t.Errorf("%s = %q, want %q", c.keyHeader, got, c.keyValue)
			}
			if c.keyHeader != "Authorization" && api.header.Get("Authorization") != "" {
				t.Errorf("Authorization also sent: %q", api.header.Get("Authorization"))
			}
			if api.body["model"] != "request-model" {
				t.Errorf("model = %v", api.body["model"])
			}
			if api.body[c.tokensParam] != float64(100) {
				t.Errorf("%s = %v", c.tokensParam, api.body[c.tokensParam])
			}
			for _, other := range []string{"max_tokens", "max_completion_tokens"} {
				if _, ok := api.body[other]; ok && other != c.tokensParam {
					t.Errorf("%s also sent", other)
				}
			}
			if api.body["reasoning_effort"] != c.maxEffort {
				t.Errorf("reasoning_effort = %v, want %s", api.body["reasoning_effort"], c.maxEffort)
			}
			if _, ok := api.body["stream_options"]; ok != c.streamUsage {
				t.Errorf("stream_options sent = %v, want %v", ok, c.streamUsage)
			}
		})
	}
}

func TestOpenAICompatibleRefusal(t *testing.T) {
	bodies := map[string]string{
		"refusal field":  `{"choices":[{"message":{"content":null,"refusal":"I can't help with that."},"finish_reason":"stop"}]}`,
		"content filter": `{"choices":[{"message":{"content":""},"finish_reason":"content_filter"}]}`,
	}

	for name, body := range bodies {
		api := newFakeOpenAI(t, http.StatusOK, "application/json", body)

		response, err := api.provider(t, Config{Provider: ProviderOpenAI, Model: "m"}).Chat(context.Background(), ChatRequest{Messages: userMessage("Hi")})
		if !errors.Is(err, ErrRefused) {
			t.Fatalf("%s: err = %v, want ErrRefused", name, err)
		}
		if response == nil || response.StopReason != StopRefusal {
			t.Errorf("%s: response = %+v", name, response)
		}
	}
}

func TestOpenAICompatibleAPIError(t *testing.T) {
	cases := map[string]struct {
		status    int
		body      string
		wantType  string
		wantMsg   string
		retryable bool
	}{
		"openai shape":  {400, `{"error":{"message":"bad model","type":"invalid_request_error","param":null,"code":"model_not_found"}}`, "invalid_request_error", "bad model", false},
		"code only":     {429, `{"error":{"message":"slow down","code":"rate_limit_exceeded"}}`, "rate_limit_exceeded", "slow down", true},
		"google shape":  {400, `[{"error":{"code":400,"message":"API key not valid","status":"INVALID_ARGUMENT"}}]`, "INVALID_ARGUMENT", "API key not valid", false},
		"not json":      {503, `upstream unavailable`, "", "upstream unavailable", true},
		"empty body":    {404, ``, "", "404 Not Found", false},
		"numeric codes": {500, `{"error":{"message":"boom","code":500}}`, "", "boom", true},
	}

	for name, c := range cases {
		api := newFakeOpenAI(t, c.status, "application/json", c.body)

		_, err := api.provider(t, Config{Provider: ProviderOpenAI, Model: "m"}).Chat(context.Background(), ChatRequest{Messages: userMessage("Hi")})

		var providerErr *ProviderError
		if !errors.As(err, &providerErr) {
			t.Fatalf("%s: err = %v, want *ProviderError", name, err)
		}
		if providerErr.HTTPStatus != c.status || providerErr.Type != c.wantType || providerErr.Message != c.wantMsg || providerErr.RequestID != "req_1" {
			t.Errorf("%s: providerErr = %+v", name, providerErr)
		}
		if providerErr.Retryable() != c.retryable {
			t.Errorf("%s: Retryable = %v", name, providerErr.Retryable())
		}
	}
}

func TestOpenAICompatibleStreamError(t *testing.T) {
	streams := map[string]string{
		"error chunk": sse(`{"choices":[{"delta":{"content":"partial"}}]}`) + sse(`{"error":{"message":"overloaded","type":"server_error"}}`),
		"error event": sse(`{"choices":[{"delta":{"content":"partial"}}]}`) + "event: error\ndata: {\"code\":\"service_overloaded\",\"message\":\"overloaded\"}\n\n",
	}

	for name, stream := range streams {
		api := newFakeOpenAI(t, http.StatusOK, "text/event-stream", stream)

		_, err := api.provider(t, Config{Provider: ProviderMeta, Model: "m"}).ChatStream(context.Background(),
			ChatRequest{Messages: userMessage("Hi")}, func(string) {})

		var providerErr *ProviderError
		if !errors.As(err, &providerErr) {
			t.Fatalf("%s: err = %v, want *ProviderError", name, err)
		}
		if providerErr.Message != "overloaded" || providerErr.Provider != ProviderMeta {
			t.Errorf("%s: providerErr = %+v", name, providerErr)
		}
	}
}

func TestOpenAICompatibleSetup(t *testing.T) {
	for _, dialect := range openAIDialects {
		t.Setenv(dialect.keyEnv, "")
	}

	// No key in the config or the environment.
	if _, err := New(Config{Provider: ProviderOpenAI}); err == nil || !strings.Contains(err.Error(), "OPENAI_API_KEY") {
		t.Errorf("no key: err = %v", err)
	}

	// Azure has no shared endpoint.
	if _, err := New(Config{Provider: ProviderMicrosoft, APIKey: "k"}); err == nil || !strings.Contains(err.Error(), "baseURL") {
		t.Errorf("microsoft without baseURL: err = %v", err)
	}

	// The key comes from the provider's environment variable.
	api := newFakeOpenAI(t, http.StatusOK, "application/json", openAIReplyJSON)
	t.Setenv("DEEPSEEK_API_KEY", "env-key")
	provider, err := New(Config{Provider: ProviderDeepSeek, BaseURL: api.URL, Model: "m"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := provider.Chat(context.Background(), ChatRequest{Messages: userMessage("Hi")}); err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if api.header.Get("Authorization") != "Bearer env-key" {
		t.Error("the key from the environment was not sent")
	}

	// No model in the config or the request.
	provider, err = New(Config{Provider: ProviderGoogle, APIKey: "k"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := provider.Chat(context.Background(), ChatRequest{Messages: userMessage("Hi")}); !errors.Is(err, ErrInvalidRequest) {
		t.Errorf("no model: err = %v, want ErrInvalidRequest", err)
	}

	// Every built-in provider is registered.
	registered := strings.Join(Providers(), ",")
	for _, name := range []string{ProviderAnthropic, ProviderOpenAI, ProviderGoogle, ProviderDeepSeek, ProviderMeta, ProviderMicrosoft} {
		if !strings.Contains(registered, name) {
			t.Errorf("%s is not registered (%s)", name, registered)
		}
	}
}
