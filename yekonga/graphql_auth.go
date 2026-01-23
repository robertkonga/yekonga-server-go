package yekonga

import (
	"errors"
	"net/http"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/console"
	"github.com/robertkonga/yekonga-server-go/plugins/graphql"
)

func (g *GraphqlAutoBuild) GetAuthQuery() *graphql.Object {
	var foreignKey string
	var targetKey string
	var name string = "User"

	var fields = make(graphql.Fields)
	fields["profile"] = &graphql.Field{
		Type: UserProfileType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var input map[string]interface{} = g.getInputData(p.Args)
			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			user := model.FindOne(input)

			return user, nil
		},
	}

	var queryType = graphql.NewObject(
		graphql.ObjectConfig{
			Name:   "Query",
			Fields: fields,
		})

	return queryType
}

func (g *GraphqlAutoBuild) GetAuthMutation() *graphql.Object {
	var fields = make(graphql.Fields)

	fields["otp"] = _otp(g)
	fields["login"] = _login(g)
	fields["refreshToken"] = _refreshToken(g)
	fields["socialLogin"] = _socialLogin(g)
	fields["contactOTP"] = _contactOTP(g)
	fields["contactVerify"] = _contactVerify(g)
	fields["registration"] = _registration(g)
	fields["resetPassword"] = _resetPassword(g)
	fields["confirmToken"] = _confirmToken(g)
	fields["changePassword"] = _changePassword(g)
	fields["profile"] = _profile(g)
	fields["switchAccount"] = _switchAccount(g)

	var mutationType = graphql.NewObject(
		graphql.ObjectConfig{
			Name:   "Mutation",
			Fields: fields,
		})

	return mutationType
}

// otp ( input: OtpInput!, type: UsernameIdentifier ): ActionResponse,
func _otp(g *GraphqlAutoBuild) *graphql.Field {
	// var foreignKey string
	// var targetKey string

	return &graphql.Field{
		Type: ActionResponseType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: OtpInput,
			},
			"type": &graphql.ArgumentConfig{
				Type: UsernameIdentifierEnum,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			req, _ := p.Context.Value(RequestContextKey).(*RequestContext)
			var result = ActionResponse{
				Message: "Fail",
			}
			var user datatype.DataMap
			var data map[string]interface{} = g.getInputData(p.Args)
			var username = helper.GetValueOfString(data, "username")
			var usernameType = helper.GetValueOfString(data, "usernameType")
			if helper.IsPhone(username) {
				username = helper.PhoneFormat(username)
			}
			if helper.IsNotEmpty(username) {
				user = g.yekonga.GetUser(username, true)
			}
			userId := helper.GetValueOfString(user, "_id")

			g.yekonga.RecordLoginAttempt("otp", p.Context, AttemptData{
				UserID:   userId,
				Username: username,
			})

			triggerResult, _ := g.yekonga.authTriggerCallback(BeforeOtpTriggerAction, req, &QueryContext{
				Data:  data,
				Input: data,
			})

			if v, ok := triggerResult.(bool); ok && !v {
				return nil, errors.New("Rejected by BeforeOtpTriggerAction")
			}

			if user != nil {
				result.Status = true
				result.Message = "Success"

				otpCode := helper.GetValueOfString(user, "otpCode")
				phone := helper.GetValueOfString(user, "phone")
				email := helper.GetValueOfString(user, "email")
				whatsapp := helper.GetValueOfString(user, "whatsapp")

				if helper.IsPhone(username) {
					if usernameType == "whatsapp" {
						whatsapp = username
					} else {
						phone = username
					}
				} else if helper.IsEmail(username) {
					email = username
				}

				if helper.IsEmpty(user["otpCode"]) {
					otpCode = helper.GetRandomInt(4)
					otpCreatedAt := helper.GetTimestamp(nil)

					g.yekonga.ModelQuery("User").Where("id", userId).Update(datatype.DataMap{
						"otpCode":      otpCode,
						"otpCreatedAt": otpCreatedAt,
					}, nil)
				}

				message := otpCode + " is your verification code. For security, do not share this code."

				g.yekonga.Notify(&NotifiedUser{
					UserID:   userId,
					Email:    email,
					Phone:    phone,
					Whatsapp: whatsapp,
				}, NotificationParams{
					Title:    "OTP",
					Text:     message,
					HTML:     message,
					Whatsapp: message,
				})

			}

			g.yekonga.authTriggerCallback(AfterOtpTriggerAction, req, &QueryContext{
				Data:  result,
				Input: data,
			})

			return result.ToMap(), nil
		},
	}
}

func _login(g *GraphqlAutoBuild) *graphql.Field {
	// var foreignKey string
	// var targetKey string
	// var name string = "User"
	var targetType *graphql.Object

	if g.yekonga.Config.SecureAuthentication {
		targetType = CredentialTokenType
	} else {
		targetType = UserProfileType
	}

	return &graphql.Field{
		Type: targetType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: LoginInput,
			},
			"type": &graphql.ArgumentConfig{
				Type: UsernameIdentifierEnum,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			req, _ := p.Context.Value(RequestContextKey).(*RequestContext)
			var input map[string]interface{} = g.getInputData(p.Args)
			// var type = g.getParamValue(p.Args, "type")
			var user datatype.DataMap
			var username = helper.GetValueOfString(input, "username")
			var usernameType = helper.GetValueOfString(input, "usernameType")
			var password = helper.GetValueOfString(input, "password")
			var loginType = helper.GetValueOfString(input, "type")
			var rememberMe = helper.GetValueOfBoolean(input, "rememberMe")

			cookieEnabled := ""
			cookie, err := req.Request.HttpRequest.Cookie(COOKIE_ENABLED_KEY)
			if err == nil {
				cookieEnabled = cookie.Value
			}

			if helper.IsPhone(username) {
				username = helper.PhoneFormat(username)
			}
			if helper.IsNotEmpty(username) {
				user = g.yekonga.GetUser(username, false)
			}

			triggerResult, _ := g.yekonga.authTriggerCallback(BeforeLoginTriggerAction, req, &QueryContext{
				Data:  user,
				Input: input,
			})

			if v, ok := triggerResult.(bool); ok && !v {
				return nil, errors.New("Rejected by before BeforeLoginTriggerAction")
			} else if v, ok := triggerResult.(datatype.DataMap); ok {
				user = v
			}

			if helper.IsNotEmpty(user) {
				attemptData := AttemptData{
					Username:     username,
					UsernameType: usernameType,
					Password:     password,
					LoginType:    loginType,
					RememberMe:   rememberMe,
				}

				u, e := g.yekonga.AttemptLogin(p.Context, attemptData)

				if helper.IsNotEmpty(u) {
					user = *u
					userId := helper.GetValueOfString(user, "id")

					attemptData.UserID = userId
					attemptData.ProfileID = helper.GetValueOfString(user, "profileId")

					g.yekonga.RecordLoginAttempt("success", p.Context, attemptData)
					g.yekonga.authTriggerCallback(AfterLoginTriggerAction, req, &QueryContext{
						Data:  user,
						Input: input,
					})

					if req.Request != nil {
						accessToken := helper.GetValueOfString(user, "token")
						profileId := helper.GetValueOfString(user, "profileId")
						tenantId := helper.GetValueOfString(user, "tenantId")
						adminId := helper.GetValueOfString(user, "adminId")

						refreshToken := g.yekonga.getRefreshToken(*req.Client, TokenPayload{
							Domain:      req.Client.OriginDomain(),
							TenantId:    tenantId,
							ProfileId:   profileId,
							UserId:      userId,
							AdminId:     adminId,
							Roles:       make([]string, 0), // ["admin", "finance"],
							Permissions: make([]string, 0), // ["payroll.read", "asset.write"],
							ExpiresAt:   helper.GetTimestamp(nil).Add(time.Minute * 15),
						}, rememberMe)

						if helper.IsEmpty(cookieEnabled) {
							user[helper.ToVariable(string(AccessTokenKey))] = accessToken
							user[helper.ToVariable(string(RefreshTokenKey))] = refreshToken
						} else {
							user[helper.ToVariable(string(AccessTokenKey))] = "Cookie is set"
							user[helper.ToVariable(string(RefreshTokenKey))] = "Cookie is set"
						}

						g.yekonga.setAuthCookies(req, accessToken, refreshToken)
					}

					return user, nil
				} else if e != nil {
					g.yekonga.RecordLoginAttempt("fail", p.Context, AttemptData{
						Username:     username,
						UsernameType: usernameType,
						LoginType:    loginType,
					})

					return nil, e
				}
			}

			g.yekonga.RecordLoginAttempt("fail", p.Context, AttemptData{
				Username:     username,
				UsernameType: usernameType,
				LoginType:    loginType,
			})

			return nil, nil
		},
	}
}

func _refreshToken(g *GraphqlAutoBuild) *graphql.Field {
	// var foreignKey string
	// var targetKey string
	// var name string = "User"
	var targetType *graphql.Object

	if g.yekonga.Config.SecureAuthentication {
		targetType = CredentialTokenType
	} else {
		targetType = UserProfileType
	}

	return &graphql.Field{
		Type: targetType,
		Args: graphql.FieldConfigArgument{
			"refreshToken": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			req, _ := p.Context.Value(RequestContextKey).(*RequestContext)
			refreshToken := helper.GetValueOfString(p.Args, "refreshToken")

			if g.yekonga.Config.SecureAuthentication {
				result, status := req.App.refreshTokenProcess(req.Request, req.Response, refreshToken)

				if status == http.StatusOK {
					return result, nil
				}

				if err, ok := result["error"]; ok {
					return nil, errors.New(helper.ToString(err))
				}
				return nil, nil
			}

			return nil, nil

		},
	}
}

// socialLogin ( input: SocialLoginInput! ): Profile,
func _socialLogin(g *GraphqlAutoBuild) *graphql.Field {
	var foreignKey string
	var targetKey string
	var name string = "User"

	return &graphql.Field{
		Type: UserProfileType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: SocialLoginInput,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var input map[string]interface{} = g.getInputData(p.Args)
			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			user := model.FindOne(input)

			return user, nil
		},
	}
}

// contactOTP ( input: ConcatOtpInput!, type: UsernameIdentifier! ): ActionResponse,
func _contactOTP(g *GraphqlAutoBuild) *graphql.Field {
	var foreignKey string
	var targetKey string
	var name string = "User"

	return &graphql.Field{
		Type: ActionResponseType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: ConcatOtpInput,
			},
			"type": &graphql.ArgumentConfig{
				Type: UsernameIdentifierEnum,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var input map[string]interface{} = g.getInputData(p.Args)
			// var type = g.getParamValue(p.Args, "type")
			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			user := model.FindOne(input)

			return user, nil
		},
	}
}

// contactVerify ( input: ContactVerifyInput!, type: UsernameIdentifier! ): ActionResponse,
func _contactVerify(g *GraphqlAutoBuild) *graphql.Field {
	var foreignKey string
	var targetKey string
	var name string = "User"

	return &graphql.Field{
		Type: ActionResponseType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: ContactVerifyInput,
			},
			"type": &graphql.ArgumentConfig{
				Type: UsernameIdentifierEnum,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var input map[string]interface{} = g.getInputData(p.Args)
			// var type = g.getParamValue(p.Args, "type")
			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			user := model.FindOne(input)

			return user, nil
		},
	}
}

// registration ( input: RegistrationInput! ): Profile,
func _registration(g *GraphqlAutoBuild) *graphql.Field {
	var foreignKey string
	var targetKey string
	var name string = "User"

	return &graphql.Field{
		Type: UserProfileType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: LoginInput,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var input map[string]interface{} = g.getInputData(p.Args)
			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			user := model.FindOne(input)

			return user, nil
		},
	}
}

// resetPassword ( input: ResetPasswordInput! ): ActionResponse,
func _resetPassword(g *GraphqlAutoBuild) *graphql.Field {
	var foreignKey string
	var targetKey string
	var name string = "User"

	return &graphql.Field{
		Type: ActionResponseType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: ResetPasswordInput,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var data map[string]interface{} = g.getInputData(p.Args)
			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			user := model.FindOne(data)

			return user, nil
		},
	}
}

// confirmToken ( input: ConfirmTokenInput! ): ConfirmTokenResponse,
func _confirmToken(g *GraphqlAutoBuild) *graphql.Field {
	var foreignKey string
	var targetKey string
	var name string = "User"

	return &graphql.Field{
		Type: ConfirmTokenResponseType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: ConfirmTokenInput,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var input map[string]interface{} = g.getInputData(p.Args)
			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			user := model.FindOne(input)

			return user, nil
		},
	}
}

// changePassword ( input: ChangePasswordInput! ): ActionResponse,
func _changePassword(g *GraphqlAutoBuild) *graphql.Field {
	var foreignKey string
	var targetKey string
	var name string = "User"

	return &graphql.Field{
		Type: ActionResponseType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: ChangePasswordInput,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var input map[string]interface{} = g.getInputData(p.Args)
			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			user := model.FindOne(input)

			return user, nil
		},
	}
}

// profile ( input: ProfileInput! ): Profile,
func _profile(g *GraphqlAutoBuild) *graphql.Field {
	var foreignKey string
	var targetKey string
	var name string = "User"

	return &graphql.Field{
		Type: UserProfileType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: UserProfileInput,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var input map[string]interface{} = g.getInputData(p.Args)
			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			user := model.Create(input)

			return user, nil
		},
	}
}

// switchAccount ( profileId: String, userId: String, token: String ): Profile,
func _switchAccount(g *GraphqlAutoBuild) *graphql.Field {
	var foreignKey string
	var targetKey string
	var name string = "User"

	return &graphql.Field{
		Type: UserProfileType,
		Args: graphql.FieldConfigArgument{
			"profileId": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"userId": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
			"token": &graphql.ArgumentConfig{
				Type: graphql.String,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var data map[string]interface{} = g.getInputData(p.Args)
			var profileId = g.getParamValue(p.Args, "profileId")
			var userId = g.getParamValue(p.Args, "userId")
			var token = g.getParamValue(p.Args, "token")

			console.Success("_switchAccount.token", token)
			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			model.Where("profileId", profileId)
			model.Where("userId", userId)

			user := model.FindOne(data)

			return user, nil
		},
	}
}
