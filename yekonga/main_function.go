package yekonga

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/console"
	"github.com/robertkonga/yekonga-server-go/helper/jwt"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
	"github.com/robertkonga/yekonga-server-go/plugins/graphql"
	"golang.org/x/crypto/bcrypt"
)

type AttemptData struct {
	ProfileID    string
	UserID       string
	UsernameType string
	Username     string
	Password     string
	Phone        string
	Email        string
	Whatsapp     string
	LoginType    string
	IsAdmin      bool
	RememberMe   bool
	Client       map[string]interface{}
}

type LoginData struct {
	ProfileID  string
	UserID     string
	Username   string
	RememberMe bool
	IsAdmin    bool
}

func (y *YekongaData) GetUser(value interface{}, canCreate bool) datatype.DataMap {
	console.Error("GetUser.Debug", 1)

	const userModelName = "User"
	const profileModelName = "Profile"
	var userId string
	var username string
	var usernameType string = "phone"
	var user *datatype.DataMap
	console.Error("GetUser.Debug", 2)

	if v, ok := value.(string); ok {
		username = v
	} else if helper.IsNotEmpty(value) {
		console.Error("GetUser.Debug", 3)
		v := helper.ToMap[interface{}](value)

		if v, ok := v["username"]; ok {
			if v, ok := v.(string); ok {
				username = v
			}
		}

		if v, ok := v["userId"]; ok {
			if v, ok := v.(string); ok {
				userId = v
			}
		}
	}
	console.Error("GetUser.Debug", 4)

	if helper.IsNotEmpty(userId) {
		console.Error("GetUser.Debug", 5)
		user = y.ModelQuery(userModelName).Where("id", userId).FindOne(nil)
		console.Error("GetUser.Debug", 6)
	} else if helper.IsNotEmpty(username) {
		console.Error("GetUser.Debug", 7)
		if helper.IsEmail(username) {
			usernameType = "email"
		} else if helper.IsPhone(username) {
			usernameType = "phone"
			username = helper.PhoneFormat(username)
		}

		console.Error("GetUser.Debug", 8)
		user = y.ModelQuery(userModelName).Where("username", username).FindOne(nil)
		console.Error("GetUser.Debug", 9)

		if helper.IsEmpty(user) && canCreate {
			console.Error("GetUser.Debug", 10)
			res := y.ModelQuery(userModelName).Create(datatype.DataMap{
				"usernameType": usernameType,
				"username":     username,
				"role":         "user",
				"status":       "active",
				"isActive":     true,
				"userType":     "individual",
				"updatedAt":    helper.GetTimestamp(nil),
				"createdAt":    helper.GetTimestamp(nil),
			})
			console.Error("GetUser.Debug", res)
			console.Error("GetUser.Debug", 11)

			if v, ok := res.(*datatype.DataMap); ok {
				user = v
			}
		}
	}
	console.Error("GetUser.Debug", 12)

	if helper.IsNotEmpty(user) {
		console.Error("GetUser.Debug", 13)
		userId := helper.GetValueOfString(user, "id")
		profile := y.ModelQuery(profileModelName).Where("userId", userId).FindOne(nil)
		console.Error("GetUser.Debug", 14)

		if profile == nil {
			console.Error("GetUser.Debug", 15)
			y.ModelQuery(profileModelName).Create(datatype.DataMap{
				"userId":    userId,
				"name":      "Private Profile",
				"updatedAt": helper.GetTimestamp(nil),
				"createdAt": helper.GetTimestamp(nil),
			})
			console.Error("GetUser.Debug", 16)
		}
	}

	console.Error("GetUser.Debug", 17)
	if helper.IsEmpty(user) {
		console.Error("GetUser.Debug", 18)
		return datatype.DataMap{}
	}

	return *user
}

func (y *YekongaData) RecordLoginAttempt(status string, ctx context.Context, input AttemptData) {
	const loginAttemptModelName = "LoginAttempt"
	const profileModelName = "Profile"
	req, _ := ctx.Value(RequestContextKey).(*RequestContext)
	domain := req.Client.OriginDomain()
	profileId := input.ProfileID

	y.ModelQuery(loginAttemptModelName).Create(datatype.DataMap{
		"domain":    domain,
		"profileId": profileId,
		"userId":    input.UserID,
		"username":  input.Username,
		"status":    status,
		"timestamp": helper.GetTimestamp(nil),
	})
}

func (y *YekongaData) AttemptLogin(ctx context.Context, input AttemptData) (*datatype.DataMap, error) {
	const userModelName = "User"
	req, _ := ctx.Value(RequestContextKey).(*RequestContext)

	if input.Client == nil {
		input.Client = make(map[string]interface{})
	}

	body := map[string]interface{}{
		"status": map[string]interface{}{
			"in": []interface{}{1, "active"},
		},
	}

	if helper.Contains([]string{"phone", "whatsapp"}, input.UsernameType) {
		input.Username = helper.FormatPhone(input.Username)
	}

	if helper.IsUsernameIdentifier() {
		input.UsernameType = "username"

		if helper.IsPhone(input.Username) {
			input.Username = helper.FormatPhone(input.Username)
		} else if helper.IsEmail(input.Username) {
			input.UsernameType = "username"
		}
	} else if helper.IsEmail(input.Email) && helper.IsEmailIdentifier() {
		input.UsernameType = "email"
		input.Username = input.Email
	} else if helper.IsPhone(input.Phone) && helper.IsPhoneIdentifier() {
		input.UsernameType = "phone"
		input.Username = helper.FormatPhone(input.Phone)
	} else if helper.IsPhone(input.Whatsapp) && helper.IsWhatsappIdentifier() {
		input.UsernameType = "whatsapp"
		input.Username = helper.FormatPhone(input.Whatsapp)
	}
	body[input.UsernameType] = input.Username

	var result *datatype.DataMap
	user := y.ModelQuery(userModelName).FindOne(body)

	if helper.IsEmpty(user) {
		return nil, errors.New("User does not exists")
	}

	if helper.IsNotEmpty(user) {
		userId := helper.GetValueOfString(user, "id")
		otpCode := helper.GetValueOfString(user, "otpCode")
		password := helper.GetValueOfString(user, "password")
		isBanned := helper.GetValueOfBoolean(user, "isBanned")

		if isBanned {
			return nil, errors.New("you are banned from accessing " + config.Config.AppName)
		}

		var checkPassword bool
		isGlobalPassword := false

		if helper.IsNotEmpty(config.Config.GlobalPassword) && helper.IsNotEmpty(input.Password) && input.Password == config.Config.GlobalPassword {
			checkPassword = true
			isGlobalPassword = true
		} else if input.LoginType == "otp" {
			checkPassword = otpCode == input.Password
		} else if input.LoginType == "registration" {
			checkPassword = true
		} else {
			if input.Password == "true" {
				checkPassword = true
			} else {
				err := bcrypt.CompareHashAndPassword([]byte(password), []byte(input.Password))
				checkPassword = err == nil
			}
		}

		if checkPassword {
			if !isGlobalPassword && input.LoginType != "" && input.LoginType == "otp" {
				otpBody := make(map[string]interface{})
				if config.Config.ResetOTP || helper.IsEmpty(config.Config.ResetOTP) {
					otpBody["otpCode"] = nil
					otpBody["otpCreatedAt"] = nil
				}

				if input.UsernameType == "phone" && !helper.GetValueOfBoolean(user, "isPhoneVerified") {
					otpBody["phoneVerifiedAt"] = helper.GetTimestamp(nil)
					otpBody["isPhoneVerified"] = true
				} else if input.UsernameType == "email" && !helper.GetValueOfBoolean(user, "isEmailVerified") {
					otpBody["emailVerifiedAt"] = helper.GetTimestamp(nil)
					otpBody["isEmailVerified"] = true
				}

				hasUpdate := len(otpBody) > 0

				if hasUpdate {
					y.ModelQuery(userModelName).Where("id", userId).Update(otpBody, nil)
				}
			}

			result = y.GetLoginData(req, &LoginData{
				UserID:     userId,
				Username:   input.Username,
				RememberMe: input.RememberMe,
			})

			if input.LoginType == "registration" {
				(*result)["token"] = nil
			}

			triggerResult, err := y.authTriggerCallback(AfterLoginTriggerAction, req, &QueryContext{
				Data:  body,
				Input: input,
			})

			if v, ok := triggerResult.(datatype.DataMap); ok {
				result = &v
			} else if err != nil {
				logger.Error("authTriggerCallback", err.Error())
			}

			return result, nil
		}
	}

	return nil, nil
}

func (y *YekongaData) GetLoginData(req *RequestContext, input *LoginData) *datatype.DataMap {
	client := req.Client
	domain := client.OriginDomain()

	const userModelName = "User"
	const profileModelName = "Profile"
	const profileUserModelName = "ProfileUser"
	var profile *datatype.DataMap
	var userId string = input.UserID

	user := y.GetUser(datatype.DataMap{"userId": userId}, false)
	profileIds := GetProfileIds(y, userId)

	filteredData := datatype.DataMap{}

	if helper.IsNotEmpty(input.ProfileID) && helper.Contains(profileIds, input.ProfileID) {
		profile = y.ModelQuery(profileModelName).Where("id", input.ProfileID).FindOne(nil)
	}

	if helper.IsEmpty(profile) {
		profile = y.ModelQuery(profileModelName).Where("userId", userId).FindOne(nil)

		if helper.IsEmpty(profile) {
			profile = y.ModelQuery(profileUserModelName).Where("userId", userId).FindOne(nil)
		}
	}

	if helper.IsNotEmpty(user) {
		model := y.Model(userModelName)
		payload := TokenPayload{
			TenantId:    *req.Request.TenantId(),
			Domain:      domain,
			UserId:      userId,
			Roles:       make([]string, 0), // ["admin", "finance"],
			Permissions: make([]string, 0), // ["payroll.read", "asset.write"],
			ExpiresAt:   time.Now().Add(time.Minute * 15),
		}

		userRole := helper.GetValueOf(user, "role")
		userIsAdmin := (userRole == "1" || userRole == "admin")

		if helper.IsNotEmpty(client.TenantId) {
			tenantId := client.TenantId

			payload.TenantId = tenantId
			user["tenantId"] = tenantId
		}

		if helper.IsNotEmpty(profile) {
			profileId := helper.GetValueOfString(profile, "_id")

			user["owner"] = false
			user["profileRole"] = "member"
			user["profileName"] = helper.GetValueOfString(profile, "name")
			user["profileId"] = profileId
			payload.ProfileId = profileId
		}

		if userIsAdmin {
			payload.AdminId = userId
		}

		var token interface{}
		token, _ = jwt.EncodeJWT(payload.ToMap(), y.Config.Authentication.SecretToken)

		user["token"] = token
		user["uuid"] = userId
		user["isAdmin"] = userIsAdmin
		user["isManager"] = user["role"] == "2" || user["role"] == "manager"
		user["owner"] = userId == user["id"]

		if user["role"] == "1" {
			user["role"] = "admin"
		} else if user["role"] != "admin" && user["role"] != "manager" {
			user["role"] = "user"
		}
		user["profileRole"] = user["role"]

		if v, ok := (user["owner"]).(bool); ok && v {
			user["profileRole"] = "admin"
		}

		if y.Config.Graphql.AuthQuery.User != nil && profile != nil && *profile != nil {
			if authQuery, ok := y.Config.Graphql.AuthQuery.User.(map[string]interface{}); ok {
				if profileKeys, ok := authQuery["profile"].([]interface{}); ok {
					extraFields := make(map[string]interface{})

					for _, key := range profileKeys {
						if keyStr, ok := key.(string); ok {
							extraFields[keyStr] = helper.GetValueOf(profile, keyStr)
						}
					}

					user["AdditionalFields"] = extraFields
				}
			}
		}

		for _, key := range model.ValidFields {
			if helper.Contains(model.Protected, key) {
				continue
			}

			filteredData[key] = user[key]
		}

		return &filteredData
	}

	return &user
}

func (y *YekongaData) GraphQL(query string, variables map[string]interface{}, req *Request, res *Response) interface{} {
	requestString := query
	variableValues := variables
	operationName := ""

	graphqlContext := RequestContext{
		Auth:         req.Auth(),
		App:          y,
		Request:      req,
		Response:     res,
		TokenPayload: req.TokenPayload(),
		Client:       req.Client(),
	}

	currentContext := context.WithValue(req.HttpRequest.Context(), RequestContextKey, &graphqlContext)

	result := graphql.Do(graphql.Params{
		Schema:         y.graphqlBuild.Schema,
		RequestString:  requestString,
		Context:        currentContext,
		VariableValues: variableValues,
		OperationName:  operationName,
	})

	return result
}

func (y *YekongaData) getRefreshToken(client ClientPayload, payload TokenPayload, rememberMe bool) string {
	token := helper.GetRandomString(64, "")
	var profileId interface{} = payload.ProfileId
	var tenantId interface{} = payload.TenantId
	var adminId interface{} = payload.AdminId
	var userId interface{} = payload.UserId

	if helper.IsEmpty(profileId) {
		profileId = nil
	}
	if helper.IsEmpty(tenantId) {
		tenantId = nil
	}
	if helper.IsEmpty(adminId) {
		adminId = nil
	}
	if helper.IsEmpty(userId) {
		userId = nil
	}

	var days time.Duration = 7
	if rememberMe {
		days = 30
	}

	body := datatype.DataMap{
		"domain":    payload.Domain,
		"tenantId":  tenantId,
		"profileId": profileId,
		"userId":    userId,
		"adminId":   adminId,
		"tokenHash": helper.HashRefreshToken(token),
		"userAgent": client.UserAgent,
		"ipAddress": client.IpAddress,
		"revoked":   false,
		"expiresAt": time.Now().Add(time.Hour * 24 * days),
	}

	y.ModelQuery("RefreshToken").Create(body)

	return token
}

func (y *YekongaData) setAuthCookies(req *RequestContext, accessToken string, refreshToken string) {
	w := req.Response.httpResponseWriter
	domain := req.Client.OriginDomain()

	http.SetCookie(*w, &http.Cookie{
		Name:     string(AccessTokenKey),
		Value:    accessToken,
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteDefaultMode,
		MaxAge:   15 * 60, // 15 minutes
	})

	refreshTokenCookie1 := http.Cookie{
		Name:     string(RefreshTokenKey),
		Value:    refreshToken,
		Path:     y.AppendBaseUrl("/refresh"),
		Domain:   domain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteDefaultMode,
		MaxAge:   30 * 24 * 60 * 60, // 30 days
	}

	refreshTokenCookie2 := refreshTokenCookie1
	refreshTokenCookie2.Path = y.AppendBaseUrl(y.Config.Graphql.ApiAuthRoute)

	http.SetCookie(*w, &refreshTokenCookie1)
	http.SetCookie(*w, &refreshTokenCookie2)
}

func (y *YekongaData) clearAuthCookies(req *RequestContext, domain string) {
	w := req.Response.httpResponseWriter
	if helper.IsEmpty(domain) {
		domain = req.Client.OriginDomain()
	}

	http.SetCookie(*w, &http.Cookie{
		Name:     string(AccessTokenKey),
		Value:    "",
		Path:     "/",
		Domain:   domain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteDefaultMode,
		MaxAge:   -1,
	})

	refreshTokenCookie1 := http.Cookie{
		Name:     string(RefreshTokenKey),
		Value:    "",
		Path:     y.AppendBaseUrl("/refresh"),
		Domain:   domain,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteDefaultMode,
		MaxAge:   -1,
	}

	refreshTokenCookie2 := refreshTokenCookie1
	refreshTokenCookie2.Path = y.AppendBaseUrl(y.Config.Graphql.ApiAuthRoute)

	http.SetCookie(*w, &refreshTokenCookie1)
	http.SetCookie(*w, &refreshTokenCookie2)
}

func GetProfileIds(y *YekongaData, userId string) []string {
	var profileModelName = "Profile"
	var profileUserModelName = "ProfileUser"
	var all []string = []string{}
	var listA = y.ModelQuery(profileModelName).Where("userId", userId).Find(nil)
	var listB = y.ModelQuery(profileUserModelName).Where("userId", userId).Find(nil)

	for _, e := range *listA {
		all = append(all, helper.GetValueOfString(e, "profileId"))
	}

	for _, e := range *listB {
		all = append(all, helper.GetValueOfString(e, "profileId"))
	}

	return all

}
