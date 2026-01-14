package yekonga

import (
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
	var token string
	var tokenPayload datatype.DataMap
	var values []string = req.HttpRequest.Header["Authorization"]

	if len(values) > 0 {
		token = values[0]
	}

	if strings.TrimSpace(token) != "" {
		_, tokenPayload = jwt.DecodeJWT(token)

		req.SetContext(string(TokenKey), token)
		req.SetContextObject(string(TokenPayloadKey), tokenPayload)
	} else {
		req.SetContext(string(TokenKey), "test-token")
		req.SetContextObject(string(TokenPayloadKey), datatype.DataMap{
			"userId": "275c553cf22fdd9e25e0ec4e",
		})
	}

	return nil
}

// Middleware to add user info as a map
func UserInfoMiddleware(req *Request, res *Response) error {
	var userInfo *datatype.DataMap
	tokenPayload := req.GetContextObject(string(TokenPayloadKey))

	if tokenPayload != nil {
		if v, ok := tokenPayload["userId"]; ok {
			if vi, oki := v.(string); oki {
				userInfo = req.App.ModelQuery("User").FindOne(datatype.DataMap{
					"_id": helper.ObjectID(vi),
				})

				if *userInfo != nil {
					(*userInfo)["id"] = (*userInfo)["_id"]
				}
			}
		}
	} else {
		userInfo = &datatype.DataMap{
			"id":        "1",
			"profileId": "1",
			"firstName": "Robert",
			"lastName":  "Konga",
			"phone":     "255755257511",
			"email":     "john.doe@example.com",
			"role":      "user",
		}
	}

	req.SetContext(string(UserInfoPayloadKey), *userInfo)
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
