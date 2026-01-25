package config

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

type SMSGatewayConfig struct { // SMS gateway configuration
	Provider  GatewayProvider `json:"provider"`  // SMS provider (e.g., beem, infobip)
	BaseURL   string          `json:"baseURL"`   // Base URL for the SMS gateway API
	Sender    string          `json:"sender"`    // Sender ID or name for SMS messages
	APIKey    string          `json:"apiKey"`    // API key for authentication
	SecretKey string          `json:"secretKey"` // Secret key for authentication
	Username  string          `json:"username"`  // Username for authentication
	Password  string          `json:"password"`  // Password for authentication
}

type WhatsappGatewayConfig struct { // WhatsApp gateway configuration
	Provider GatewayProvider `json:"provider"` // WhatsApp provider (e.g., infobip)
	Sender   string          `json:"sender"`   // Sender ID or name for WhatsApp messages
	Sandbox  string          `json:"sandbox"`  // Sandbox environment indicator
	BaseURL  string          `json:"baseURL"`  // Base URL for the WhatsApp gateway API
	APIKey   string          `json:"apiKey"`   // API key for authentication
}

// SMTPConfig holds SMTP configuration
type SMTPConfig struct {
	Service  string `json:"service"`  // SMTP service provider
	Host     string `json:"host"`     // SMTP host address
	Port     int    `json:"port"`     // SMTP port
	Secure   bool   `json:"secure"`   // Enable secure connection (SSL/TLS)
	From     string `json:"from"`     // Sender email address
	Domain   string `json:"domain"`   // Sending domain
	Username string `json:"username"` // SMTP username
	Password string `json:"password"` // SMTP password
	APIKey   string `json:"apiKey"`   // API key for transactional email services
}

type Branding struct { // Branding configuration for the application
	LogoUrl             string `json:"logoUrl"`             // URL to the application logo
	FaviconUrl          string `json:"faviconUrl"`          // URL to the favicon
	PrimaryColor        string `json:"primaryColor"`        // Primary color for the application UI
	SecondaryColor      string `json:"secondaryColor"`      // Secondary color for the application UI
	DarkBackgroundColor string `json:"darkBackgroundColor"` // Background color for dark mode
}

type YekongaConfig struct {
	AppName                 string   `json:"appName"`                 // Name of the application
	Version                 string   `json:"version"`                 // Version of the application
	Description             string   `json:"description"`             // Description of the application
	AppKey                  string   `json:"appKey"`                  // Key for the application
	MasterKey               string   `json:"masterKey"`               // Master key for the application
	EnableAppKey            bool     `json:"enableAppKey"`            // Enable or disable app key usage
	ConnectionID            string   `json:"connectionID"`            // Connection ID
	UserIdentifiers         []string `json:"userIdentifiers"`         // List of user identifiers
	Domain                  string   `json:"domain"`                  // Application domain
	Protocol                string   `json:"protocol"`                // Protocol (e.g., http, https)
	DomainAlias             []string `json:"domainAlias"`             // List of domain aliases
	Address                 string   `json:"address"`                 // Application address
	BaseUrl                 string   `json:"baseURL"`                 // Base URL of the application
	RestApi                 string   `json:"restAPI"`                 // REST API endpoint
	RestAuthApi             string   `json:"restAuthAPI"`             // REST authentication API endpoint
	TokenKey                string   `json:"tokenKey"`                // Key for generating tokens
	PdfInstances            int      `json:"pdfInstances"`            // Number of PDF instances
	TokenExpireTime         int      `json:"tokenExpireTime"`         // Token expiration time in minutes
	SecureOnly              bool     `json:"secureOnly"`              // Enforce secure connections only
	Debug                   bool     `json:"debug"`                   // Enable or disable debug mode
	Cors                    bool     `json:"cors"`                    // Enable or disable CORS
	ResetOTP                bool     `json:"resetOTP"`                // Enable or disable OTP reset
	Environment             string   `json:"environment"`             // Application environment (e.g., development, production)
	HasTenant               bool     `json:"hasTenant"`               // Enable multi-tenancy
	SecureAuthentication    bool     `json:"secureAuthentication"`    // Enable or disable secure authentication
	IsAuthorizationServer   bool     `json:"isAuthorizationServer"`   // Designate as an authorization server
	MustAuthorized          bool     `json:"mustAuthorized"`          // Require authorization for all requests
	HasCronjob              bool     `json:"hasCronjob"`              // Require authorization for all requests
	RegisterUserOnOtp       bool     `json:"registerUserOnOtp"`       // Register user automatically on OTP verification
	SendOtpToSmsAndWhatsapp bool     `json:"sendOtpToSmsAndWhatsapp"` // Send OTP via SMS and WhatsApp
	EndToEndEncryption      bool     `json:"endToEndEncryption"`      // Enable end-to-end encryption
	AuthPlaygroundEnable    bool     `json:"authPlaygroundEnable"`    // Enable authentication playground
	ApiPlaygroundEnable     bool     `json:"apiPlaygroundEnable"`     // Enable API playground
	EnableDashboard         bool     `json:"enableDashboard"`         // Enable admin dashboard
	AllowCreateFrontend     bool     `json:"allowCreateFrontend"`     // Allow frontend creation
	NamingConvention        string   `json:"namingConvention"`        // Naming convention for database tables and fields
	ColumnNamingConvention  string   `json:"columnNamingConvention"`  // Naming convention for database columns
	NamingConventionOptions []string `json:"namingConventionOptions"` // Options for naming conventions
	Public                  []string `json:"public"`                  // List of public routes/endpoints
	Cloud                   string   `json:"cloud"`                   // Cloud provider configuration
	LogFile                 string   `json:"logFile"`                 // Path to the log file
	IndexTemplate           string   `json:"indexTemplate"`           // Path to the index HTML template
	EmailTemplate           string   `json:"emailTemplate"`           // Path to the email HTML template
	GoogleApiKey            string   `json:"googleApiKey"`            // Google API key
	GoogleApiKeyAlt         string   `json:"googleApiKeyAlt"`         // Alternative Google API key
	GoogleClientId          string   `json:"googleClientId"`          // Google OAuth client ID
	GoogleClientSecret      string   `json:"googleClientSecret"`      // Google OAuth client secret
	GlobalPassword          string   `json:"globalPassword"`          // Global password for certain operations
	Branding                Branding `json:"branding"`                // Branding configuration
	Permissions             struct { // Permissions configuration
		AuthActions  []string `json:"authActions"`  // List of actions requiring authentication
		GuestActions []string `json:"guestActions"` // List of actions accessible to guests
	}
	Graphql struct { // GraphQL configuration
		ApiRoute            string      `json:"apiRoute"`            // GraphQL API route
		ApiAuthRoute        string      `json:"apiAuthRoute"`        // GraphQL authentication API route
		CustomTypes         string      `json:"customTypes"`         // Path to custom GraphQL types
		CustomResolvers     string      `json:"customResolvers"`     // Path to custom GraphQL resolvers
		CustomAuthTypes     string      `json:"customAuthTypes"`     // Path to custom authenticated GraphQL types
		CustomAuthResolvers string      `json:"customAuthResolvers"` // Path to custom authenticated GraphQL resolvers
		EnabledForClasses   interface{} `json:"enabledForClasses"`   // GraphQL enabled for specific classes
		DisabledForClasses  interface{} `json:"disabledForClasses"`  // GraphQL disabled for specific classes
		AuthResolvers       interface{} `json:"authResolvers"`       // Authenticated GraphQL resolvers
		AuthClasses         interface{} `json:"authClasses"`         // Authenticated GraphQL classes
		GuestResolvers      interface{} `json:"guestResolvers"`      // Guest GraphQL resolvers
		GuestClasses        interface{} `json:"guestClasses"`        // Guest GraphQL classes
		AuthQuery           struct {    // Authenticated GraphQL queries
			User    interface{} `json:"user"`    // User-related queries
			Account interface{} `json:"account"` // Account-related queries
		}
	}
	Database struct { // Database configuration
		Kind             DatabaseType `json:"kind"`             // Type of database (e.g., mongodb, sql, mysql, local)
		Srv              bool         `json:"srv"`              // Enable SRV record lookup for MongoDB
		Host             string       `json:"host"`             // Database host address
		Port             string       `json:"port"`             // Database port
		DatabaseName     string       `json:"databaseName"`     // Name of the database
		Username         interface{}  `json:"username"`         // Database username
		Password         interface{}  `json:"password"`         // Database password
		Prefix           string       `json:"prefix"`           // Table/collection name prefix
		GenerateID       bool         `json:"generateID"`       // Automatically generate IDs for new records
		GenerateIDLength int          `json:"generateIDLength"` // Length of generated IDs
	}
	Authentication struct { // Authentication configuration
		SaltRound   int    `json:"saltRound"`   // Number of salt rounds for password hashing
		Algorithm   string `json:"algorithm"`   // Hashing algorithm for passwords
		SecretToken string `json:"secretToken"` // Secret key for JWT or session tokens
		CryptoJsKey string `json:"cryptoJsKey"` // Cryptographic key for client-side encryption
		CryptoJsIv  string `json:"cryptoJsIv"`  // Initialization vector for client-side encryption
	}
	Ports struct { // Port configuration
		Secure    bool `json:"secure"`    // Enable secure ports (HTTPS/SSL)
		Server    int  `json:"server"`    // HTTP server port
		SSLServer int  `json:"sslServer"` // HTTPS/SSL server port
		Redis     int  `json:"redis"`     // Redis server port
	}
	Mail struct { // Mail configuration
		Smtp SMTPConfig `json:"smtp"` // SMTP server configuration
	}
	ApiGateway struct { // API Gateway configuration for external services
		SMS      SMSGatewayConfig      `json:"sms"`      // SMS gateway configuration
		Whatsapp WhatsappGatewayConfig `json:"whatsapp"` // WhatsApp gateway configuration
	}
	AdminCredential struct { // Admin user credentials
		Username interface{} `json:"username"` // Admin username
		Password interface{} `json:"password"` // Admin password
	}
}
