package whatsapp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/robertkonga/yekonga-server-go/gateway/setting"
	"github.com/robertkonga/yekonga-server-go/helper"
)

// Ensure InfobipProvider supports template management.
var _ setting.WhatsappTemplateManager = (*InfobipProvider)(nil)

// InfobipError is an error body returned by the Infobip API. Both the classic
// requestError.serviceException shape and the newer errorCode/description
// shape are decoded into it.
type InfobipError struct {
	StatusCode int
	Code       string
	Text       string
	Details    []string
}

func (e *InfobipError) Error() string {
	msg := fmt.Sprintf("infobip whatsapp: %s (%s, HTTP %d)", e.Text, e.Code, e.StatusCode)
	if len(e.Details) > 0 {
		msg += ": " + strings.Join(e.Details, "; ")
	}
	return msg
}

// infobipTemplate is a template as Infobip returns it.
type infobipTemplate struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Language  string                 `json:"language"`
	Status    string                 `json:"status"`
	Category  string                 `json:"category"`
	Structure map[string]interface{} `json:"structure"`
}

// ListTemplates returns the sender's templates. Infobip has no server-side
// filtering or paging for this endpoint, so the params are applied locally
// and Paging is always empty.
// GET /whatsapp/2/senders/{sender}/templates
func (i *InfobipProvider) ListTemplates(params setting.WhatsappTemplateListParams) (*setting.WhatsappTemplateList, error) {
	endpoint, err := i.templatesURL()
	if err != nil {
		return nil, err
	}

	var response struct {
		Templates []infobipTemplate `json:"templates"`
	}
	if err := i.templateRequest(http.MethodGet, endpoint, nil, &response); err != nil {
		return nil, err
	}

	result := &setting.WhatsappTemplateList{Data: []setting.WhatsappTemplate{}}
	for _, t := range response.Templates {
		if helper.IsNotEmpty(params.Name) && !strings.Contains(t.Name, params.Name) {
			continue
		}
		if helper.IsNotEmpty(params.Status) && !strings.EqualFold(t.Status, params.Status) {
			continue
		}
		if helper.IsNotEmpty(params.Category) && !strings.EqualFold(t.Category, params.Category) {
			continue
		}
		if helper.IsNotEmpty(params.Language) && !strings.EqualFold(t.Language, params.Language) {
			continue
		}
		result.Data = append(result.Data, fromInfobipTemplate(t))
		if params.Limit > 0 && len(result.Data) >= params.Limit {
			break
		}
	}

	return result, nil
}

// GetTemplate returns a single template by its ID. It is looked up from the
// template list, which works on every Infobip account.
func (i *InfobipProvider) GetTemplate(id string) (*setting.WhatsappTemplate, error) {
	if helper.IsEmpty(id) {
		return nil, errors.New("infobip whatsapp: template id is required")
	}

	list, err := i.ListTemplates(setting.WhatsappTemplateListParams{})
	if err != nil {
		return nil, err
	}
	for _, t := range list.Data {
		if t.ID == id {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("infobip whatsapp: template %s not found", id)
}

// CreateTemplate submits a new template; Infobip forwards it to Meta for review.
// POST /whatsapp/2/senders/{sender}/templates
func (i *InfobipProvider) CreateTemplate(template setting.WhatsappTemplate) (*setting.WhatsappTemplateResult, error) {
	endpoint, err := i.templatesURL()
	if err != nil {
		return nil, err
	}

	if helper.IsEmpty(template.Name) || helper.IsEmpty(template.Language) || helper.IsEmpty(template.Category) {
		return nil, errors.New("infobip whatsapp: template name, language and category are required")
	}
	if len(template.Components) == 0 && len(template.Structure) == 0 {
		return nil, errors.New("infobip whatsapp: template needs components (at least a BODY) or a structure")
	}

	body := map[string]interface{}{
		"name":      template.Name,
		"language":  template.Language,
		"category":  strings.ToUpper(template.Category),
		"structure": toInfobipStructure(template),
	}

	var created infobipTemplate
	if err := i.templateRequest(http.MethodPost, endpoint, body, &created); err != nil {
		return nil, err
	}

	return &setting.WhatsappTemplateResult{
		Success:  helper.IsNotEmpty(created.ID),
		ID:       created.ID,
		Status:   created.Status,
		Category: created.Category,
	}, nil
}

// UpdateTemplate edits an existing template. Only category and
// components/structure are sent; name and language cannot be changed. Infobip
// only allows editing APPROVED, REJECTED or PAUSED templates, and not the
// category of an approved one.
// PATCH /whatsapp/2/senders/{sender}/templates/{id}
func (i *InfobipProvider) UpdateTemplate(id string, template setting.WhatsappTemplate) (*setting.WhatsappTemplateResult, error) {
	endpoint, err := i.templatesURL()
	if err != nil {
		return nil, err
	}
	if helper.IsEmpty(id) {
		return nil, errors.New("infobip whatsapp: template id is required")
	}

	body := map[string]interface{}{}
	if helper.IsNotEmpty(template.Category) {
		body["category"] = strings.ToUpper(template.Category)
	}
	if len(template.Components) > 0 || len(template.Structure) > 0 {
		body["structure"] = toInfobipStructure(template)
	}
	if len(body) == 0 {
		return nil, errors.New("infobip whatsapp: nothing to update (set category, components or structure)")
	}

	var updated infobipTemplate
	if err := i.templateRequest(http.MethodPatch, endpoint+"/"+url.PathEscape(id), body, &updated); err != nil {
		return nil, err
	}

	result := &setting.WhatsappTemplateResult{
		Success:  true,
		ID:       updated.ID,
		Status:   updated.Status,
		Category: updated.Category,
	}
	if helper.IsEmpty(result.ID) {
		result.ID = id
	}
	return result, nil
}

// DeleteTemplate deletes a template by name. Infobip deletes by name only, so
// id is ignored and every language version is removed.
// DELETE /whatsapp/2/senders/{sender}/templates/{templateName}
func (i *InfobipProvider) DeleteTemplate(name, id string) (*setting.WhatsappTemplateResult, error) {
	endpoint, err := i.templatesURL()
	if err != nil {
		return nil, err
	}
	if helper.IsEmpty(name) {
		return nil, errors.New("infobip whatsapp: template name is required")
	}

	if err := i.templateRequest(http.MethodDelete, endpoint+"/"+url.PathEscape(name), nil, nil); err != nil {
		return nil, err
	}
	return &setting.WhatsappTemplateResult{Success: true, ID: id}, nil
}

func (i *InfobipProvider) templatesURL() (string, error) {
	sender := i.DefaultConfig.Sender
	if helper.IsEmpty(sender) {
		return "", errors.New("infobip whatsapp: sender is required for template management")
	}
	if helper.IsEmpty(i.DefaultConfig.BaseURL) {
		return "", errors.New("infobip whatsapp: baseURL is required")
	}
	return fmt.Sprintf("%s/whatsapp/2/senders/%s/templates", withScheme(i.DefaultConfig.BaseURL), url.PathEscape(sender)), nil
}

// templateRequest sends a request and decodes a successful response into out.
// An error body is returned as *InfobipError.
func (i *InfobipProvider) templateRequest(method, endpoint string, body interface{}, out interface{}) error {
	if helper.IsEmpty(i.DefaultConfig.APIKey) {
		return errors.New("infobip whatsapp: apiKey is required")
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
	req.Header.Set("Authorization", "App "+i.DefaultConfig.APIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := i.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 300 {
		return parseInfobipError(resp.StatusCode, data)
	}

	if out != nil && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("infobip whatsapp: failed to parse response: %w", err)
		}
	}
	return nil
}

func parseInfobipError(statusCode int, data []byte) error {
	var envelope struct {
		RequestError *struct {
			ServiceException struct {
				MessageID        string      `json:"messageId"`
				Text             string      `json:"text"`
				ValidationErrors interface{} `json:"validationErrors"`
			} `json:"serviceException"`
		} `json:"requestError"`
		ErrorCode   string `json:"errorCode"`
		Description string `json:"description"`
		Violations  []struct {
			Property  string `json:"property"`
			Violation string `json:"violation"`
		} `json:"violations"`
	}

	apiErr := &InfobipError{StatusCode: statusCode}
	if err := json.Unmarshal(data, &envelope); err == nil {
		switch {
		case envelope.RequestError != nil:
			se := envelope.RequestError.ServiceException
			apiErr.Code, apiErr.Text = se.MessageID, se.Text
			if se.ValidationErrors != nil {
				if m, ok := se.ValidationErrors.(map[string]interface{}); ok {
					for field, v := range m {
						apiErr.Details = append(apiErr.Details, fmt.Sprintf("%s: %v", field, v))
					}
				} else {
					apiErr.Details = append(apiErr.Details, fmt.Sprint(se.ValidationErrors))
				}
			}
		case helper.IsNotEmpty(envelope.ErrorCode) || helper.IsNotEmpty(envelope.Description):
			apiErr.Code, apiErr.Text = envelope.ErrorCode, envelope.Description
			for _, v := range envelope.Violations {
				apiErr.Details = append(apiErr.Details, fmt.Sprintf("%s: %s", v.Property, v.Violation))
			}
		}
	}
	if helper.IsEmpty(apiErr.Text) {
		apiErr.Text = strings.TrimSpace(string(data))
		if apiErr.Text == "" {
			apiErr.Text = http.StatusText(statusCode)
		}
	}
	return apiErr
}

// ---- Components (Meta shape) <-> Infobip structure ----

func isAuthenticationTemplate(t setting.WhatsappTemplate) bool {
	if strings.EqualFold(t.Category, "AUTHENTICATION") {
		return true
	}
	for _, c := range t.Components {
		if c.AddSecurityRecommendation != nil || c.CodeExpirationMinutes != nil {
			return true
		}
		for _, b := range c.Buttons {
			if typ, _ := b["type"].(string); strings.EqualFold(typ, "OTP") {
				return true
			}
		}
	}
	return false
}

// toInfobipStructure builds Infobip's {header, body, footer, buttons} from
// Meta-style components, unless a native Structure was supplied.
func toInfobipStructure(t setting.WhatsappTemplate) map[string]interface{} {
	if len(t.Structure) > 0 {
		return t.Structure
	}

	auth := isAuthenticationTemplate(t)
	structure := map[string]interface{}{}

	for _, c := range t.Components {
		switch strings.ToUpper(c.Type) {
		case "HEADER":
			format := strings.ToUpper(c.Format)
			if format == "" {
				format = "TEXT"
			}
			header := map[string]interface{}{"format": format}
			if format == "TEXT" {
				header["text"] = c.Text
			}
			if ex := firstExample(c.Example, "header_text", "header_url", "header_handle"); ex != "" {
				header["example"] = ex
			}
			structure["header"] = header

		case "BODY":
			body := map[string]interface{}{}
			if auth {
				if c.AddSecurityRecommendation != nil {
					body["addSecurityRecommendation"] = *c.AddSecurityRecommendation
				}
			} else {
				body["text"] = c.Text
				if examples := exampleList(c.Example, "body_text"); len(examples) > 0 {
					body["examples"] = examples
				}
			}
			structure["body"] = body

		case "FOOTER":
			footer := map[string]interface{}{}
			if auth {
				if c.CodeExpirationMinutes != nil {
					footer["codeExpirationMinutes"] = *c.CodeExpirationMinutes
				}
			} else {
				footer["text"] = c.Text
			}
			structure["footer"] = footer

		case "BUTTONS":
			buttons := make([]map[string]interface{}, 0, len(c.Buttons))
			for _, b := range c.Buttons {
				buttons = append(buttons, toInfobipButton(b, auth))
			}
			structure["buttons"] = buttons
		}
	}

	if auth {
		// Infobip requires body and footer on authentication templates.
		if _, ok := structure["body"]; !ok {
			structure["body"] = map[string]interface{}{}
		}
		if _, ok := structure["footer"]; !ok {
			structure["footer"] = map[string]interface{}{}
		}
	}

	return structure
}

// toInfobipButton converts a Meta-style button (snake_case, example as array)
// into Infobip's (camelCase, example as string). Authentication buttons use
// otpType as their discriminator, so Meta's "type": "OTP" is dropped.
func toInfobipButton(b map[string]interface{}, auth bool) map[string]interface{} {
	out := make(map[string]interface{}, len(b))
	for k, v := range b {
		key := snakeToCamel(k)
		if key == "example" {
			v = firstExample(v)
		}
		out[key] = v
	}

	if typ, ok := out["type"].(string); ok {
		out["type"] = strings.ToUpper(typ)
	}
	if otp, ok := out["otpType"].(string); ok {
		out["otpType"] = strings.ToUpper(otp)
	}
	if auth && out["type"] == "OTP" {
		delete(out, "type")
	}
	return out
}

// fromInfobipTemplate converts an Infobip template into the shared Meta-style shape.
func fromInfobipTemplate(t infobipTemplate) setting.WhatsappTemplate {
	result := setting.WhatsappTemplate{
		ID:        t.ID,
		Name:      t.Name,
		Language:  t.Language,
		Category:  t.Category,
		Status:    t.Status,
		Structure: t.Structure,
	}
	auth := strings.EqualFold(t.Category, "AUTHENTICATION")

	if h, ok := t.Structure["header"].(map[string]interface{}); ok {
		c := setting.WhatsappTemplateComponent{
			Type:   "HEADER",
			Format: helper.GetValueOfString(h, "format"),
			Text:   helper.GetValueOfString(h, "text"),
		}
		if ex := helper.GetValueOfString(h, "example"); ex != "" {
			if strings.EqualFold(c.Format, "TEXT") {
				c.Example = map[string]interface{}{"header_text": []string{ex}}
			} else {
				c.Example = ex
			}
		}
		result.Components = append(result.Components, c)
	}

	if b, ok := t.Structure["body"].(map[string]interface{}); ok {
		c := setting.WhatsappTemplateComponent{Type: "BODY", Text: helper.GetValueOfString(b, "text")}
		if examples := toStringSlice(b["examples"]); len(examples) > 0 {
			c.Example = map[string]interface{}{"body_text": [][]string{examples}}
		}
		if v, ok := b["addSecurityRecommendation"].(bool); ok {
			c.AddSecurityRecommendation = &v
		}
		result.Components = append(result.Components, c)
	}

	if f, ok := t.Structure["footer"].(map[string]interface{}); ok {
		c := setting.WhatsappTemplateComponent{Type: "FOOTER", Text: helper.GetValueOfString(f, "text")}
		if v, ok := f["codeExpirationMinutes"].(float64); ok {
			minutes := int(v)
			c.CodeExpirationMinutes = &minutes
		}
		result.Components = append(result.Components, c)
	}

	if list, ok := t.Structure["buttons"].([]interface{}); ok && len(list) > 0 {
		c := setting.WhatsappTemplateComponent{Type: "BUTTONS"}
		for _, item := range list {
			b, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			button := make(map[string]interface{}, len(b))
			for k, v := range b {
				button[camelToSnake(k)] = v
			}
			if _, hasOtp := button["otp_type"]; (auth || hasOtp) && button["type"] == nil {
				button["type"] = "OTP"
			}
			if ex, ok := button["example"].(string); ok && button["type"] == "URL" {
				button["example"] = []string{ex}
			}
			c.Buttons = append(c.Buttons, button)
		}
		result.Components = append(result.Components, c)
	}

	return result
}

// firstExample returns the first sample value from a string, a list, a nested
// list, or a map keyed by one of keys (e.g. {"header_text": ["John"]}).
func firstExample(example interface{}, keys ...string) string {
	switch v := example.(type) {
	case nil:
		return ""
	case string:
		return v
	case []string:
		if len(v) > 0 {
			return v[0]
		}
	case [][]string:
		if len(v) > 0 && len(v[0]) > 0 {
			return v[0][0]
		}
	case []interface{}:
		if len(v) > 0 {
			return firstExample(v[0])
		}
	case map[string]interface{}:
		for _, k := range keys {
			if val, ok := v[k]; ok {
				return firstExample(val)
			}
		}
	}
	return ""
}

// exampleList returns body sample values from a flat list, Meta's nested
// [["a","b"]] form, or a map keyed by key (e.g. {"body_text": [["a","b"]]}).
func exampleList(example interface{}, key string) []string {
	switch v := example.(type) {
	case []string:
		return v
	case [][]string:
		if len(v) > 0 {
			return v[0]
		}
	case []interface{}:
		if len(v) > 0 {
			if _, nested := v[0].([]interface{}); nested {
				return toStringSlice(v[0])
			}
		}
		return toStringSlice(v)
	case map[string]interface{}:
		return exampleList(v[key], key)
	}
	return nil
}

func snakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}
