package Yekonga

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/robertkonga/yekonga-server/datatype"
	"github.com/robertkonga/yekonga-server/helper"
)

type RequestContext struct {
	Auth             *AuthPayload
	App              *YekongaData
	Request          *Request
	Client           *ClientPayload
	TokenPayload     *TokenPayload
	QuerySelectors   map[uint][]string
	QueryRelatedData datatype.JsonObject
	QueryWhereData   datatype.JsonObject
	mut              sync.RWMutex
}

type AuthPayload struct {
	ID           string
	ProfileID    string
	Username     string
	UsernameType string
	FirstName    string
	LastName     string
	Phone        string
	Email        string
	Whatsapp     string
	Role         string
	Extracts     map[string]interface{}
}

func (a *AuthPayload) ToMap() map[string]interface{} {
	// // Marshal the struct into JSON
	// jsonData, _ := json.Marshal(a)

	// // Unmarshal the JSON into a map[string]interface{}
	// var result map[string]interface{}
	// json.Unmarshal(jsonData, &result)

	result := map[string]interface{}{
		"id":           a.ID,
		"profileId":    a.ProfileID,
		"username":     a.Username,
		"usernameType": a.UsernameType,
		"firstName":    a.FirstName,
		"lastName":     a.LastName,
		"phone":        a.Phone,
		"email":        a.Email,
		"whatsapp":     a.Whatsapp,
		"role":         a.Role,
		"extracts":     a.Extracts,
	}

	return result
}

func (a *AuthPayload) ToJson() string {
	result, _ := json.Marshal(a.ToMap())

	return string(result)
}

type TokenPayload struct {
	Domain      string
	ProfileId   string
	UserId      string
	AdminId     string
	ExpiredTime string
}

func (a *TokenPayload) ToMap() map[string]interface{} {
	// // Marshal the struct into JSON
	// jsonData, _ := json.Marshal(a)

	// // Unmarshal the JSON into a map[string]interface{}
	// var result map[string]interface{}
	// json.Unmarshal(jsonData, &result)

	result := map[string]interface{}{
		"domain":      a.Domain,
		"userId":      a.UserId,
		"profileId":   a.ProfileId,
		"adminId":     a.AdminId,
		"expiredTime": a.ExpiredTime,
	}

	return result
}

func (a *TokenPayload) ToJson() string {
	result, _ := json.Marshal(a.ToMap())

	return string(result)
}

type ClientPayload struct {
	Origin string
	Host   string
	Port   string
	Proto  string
	Path   string
	Method string
}

func (a *ClientPayload) ToMap() map[string]interface{} {
	// // Marshal the struct into JSON
	// jsonData, _ := json.Marshal(a)

	// // Unmarshal the JSON into a map[string]interface{}
	// var result map[string]interface{}
	// json.Unmarshal(jsonData, &result)

	result := map[string]interface{}{
		"origin":   a.Origin,
		"host":     a.Host,
		"port":     a.Port,
		"protocol": a.Proto,
	}

	return result
}

func (a *ClientPayload) ToJson() string {
	result, _ := json.Marshal(a.ToMap())

	return string(result)
}

// Request represents an HTTP request with additional context
type Request struct {
	HttpRequest   *http.Request
	RawBody       interface{}
	Context       datatype.Context
	ContextObject datatype.ContextObject
	Params        map[string]string
	next          func(Request, Response)
	mut           sync.RWMutex
	App           *YekongaData
}

// Request methods
func (r *Request) SetContext(key string, value interface{}) {
	r.mut.Lock()
	defer r.mut.Unlock()
	if r.Context == nil {
		r.Context = make(datatype.Context)
	}

	r.Context[key] = value
}

func (r *Request) GetContext(key string) interface{} {
	r.mut.RLock()
	defer r.mut.RUnlock()
	if r.Context == nil {
		return nil
	}

	return r.Context[key]
}

// Request methods
func (r *Request) SetContextObject(key string, value map[string]interface{}) {
	r.mut.Lock()
	defer r.mut.Unlock()
	if r.ContextObject == nil {
		r.ContextObject = make(map[string]map[string]interface{})
	}

	r.ContextObject[key] = value
}

func (r *Request) GetContextObject(key string) map[string]interface{} {
	r.mut.RLock()
	defer r.mut.RUnlock()
	if r.ContextObject == nil {
		return nil
	}

	return r.ContextObject[key]
}

func (r *Request) Param(name string) string {
	if r.Params == nil {
		return ""
	}

	return r.Params[name]
}

// New query parameter methods for Request
func (r *Request) Query(key string) string {
	return r.HttpRequest.URL.Query().Get(key)
}

func (r *Request) QueryInt(key string, defaultValue int) int {
	value := r.Query(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return intValue
}

func (r *Request) QueryFloat(key string, defaultValue float64) float64 {
	value := r.Query(key)
	if value == "" {
		return defaultValue
	}

	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}
	return floatValue
}

func (r *Request) QueryBool(key string, defaultValue bool) bool {
	value := r.Query(key)
	if value == "" {
		return defaultValue
	}

	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return boolValue
}

func (r *Request) QueryArray(key string) []string {
	return r.HttpRequest.URL.Query()[key]
}

func (r *Request) QueryMap() map[string][]string {
	return r.HttpRequest.URL.Query()
}

func (r *Request) Next(req Request, res Response) error {
	if r.next != nil {
		r.next(req, res)
	}
	return nil
}

func (r *Request) Auth() *AuthPayload {
	var typeValue AuthPayload
	data, exists := r.Context[string(UserInfoPayloadKey)]

	if !exists {
		return nil
	}

	json.Unmarshal(helper.ToByte(data), &typeValue)

	return &typeValue
}

func (r *Request) Client() *ClientPayload {
	protoList := strings.Split(strings.ToLower(r.HttpRequest.Proto), "/")
	hostList := strings.Split(strings.ToLower(r.HttpRequest.Host), ":")
	proto := protoList[0]
	host := hostList[0]
	port := hostList[len(hostList)-1]

	return &ClientPayload{
		Host:   host,
		Proto:  proto,
		Port:   port,
		Path:   r.HttpRequest.URL.Path,
		Method: strings.ToLower(r.HttpRequest.Method),
		Origin: r.HttpRequest.Header.Get("origin"),
	}
}

func (r *Request) TokenPayload() *TokenPayload {
	var typeValue TokenPayload
	data, exists := r.Context[string(UserInfoPayloadKey)]
	if !exists {
		return nil
	}

	var body map[string]interface{}
	if m, ok := data.(map[string]interface{}); ok {
		body = m
	}

	json.Unmarshal(helper.ToByte(body), &typeValue)

	return &typeValue
}

func (r *Request) Token() string {
	// Implementation for JWT token parsing
	return r.GetHeader("token")
}

func (r *Request) Body() interface{} {
	return r.RawBody
}

func (r *Request) Header(key string) string {
	return r.HttpRequest.Header.Get(key)
}

func (r *Request) GetHeader(key string) string {
	return r.HttpRequest.Header.Get(key)
}

func (r *Request) SetHeader(key, value string) {
	r.HttpRequest.Header.Set(key, value)
}
