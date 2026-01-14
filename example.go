package main

// func main() {
// 	Yekonga.ServerConfig("./config.json", "./database.json")

// 	Yekonga.Server.Define("getDomain", func(i interface{}, rc *Yekonga.RequestContext) (interface{}, error) {
// 		return "yekonga.co.tz", nil
// 	})

// 	Yekonga.Server.AfterFind("User", nil, nil, func(rc *Yekonga.RequestContext, qc *Yekonga.QueryContext) (interface{}, error) {
// 		// logger.Success("Parent", qc.Parent)
// 		// logger.Success("Data", qc.Data)

// 		return nil, nil
// 	})

// 	Yekonga.Server.Get("/", func(req *Yekonga.Request, res *Yekonga.Response) {
// 		res.File("index.html")
// 	})

// 	Yekonga.Server.Get("/me", func(req *Yekonga.Request, res *Yekonga.Response) {
// 		res.Json(req.Auth().ToMap())
// 	})

// 	Yekonga.Server.Get("/list", func(req *Yekonga.Request, res *Yekonga.Response) {
// 		// start := time.Now()
// 		// helper.TrackTime(&start, "list 1")
// 		var list = Yekonga.Server.ModelQuery("AuditTrail").Take(100).Find(nil)
// 		// helper.TrackTime(&start, "list 2")

// 		res.Json(list)
// 	})

// 	Yekonga.Server.Get("/domain", func(req *Yekonga.Request, res *Yekonga.Response) {
// 		// start := time.Now()
// 		// helper.TrackTime(&start, "list 1")
// 		list, _ := Yekonga.Server.Run("getDomain", nil, &Yekonga.RequestContext{}, 0)
// 		// helper.TrackTime(&start, "list 2")

// 		res.Text(helper.ToJson(list))
// 	})

// 	// Example routes with parameters
// 	Yekonga.Server.Get("/user/:id", func(req *Yekonga.Request, res *Yekonga.Response) {
// 		userId := req.Param("id")
// 		res.Json(map[string]string{"userId": userId})
// 	})

// 	Yekonga.Server.Get("/user/first/:id", func(req *Yekonga.Request, res *Yekonga.Response) {
// 		userId := req.Param("id")
// 		res.Text(userId)
// 	})

// 	// user := ye.GetUser(datatype.DataMap{
// 	// 	"username": "0955257599",
// 	// })
// 	// console.Log("user", user)
// 	// testQueryParser()
// 	// data, _ := helper.LoadJSONFile("graphql.json")
// 	// result := helper.ExtractGraphqlQuery(data, 0)
// 	// logger.Error(result)

// 	// Start the server
// 	Yekonga.Server.Start(nil)
// }

// func testQueryParser() {
// 	query := `query {
// 		users {
// 			id,
// 			name,
// 			p:phone
// 		},

// 	}`
// 	doc := graphql.Parser(query)

// 	fmt.Println(helper.ToJson(doc))
// }
