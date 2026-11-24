package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// sseEvent is one server-sent event of a streamed Messages API reply.
type sseEvent struct {
	name string
	data string
}

func messageStart(model string) sseEvent {
	return sseEvent{"message_start", fmt.Sprintf(`{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":%q,"content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":12,"output_tokens":1}}}`, model)}
}

func textBlock(index int, parts ...string) []sseEvent {
	events := []sseEvent{{"content_block_start", fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"text","text":""}}`, index)}}
	for _, part := range parts {
		events = append(events, sseEvent{"content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"text_delta","text":%q}}`, index, part)})
	}
	return append(events, sseEvent{"content_block_stop", fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, index)})
}

func thinkingBlock(index int) []sseEvent {
	return []sseEvent{
		{"content_block_start", fmt.Sprintf(`{"type":"content_block_start","index":%d,"content_block":{"type":"thinking","thinking":"","signature":""}}`, index)},
		{"content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":%d,"delta":{"type":"signature_delta","signature":"sig"}}`, index)},
		{"content_block_stop", fmt.Sprintf(`{"type":"content_block_stop","index":%d}`, index)},
	}
}

func messageEnd(stopReason, stopDetails string) []sseEvent {
	return []sseEvent{
		{"message_delta", fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":%q,"stop_sequence":null,"stop_details":%s},"usage":{"output_tokens":7}}`, stopReason, stopDetails)},
		{"message_stop", `{"type":"message_stop"}`},
	}
}

// fakeAPI serves events as the reply to every Messages request and records
// the last request it received.
type fakeAPI struct {
	*httptest.Server
	body   map[string]interface{}
	header http.Header
}

func newFakeAPI(t *testing.T, status int, errorBody string, events ...sseEvent) *fakeAPI {
	t.Helper()

	api := &fakeAPI{}
	api.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		api.body = map[string]interface{}{}
		json.Unmarshal(raw, &api.body)
		api.header = r.Header.Clone()

		if status != http.StatusOK {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			io.WriteString(w, errorBody)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		for _, e := range events {
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", e.name, e.data)
		}
	}))
	t.Cleanup(api.Close)

	return api
}

func (api *fakeAPI) provider(t *testing.T, cfg Config) Provider {
	t.Helper()

	cfg.APIKey = "test-key"
	cfg.BaseURL = api.URL

	provider, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return provider
}

func events(groups ...[]sseEvent) []sseEvent {
	var all []sseEvent
	for _, g := range groups {
		all = append(all, g...)
	}
	return all
}

func userMessage(text string) []Message {
	return []Message{{Role: RoleUser, Content: text}}
}

func TestAnthropicChat(t *testing.T) {
	api := newFakeAPI(t, http.StatusOK, "", events(
		[]sseEvent{messageStart("claude-opus-5-5")},
		thinkingBlock(0),
		textBlock(1, "Hello", ", world"),
		messageEnd("end_turn", "null"),
	)...)

	var streamed []string
	response, err := api.provider(t, Config{System: "Be brief."}).ChatStream(context.Background(), ChatRequest{
		Messages: []Message{
			{Role: RoleUser, Content: "Hi"},
			{Role: RoleAssistant, Content: "Hey"},
			{Role: RoleUser, Content: "Say hello"},
		},
	}, func(text string) { streamed = append(streamed, text) })
	if err != nil {
		t.Fatalf("ChatStream: %v", err)
	}

	if response.Text != "Hello, world" {
		t.Errorf("Text = %q", response.Text)
	}
	if got := strings.Join(streamed, "|"); got != "Hello|, world" {
		t.Errorf("streamed = %q", got)
	}
	if response.Model != "claude-opus-5-5" || response.StopReason != "end_turn" || response.Provider != ProviderAnthropic {
		t.Errorf("response = %+v", response)
	}
	if response.Usage.InputTokens != 12 || response.Usage.OutputTokens != 7 {
		t.Errorf("usage = %+v", response.Usage)
	}

	if api.body["model"] != AnthropicDefaultModel {
		t.Errorf("model = %v", api.body["model"])
	}
	if api.body["max_tokens"] != float64(AnthropicDefaultMaxTokens) {
		t.Errorf("max_tokens = %v", api.body["max_tokens"])
	}
	if api.body["stream"] != true {
		t.Errorf("stream = %v", api.body["stream"])
	}
	if system, _ := json.Marshal(api.body["system"]); !strings.Contains(string(system), "Be brief.") {
		t.Errorf("system = %s", system)
	}
	if _, ok := api.body["output_config"]; ok {
		t.Errorf("output_config sent without an effort: %v", api.body["output_config"])
	}

	messages, _ := api.body["messages"].([]interface{})
	if len(messages) != 3 {
		t.Fatalf("messages = %v", api.body["messages"])
	}
	for i, role := range []string{"user", "assistant", "user"} {
		if got := messages[i].(map[string]interface{})["role"]; got != role {
			t.Errorf("messages[%d].role = %v, want %s", i, got, role)
		}
	}

	if api.header.Get("Authorization") != "Bearer test-key" {
		t.Errorf("the configured API key was not sent")
	}
	// The default model gets server-side fallbacks.
	if api.body["fallbacks"] != "default" {
		t.Errorf("fallbacks = %v", api.body["fallbacks"])
	}
	if beta := api.header.Get("Anthropic-Beta"); !strings.Contains(beta, "server-side-fallback-2026-07-01") {
		t.Errorf("anthropic-beta = %q", beta)
	}
}

func TestAnthropicRequestOverrides(t *testing.T) {
	api := newFakeAPI(t, http.StatusOK, "", events(
		[]sseEvent{messageStart("claude-haiku-5-5")},
		textBlock(0, "ok"),
		messageEnd("end_turn", "null"),
	)...)

	provider := api.provider(t, Config{Model: "claude-opus-5-5", System: "config prompt", MaxTokens: 500, Effort: "low"})
	_, err := provider.Chat(context.Background(), ChatRequest{
		Model:     "claude-haiku-5-5",
		System:    "request prompt",
		MaxTokens: 100,
		Effort:    "HIGH",
		Messages:  userMessage("Hi"),
	})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if api.body["model"] != "claude-haiku-5-5" || api.body["max_tokens"] != float64(100) {
		t.Errorf("model/max_tokens = %v/%v", api.body["model"], api.body["max_tokens"])
	}
	if system, _ := json.Marshal(api.body["system"]); !strings.Contains(string(system), "request prompt") {
		t.Errorf("system = %s", system)
	}
	if config, _ := api.body["output_config"].(map[string]interface{}); config["effort"] != "high" {
		t.Errorf("output_config = %v", api.body["output_config"])
	}
	// This model has no server-side fallback, so none is requested.
	if _, ok := api.body["fallbacks"]; ok {
		t.Errorf("fallbacks sent for a model without them: %v", api.body["fallbacks"])
	}
	if beta := api.header.Get("Anthropic-Beta"); strings.Contains(beta, "server-side-fallback") {
		t.Errorf("anthropic-beta = %q", beta)
	}
}

func TestAnthropicDisableFallback(t *testing.T) {
	api := newFakeAPI(t, http.StatusOK, "", events(
		[]sseEvent{messageStart("claude-opus-5-5")},
		textBlock(0, "ok"),
		messageEnd("end_turn", "null"),
	)...)

	if _, err := api.provider(t, Config{DisableFallback: true}).Chat(context.Background(), ChatRequest{Messages: userMessage("Hi")}); err != nil {
		t.Fatalf("Chat: %v", err)
	}

	if _, ok := api.body["fallbacks"]; ok {
		t.Errorf("fallbacks sent although disabled: %v", api.body["fallbacks"])
	}
}

func TestAnthropicRefusal(t *testing.T) {
	api := newFakeAPI(t, http.StatusOK, "", events(
		[]sseEvent{messageStart("claude-opus-5-5")},
		messageEnd("refusal", `{"type":"refusal","category":"cyber","explanation":"declined"}`),
	)...)

	response, err := api.provider(t, Config{}).Chat(context.Background(), ChatRequest{Messages: userMessage("Hi")})
	if !errors.Is(err, ErrRefused) {
		t.Fatalf("err = %v, want ErrRefused", err)
	}
	if response == nil || response.StopReason != "refusal" || response.RefusalCategory != "cyber" {
		t.Errorf("response = %+v", response)
	}
}

func TestAnthropicAPIError(t *testing.T) {
	api := newFakeAPI(t, http.StatusBadRequest, `{"type":"error","error":{"type":"invalid_request_error","message":"bad model"},"request_id":"req_1"}`)

	_, err := api.provider(t, Config{}).Chat(context.Background(), ChatRequest{Messages: userMessage("Hi")})

	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("err = %v, want *ProviderError", err)
	}
	if providerErr.HTTPStatus != http.StatusBadRequest || providerErr.Type != "invalid_request_error" || providerErr.Message != "bad model" {
		t.Errorf("providerErr = %+v", providerErr)
	}
	if providerErr.Retryable() {
		t.Error("a 400 must not be retryable")
	}
}

func TestInvalidRequest(t *testing.T) {
	provider, err := New(Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for name, request := range map[string]ChatRequest{
		"no messages":   {},
		"unknown role":  {Messages: []Message{{Role: "system", Content: "Hi"}}},
		"empty content": {Messages: []Message{{Role: RoleUser, Content: " "}}},
	} {
		if _, err := provider.Chat(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
			t.Errorf("%s: err = %v, want ErrInvalidRequest", name, err)
		}
	}
}

func TestUnknownProvider(t *testing.T) {
	if _, err := New(Config{Provider: "nope"}); !errors.Is(err, ErrUnknownProvider) {
		t.Errorf("err = %v, want ErrUnknownProvider", err)
	}
}

func TestRegisterFactory(t *testing.T) {
	RegisterFactory(" Custom ", func(cfg Config) (Provider, error) {
		return &AnthropicProvider{config: cfg}, nil
	})
	t.Cleanup(func() {
		registryMu.Lock()
		delete(registry, "custom")
		registryMu.Unlock()
	})

	provider, err := New(Config{Provider: "CUSTOM"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := provider.(*AnthropicProvider).config.Provider; got != "custom" {
		t.Errorf("config.Provider = %q", got)
	}
}
