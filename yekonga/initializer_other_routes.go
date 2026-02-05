package yekonga

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/console"
)

type WebConfig struct {
	config                map[string]any
	profileConfig         map[string]any
	systemLanguages       []map[string]any
	systemDefaultLanguage []map[string]any
	systemPermissions     []map[string]any
	systemTemplateConfig  map[string]any
}

func (y *YekongaData) initializerConfig(req *Request) WebConfig {
	locale := "en"
	// auth := *req.Auth()
	client := *req.Client()
	config := map[string]any{}
	profileConfig := map[string]any{}
	systemTemplateConfig := map[string]any{}
	systemPermissions := []map[string]any{}
	baseUrl := client.Proto + "://" + client.Host + ":" + client.Port

	systemLanguages := []map[string]interface{}{}
	systemDefaultLanguage := []map[string]interface{}{}

	listA := y.ModelQuery("TranslatorLanguage").Find(nil)
	countA := len(*listA)

	for i := 0; i < countA; i++ {
		e := (*listA)[i]
		d := datatype.DataMap{
			"locale": e["locale"],
			"name":   e["name"],
			"flag":   e["flag"],
			"id":     e["id"],
		}

		systemLanguages = append(systemLanguages, d)
	}

	listB := y.ModelQuery("TranslatorTranslation").Find(map[string]any{"locale": locale})
	countB := len(*listB)

	for i := 0; i < countB; i++ {
		e := (*listB)[i]
		d := datatype.DataMap{
			"id":           e["id"],
			"locale":       e["locale"],
			"name":         e["name"],
			"namespace":    e["namespace"],
			"group":        e["group"],
			"item":         e["item"],
			"descriptions": e["descriptions"],
			"text":         e["text"],
			"locked":       e["locked"],
		}

		systemDefaultLanguage = append(systemDefaultLanguage, d)
	}

	config["baseUrl"] = baseUrl
	config["host"] = client.Host
	config["appName"] = y.Config.AppName
	config["endToEndEncryption"] = y.Config.EndToEndEncryption
	config["apiRoute"] = y.Config.Graphql.ApiRoute
	config["authRoute"] = y.Config.Graphql.ApiAuthRoute
	config["googleApiKey"] = y.Config.GoogleApiKey
	config["googleClientId"] = y.Config.GoogleClientId
	config["googleClientSecret"] = y.Config.GoogleClientSecret
	config["cryptojsKey"] = y.Config.Authentication.CryptoJsKey
	config["cryptojsIv"] = y.Config.Authentication.CryptoJsIv
	config["userIdentifiers"] = y.Config.UserIdentifiers
	config["locale"] = locale
	config["language"] = locale

	return WebConfig{
		config:                config,
		profileConfig:         profileConfig,
		systemLanguages:       systemLanguages,
		systemDefaultLanguage: systemDefaultLanguage,
		systemPermissions:     systemPermissions,
		systemTemplateConfig:  systemTemplateConfig,
	}
}

func (y *YekongaData) initializerOtherRoutes() {

	y.All("/upload", func(req *Request, res *Response) {
		uploadFileHandler(*res.httpResponseWriter, req.HttpRequest)
	})

	y.All("/upload-files", func(req *Request, res *Response) {
		uploadMultipleFileHandler(*res.httpResponseWriter, req.HttpRequest)
	})

	y.All("/languages", func(req *Request, res *Response) {
		languages := []map[string]interface{}{}

		list := y.ModelQuery("TranslatorLanguage").Find(nil)
		count := len(*list)

		for i := 0; i < count; i++ {
			e := (*list)[i]
			d := datatype.DataMap{
				"locale": e["locale"],
				"name":   e["name"],
				"flag":   e["flag"],
				"id":     e["locale"],
			}

			languages = append(languages, d)
		}

		res.Json(languages)
	})

	y.All("/translations/:locale", func(req *Request, res *Response) {
		locale := req.Param("locale")
		translations := []map[string]interface{}{}
		list := y.ModelQuery("TranslatorTranslation").Find(map[string]any{"locale": locale})
		count := len(*list)

		for i := 0; i < count; i++ {
			e := (*list)[i]
			d := datatype.DataMap{
				"id":           e["id"],
				"locale":       e["locale"],
				"name":         e["name"],
				"namespace":    e["namespace"],
				"group":        e["group"],
				"item":         e["item"],
				"descriptions": e["descriptions"],
				"text":         e["text"],
				"locked":       e["locked"],
			}

			translations = append(translations, d)
		}

		res.Json(translations)
	})

	y.All("/config/data", func(req *Request, res *Response) {
		wetConfig := y.initializerConfig(req)

		res.Json(map[string]interface{}{
			"config":                wetConfig.config,
			"profileConfig":         wetConfig.profileConfig,
			"systemLanguages":       wetConfig.systemLanguages,
			"systemDefaultLanguage": wetConfig.systemDefaultLanguage,
			"systemPermissions":     wetConfig.systemPermissions,
			"systemTemplateConfig":  wetConfig.systemTemplateConfig,
		})
	})

	y.All("/config/report", func(req *Request, res *Response) {
		config := map[string]any{"reportConfig": nil}

		res.Json(config)
	})

	y.All("/permissions", func(req *Request, res *Response) {
		config := map[string]any{}

		auth := req.Auth()
		if auth == nil {
			config["error"] = "You must login first"
		} else {
			config["permissions"] = []interface{}{}
		}

		res.Json(config)
	})

	y.All("/config", func(req *Request, res *Response) {
		wetConfig := y.initializerConfig(req)

		content := "window['systemLanguages'] = " + helper.ToJson(wetConfig.systemLanguages) + ";\n" +
			"window['systemDefaultLanguage'] = " + helper.ToJson(wetConfig.systemDefaultLanguage) + ";\n" +
			"window['systemTemplateConfig'] = {};\n" +
			"window['systemPermissions'] = " + helper.ToJson(wetConfig.systemPermissions) + ";\n" +
			"window['systemConfig'] = " + helper.ToJson(wetConfig.config) + ";\n" +
			"window['ProfileConfig'] = " + helper.ToJson(wetConfig.profileConfig) + ";\n"

		res.Header("Cache-Control", "public, max-age=0")
		res.Header("content-type", "application/javascript; charset=UTF-8")
		res.Byte([]byte(content))
	})

	y.All("/download/:filename", func(req *Request, res *Response) {
		filename := req.Param("filename")
		title := req.Query("title")
		// console.Log(filename)

		publicDir, _ := helper.GetPublicPath()
		file := path.Join(publicDir, "tmp", filename)

		console.Log(file)

		res.Download(file, title)
	})

	y.Get("/yekonga/yekonga.js", func(req *Request, res *Response) {
		scripts := ""
		client := *req.Client()
		var content []byte

		content, _ = StaticFS.ReadFile("static/dk/axios.min.js")
		scripts += string(content) + "\n"

		content, _ = StaticFS.ReadFile("static/sdk/simplepeer.min.js")
		scripts += string(content) + "\n"

		content, _ = StaticFS.ReadFile("static/sdk/yekonga.io.js")
		scripts += string(content) + "\n"

		var config string = "window.YekongaServer = {};\n" +
			"window.YekongaServer.Applications = {};\n" +
			"window.socket = null;\n" +
			"window.socketSystem = null;\n" +
			"window.YekongaServer.Host = '" + client.Host + "';\n" +
			"window.YekongaServer.Proto = '" + client.Proto + "';\n" +
			"window.YekongaServer.Port = '" + client.Port + "';\n" +
			"window.YekongaServer.graphql = '" + y.Config.Graphql.ApiRoute + "';"

		scripts += config + "\n"

		content, _ = StaticFS.ReadFile("static/sdk/webRTC.js")
		scripts += string(content) + "\n"

		content, _ = StaticFS.ReadFile("static/sdk/yekonga.js")
		scripts += string(content) + "\n"

		res.Header("Cache-Control", "public, max-age=0")
		res.Header("content-type", "application/javascript")
		res.Byte([]byte(scripts))
	})

	y.Get("/theme.css", runCustomCSS(y))
	y.Get("/custom-style.css", runCustomCSS(y))

	if y.Config.ApiPlaygroundEnable || y.Config.AuthPlaygroundEnable {
		y.Get("/playground", func(req *Request, res *Response) {
			content, _ := StaticFS.ReadFile("static/playground/index.html")
			html := string(content)

			apiRoute := y.AppendBaseUrl(y.Config.Graphql.ApiRoute)
			baseUrl := y.AppendBaseUrl("")
			data := map[string]interface{}{
				"apiRoute": apiRoute,
				"baseUrl":  baseUrl,
			}
			html = helper.TextTemplate(html, data, nil)

			res.Html(html)
		})

		y.Get("/playground/font.css", func(req *Request, res *Response) {
			content, _ := StaticFS.ReadFile("static/playground/font.css")

			res.Header("content-type", "text/css; charset=utf-8")
			res.Byte(content)
		})

		y.Get("/playground/index.css", func(req *Request, res *Response) {
			content, _ := StaticFS.ReadFile("static/playground/index.css")

			res.Header("content-type", "text/css; charset=utf-8")
			res.Byte(content)
		})

		y.Get("/playground/favicon.png", func(req *Request, res *Response) {
			content, _ := StaticFS.ReadFile("static/playground/fovicon.png")

			res.Header("content-type", "image/png")
			res.Byte(content)
		})

		y.Get("/playground/middleware.js", func(req *Request, res *Response) {
			content, _ := StaticFS.ReadFile("static/playground/middleware.js")

			res.Header("content-type", "text/javascript")
			res.Byte(content)
		})
	}

	y.Get("/placeholder.jpg", func(req *Request, res *Response) {
		content, _ := StaticFS.ReadFile("static/placeholder.jpg")

		res.Header("content-type", "image/jpeg")
		res.Byte(content)
	})
}

func (y *YekongaData) initializerSocketRoutes() {

	y.Get("/yekonga.io/yekonga.io.js", func(req *Request, res *Response) {
		scripts := ""
		var content []byte

		content, _ = StaticFS.ReadFile("static/sdk/yekonga.io.js")
		scripts += string(content) + "\n"

		res.Header("Cache-Control", "public, max-age=0")
		res.Header("content-type", "application/javascript")
		res.Byte([]byte(scripts))
	})

	y.All("/yekonga.io/", func(req *Request, res *Response) {

		(*res.httpResponseWriter).Header().Set("access-control-allow-origin", "*")
		(*res.httpResponseWriter).Header().Add("access-control-allow-headers", "content-type, authorization, x-requested-with, x-csrf-token, timezone, upgrade-insecure-requests")

		y.socketServer.ServeWS(req, res)
	})
}

func runCustomCSS(y *YekongaData) Handler {
	return func(req *Request, res *Response) {
		content := "/* @charset \"UTF-8\"; */" +
			"" +
			""

		custom, err := y.CustomCSS(req, res)
		if err == nil && helper.IsNotEmpty(custom) {

			if str, ok := custom.(string); ok {
				content += str
			}
		}

		res.Header("Cache-Control", "public, max-age=0")
		res.Header("content-type", "text/css; charset=utf-8")
		res.Byte([]byte(content))
	}
}

func uploadFileHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Parse the multipart form (max 32MB in memory)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Form too large", http.StatusBadRequest)
		return
	}

	// 2. Retrieve the file from form data
	file, handler, err := r.FormFile("file")

	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)

	// 3. Create the destination directory if it doesn't exist
	uploadDir := helper.GetPath("public/uploads")
	os.MkdirAll(uploadDir, os.ModePerm)

	// 4. Create a local file to save the uploaded data
	dst, err := os.Create(filepath.Join(uploadDir, handler.Filename))
	if err != nil {
		http.Error(w, "Error saving the file", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 5. Copy the uploaded file to the destination
	if _, err := io.Copy(dst, file); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Fprintf(w, "Successfully Uploaded File\n")
}

func uploadMultipleFileHandler(w http.ResponseWriter, r *http.Request) {
	// 1. Parse the multipart form (max 32MB in memory)
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Form too large", http.StatusBadRequest)
		return
	}

	// 2. Get the files from the specific key
	files := r.MultipartForm.File["files"]

	uploadDir := helper.GetPath("public/uploads")
	os.MkdirAll(uploadDir, os.ModePerm)

	for _, fileHeader := range files {
		// Open the uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// 3. Create destination path
		dstPath := filepath.Join(uploadDir, fileHeader.Filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			http.Error(w, "Unable to save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// 4. Copy the data
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Printf("Saved: %s\n", fileHeader.Filename)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "All files uploaded successfully")
}
