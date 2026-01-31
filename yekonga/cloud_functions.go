package yekonga

import (
	"errors"
	"fmt"
	"time"

	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
)

const (
	CustomCSS    string = "__SET_CUSTOM_CSS__"
	CustomConfig string = "__SET_CUSTOM_CONFIG__"
)

type TriggerAction string

const (
	BeforeOtpTriggerAction TriggerAction = "BeforeOtp"
	AfterOtpTriggerAction  TriggerAction = "AfterOtp"

	BeforeLoginTriggerAction TriggerAction = "BeforeLogin"
	AfterLoginTriggerAction  TriggerAction = "AfterLogin"

	// Before ALL
	BeforeFindTriggerAllAction TriggerAction = "AllBeforeFind"
	AfterFindTriggerAllAction  TriggerAction = "AllAfterFind"

	BeforeCreateTriggerAllAction TriggerAction = "AllBeforeCreate"
	AfterCreateTriggerAllAction  TriggerAction = "AllAfterCreate"

	BeforeUpdateTriggerAllAction TriggerAction = "AllBeforeUpdate"
	AfterUpdateTriggerAllAction  TriggerAction = "AllAfterUpdate"

	BeforeDeleteTriggerAllAction TriggerAction = "AllBeforeDelete"
	AfterDeleteTriggerAllAction  TriggerAction = "AllAfterDelete"

	// Before Specific model
	BeforeFindTriggerAction TriggerAction = "BeforeFind"
	AfterFindTriggerAction  TriggerAction = "AfterFind"

	BeforeCreateTriggerAction TriggerAction = "BeforeCreate"
	AfterCreateTriggerAction  TriggerAction = "AfterCreate"

	BeforeUpdateTriggerAction TriggerAction = "BeforeUpdate"
	AfterUpdateTriggerAction  TriggerAction = "AfterUpdate"

	BeforeDeleteTriggerAction TriggerAction = "BeforeDelete"
	AfterDeleteTriggerAction  TriggerAction = "AfterDelete"
)

type ContextKey string

const (
	AccessTokenKey     ContextKey = "access_token"
	RefreshTokenKey    ContextKey = "refresh_token"
	UserInfoPayloadKey ContextKey = "userInfoPayload"
	ClientPayloadKey   ContextKey = "clientPayload"
	TokenPayloadKey    ContextKey = "tokenPayload"
	CurrentTenantId    ContextKey = "currentTenantId"
	RequestKey         ContextKey = "requestObject"
	YekongaKey         ContextKey = "yekongaObject"
	RequestContextKey  ContextKey = "requestContext"
	ResponseContextKey ContextKey = "responseContext"
)

type PrimaryCloudKey string

const (
	SendSMSCloudFunctionKey      PrimaryCloudKey = "SendSMS"
	SendEmailCloudFunctionKey    PrimaryCloudKey = "EmailSMS"
	SendWhatsappCloudFunctionKey PrimaryCloudKey = "WhatsappSMS"
)

// AddCloudFunction registers a new cloud function
func (y *YekongaData) Define(name string, fn CloudFunction) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if _, exists := y.functions[name]; exists {
		return fmt.Errorf("cloud function %s already exists", name)
	}

	y.functions[name] = fn
	logger.Error("Registered cloud function", name)
	return nil
}

func (y *YekongaData) Run(name string, data interface{}, ctx *RequestContext, timeout time.Duration) (interface{}, error) {
	y.mut.RLock()
	fun, exists := y.functions[name]
	y.mut.RUnlock()

	if exists {
		if ctx == nil {
			ctx = &RequestContext{}
		}

		return fun(data, ctx)
	}

	return nil, nil
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) Action(model string, action string, accessRole interface{}, route interface{}, fn ActionCloudFunction) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if y.graphqlActionFunctions[model] == nil {
		y.graphqlActionFunctions[model] = make(map[string]map[string]ActionCloudFunction)
	}

	if y.graphqlActionFunctions[model][action] == nil {
		y.graphqlActionFunctions[model][action] = make(map[string]ActionCloudFunction)
	}

	actionAccess := ""

	if v, ok := accessRole.(string); ok {
		actionAccess = v
	}

	if v, ok := route.(string); ok {
		if helper.IsEmpty(actionAccess) {
			actionAccess = v
		} else {
			actionAccess += "_" + v
		}
	}

	actionAccess = helper.ToSlug(actionAccess)

	if _, exists := y.graphqlActionFunctions[model][action][actionAccess]; exists {
		return fmt.Errorf("cloud function %s -> %v -> %v already exists", model, action, accessRole)
	}

	y.graphqlActionFunctions[model][action][actionAccess] = fn
	logger.Error("Registered cloud function: %s -> %v -> %v", model, action, accessRole)
	return nil
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) actionCallback(model string, action string, ctxRequest *RequestContext, ctxQuery *QueryContext) (interface{}, error) {
	y.mut.RLock()

	if y.graphqlActionFunctions[model] == nil {
		y.mut.RUnlock()
		return nil, errors.New("not exists")
	}

	if y.graphqlActionFunctions[model][action] == nil {
		y.mut.RUnlock()
		return nil, errors.New("not exists")
	}

	actionAccess := ctxQuery.AccessRole

	if helper.IsEmpty(actionAccess) {
		actionAccess = ctxQuery.Route
	} else {
		actionAccess += "_" + ctxQuery.Route
	}

	actionAccess = helper.ToSlug(actionAccess)

	if _, exists := y.graphqlActionFunctions[model][action][actionAccess]; exists {
		var result interface{}
		var err error

		result, err = y.graphqlActionFunctions[model][action][actionAccess](ctxRequest, ctxQuery)

		y.mut.RUnlock()

		return result, err
	}

	return nil, errors.New("not exists")
}

func (y *YekongaData) BeforeOtp(fn TriggerFunction) interface{} {
	return y.setAuthTrigger(BeforeOtpTriggerAction, fn)
}

func (y *YekongaData) AfterOtp(fn TriggerFunction) interface{} {
	return y.setAuthTrigger(AfterOtpTriggerAction, fn)
}

func (y *YekongaData) BeforeLogin(fn TriggerFunction) interface{} {
	return y.setAuthTrigger(BeforeLoginTriggerAction, fn)
}

func (y *YekongaData) AfterLogin(fn TriggerFunction) interface{} {
	return y.setAuthTrigger(AfterLoginTriggerAction, fn)
}

func (y *YekongaData) BeforeFindAll(fn TriggerAllFunction) interface{} {
	return y.setTriggerAll(BeforeFindTriggerAllAction, fn)
}

func (y *YekongaData) BeforeFind(model string, accessRole interface{}, route interface{}, fn TriggerFunction) interface{} {
	return y.setTrigger(model, BeforeFindTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) AfterFindAll(fn TriggerAllFunction) interface{} {
	return y.setTriggerAll(AfterFindTriggerAllAction, fn)
}

func (y *YekongaData) AfterFind(model string, accessRole interface{}, route interface{}, fn TriggerFunction) interface{} {
	return y.setTrigger(model, AfterFindTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) BeforeCreateAll(fn TriggerAllFunction) interface{} {
	return y.setTriggerAll(BeforeCreateTriggerAllAction, fn)
}

func (y *YekongaData) BeforeCreate(model string, accessRole interface{}, route interface{}, fn TriggerFunction) interface{} {
	return y.setTrigger(model, BeforeCreateTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) AfterCreateAll(fn TriggerAllFunction) interface{} {
	return y.setTriggerAll(AfterCreateTriggerAllAction, fn)
}

func (y *YekongaData) AfterCreate(model string, accessRole interface{}, route interface{}, fn TriggerFunction) interface{} {
	return y.setTrigger(model, AfterCreateTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) BeforeUpdateAll(fn TriggerAllFunction) interface{} {
	return y.setTriggerAll(BeforeUpdateTriggerAllAction, fn)
}

func (y *YekongaData) BeforeUpdate(model string, accessRole interface{}, route interface{}, fn TriggerFunction) interface{} {
	return y.setTrigger(model, BeforeUpdateTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) AfterUpdateAll(fn TriggerAllFunction) interface{} {
	return y.setTriggerAll(AfterUpdateTriggerAllAction, fn)
}

func (y *YekongaData) AfterUpdate(model string, accessRole interface{}, route interface{}, fn TriggerFunction) interface{} {
	return y.setTrigger(model, AfterUpdateTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) BeforeDeleteAll(fn TriggerAllFunction) interface{} {
	return y.setTriggerAll(BeforeDeleteTriggerAllAction, fn)
}

func (y *YekongaData) BeforeDelete(model string, accessRole interface{}, route interface{}, fn TriggerFunction) interface{} {
	return y.setTrigger(model, BeforeDeleteTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) AfterDeleteAll(fn TriggerAllFunction) interface{} {
	return y.setTriggerAll(AfterDeleteTriggerAllAction, fn)
}

func (y *YekongaData) AfterDelete(model string, accessRole interface{}, route interface{}, fn TriggerFunction) interface{} {
	return y.setTrigger(model, AfterDeleteTriggerAction, accessRole, route, fn)
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) setTrigger(model string, action TriggerAction, accessRole interface{}, route interface{}, fn TriggerFunction) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if y.triggerFunctions == nil {
		y.triggerFunctions = make(map[string]map[TriggerAction]map[string]TriggerFunction)
	}

	if y.triggerFunctions[model] == nil {
		y.triggerFunctions[model] = map[TriggerAction]map[string]TriggerFunction{}
	}

	if y.triggerFunctions[model][action] == nil {
		y.triggerFunctions[model][action] = map[string]TriggerFunction{}
	}

	actionAccess := ""

	if v, ok := accessRole.(string); ok {
		actionAccess = v
	}

	if v, ok := route.(string); ok {
		if helper.IsEmpty(actionAccess) {
			actionAccess = v
		} else {
			actionAccess += "_" + v
		}
	}

	actionAccess = helper.ToSlug(actionAccess)

	if _, exists := y.triggerFunctions[model][action][actionAccess]; exists {
		logger.Error("cloud function %s -> %v -> %v ->  %v already exists", model, action, accessRole, route)
		return nil
	}

	y.triggerFunctions[model][action][actionAccess] = fn
	logger.Warn("Registered cloud function: %s -> %v -> %v -> %v", model, action, accessRole, route)
	return nil
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) triggerCallback(model string, action TriggerAction, ctxRequest *RequestContext, ctxQuery *QueryContext) (interface{}, error) {
	y.mut.RLock()
	defer y.mut.RUnlock()

	if y.triggerFunctions[model] == nil {
		return nil, fmt.Errorf("%v model not exists, action %v", model, string(action))
	}

	if y.triggerFunctions[model][action] == nil {
		return nil, fmt.Errorf("%v -> %v action not exists", model, action)
	}

	actionAccess := ctxQuery.AccessRole

	if helper.IsEmpty(actionAccess) {
		actionAccess = ctxQuery.Route
	} else {
		actionAccess += "_" + ctxQuery.Route
	}

	actionAccess = helper.ToSlug(actionAccess)

	if _, exists := y.triggerFunctions[model][action][actionAccess]; exists {
		var result interface{}
		var err error

		result, err = y.triggerFunctions[model][action][actionAccess](ctxRequest, ctxQuery)

		return result, err
	}

	return nil, errors.New("not exists")
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) setAuthTrigger(action TriggerAction, fn TriggerFunction) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if y.authTriggerFunctions == nil {
		y.authTriggerFunctions = make(map[TriggerAction]TriggerFunction)
	}

	y.authTriggerFunctions[action] = fn
	logger.Warn("Registered Auth cloud function: %s", action)

	return nil
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) authTriggerCallback(action TriggerAction, ctxRequest *RequestContext, ctxQuery *QueryContext) (interface{}, error) {
	y.mut.RLock()
	defer y.mut.RUnlock()

	if _, exists := y.authTriggerFunctions[action]; exists {
		var result interface{}
		var err error

		result, err = y.authTriggerFunctions[action](ctxRequest, ctxQuery)

		return result, err
	}

	return nil, errors.New("not exists")
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) setTriggerAll(action TriggerAction, fn TriggerAllFunction) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if y.triggerAllFunctions == nil {
		y.triggerAllFunctions = map[TriggerAction]TriggerAllFunction{}
	}

	if y.triggerAllFunctions[action] == nil {
		y.triggerAllFunctions[action] = fn

		logger.Warn("Registered cloud function: %s", action)
	} else {
		logger.Error("Registered Trigger All function: %s All ready exists", action)
	}

	return nil
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) triggerAllCallback(action TriggerAction, model *DataModel, ctxRequest *RequestContext, ctxQuery *QueryContext) (interface{}, error) {
	y.mut.RLock()
	defer y.mut.RUnlock()

	if y.triggerAllFunctions[action] == nil {
		return nil, fmt.Errorf("%v -> action not exists", action)
	}

	if _, exists := y.triggerAllFunctions[action]; exists {
		var result interface{}
		var err error
		result, err = y.triggerAllFunctions[action](model, ctxRequest, ctxQuery)

		return result, err
	}

	return nil, errors.New("not exists")
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) SetCustomCSS(fn SystemHandler) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if _, exists := y.systemFunctions[CustomCSS]; exists {
		return fmt.Errorf("cloud function %s already exists", CustomCSS)
	}

	y.systemFunctions[CustomCSS] = fn
	logger.Error("Registered system cloud function SetCustomCSS")
	return nil
}

func (y *YekongaData) CustomCSS(req *Request, res *Response) (interface{}, error) {
	y.mut.RLock()
	fun, exists := y.systemFunctions[CustomCSS]
	y.mut.RUnlock()

	if exists {
		return fun(req, res)
	}

	return nil, errors.New("Custom CSS not set")
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) SetCustomConfig(fn SystemHandler) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if _, exists := y.systemFunctions[CustomConfig]; exists {
		return fmt.Errorf("cloud function %s already exists", CustomConfig)
	}

	y.systemFunctions[CustomConfig] = fn
	logger.Error("Registered system cloud function SetCustomConfig")
	return nil
}

func (y *YekongaData) CustomConfig(req *Request, res *Response) (interface{}, error) {
	y.mut.RLock()
	fun, exists := y.systemFunctions[CustomConfig]
	y.mut.RUnlock()

	if exists {
		return fun(req, res)
	}

	return nil, errors.New("Custom CONFIG not set")
}
