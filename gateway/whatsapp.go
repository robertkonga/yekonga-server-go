package gateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/gateway/setting"
	"github.com/robertkonga/yekonga-server-go/gateway/whatsapp"
	"github.com/robertkonga/yekonga-server-go/helper"
)

// NewWhatsappTemplateManager returns the template API of the configured
// WhatsApp provider, or an error if that provider doesn't support it.
func NewWhatsappTemplateManager(defaultConfig *config.WhatsappGatewayConfig) (setting.WhatsappTemplateManager, error) {
	manager, ok := NewWhatsappProvider(defaultConfig).(setting.WhatsappTemplateManager)
	if !ok {
		return nil, fmt.Errorf("whatsapp provider %q does not support template management", defaultConfig.Provider)
	}
	return manager, nil
}

// NewWhatsappProvider creates a new WhatsApp control instance
func NewWhatsappProvider(defaultConfig *config.WhatsappGatewayConfig) setting.WhatsappProvider {
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	if defaultConfig.Provider == config.ProviderMeta {
		// Sender is the Meta Phone Number ID, so there is no sensible default.
		if helper.IsEmpty(defaultConfig.BaseURL) {
			defaultConfig.BaseURL = whatsapp.MetaDefaultBaseURL
		}

		return &whatsapp.MetaProvider{
			DefaultConfig: defaultConfig,
			HTTPClient:    httpClient,
		}
	}

	if helper.IsEmpty(defaultConfig.Sender) {
		defaultConfig.Sender = "441134960000"
	}

	if defaultConfig.Provider == config.ProviderInfobip {
		if helper.IsEmpty(defaultConfig.BaseURL) {
			defaultConfig.BaseURL = "43vk1n.api.infobip.com"
		}
	}

	return &whatsapp.InfobipProvider{
		DefaultConfig: defaultConfig,
		HTTPClient:    httpClient,
	}
}
