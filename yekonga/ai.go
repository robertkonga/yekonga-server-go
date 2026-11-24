package yekonga

import (
	"context"

	"github.com/robertkonga/yekonga-server-go/gateway"
	"github.com/robertkonga/yekonga-server-go/gateway/ai"
)

// AI returns the AI provider configured in config.apiGateway.ai. It is built
// on first use and reused afterwards.
func (y *YekongaData) AI() (ai.Provider, error) {
	y.aiProviderOnce.Do(func() {
		y.aiProvider, y.aiProviderErr = gateway.NewAIProvider(&y.Config.ApiGateway.AI)
	})

	return y.aiProvider, y.aiProviderErr
}

// AIChat sends a conversation to the configured AI provider and returns its
// reply. Pass the request's context (req.HttpRequest.Context()) so the call
// stops when the client disconnects.
func (y *YekongaData) AIChat(ctx context.Context, request ai.ChatRequest) (*ai.ChatResponse, error) {
	provider, err := y.AI()
	if err != nil {
		return nil, err
	}

	return provider.Chat(ctx, request)
}

// AIChatStream is AIChat, also passing the reply to onText piece by piece as
// it is generated, e.g. to forward it over a socket.
func (y *YekongaData) AIChatStream(ctx context.Context, request ai.ChatRequest, onText func(text string)) (*ai.ChatResponse, error) {
	provider, err := y.AI()
	if err != nil {
		return nil, err
	}

	return provider.ChatStream(ctx, request, onText)
}

// AIAsk sends a single prompt to the configured AI provider and returns the
// text of its reply.
func (y *YekongaData) AIAsk(ctx context.Context, prompt string) (string, error) {
	response, err := y.AIChat(ctx, ai.ChatRequest{
		Messages: []ai.Message{{Role: ai.RoleUser, Content: prompt}},
	})
	if err != nil {
		return "", err
	}

	return response.Text, nil
}
