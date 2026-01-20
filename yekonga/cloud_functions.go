package yekonga

import (
	"errors"
	"fmt"
	"time"

	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
)

type TriggerAction string

const (
	BeforeOtpTriggerAction TriggerAction = "BeforeOtp"
	AfterOtpTriggerAction  TriggerAction = "AfterOtp"

	BeforeLoginTriggerAction TriggerAction = "BeforeLogin"
	AfterLoginTriggerAction  TriggerAction = "AfterLogin"

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
func (y *YekongaData) SetSendSms(fn CloudFunction) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if _, exists := y.primaryFunctions[SendSMSCloudFunctionKey]; exists {
		return fmt.Errorf("cloud function %s already exists", SendSMSCloudFunctionKey)
	}

	y.primaryFunctions[SendSMSCloudFunctionKey] = fn
	logger.Error("Registered cloud function", SendSMSCloudFunctionKey)
	return nil
}

func (y *YekongaData) SendSms(data interface{}, ctx *RequestContext, timeout time.Duration) (interface{}, error) {
	y.mut.RLock()
	fun, exists := y.primaryFunctions[SendSMSCloudFunctionKey]
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
func (y *YekongaData) SetSendEmail(fn CloudFunction) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if _, exists := y.primaryFunctions[SendEmailCloudFunctionKey]; exists {
		return fmt.Errorf("cloud function %s already exists", SendEmailCloudFunctionKey)
	}

	y.primaryFunctions[SendEmailCloudFunctionKey] = fn
	logger.Error("Registered cloud function", SendEmailCloudFunctionKey)
	return nil
}

func (y *YekongaData) SendEmail(data interface{}, ctx *RequestContext, timeout time.Duration) (interface{}, error) {
	y.mut.RLock()
	fun, exists := y.primaryFunctions[SendSMSCloudFunctionKey]
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
func (y *YekongaData) SetSendWhatsapp(fn CloudFunction) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if _, exists := y.primaryFunctions[SendWhatsappCloudFunctionKey]; exists {
		return fmt.Errorf("cloud function %s already exists", SendWhatsappCloudFunctionKey)
	}

	y.primaryFunctions[SendWhatsappCloudFunctionKey] = fn
	logger.Error("Registered cloud function", SendWhatsappCloudFunctionKey)
	return nil
}

func (y *YekongaData) SendWhatsapp(data interface{}, ctx *RequestContext, timeout time.Duration) (interface{}, error) {
	y.mut.RLock()
	fun, exists := y.primaryFunctions[SendWhatsappCloudFunctionKey]
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

func (y *YekongaData) BeforeOtp(fn TriggerCloudFunction) interface{} {
	return y.setAuthTrigger(BeforeOtpTriggerAction, fn)
}

func (y *YekongaData) AfterOtp(fn TriggerCloudFunction) interface{} {
	return y.setAuthTrigger(AfterOtpTriggerAction, fn)
}

func (y *YekongaData) BeforeLogin(fn TriggerCloudFunction) interface{} {
	return y.setAuthTrigger(BeforeLoginTriggerAction, fn)
}

func (y *YekongaData) AfterLogin(fn TriggerCloudFunction) interface{} {
	return y.setAuthTrigger(AfterLoginTriggerAction, fn)
}

func (y *YekongaData) BeforeFind(model string, accessRole interface{}, route interface{}, fn TriggerCloudFunction) interface{} {
	return y.setTrigger(model, BeforeFindTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) AfterFind(model string, accessRole interface{}, route interface{}, fn TriggerCloudFunction) interface{} {
	return y.setTrigger(model, AfterFindTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) BeforeCreate(model string, accessRole interface{}, route interface{}, fn TriggerCloudFunction) interface{} {
	return y.setTrigger(model, BeforeCreateTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) AfterCreate(model string, accessRole interface{}, route interface{}, fn TriggerCloudFunction) interface{} {
	return y.setTrigger(model, AfterCreateTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) BeforeUpdate(model string, accessRole interface{}, route interface{}, fn TriggerCloudFunction) interface{} {
	return y.setTrigger(model, BeforeUpdateTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) AfterUpdate(model string, accessRole interface{}, route interface{}, fn TriggerCloudFunction) interface{} {
	return y.setTrigger(model, AfterUpdateTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) BeforeDelete(model string, accessRole interface{}, route interface{}, fn TriggerCloudFunction) interface{} {
	return y.setTrigger(model, BeforeDeleteTriggerAction, accessRole, route, fn)
}

func (y *YekongaData) AfterDelete(model string, accessRole interface{}, route interface{}, fn TriggerCloudFunction) interface{} {
	return y.setTrigger(model, AfterDeleteTriggerAction, accessRole, route, fn)
}

// AddCloudFunction registers a new cloud function
func (y *YekongaData) setTrigger(model string, action TriggerAction, accessRole interface{}, route interface{}, fn TriggerCloudFunction) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if y.triggerFunctions == nil {
		y.triggerFunctions = make(map[string]map[TriggerAction]map[string]TriggerCloudFunction)
	}

	if y.triggerFunctions[model] == nil {
		y.triggerFunctions[model] = map[TriggerAction]map[string]TriggerCloudFunction{}
	}

	if y.triggerFunctions[model][action] == nil {
		y.triggerFunctions[model][action] = map[string]TriggerCloudFunction{}
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
func (y *YekongaData) setAuthTrigger(action TriggerAction, fn TriggerCloudFunction) error {
	y.mut.Lock()
	defer y.mut.Unlock()

	if y.authTriggerFunctions == nil {
		y.authTriggerFunctions = make(map[TriggerAction]TriggerCloudFunction)
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
