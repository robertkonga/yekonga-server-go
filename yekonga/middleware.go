package yekonga

import (
	"encoding/json"
	"errors"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/jwt"
)

// Define custom middleware keys
type MiddlewareType string

const (
	GlobalMiddleware  MiddlewareType = "global"
	InitMiddleware    MiddlewareType = "init"
	PreloadMiddleware MiddlewareType = "preload"
	CatchMiddleware   MiddlewareType = "catch"
)

func BillingMiddleware(req *Request, res *Response) (int, error) {
	return http.StatusOK, nil
}

// Middleware to set client detail
func ClientMiddleware(req *Request, res *Response) (int, error) {
	r := req.HttpRequest
	protoList := strings.Split(strings.ToLower(r.Proto), "/")
	hostList := strings.Split(strings.ToLower(r.Host), ":")
	proto := protoList[0]
	host := hostList[0]
	port := ""
	ipAddress := helper.GetClientIP(r)
	origin := r.Header.Get("origin")

	if len(hostList) > 1 {
		port = hostList[len(hostList)-1]
	}

	if helper.IsEmpty(origin) {
		referer := r.Header.Get("referer")
		origin = helper.ExtractDomain(referer)

		if helper.IsEmpty(origin) {
			origin = proto + "://" + host
		}
	}

	client := ClientPayload{
		Host:      host,
		Proto:     proto,
		Port:      port,
		Path:      r.URL.Path,
		Method:    strings.ToLower(r.Method),
		Origin:    origin,
		UserAgent: r.UserAgent(),
		IpAddress: ipAddress,
	}

	req.SetContext(string(ClientPayloadKey), client)

	return http.StatusOK, nil
}

func TenantCatchMiddleware(req *Request, res *Response) (int, error) {
	tenantModelName := "Tenant"
	client := *req.Client()
	host := client.Host
	tenantId := ""

	if req.App.Config.HasTenant {
		tenant := req.App.ModelQuery(tenantModelName).FindOne(datatype.DataMap{
			"domain": host,
		})

		if helper.IsNotEmpty(tenant) {
			tenantId = helper.GetValueOfString(tenant, "_id")
		} else {
			tenant = req.App.ModelQuery(tenantModelName).FindOne(datatype.DataMap{
				"subdomain": host,
			})

			if helper.IsNotEmpty(tenant) {
				tenantId = helper.GetValueOfString(tenant, "_id")
			}
		}

		if helper.IsEmpty(tenantId) && req.App.Config.TenantOnly {
			return http.StatusBadRequest, errors.New("tenant not found for the request")
		}
	}

	if req.App.Config.HasTenantCatch {
		tenant, err := req.App.FetchTenantByDomain(host, req, res)

		if err == nil {
			if helper.IsNotEmpty(tenant) {
				tenantId = helper.GetValueOfString(tenant, "tenantId")
			}
		}
	}

	if helper.IsNotEmpty(tenantId) {
		client.TenantId = tenantId

		req.SetTenantId(tenantId)
		req.SetContext(string(ClientPayloadKey), client)
	}

	return http.StatusOK, nil
}

// Middleware to add token as a string
func TokenMiddleware(req *Request, res *Response) (int, error) {
	app := req.App
	config := req.App.Config
	clientPayload := req.Client()
	domain := clientPayload.OriginDomain()

	var isValid bool
	var accessToken string
	var refreshToken string
	var tokenPayloadMap datatype.DataMap
	var tokenPayload TokenPayload

	accessCookie, err := req.HttpRequest.Cookie(string(AccessTokenKey))
	if err == nil {
		accessToken = accessCookie.Value
	}

	refreshCookie, err := req.HttpRequest.Cookie(string(RefreshTokenKey))
	if err == nil {
		refreshToken = refreshCookie.Value
	}

	if helper.IsEmpty(accessToken) {
		var values []string = req.HttpRequest.Header["Authorization"]

		if len(values) > 0 {
			accessToken = strings.Trim(strings.Split(values[0], "Bearer")[1], " ")
		}
	}

	if helper.IsNotEmpty(accessToken) {
		isValid, tokenPayloadMap = jwt.DecodeJWT(accessToken, config.Authentication.SecretToken)

		if !isValid || helper.IsEmpty(tokenPayloadMap) {
			requestContext := &RequestContext{
				App:      req.App,
				Auth:     req.Auth(),
				Client:   req.Client(),
				Request:  req,
				Response: res,
			}
			req.App.clearAuthCookies(requestContext, domain)

			return http.StatusUnauthorized, errors.New("Access token invalid")
		}

		json.Unmarshal([]byte(helper.ToJson(tokenPayloadMap)), &tokenPayload)

		if tokenPayload.ExpiresAt.Before(helper.GetTimestamp(nil)) {
			return http.StatusUnauthorized, errors.New("Token expired")
		}

		if domain != tokenPayload.Domain {
			requestContext := &RequestContext{
				App:      req.App,
				Auth:     req.Auth(),
				Client:   req.Client(),
				Request:  req,
				Response: res,
			}
			req.App.clearAuthCookies(requestContext, tokenPayload.Domain)

			return http.StatusUnauthorized, errors.New("Domain mismatch expired")
		}

		if req.App.Config.TenantOnly {
			if helper.IsNotEmpty(tokenPayload) && helper.IsEmpty(tokenPayload.TenantId) {
				return http.StatusBadRequest, errors.New("tenant not found for the request")
			}
		}

		req.SetContext(string(AccessTokenKey), accessToken)
		req.SetContext(string(TokenPayloadKey), tokenPayload)

		if helper.IsNotEmpty(tokenPayload.TenantId) {
			req.SetTenantId(tokenPayload.TenantId)
		}
	}

	if helper.IsNotEmpty(refreshToken) {
		req.SetContext(string(RefreshTokenKey), refreshToken)
	}

	if config.AuthorizedOnly {
		paths := []string{
			app.AppendBaseUrl("/me"),
			app.AppendBaseUrl(config.RestApi),
			app.AppendBaseUrl(config.Graphql.ApiRoute),
		}
		currentPath := req.HttpRequest.URL.Path

		if helper.Contains(paths, currentPath) && helper.IsEmpty(accessToken) {
			return http.StatusUnauthorized, errors.New("Must be authorized/login")
		}
	}

	return http.StatusOK, nil
}

// Middleware to add user info as a map
func UserInfoMiddleware(req *Request, res *Response) (int, error) {
	var userInfo *datatype.DataMap
	const userModelName = "User"
	tokenPayload := req.GetContext(string(TokenPayloadKey))

	if helper.IsNotEmpty(tokenPayload) {
		if payload, ok := tokenPayload.(TokenPayload); ok {
			id := payload.UserId

			if helper.IsNotEmpty(id) {
				if req.App.Config.IsAuthorizationServer {
					userInfo = req.App.ModelQuery(userModelName).FindOne(datatype.DataMap{
						"_id": helper.ObjectID(id),
					})
				} else {
					userInfo = &datatype.DataMap{
						"_id":         id,
						"id":          id,
						"userId":      payload.UserId,
						"profileId":   payload.ProfileId,
						"tenantId":    payload.TenantId,
						"adminId":     payload.AdminId,
						"roles":       payload.Roles,
						"permissions": payload.Permissions,
					}
				}

			}
		}
	}

	if userInfo != nil {
		req.SetContext(string(UserInfoPayloadKey), *userInfo)
	}

	return http.StatusOK, nil
}

func ApplicationKeyMiddleware(req *Request, res *Response) (int, error) {
	var appKey string = req.GetHeader("application-key")
	var path string = req.HttpRequest.URL.Path
	var extension string = filepath.Ext(path)
	var allowed = []string{
		".css",
		".js",
		".ico",
		".pdf",
		".flv",
		".jpg",
		".jpeg",
		".png",
		".gif",
		".webp",
		".woff2",
		".woff",
		".ttf",
		".eot",
	}

	if !helper.Contains(allowed, extension) {
		if helper.IsEmpty(appKey) {
			appKey = req.Query("application-key")

			if helper.IsEmpty(appKey) {
				appKey = req.Query("app-key")
			}
		}

		if req.App.Config.EnableAppKey {
			if helper.IsNotEmpty(appKey) {
				if appKey != req.App.Config.AppKey {
					return http.StatusUnauthorized, errors.New("application key invalid")
				}
			} else {
				return http.StatusUnauthorized, errors.New("application key not provided")
			}
		}
	}

	return http.StatusOK, nil
}
