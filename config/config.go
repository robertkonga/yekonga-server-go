package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

type DatabaseType string

const (
	DBTypeMongodb DatabaseType = "mongodb"
	DBTypeSql     DatabaseType = "sql"
	DBTypeMysql   DatabaseType = "mysql"
	DBTypeLocal   DatabaseType = "local"
)

type GatewayProvider string

const (
	ProviderBeem    GatewayProvider = "beem"
	ProviderInfobip GatewayProvider = "infobip"
	ProviderAlibaba GatewayProvider = "alibaba"
)

type SMSGatewayConfig struct {
	Provider  GatewayProvider
	BaseURL   string
	Sender    string
	APIKey    string
	SecretKey string
	Username  string
	Password  string
}

type WhatsappGatewayConfig struct {
	Provider GatewayProvider
	Sender   string
	Sandbox  string
	BaseURL  string
	APIKey   string
}

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Service  string `json:"service"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Secure   bool   `json:"secure"`
	From     string `json:"from"`
	Domain   string `json:"domain"`
	Username string `json:"username"`
	Password string `json:"password"`
	APIKey   string `json:"apiKey"`
}

type YekongaConfig struct {
	AppName                 string
	Version                 string
	Description             string
	LogoUrl                 string
	FaviconUrl              string
	PrimaryColor            string
	SecondaryColor          string
	DarkBackgroundColor     string
	AppKey                  string
	MasterKey               string
	EnableAppKey            bool
	ConnectionID            string
	UserIdentifiers         []string
	Domain                  string
	Protocol                string
	DomainAlias             []string
	Address                 string
	BaseUrl                 string
	RestApi                 string
	RestAuthApi             string
	TokenKey                string
	PdfInstances            int
	TokenExpireTime         int
	SecureOnly              bool
	Debug                   bool
	Cors                    bool
	ResetOTP                bool
	Environment             string
	SecureAuthentication    bool
	IsAuthorizationServer   bool
	HasTenant               bool
	RegisterUserOnOtp       bool
	SendOtpToSmsAndWhatsapp bool
	EndToEndEncryption      bool
	AuthPlaygroundEnable    bool
	ApiPlaygroundEnable     bool
	EnableDashboard         bool
	AllowCreateFrontend     bool
	NamingConvention        string
	ColumnNamingConvention  string
	NamingConventionOptions []string
	Public                  []string
	Cloud                   string
	LogFile                 string
	IndexTemplate           string
	EmailTemplate           string
	GoogleApiKey            string
	GoogleApiKeyAlt         string
	GoogleClientId          string
	GoogleClientSecret      string
	GlobalPassword          string
	Permissions             struct {
		AuthActions  []string
		GuestActions []string
	}
	Graphql struct {
		ApiRoute            string
		ApiAuthRoute        string
		CustomTypes         string
		CustomResolvers     string
		CustomAuthTypes     string
		CustomAuthResolvers string
		EnabledForClasses   interface{}
		DisabledForClasses  interface{}
		AuthResolvers       interface{}
		AuthClasses         interface{}
		GuestResolvers      interface{}
		GuestClasses        interface{}
		AuthQuery           struct {
			User    interface{}
			Account interface{}
		}
	}
	Database struct {
		Kind             DatabaseType
		Srv              bool
		Host             string
		Port             string
		DatabaseName     string
		Username         interface{}
		Password         interface{}
		Prefix           string
		GenerateID       bool
		GenerateIDLength int
	}
	Authentication struct {
		SaltRound   int
		Algorithm   string
		SecretToken string
		CryptoJsKey string
		CryptoJsIv  string
	}
	Ports struct {
		Secure    bool
		Server    int
		SSLServer int
		Redis     int
	}

	Mail struct {
		Smtp SMTPConfig
	}
	ApiGateway struct {
		SMS      SMSGatewayConfig
		Whatsapp WhatsappGatewayConfig
	}
	AdminCredential struct {
		Username interface{}
		Password interface{}
	}
}

var Config *YekongaConfig

func NewYekongaConfig(file string) *YekongaConfig {
	var config YekongaConfig
	if !FileExists(file) {
		file = GetPath(file)
	}

	data, err := LoadJSONFile(file)
	if err != nil {
		fmt.Println(err)
	}
	Config = &config

	json.Unmarshal(ToByte(data), Config)

	return Config
}

func ToByte(data interface{}) []byte {
	jsonData, _ := json.Marshal(data)

	return jsonData
}

func LoadJSONFile(filename string) (map[string]interface{}, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}

	return data, nil
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

func GetPath(relativePath string) string {
	ex, err := os.Executable()
	if err != nil {
		log.Fatalf("Error getting executable path: %v", err)
	}

	// 2. Get the directory of the executable
	exPath := filepath.Dir(ex)

	// 3. Join the executable's directory with the relative path
	absolutePath := filepath.Join(exPath, relativePath)

	if err != nil {
		log.Fatalf("Error getting absolute path: %v", err)
	}

	return absolutePath
}
