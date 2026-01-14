package yekonga

import (
	"github.com/robertkonga/yekonga-server/datatype"
	"github.com/robertkonga/yekonga-server/helper"
	"github.com/robertkonga/yekonga-server/plugins/graphql"
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
			var result = ActionResponse{
				Message: "Fail",
			}
			var user datatype.DataMap
			var data map[string]interface{} = g.getInputData(p.Args)
			var username = g.getParamValue(data, "username")
			if helper.IsPhone(username) {
				username = helper.PhoneFormat(username)
			}
			if helper.IsNotEmpty(username) {
				user = g.yekonga.GetUser(username, true)
			}

			if user != nil {
				result.Status = true
				result.Message = "Success"

				userId := helper.ObjectID(helper.GetValueOfString(user, "_id"))
				otpCode := helper.GetValueOfString(user, "otpCode")

				if helper.IsEmpty(user["otpCode"]) {
					otpCode = helper.GetRandomInt(4)
					otpCreatedAt := helper.GetTimestamp(nil)

					g.yekonga.ModelQuery("User").Where("id", userId).Update(datatype.DataMap{
						"otpCode":      otpCode,
						"otpCreatedAt": otpCreatedAt,
					}, nil)
				}

				message := otpCode + " is your verification code. For security, do not share this code."

				g.yekonga.Notify(&User{
					UserID:   userId.String(),
					Email:    helper.GetValueOfString(user, "email"),
					Phone:    helper.GetValueOfString(user, "phone"),
					Whatsapp: helper.GetValueOfString(user, "whatsapp"),
				}, NotificationParams{
					Title:    "OTP",
					Text:     message,
					HTML:     message,
					Whatsapp: message,
				})

			}

			return result.ToMap(), nil
		},
	}
}

// login ( input: LoginInput!, type: UsernameIdentifier ): Profile,
func _login(g *GraphqlAutoBuild) *graphql.Field {
	// var foreignKey string
	// var targetKey string
	// var name string = "User"

	return &graphql.Field{
		Type: UserProfileType,
		Args: graphql.FieldConfigArgument{
			"input": &graphql.ArgumentConfig{
				Type: LoginInput,
			},
			"type": &graphql.ArgumentConfig{
				Type: UsernameIdentifierEnum,
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var input map[string]interface{} = g.getInputData(p.Args)
			// var type = g.getParamValue(p.Args, "type")
			var user datatype.DataMap
			var username = helper.GetValueOfString(input, "username")
			var usernameType = helper.GetValueOfString(input, "usernameType")
			var password = helper.GetValueOfString(input, "password")
			var loginType = helper.GetValueOfString(input, "type")

			if helper.IsPhone(username) {
				username = helper.PhoneFormat(username)
			}
			if helper.IsNotEmpty(username) {
				user = g.yekonga.GetUser(username, false)
			}

			if user != nil {
				u, e := g.yekonga.AttemptLogin(p.Context, AttemptData{
					Username:     username,
					UsernameType: usernameType,
					Password:     password,
					LoginType:    loginType,
				})

				if u != nil {
					userId := helper.GetValueOfString(u, "id")
					user := g.yekonga.GetLoginData(p.Context, &LoginData{
						UserID: userId,
					})

					return user, nil
				} else if e != nil {
					return nil, e
				}
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
			// var token = g.getParamValue(p.Args, "token")

			var model = g.yekonga.ModelQuery(name)

			g.setModelParams(model, &p, foreignKey, targetKey)
			model.Where("profileId", profileId)
			model.Where("userId", userId)

			user := model.FindOne(data)

			return user, nil
		},
	}
}
