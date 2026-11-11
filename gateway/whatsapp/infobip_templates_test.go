package whatsapp

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/gateway/setting"
)

func newInfobipTemplateServer(t *testing.T, status int, reply string, call *graphCall) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "App KEY" {
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

func newInfobipTemplateProvider(baseURL string) *InfobipProvider {
	return &InfobipProvider{
		DefaultConfig: &config.WhatsappGatewayConfig{
			Provider: config.ProviderInfobip,
			BaseURL:  baseURL,
			APIKey:   "KEY",
			Sender:   "255700000000",
		},
		HTTPClient: http.DefaultClient,
	}
}

const infobipTemplatesReply = `{"templates":[
 {"id":"111","businessAccountId":9,"name":"order_update","language":"en","status":"APPROVED","category":"UTILITY",
  "structure":{"header":{"format":"TEXT","text":"Order {{1}}","example":"A100"},
   "body":{"text":"Hi {{1}}, shipped","examples":["John"]},"footer":{"text":"Thanks"},
   "buttons":[{"type":"URL","text":"Track","url":"https://x.co/{{1}}","example":"https://x.co/A100"},{"type":"PHONE_NUMBER","text":"Call","phoneNumber":"255700000001"}],"type":"TEXT"}},
 {"id":"222","name":"login_code","language":"en","status":"PENDING","category":"AUTHENTICATION",
  "structure":{"body":{"addSecurityRecommendation":true},"footer":{"codeExpirationMinutes":5},"buttons":[{"otpType":"COPY_CODE","text":"Copy"}]}}
]}`

func TestInfobipListTemplates(t *testing.T) {
	var call graphCall
	srv := newInfobipTemplateServer(t, 200, infobipTemplatesReply, &call)
	defer srv.Close()

	p := newInfobipTemplateProvider(srv.URL)
	list, err := p.ListTemplates(setting.WhatsappTemplateListParams{})
	if err != nil {
		t.Fatal(err)
	}
	if call.Method != "GET" || call.Path != "/whatsapp/2/senders/255700000000/templates" {
		t.Fatalf("unexpected call: %+v", call)
	}
	if len(list.Data) != 2 {
		t.Fatalf("want 2 templates, got %d", len(list.Data))
	}

	order := list.Data[0]
	if order.ID != "111" || len(order.Components) != 4 || order.Structure == nil {
		t.Fatalf("unexpected template: %+v", order)
	}
	if order.Components[1].Type != "BODY" || exampleList(order.Components[1].Example, "body_text")[0] != "John" {
		t.Fatalf("unexpected body: %+v", order.Components[1])
	}
	buttons := order.Components[3].Buttons
	if buttons[1]["phone_number"] != "255700000001" || buttons[0]["example"].([]string)[0] != "https://x.co/A100" {
		t.Fatalf("unexpected buttons: %+v", buttons)
	}

	auth := list.Data[1]
	if *auth.Components[0].AddSecurityRecommendation != true || *auth.Components[1].CodeExpirationMinutes != 5 ||
		auth.Components[2].Buttons[0]["type"] != "OTP" || auth.Components[2].Buttons[0]["otp_type"] != "COPY_CODE" {
		t.Fatalf("unexpected auth template: %+v", auth)
	}

	filtered, _ := p.ListTemplates(setting.WhatsappTemplateListParams{Status: "pending"})
	if len(filtered.Data) != 1 || filtered.Data[0].ID != "222" {
		t.Fatalf("status filter failed: %+v", filtered.Data)
	}

	one, err := p.GetTemplate("222")
	if err != nil || one.Name != "login_code" {
		t.Fatalf("GetTemplate: %+v, %v", one, err)
	}
}

func TestInfobipCreateTemplate(t *testing.T) {
	var call graphCall
	srv := newInfobipTemplateServer(t, 200, `{"id":"333","name":"order_update","language":"en","status":"PENDING","category":"UTILITY","structure":{}}`, &call)
	defer srv.Close()

	res, err := newInfobipTemplateProvider(srv.URL).CreateTemplate(setting.WhatsappTemplate{
		Name: "order_update", Language: "en", Category: "utility",
		Components: []setting.WhatsappTemplateComponent{
			{Type: "HEADER", Format: "TEXT", Text: "Order {{1}}", Example: map[string]interface{}{"header_text": []string{"A100"}}},
			{Type: "BODY", Text: "Hi {{1}}", Example: map[string]interface{}{"body_text": [][]string{{"John"}}}},
			{Type: "BUTTONS", Buttons: []map[string]interface{}{
				{"type": "URL", "text": "Track", "url": "https://x.co/{{1}}", "example": []string{"https://x.co/A100"}},
				{"type": "PHONE_NUMBER", "text": "Call", "phone_number": "255700000001"},
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Success || res.ID != "333" || res.Status != "PENDING" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if call.Method != "POST" || call.Body["category"] != "UTILITY" {
		t.Fatalf("unexpected call: %+v", call)
	}

	s := call.Body["structure"].(map[string]interface{})
	if s["header"].(map[string]interface{})["example"] != "A100" {
		t.Fatalf("unexpected header: %v", s["header"])
	}
	if s["body"].(map[string]interface{})["examples"].([]interface{})[0] != "John" {
		t.Fatalf("unexpected body: %v", s["body"])
	}
	btns := s["buttons"].([]interface{})
	if btns[0].(map[string]interface{})["example"] != "https://x.co/A100" || btns[1].(map[string]interface{})["phoneNumber"] != "255700000001" {
		t.Fatalf("unexpected buttons: %v", btns)
	}
}

func TestInfobipCreateAuthTemplate(t *testing.T) {
	var call graphCall
	srv := newInfobipTemplateServer(t, 200, `{"id":"444","status":"PENDING","category":"AUTHENTICATION"}`, &call)
	defer srv.Close()

	yes, five := true, 5
	_, err := newInfobipTemplateProvider(srv.URL).CreateTemplate(setting.WhatsappTemplate{
		Name: "login_code", Language: "en", Category: "AUTHENTICATION",
		Components: []setting.WhatsappTemplateComponent{
			{Type: "BODY", AddSecurityRecommendation: &yes},
			{Type: "FOOTER", CodeExpirationMinutes: &five},
			{Type: "BUTTONS", Buttons: []map[string]interface{}{{"type": "OTP", "otp_type": "copy_code", "text": "Copy"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	s := call.Body["structure"].(map[string]interface{})
	btn := s["buttons"].([]interface{})[0].(map[string]interface{})
	if s["body"].(map[string]interface{})["addSecurityRecommendation"] != true ||
		s["footer"].(map[string]interface{})["codeExpirationMinutes"] != float64(5) ||
		btn["otpType"] != "COPY_CODE" || btn["type"] != nil {
		t.Fatalf("unexpected auth structure: %v", s)
	}
}

func TestInfobipUpdateAndDeleteTemplate(t *testing.T) {
	var call graphCall
	srv := newInfobipTemplateServer(t, 200, `{"id":"111","status":"PENDING","category":"UTILITY"}`, &call)
	defer srv.Close()
	p := newInfobipTemplateProvider(srv.URL)

	res, err := p.UpdateTemplate("111", setting.WhatsappTemplate{
		Name:       "ignored",
		Structure:  map[string]interface{}{"body": map[string]interface{}{"text": "raw"}},
		Components: []setting.WhatsappTemplateComponent{{Type: "BODY", Text: "unused"}},
	})
	if err != nil || !res.Success || res.Status != "PENDING" {
		t.Fatalf("update: %+v, %v", res, err)
	}
	if call.Method != "PATCH" || call.Path != "/whatsapp/2/senders/255700000000/templates/111" || call.Body["name"] != nil ||
		call.Body["structure"].(map[string]interface{})["body"].(map[string]interface{})["text"] != "raw" {
		t.Fatalf("unexpected update call: %+v", call)
	}

	call = graphCall{}
	if _, err := p.DeleteTemplate("order_update", ""); err != nil {
		t.Fatal(err)
	}
	if call.Method != "DELETE" || call.Path != "/whatsapp/2/senders/255700000000/templates/order_update" {
		t.Fatalf("unexpected delete call: %+v", call)
	}
}

func TestInfobipTemplateErrors(t *testing.T) {
	cases := map[string]string{
		"classic": `{"requestError":{"serviceException":{"messageId":"BAD_REQUEST","text":"Bad request","validationErrors":{"name":["must match pattern"]}}}}`,
		"new":     `{"errorCode":"E400","description":"Bad request","action":"Check the request","violations":[{"property":"name","violation":"must match pattern"}],"resources":[]}`,
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			var call graphCall
			srv := newInfobipTemplateServer(t, 400, reply, &call)
			defer srv.Close()

			_, err := newInfobipTemplateProvider(srv.URL).ListTemplates(setting.WhatsappTemplateListParams{})
			var apiErr *InfobipError
			if !errors.As(err, &apiErr) || apiErr.Text != "Bad request" || apiErr.StatusCode != 400 || len(apiErr.Details) != 1 {
				t.Fatalf("unexpected error: %#v", err)
			}
		})
	}
}
