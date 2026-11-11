package whatsapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/robertkonga/yekonga-server-go/gateway/setting"
	"github.com/robertkonga/yekonga-server-go/helper"
)

// Ensure MetaProvider supports template management.
var _ setting.WhatsappTemplateManager = (*MetaProvider)(nil)

// Error makes MetaError usable as a Go error.
func (e *MetaError) Error() string {
	msg := fmt.Sprintf("meta whatsapp: %s (code %d", e.Message, e.Code)
	if e.ErrorSubcode != 0 {
		msg += fmt.Sprintf(", subcode %d", e.ErrorSubcode)
	}
	msg += ")"
	if e.ErrorData != nil && helper.IsNotEmpty(e.ErrorData.Details) {
		msg += ": " + e.ErrorData.Details
	}
	return msg
}

// ListTemplates returns one page of the business account's message templates.
// GET /{waba-id}/message_templates
func (m *MetaProvider) ListTemplates(params setting.WhatsappTemplateListParams) (*setting.WhatsappTemplateList, error) {
	endpoint, err := m.templatesURL()
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if helper.IsNotEmpty(params.Name) {
		query.Set("name", params.Name)
	}
	if helper.IsNotEmpty(params.Status) {
		query.Set("status", strings.ToUpper(params.Status))
	}
	if helper.IsNotEmpty(params.Category) {
		query.Set("category", strings.ToUpper(params.Category))
	}
	if helper.IsNotEmpty(params.Language) {
		query.Set("language", params.Language)
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if helper.IsNotEmpty(params.After) {
		query.Set("after", params.After)
	}
	if helper.IsNotEmpty(params.Before) {
		query.Set("before", params.Before)
	}
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var result setting.WhatsappTemplateList
	if err := m.graphRequest(http.MethodGet, endpoint, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetTemplate returns a single template by its ID.
// GET /{template-id}
func (m *MetaProvider) GetTemplate(id string) (*setting.WhatsappTemplate, error) {
	if helper.IsEmpty(id) {
		return nil, errors.New("meta whatsapp: template id is required")
	}

	var result setting.WhatsappTemplate
	endpoint := fmt.Sprintf("%s/%s", m.graphBaseURL(), url.PathEscape(id))
	if err := m.graphRequest(http.MethodGet, endpoint, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTemplate submits a new template for Meta review. The result's Status
// is usually PENDING; it becomes usable once APPROVED.
// POST /{waba-id}/message_templates
func (m *MetaProvider) CreateTemplate(template setting.WhatsappTemplate) (*setting.WhatsappTemplateResult, error) {
	endpoint, err := m.templatesURL()
	if err != nil {
		return nil, err
	}

	if helper.IsEmpty(template.Name) || helper.IsEmpty(template.Language) || helper.IsEmpty(template.Category) {
		return nil, errors.New("meta whatsapp: template name, language and category are required")
	}
	if len(template.Components) == 0 {
		return nil, errors.New("meta whatsapp: template needs at least a BODY component")
	}

	body := map[string]interface{}{
		"name":       template.Name,
		"language":   template.Language,
		"category":   strings.ToUpper(template.Category),
		"components": normalizeComponents(template.Components),
	}
	if helper.IsNotEmpty(template.ParameterFormat) {
		body["parameter_format"] = strings.ToUpper(template.ParameterFormat)
	}

	var result setting.WhatsappTemplateResult
	if err := m.graphRequest(http.MethodPost, endpoint, body, &result); err != nil {
		return nil, err
	}
	result.Success = helper.IsNotEmpty(result.ID)
	return &result, nil
}

// UpdateTemplate edits an existing template. Only category, components and
// parameterFormat are sent; name and language cannot be changed. Meta limits
// how often a template can be edited and re-reviews it afterwards.
// POST /{template-id}
func (m *MetaProvider) UpdateTemplate(id string, template setting.WhatsappTemplate) (*setting.WhatsappTemplateResult, error) {
	if helper.IsEmpty(id) {
		return nil, errors.New("meta whatsapp: template id is required")
	}

	body := map[string]interface{}{}
	if helper.IsNotEmpty(template.Category) {
		body["category"] = strings.ToUpper(template.Category)
	}
	if len(template.Components) > 0 {
		body["components"] = normalizeComponents(template.Components)
	}
	if helper.IsNotEmpty(template.ParameterFormat) {
		body["parameter_format"] = strings.ToUpper(template.ParameterFormat)
	}
	if len(body) == 0 {
		return nil, errors.New("meta whatsapp: nothing to update (set category, components or parameterFormat)")
	}

	var result setting.WhatsappTemplateResult
	endpoint := fmt.Sprintf("%s/%s", m.graphBaseURL(), url.PathEscape(id))
	if err := m.graphRequest(http.MethodPost, endpoint, body, &result); err != nil {
		return nil, err
	}
	result.ID = id
	return &result, nil
}

// DeleteTemplate deletes a template. With only name, every language version is
// deleted; with id as well, only that version is.
// DELETE /{waba-id}/message_templates?name=...&hsm_id=...
func (m *MetaProvider) DeleteTemplate(name, id string) (*setting.WhatsappTemplateResult, error) {
	endpoint, err := m.templatesURL()
	if err != nil {
		return nil, err
	}
	if helper.IsEmpty(name) {
		return nil, errors.New("meta whatsapp: template name is required")
	}

	query := url.Values{"name": {name}}
	if helper.IsNotEmpty(id) {
		query.Set("hsm_id", id)
	}

	var result setting.WhatsappTemplateResult
	if err := m.graphRequest(http.MethodDelete, endpoint+"?"+query.Encode(), nil, &result); err != nil {
		return nil, err
	}
	result.ID = id
	return &result, nil
}

// normalizeComponents upper-cases the enum fields Meta expects.
func normalizeComponents(components []setting.WhatsappTemplateComponent) []setting.WhatsappTemplateComponent {
	out := make([]setting.WhatsappTemplateComponent, len(components))
	for i, c := range components {
		c.Type = strings.ToUpper(c.Type)
		c.Format = strings.ToUpper(c.Format)
		out[i] = c
	}
	return out
}

func (m *MetaProvider) graphBaseURL() string {
	baseURL := m.DefaultConfig.BaseURL
	if helper.IsEmpty(baseURL) {
		baseURL = MetaDefaultBaseURL
	}
	return withScheme(baseURL)
}

func (m *MetaProvider) templatesURL() (string, error) {
	wabaID := m.DefaultConfig.BusinessAccountID
	if helper.IsEmpty(wabaID) {
		return "", errors.New("meta whatsapp: businessAccountId (WABA ID) is required for template management")
	}
	return fmt.Sprintf("%s/%s/message_templates", m.graphBaseURL(), url.PathEscape(wabaID)), nil
}

// graphRequest sends a Graph API request and decodes a successful response
// into out. A Graph error body is returned as *MetaError.
func (m *MetaProvider) graphRequest(method, endpoint string, body interface{}, out interface{}) error {
	if helper.IsEmpty(m.DefaultConfig.APIKey) {
		return errors.New("meta whatsapp: apiKey (access token) is required")
	}

	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+m.DefaultConfig.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var envelope struct {
		Error *MetaError `json:"error"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Error != nil {
		return envelope.Error
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("meta whatsapp: unexpected HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}

	if out != nil {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("meta whatsapp: failed to parse response: %w", err)
		}
	}
	return nil
}
