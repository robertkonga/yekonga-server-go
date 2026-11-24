package whatsapp

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/gateway/setting"
	"github.com/robertkonga/yekonga-server-go/helper"
)

// MetaDefaultBaseURL is the Graph API host and version used when baseURL is not configured.
const MetaDefaultBaseURL = "graph.facebook.com/v23.0"

// MetaProvider sends WhatsApp messages directly through Meta's WhatsApp Business Cloud API.
//
// Config mapping (apiGateway.whatsapp in config.json):
//   - provider:    "meta"
//   - sender:      WhatsApp Phone Number ID (not the phone number itself)
//   - apiKey:      system-user / permanent access token
//   - baseURL:     optional, defaults to graph.facebook.com/v23.0
//   - secretKey:   optional App Secret, used by ValidateSignature for webhooks
//   - verifyToken: optional token used by VerifyWebhook for the subscription handshake
type MetaProvider struct {
	DefaultConfig *config.WhatsappGatewayConfig
	HTTPClient    *http.Client
}

// metaSendResponse is the body returned by POST /{phone-number-id}/messages.
type metaSendResponse struct {
	MessagingProduct string `json:"messaging_product,omitempty"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts,omitempty"`
	Messages []struct {
		ID            string `json:"id"`
		MessageStatus string `json:"message_status,omitempty"`
	} `json:"messages,omitempty"`
	Error *MetaError `json:"error,omitempty"`
}

// MetaError is the error object returned by the Graph API.
type MetaError struct {
	Message      string `json:"message"`
	Type         string `json:"type"`
	Code         int    `json:"code"`
	ErrorSubcode int    `json:"error_subcode,omitempty"`
	FbtraceID    string `json:"fbtrace_id,omitempty"`
	ErrorData    *struct {
		MessagingProduct string `json:"messaging_product,omitempty"`
		Details          string `json:"details,omitempty"`
	} `json:"error_data,omitempty"`
}

// MetaWebhookPayload is the body Meta posts to the webhook callback URL.
type MetaWebhookPayload struct {
	Object string `json:"object"`
	Entry  []struct {
		ID      string `json:"id"`
		Changes []struct {
			Field string `json:"field"`
			Value struct {
				MessagingProduct string `json:"messaging_product"`
				Metadata         struct {
					DisplayPhoneNumber string `json:"display_phone_number"`
					PhoneNumberID      string `json:"phone_number_id"`
				} `json:"metadata"`
				Statuses []MetaWebhookStatus      `json:"statuses,omitempty"`
				Messages []map[string]interface{} `json:"messages,omitempty"`
			} `json:"value"`
		} `json:"changes"`
	} `json:"entry"`
}

// MetaWebhookStatus is a single delivery status update from a webhook.
type MetaWebhookStatus struct {
	ID          string `json:"id"`
	Status      string `json:"status"` // sent, delivered, read, failed
	Timestamp   string `json:"timestamp"`
	RecipientID string `json:"recipient_id"`
	Errors      []struct {
		Code  int    `json:"code"`
		Title string `json:"title"`
	} `json:"errors,omitempty"`
}

// Send sends a WhatsApp message via the Meta Cloud API.
//
// Config precedence: DefaultConfig < cfg argument < params.Config.
func (m *MetaProvider) Send(params setting.SendParams, cfg *config.WhatsappGatewayConfig) (*setting.SendResponse, error) {
	baseURL := m.DefaultConfig.BaseURL
	accessToken := m.DefaultConfig.APIKey
	phoneNumberID := m.DefaultConfig.Sender

	if cfg != nil {
		if helper.IsNotEmpty(cfg.BaseURL) {
			baseURL = cfg.BaseURL
		}
		if helper.IsNotEmpty(cfg.APIKey) {
			accessToken = cfg.APIKey
		}
		if helper.IsNotEmpty(cfg.Sender) {
			phoneNumberID = cfg.Sender
		}
	}

	if params.Config != nil {
		if helper.IsNotEmpty(params.Config.BaseURL) {
			baseURL = params.Config.BaseURL
		}
		if helper.IsNotEmpty(params.Config.APIKey) {
			accessToken = params.Config.APIKey
		}
		if helper.IsNotEmpty(params.Config.Sender) {
			phoneNumberID = params.Config.Sender
		}
	}

	if helper.IsEmpty(baseURL) {
		baseURL = MetaDefaultBaseURL
	}
	if helper.IsEmpty(accessToken) || helper.IsEmpty(phoneNumberID) {
		err := errors.New("meta whatsapp: apiKey (access token) and sender (phone number ID) are required")
		return &setting.SendResponse{Status: "FAILED", Message: err.Error(), Error: err.Error()}, err
	}

	to := normalizeMetaPhone(params.Phone)
	if to == "" {
		err := errors.New("meta whatsapp: recipient phone is required")
		return &setting.SendResponse{Status: "FAILED", Message: err.Error(), Error: err.Error()}, err
	}

	body, err := buildMetaMessage(params)
	if err != nil {
		return &setting.SendResponse{Status: "FAILED", Message: err.Error(), Error: err.Error()}, err
	}
	body["messaging_product"] = "whatsapp"
	body["recipient_type"] = "individual"
	body["to"] = to

	url := fmt.Sprintf("%s/%s/messages", withScheme(baseURL), phoneNumberID)

	statusCode, apiResponse, err := m.postJSON(url, body, accessToken)
	if err != nil {
		return &setting.SendResponse{Status: "FAILED", Message: err.Error(), Error: err.Error()}, err
	}

	var responseData metaSendResponse
	if err := json.Unmarshal(apiResponse, &responseData); err != nil {
		return &setting.SendResponse{
			Status:  "FAILED",
			Message: "Failed to parse response",
			Code:    statusCode,
		}, err
	}

	if responseData.Error != nil || len(responseData.Messages) == 0 {
		message := fmt.Sprintf("unexpected response (HTTP %d)", statusCode)
		code := statusCode
		if responseData.Error != nil {
			message = responseData.Error.Message
			if responseData.Error.ErrorData != nil && helper.IsNotEmpty(responseData.Error.ErrorData.Details) {
				message = fmt.Sprintf("%s: %s", message, responseData.Error.ErrorData.Details)
			}
			code = responseData.Error.Code
		}

		return &setting.SendResponse{
			Status:   "FAILED",
			Message:  message,
			Code:     code,
			Error:    message,
			Response: responseData,
		}, nil
	}

	message := responseData.Messages[0].MessageStatus
	if helper.IsEmpty(message) {
		message = "accepted"
	}

	return &setting.SendResponse{
		Status:    "SUCCESS",
		Message:   message,
		MessageID: responseData.Messages[0].ID,
		Code:      statusCode,
		Response:  responseData,
	}, nil
}

// Notification processes delivery status callbacks (see ParseWebhook to build the body).
func (m *MetaProvider) Notification(providerParams map[string]string, body *setting.NotificationBody) error {
	if body == nil || len(body.Results) == 0 {
		return nil
	}

	for _, result := range body.Results {
		if err := m.DeliveryStatus(providerParams, result); err != nil {
			fmt.Printf("Error processing delivery status: %v\n", err)
		}
	}

	return nil
}

// DeliveryStatus updates delivery status
func (m *MetaProvider) DeliveryStatus(providerParams map[string]string, result setting.NotificationResult) error {
	var status string

	switch result.Status.GroupName {
	case "DELIVERED":
		status = "delivered"
	case "UNDELIVERED":
		status = "undelivered"
	case "":
	default:
		status = strings.ToLower(result.Status.GroupName)
	}

	updateBody := make(map[string]interface{})
	if helper.IsNotEmpty(status) {
		updateBody["status"] = status
	}
	if helper.IsNotEmpty(result.SeenAt) {
		updateBody["isSeen"] = true
		updateBody["opened"] = true
	}

	if result.MessageID != "" {
		fmt.Printf("Updating notification %s with status: %s\n", result.MessageID, status)
	}

	return nil
}

// Balance is not available on the Meta Cloud API (billing lives in Business Manager).
func (m *MetaProvider) Balance() (*setting.BalanceResponse, error) {
	return &setting.BalanceResponse{
		Status:  "UNSUPPORTED",
		Balance: 0,
	}, errors.New("meta whatsapp: balance lookup is not supported by the Cloud API")
}

// VerifyWebhook handles Meta's subscription handshake (GET with hub.mode,
// hub.verify_token and hub.challenge). It returns the challenge to echo back
// with HTTP 200 and true when the token matches the configured verifyToken.
func (m *MetaProvider) VerifyWebhook(mode, token, challenge string) (string, bool) {
	expected := m.DefaultConfig.VerifyToken
	if mode != "subscribe" || helper.IsEmpty(expected) {
		return "", false
	}
	if !hmac.Equal([]byte(token), []byte(expected)) {
		return "", false
	}
	return challenge, true
}

// ValidateSignature checks the X-Hub-Signature-256 header ("sha256=<hex>")
// against the raw request body using the configured secretKey (App Secret).
func (m *MetaProvider) ValidateSignature(rawBody []byte, signatureHeader string) bool {
	secret := m.DefaultConfig.SecretKey
	if helper.IsEmpty(secret) || !strings.HasPrefix(signatureHeader, "sha256=") {
		return false
	}

	got, err := hex.DecodeString(strings.TrimPrefix(signatureHeader, "sha256="))
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	return hmac.Equal(got, mac.Sum(nil))
}

// ParseWebhook converts a Meta webhook body into the provider-neutral
// NotificationBody accepted by Notification. Only status updates are mapped;
// inbound messages are left in the returned raw payload.
func (m *MetaProvider) ParseWebhook(rawBody []byte) (*setting.NotificationBody, *MetaWebhookPayload, error) {
	var payload MetaWebhookPayload
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return nil, nil, err
	}

	body := &setting.NotificationBody{}
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			for _, s := range change.Value.Statuses {
				result := setting.NotificationResult{MessageID: s.ID}
				result.Status.Name = s.Status

				switch s.Status {
				case "delivered":
					result.Status.GroupName = "DELIVERED"
				case "read":
					result.Status.GroupName = "DELIVERED"
					result.SeenAt = s.Timestamp
				case "failed":
					result.Status.GroupName = "UNDELIVERED"
					if len(s.Errors) > 0 {
						result.Status.ID = s.Errors[0].Code
						result.Status.Description = s.Errors[0].Title
					}
				default:
					result.Status.GroupName = strings.ToUpper(s.Status)
				}

				body.Results = append(body.Results, result)
			}
		}
	}

	return body, &payload, nil
}

// buildMetaMessage turns SendParams into the type-specific part of a Cloud API message.
func buildMetaMessage(params setting.SendParams) (map[string]interface{}, error) {
	c := params.Content
	if c == nil {
		c = &setting.Content{}
	}

	// 1. Raw content map (e.g. from a whatsapp template file)
	if len(c.Content) > 0 {
		raw := c.Content

		// Already in Cloud API shape: {"type":"template","template":{...}}
		if t, ok := raw["type"].(string); ok && raw[t] != nil {
			msg := make(map[string]interface{}, len(raw))
			for k, v := range raw {
				msg[k] = v
			}
			return msg, nil
		}

		// Infobip-style template: {"templateName":..., "templateData":{"body":{"placeholders":[...]}}, "language":...}
		if name := helper.GetValueOfString(raw, "templateName"); helper.IsNotEmpty(name) {
			var placeholders []string
			if td, ok := raw["templateData"].(map[string]interface{}); ok {
				if b, ok := td["body"].(map[string]interface{}); ok {
					placeholders = toStringSlice(b["placeholders"])
				}
			}
			return metaTemplate(name, helper.GetValueOfString(raw, "language"), placeholders), nil
		}

		if text, ok := raw["text"].(string); ok && helper.IsNotEmpty(text) {
			return metaText(text), nil
		}
	}

	// 2. Template by name
	templateName := firstNonEmpty(c.TemplateName, c.Template, params.Template)
	if helper.IsNotEmpty(templateName) {
		placeholders := c.Placeholders
		if len(placeholders) == 0 && c.TemplateData != nil {
			placeholders = c.TemplateData.Body.Placeholders
		}
		if len(placeholders) == 0 {
			placeholders = params.Placeholders
		}
		msg := metaTemplate(templateName, c.Language, placeholders)
		if c.TemplateData != nil {
			addMetaTemplateExtras(msg, c.TemplateData.Header, c.TemplateData.Buttons)
		}
		return msg, nil
	}

	// 3. Media
	if helper.IsNotEmpty(c.MediaURL) {
		mediaType := strings.ToLower(c.Type)
		switch mediaType {
		case "image", "document", "video", "audio", "sticker":
		default:
			mediaType = "image"
		}

		media := map[string]interface{}{"link": c.MediaURL}
		if helper.IsNotEmpty(c.Caption) && (mediaType == "image" || mediaType == "document" || mediaType == "video") {
			media["caption"] = c.Caption
		}

		return map[string]interface{}{
			"type":    mediaType,
			mediaType: media,
		}, nil
	}

	// 4. Plain text (only deliverable inside the 24h customer service window)
	text := firstNonEmpty(c.Text, params.Text)
	if helper.IsEmpty(text) {
		return nil, errors.New("meta whatsapp: message has no text, template or media")
	}

	return metaText(text), nil
}

func metaText(text string) map[string]interface{} {
	return map[string]interface{}{
		"type": "text",
		"text": map[string]interface{}{
			"preview_url": false,
			"body":        text,
		},
	}
}

// addMetaTemplateExtras adds a template message's header and button
// parameters as Cloud API components: the header first, buttons last.
func addMetaTemplateExtras(msg map[string]interface{}, header *setting.TemplateHeader, buttons []setting.TemplateButton) {
	template, _ := msg["template"].(map[string]interface{})
	if template == nil {
		return
	}
	components, _ := template["components"].([]map[string]interface{})

	if header != nil {
		var parameter map[string]interface{}
		switch kind := strings.ToLower(header.Type); kind {
		case "text":
			if helper.IsNotEmpty(header.Placeholder) {
				parameter = map[string]interface{}{"type": "text", "text": header.Placeholder}
			}
		case "image", "video", "document":
			if helper.IsNotEmpty(header.MediaURL) {
				media := map[string]interface{}{"link": header.MediaURL}
				if kind == "document" && helper.IsNotEmpty(header.Filename) {
					media["filename"] = header.Filename
				}
				parameter = map[string]interface{}{"type": kind, kind: media}
			}
		case "location":
			if header.Latitude != nil && header.Longitude != nil {
				parameter = map[string]interface{}{"type": "location", "location": map[string]interface{}{
					"latitude": *header.Latitude, "longitude": *header.Longitude,
				}}
			}
		}
		if parameter != nil {
			components = append([]map[string]interface{}{{"type": "header", "parameters": []map[string]interface{}{parameter}}}, components...)
		}
	}

	for _, b := range buttons {
		var subType string
		var parameter map[string]interface{}
		switch strings.ToUpper(b.Type) {
		case "URL":
			subType, parameter = "url", map[string]interface{}{"type": "text", "text": b.Parameter}
		case "QUICK_REPLY":
			subType, parameter = "quick_reply", map[string]interface{}{"type": "payload", "payload": b.Parameter}
		case "COPY_CODE":
			subType, parameter = "copy_code", map[string]interface{}{"type": "coupon_code", "coupon_code": b.Parameter}
		default:
			continue
		}
		components = append(components, map[string]interface{}{
			"type": "button", "sub_type": subType, "index": strconv.Itoa(b.Index),
			"parameters": []map[string]interface{}{parameter},
		})
	}

	if len(components) > 0 {
		template["components"] = components
	}
}

func metaTemplate(name, language string, placeholders []string) map[string]interface{} {
	if helper.IsEmpty(language) {
		language = "en"
	}

	template := map[string]interface{}{
		"name":     name,
		"language": map[string]string{"code": language},
	}

	if len(placeholders) > 0 {
		parameters := make([]map[string]string, 0, len(placeholders))
		for _, p := range placeholders {
			parameters = append(parameters, map[string]string{"type": "text", "text": p})
		}
		template["components"] = []map[string]interface{}{
			{"type": "body", "parameters": parameters},
		}
	}

	return map[string]interface{}{
		"type":     "template",
		"template": template,
	}
}

// normalizeMetaPhone strips everything but digits; the Cloud API expects E.164 without "+".
func normalizeMetaPhone(phone string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, phone)
}

func withScheme(baseURL string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	return baseURL
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if helper.IsNotEmpty(v) {
			return v
		}
	}
	return ""
}

func toStringSlice(v interface{}) []string {
	switch list := v.(type) {
	case []string:
		return list
	case []interface{}:
		out := make([]string, 0, len(list))
		for _, item := range list {
			out = append(out, fmt.Sprint(item))
		}
		return out
	}
	return nil
}

// postJSON makes a POST request with a JSON body and bearer token.
func (m *MetaProvider) postJSON(url string, body interface{}, accessToken string) (int, []byte, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}
