package whatsapp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/robertkonga/yekonga-server-go/gateway/setting"
)

type graphCall struct {
	Method string
	Path   string
	Query  url.Values
	Body   map[string]interface{}
}

func newTemplateServer(t *testing.T, status int, reply string, call *graphCall) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer TOKEN" {
			t.Errorf("Authorization = %q", got)
		}
		call.Method, call.Path, call.Query = r.Method, r.URL.Path, r.URL.Query()
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			if err := json.Unmarshal(raw, &call.Body); err != nil {
				t.Fatalf("bad request body: %v", err)
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(reply))
	}))
}

func newTemplateProvider(baseURL string) *MetaProvider {
	p := newMetaTestProvider(baseURL)
	p.DefaultConfig.BusinessAccountID = "WABA"
	return p
}

func TestMetaListTemplates(t *testing.T) {
	var call graphCall
	srv := newTemplateServer(t, 200, `{"data":[{"id":"1","name":"otp","language":"en","category":"AUTHENTICATION","status":"APPROVED","components":[{"type":"BODY","text":"{{1}} is your code"}]}],"paging":{"cursors":{"after":"CUR"}}}`, &call)
	defer srv.Close()

	list, err := newTemplateProvider(srv.URL).ListTemplates(setting.WhatsappTemplateListParams{Status: "approved", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if call.Method != "GET" || call.Path != "/WABA/message_templates" || call.Query.Get("status") != "APPROVED" || call.Query.Get("limit") != "10" {
		t.Fatalf("unexpected call: %+v", call)
	}
	if len(list.Data) != 1 || list.Data[0].Name != "otp" || list.Data[0].Components[0].Text == "" || list.Paging.Cursors.After != "CUR" {
		t.Fatalf("unexpected list: %+v", list)
	}
}

func TestMetaCreateTemplate(t *testing.T) {
	var call graphCall
	srv := newTemplateServer(t, 200, `{"id":"123","status":"PENDING","category":"UTILITY"}`, &call)
	defer srv.Close()

	res, err := newTemplateProvider(srv.URL).CreateTemplate(setting.WhatsappTemplate{
		Name:     "order_update",
		Language: "en_US",
		Category: "utility",
		Components: []setting.WhatsappTemplateComponent{
			{Type: "body", Text: "Hi {{1}}, order {{2}} shipped", Example: map[string]interface{}{"body_text": [][]string{{"John", "A1"}}}},
			{Type: "BUTTONS", Buttons: []map[string]interface{}{{"type": "QUICK_REPLY", "text": "Thanks"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success || res.ID != "123" || res.Status != "PENDING" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if call.Method != "POST" || call.Path != "/WABA/message_templates" || call.Body["category"] != "UTILITY" || call.Body["name"] != "order_update" {
		t.Fatalf("unexpected call: %+v", call)
	}
	if typ := call.Body["components"].([]interface{})[0].(map[string]interface{})["type"]; typ != "BODY" {
		t.Fatalf("component type not normalized: %v", typ)
	}
}

func TestMetaUpdateTemplate(t *testing.T) {
	var call graphCall
	srv := newTemplateServer(t, 200, `{"success":true}`, &call)
	defer srv.Close()

	res, err := newTemplateProvider(srv.URL).UpdateTemplate("123", setting.WhatsappTemplate{
		Name:       "ignored",
		Components: []setting.WhatsappTemplateComponent{{Type: "BODY", Text: "New text {{1}}"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success || res.ID != "123" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if call.Method != "POST" || call.Path != "/123" || call.Body["name"] != nil || call.Body["components"] == nil {
		t.Fatalf("unexpected call: %+v", call)
	}
}

func TestMetaDeleteTemplate(t *testing.T) {
	var call graphCall
	srv := newTemplateServer(t, 200, `{"success":true}`, &call)
	defer srv.Close()

	res, err := newTemplateProvider(srv.URL).DeleteTemplate("order_update", "123")
	if err != nil || !res.Success {
		t.Fatalf("unexpected result: %+v, %v", res, err)
	}
	if call.Method != "DELETE" || call.Query.Get("name") != "order_update" || call.Query.Get("hsm_id") != "123" {
		t.Fatalf("unexpected call: %+v", call)
	}
}

func TestMetaTemplateError(t *testing.T) {
	var call graphCall
	srv := newTemplateServer(t, 400, `{"error":{"message":"Invalid parameter","type":"OAuthException","code":100,"error_subcode":2388023,"error_data":{"details":"Content in this language already exists"}}}`, &call)
	defer srv.Close()

	_, err := newTemplateProvider(srv.URL).CreateTemplate(setting.WhatsappTemplate{
		Name: "dup", Language: "en", Category: "UTILITY",
		Components: []setting.WhatsappTemplateComponent{{Type: "BODY", Text: "x"}},
	})
	var metaErr *MetaError
	if !errors.As(err, &metaErr) || metaErr.Code != 100 || metaErr.ErrorSubcode != 2388023 {
		t.Fatalf("expected MetaError, got %v", err)
	}
}

func TestMetaTemplatesRequireWABA(t *testing.T) {
	if _, err := newMetaTestProvider("http://unused").ListTemplates(setting.WhatsappTemplateListParams{}); err == nil {
		t.Fatal("expected error without businessAccountId")
	}
}
