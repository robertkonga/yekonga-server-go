package setting

import (
	"github.com/robertkonga/yekonga-server-go/config"
)

// SmsRecipient represents an SMS recipient
type SmsRecipient struct {
	RecipientID int    `json:"recipient_id"`
	DestAddr    string `json:"dest_addr"`
}

// SendRequest represents the SMS send request body
type SendRequest struct {
	SourceAddr   string    `json:"source_addr"`
	ScheduleTime string    `json:"schedule_time"`
	Message      string    `json:"message"`
	Messages     []Message `json:"messages"`

	Encoding   int            `json:"encoding"`
	Recipients []SmsRecipient `json:"recipients"`
}

// SendResponse represents the response from send API
type SendResponse struct {
	Status    string      `json:"status"`
	Message   string      `json:"message"`
	MessageID string      `json:"messageId"`
	Code      int         `json:"code"`
	RequestID string      `json:"request_id"`
	Response  interface{} `json:"response"`
	Error     string      `json:"error,omitempty"`
}

// StatusResponse represents delivery status response
type StatusResponse struct {
	Status    string      `json:"status"`
	Message   string      `json:"message"`
	RequestID string      `json:"request_id"`
	Response  interface{} `json:"response"`
}

// BalanceResponse represents balance check response
type BalanceResponse struct {
	Status   string      `json:"status"`
	Balance  float64     `json:"balance"`
	Response interface{} `json:"response"`
	Data     BalanceData `json:"data,omitempty"`
}

// BalanceData holds the balance data from API
type BalanceData struct {
	CreditBalance float64 `json:"credit_balance"`
}

// Destination represents an SMS destination
type Destination struct {
	To string `json:"to"`
}

// Message represents a single SMS message
type Message struct {
	From         string        `json:"from"`
	Text         string        `json:"text"`
	Destinations []Destination `json:"destinations"`
}

// MessageStatus represents the status of a message
type MessageStatus struct {
	GroupID     int    `json:"groupId"`
	GroupName   string `json:"groupName"`
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ServiceException represents an error from the API
type ServiceException struct {
	MessageID string `json:"messageId"`
	Text      string `json:"text"`
}

// MessageResponse represents a message response
type MessageResponse struct {
	MessageID string        `json:"messageId"`
	Status    MessageStatus `json:"status"`
	To        string        `json:"to"`
}

// RequestError represents request error
type RequestError struct {
	ServiceException ServiceException `json:"serviceException"`
}

// DeliveryResult represents a single delivery status result
type DeliveryResult struct {
	MessageID string        `json:"messageId"`
	Status    MessageStatus `json:"status"`
}

// DeliveryStatusBody represents the delivery status callback body
type DeliveryStatusBody struct {
	Results []DeliveryResult `json:"results"`
}

// Config holds WhatsApp configuration
type Config struct {
	BaseURL string
	APIKey  string
	Sender  string
}

// Content represents message content
type Content struct {
	Type         string                 `json:"type,omitempty"`
	Template     string                 `json:"template,omitempty"`
	TemplateName string                 `json:"templateName,omitempty"`
	Placeholders []string               `json:"placeholders,omitempty"`
	Text         string                 `json:"text,omitempty"`
	MediaURL     string                 `json:"mediaUrl,omitempty"`
	Caption      string                 `json:"caption,omitempty"`
	Content      map[string]interface{} `json:"content,omitempty"`
	TemplateData *TemplateData          `json:"templateData,omitempty"`
	Language     string                 `json:"language,omitempty"`
}

// TemplateData represents template data structure
type TemplateData struct {
	Body    TemplateBody     `json:"body"`
	Header  *TemplateHeader  `json:"header,omitempty"`
	Buttons []TemplateButton `json:"buttons,omitempty"`
}

// TemplateHeader fills a template's header (Infobip shape): the TEXT
// header's {{1}}, or the link of its IMAGE, VIDEO or DOCUMENT.
type TemplateHeader struct {
	Type        string   `json:"type"`                  // TEXT, IMAGE, VIDEO, DOCUMENT, LOCATION
	Placeholder string   `json:"placeholder,omitempty"` // TEXT
	MediaURL    string   `json:"mediaUrl,omitempty"`    // IMAGE, VIDEO, DOCUMENT
	Filename    string   `json:"filename,omitempty"`    // DOCUMENT
	Latitude    *float64 `json:"latitude,omitempty"`    // LOCATION
	Longitude   *float64 `json:"longitude,omitempty"`   // LOCATION
}

// TemplateButton fills one button that takes a value: a URL button's
// {{1}} suffix, a QUICK_REPLY's payload or a COPY_CODE's code. Infobip
// takes them in the template's order; Meta needs Index, the button's
// position among all the template's buttons.
type TemplateButton struct {
	Type      string `json:"type"` // URL, QUICK_REPLY, COPY_CODE
	Parameter string `json:"parameter"`
	Index     int    `json:"-"`
}

// TemplateBody represents template body with placeholders
type TemplateBody struct {
	Placeholders []string `json:"placeholders"`
}

// SendParams holds parameters for sending messages
type SendParams struct {
	Phone        string
	Text         string
	Content      *Content
	Domain       string
	SenderID     string
	Config       *Config
	Template     string
	Placeholders []string
}

// APIResponse represents API response
type APIResponse struct {
	Messages     []MessageResponse `json:"messages,omitempty"`
	RequestError *RequestError     `json:"requestError,omitempty"`
}

// NotificationResult represents delivery notification result
type NotificationResult struct {
	MessageID string        `json:"messageId"`
	Status    MessageStatus `json:"status"`
	SeenAt    string        `json:"seenAt,omitempty"`
}

// NotificationBody represents notification callback body
type NotificationBody struct {
	Results []NotificationResult `json:"results"`
}

// InfoBipBalanceResponse represents InfoBip balance API response
type InfoBipBalanceResponse struct {
	Balance  float64 `json:"balance"`
	Currency string  `json:"currency"`
}

type SMSProvider interface {
	Send(params SendParams, config *config.SMSGatewayConfig) (*SendResponse, error)
	CheckStatus(phone, requestID string, config *config.SMSGatewayConfig) (*StatusResponse, error)
	Balance() (*BalanceResponse, error)
}

// WhatsappProvider interface for different providers
type WhatsappProvider interface {
	Send(params SendParams, config *config.WhatsappGatewayConfig) (*SendResponse, error)
	Notification(providerParams map[string]string, body *NotificationBody) error
	DeliveryStatus(providerParams map[string]string, result NotificationResult) error
	Balance() (*BalanceResponse, error)
}

// WhatsappTemplateManager is implemented by WhatsApp providers that can manage
// message templates. Not every provider supports it; type-assert a
// WhatsappProvider to check.
type WhatsappTemplateManager interface {
	ListTemplates(params WhatsappTemplateListParams) (*WhatsappTemplateList, error)
	GetTemplate(id string) (*WhatsappTemplate, error)
	CreateTemplate(template WhatsappTemplate) (*WhatsappTemplateResult, error)
	// UpdateTemplate edits an existing template. Only category, components and
	// parameterFormat can change; name and language are fixed once created.
	UpdateTemplate(id string, template WhatsappTemplate) (*WhatsappTemplateResult, error)
	// DeleteTemplate deletes a template by name (all languages), or only the
	// one language version identified by id when id is set.
	DeleteTemplate(name, id string) (*WhatsappTemplateResult, error)
}

// WhatsappTemplate is a WhatsApp message template, in the Meta Cloud API shape.
type WhatsappTemplate struct {
	ID              string                      `json:"id,omitempty"`
	Name            string                      `json:"name,omitempty"`             // lowercase letters, digits and underscores
	Language        string                      `json:"language,omitempty"`         // e.g. en, en_US, sw
	Category        string                      `json:"category,omitempty"`         // MARKETING, UTILITY, AUTHENTICATION
	Status          string                      `json:"status,omitempty"`           // APPROVED, PENDING, REJECTED, PAUSED, DISABLED (read-only)
	ParameterFormat string                      `json:"parameter_format,omitempty"` // POSITIONAL ({{1}}) or NAMED ({{name}})
	Components      []WhatsappTemplateComponent `json:"components,omitempty"`
	RejectedReason  string                      `json:"rejected_reason,omitempty"` // read-only

	// Structure is the provider-native template body (Infobip only). When set on
	// create/update it is sent as-is instead of being built from Components, for
	// features Components can't express (e.g. carousel). Filled on reads.
	Structure map[string]interface{} `json:"structure,omitempty"`
}

// WhatsappTemplateComponent is one block of a template (HEADER, BODY, FOOTER, BUTTONS).
type WhatsappTemplateComponent struct {
	Type    string                   `json:"type"`             // HEADER, BODY, FOOTER, BUTTONS
	Format  string                   `json:"format,omitempty"` // HEADER only: TEXT, IMAGE, VIDEO, DOCUMENT, LOCATION
	Text    string                   `json:"text,omitempty"`
	Example interface{}              `json:"example,omitempty"` // sample values, e.g. {"body_text": [["John", "1234"]]}
	Buttons []map[string]interface{} `json:"buttons,omitempty"` // e.g. {"type": "QUICK_REPLY", "text": "Yes"}

	// AUTHENTICATION templates only
	AddSecurityRecommendation *bool `json:"add_security_recommendation,omitempty"` // BODY
	CodeExpirationMinutes     *int  `json:"code_expiration_minutes,omitempty"`     // FOOTER
}

// WhatsappTemplateListParams filters and paginates ListTemplates. All fields are optional.
type WhatsappTemplateListParams struct {
	Name     string // matches templates whose name contains this value
	Status   string
	Category string
	Language string
	Limit    int
	After    string // cursor from WhatsappTemplateList.Paging.Cursors.After
	Before   string
}

// WhatsappTemplateList is a page of templates.
type WhatsappTemplateList struct {
	Data   []WhatsappTemplate `json:"data"`
	Paging struct {
		Cursors struct {
			Before string `json:"before,omitempty"`
			After  string `json:"after,omitempty"`
		} `json:"cursors"`
		Next     string `json:"next,omitempty"`
		Previous string `json:"previous,omitempty"`
	} `json:"paging"`
}

// WhatsappTemplateResult is returned by create, update and delete.
type WhatsappTemplateResult struct {
	Success  bool   `json:"success"`
	ID       string `json:"id,omitempty"`
	Status   string `json:"status,omitempty"`
	Category string `json:"category,omitempty"`
}
