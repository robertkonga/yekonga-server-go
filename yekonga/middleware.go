package yekonga

import (
	"encoding/json"
	"errors"
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
)

// Middleware to add token as a string
func TokenMiddleware(req *Request, res *Response) error {
	var accessToken string
	var refreshToken string
	var tokenPayload datatype.DataMap

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
		_, tokenPayload = jwt.DecodeJWT(accessToken, req.App.Config.Authentication.SecretToken)

		var payload TokenPayload
		json.Unmarshal([]byte(helper.ToJson(tokenPayload)), &payload)

		if payload.ExpiresAt.Before(helper.GetTimestamp(nil)) {
			return errors.New("Token expired")
		}

		req.SetContext(string(AccessTokenKey), accessToken)
		req.SetContext(string(TokenPayloadKey), payload)
	}

	if helper.IsNotEmpty(refreshToken) {
		req.SetContext(string(RefreshTokenKey), refreshToken)
	}

	return nil
}

// Middleware to add user info as a map
func UserInfoMiddleware(req *Request, res *Response) error {
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

	return nil
}

func ApplicationKeyMiddleware(req *Request, res *Response) error {
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
					return errors.New("application key invalid")
				}
			} else {
				return errors.New("application key not provided")
			}
		}
	}

	return nil
}
