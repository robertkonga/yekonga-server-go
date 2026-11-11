package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/gateway/setting"
)

func newMetaTestServer(t *testing.T, status int, reply string, captured *map[string]interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/PNID/messages" {
			t.Errorf("path = %s, want /PNID/messages", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer TOKEN" {
			t.Errorf("Authorization = %q", got)
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, captured); err != nil {
			t.Fatalf("bad request body: %v", err)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
}

func newMetaTestProvider(baseURL string) *MetaProvider {
	return &MetaProvider{
		DefaultConfig: &config.WhatsappGatewayConfig{
			Provider:    config.ProviderMeta,
			BaseURL:     baseURL,
			APIKey:      "TOKEN",
			Sender:      "PNID",
			SecretKey:   "app-secret",
			VerifyToken: "verify-me",
		},
		HTTPClient: http.DefaultClient,
	}
}

func TestMetaSendText(t *testing.T) {
	var body map[string]interface{}
	srv := newMetaTestServer(t, 200, `{"messaging_product":"whatsapp","contacts":[{"input":"255700000000","wa_id":"255700000000"}],"messages":[{"id":"wamid.ABC","message_status":"accepted"}]}`, &body)
	defer srv.Close()

	res, err := newMetaTestProvider(srv.URL).Send(setting.SendParams{Phone: "+255 700 000 000", Text: "hello"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "SUCCESS" || res.MessageID != "wamid.ABC" {
		t.Fatalf("unexpected response: %+v", res)
	}
	if body["to"] != "255700000000" || body["type"] != "text" || body["messaging_product"] != "whatsapp" {
		t.Fatalf("unexpected request: %v", body)
	}
	if body["text"].(map[string]interface{})["body"] != "hello" {
		t.Fatalf("unexpected text: %v", body["text"])
	}
}

func TestMetaSendTemplateFromInfobipShape(t *testing.T) {
	var body map[string]interface{}
	srv := newMetaTestServer(t, 200, `{"messages":[{"id":"wamid.T"}]}`, &body)
	defer srv.Close()

	params := setting.SendParams{
		Phone: "255700000000",
		Content: &setting.Content{Content: map[string]interface{}{
			"templateName": "otp",
			"language":     "en_US",
			"templateData": map[string]interface{}{"body": map[string]interface{}{"placeholders": []interface{}{"1234"}}},
		}},
	}
	if _, err := newMetaTestProvider(srv.URL).Send(params, nil); err != nil {
		t.Fatal(err)
	}

	tpl := body["template"].(map[string]interface{})
	if body["type"] != "template" || tpl["name"] != "otp" || tpl["language"].(map[string]interface{})["code"] != "en_US" {
		t.Fatalf("unexpected template: %v", body)
	}
	param := tpl["components"].([]interface{})[0].(map[string]interface{})["parameters"].([]interface{})[0].(map[string]interface{})
	if param["text"] != "1234" {
		t.Fatalf("unexpected parameter: %v", param)
	}
}

func TestMetaSendError(t *testing.T) {
	var body map[string]interface{}
	srv := newMetaTestServer(t, 400, `{"error":{"message":"(#131030) Recipient phone number not in allowed list","type":"OAuthException","code":131030}}`, &body)
	defer srv.Close()

	res, err := newMetaTestProvider(srv.URL).Send(setting.SendParams{Phone: "255700000000", Text: "hi"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "FAILED" || res.Code != 131030 {
		t.Fatalf("unexpected response: %+v", res)
	}
}

func TestMetaWebhook(t *testing.T) {
	p := newMetaTestProvider("")

	if c, ok := p.VerifyWebhook("subscribe", "verify-me", "42"); !ok || c != "42" {
		t.Fatal("expected verification to pass")
	}
	if _, ok := p.VerifyWebhook("subscribe", "wrong", "42"); ok {
		t.Fatal("expected verification to fail")
	}

	raw := []byte(`{"object":"whatsapp_business_account","entry":[{"id":"1","changes":[{"field":"messages","value":{"statuses":[{"id":"wamid.R","status":"read","timestamp":"1700000000","recipient_id":"255700000000"}]}}]}]}`)
	mac := hmac.New(sha256.New, []byte("app-secret"))
	mac.Write(raw)
	if !p.ValidateSignature(raw, "sha256="+hex.EncodeToString(mac.Sum(nil))) {
		t.Fatal("expected signature to validate")
	}
	if p.ValidateSignature(raw, "sha256=00") {
		t.Fatal("expected bad signature to fail")
	}

	nb, _, err := p.ParseWebhook(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(nb.Results) != 1 || nb.Results[0].MessageID != "wamid.R" || nb.Results[0].Status.GroupName != "DELIVERED" || nb.Results[0].SeenAt == "" {
		t.Fatalf("unexpected parse: %+v", nb)
	}
}
