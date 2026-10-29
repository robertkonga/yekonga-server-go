package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
)

type HexColor string

// LoadTheme reads a JSON file and unmarshals it into a Theme struct.
// Empty input yields the default theme, and any field the JSON leaves
// empty is filled from DefaultTheme when T is Theme or TenantConfig.
func LoadTheme[T any](data string) (*T, error) {
	// Unmarshal JSON into Theme struct
	var theme T
	if trimmed := strings.TrimSpace(data); trimmed != "" && trimmed != "null" {
		if err := json.Unmarshal([]byte(data), &theme); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	}

	if d, ok := any(&theme).(interface{ ApplyDefaults() }); ok {
		d.ApplyDefaults()
	}

	return &theme, nil
}

// LoadTheme reads a JSON file and unmarshals it into a Theme struct
func LoadThemeFile(filePath string) (*Theme, error) {
	// filePath = getPath(filePath)
	// Open the JSON file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read the file contents
	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Unmarshal JSON into Theme struct
	var theme Theme
	if err := json.Unmarshal(bytes, &theme); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	theme.ApplyDefaults()

	return &theme, nil
}

// DefaultTheme returns the built-in theme used for any value a tenant or
// theme file does not set.
func DefaultTheme() Theme {
	return Theme{
		LightTheme: LightTheme{
			Logo: "/img/logo-dark.svg",
			Semantic: SemanticColors{
				Brand:     "#0b7dff",
				Primary:   "#0b7dff",
				Secondary: "#FFC50B",
				Success:   "#00A884",
				Danger:    "#F05252",
				Warning:   "#F6C344",
				Info:      "#4CB5FF",
			},
			Background: BackgroundColors{
				App:     "#F4F7F6",
				Body:    "#F6F6F6",
				Sidebar: "#FFFFFF",
				Header:  "#FFFFFF",
				Card:    "#FFFFFF",
				Muted:   "#EDF2F1",
			},
			Text: TextColors{
				App:           "#28251E",
				Body:          "#2D3748",
				BodyActive:    "#00A884",
				Sidebar:       "#718096",
				SidebarActive: "#28251E",
				Header:        "#28251E",
				HeaderActive:  "#00A884",
				Card:          "#28251E",
				Muted:         "#A0AEC0",
				OnBrand:       "#FFFFFF",
				OnPrimary:     "#FFFFFF",
				OnSecondary:   "#FFC50B",
				OnSuccess:     "#FFFFFF",
				OnDanger:      "#FFFFFF",
				OnWarning:     "#28251E",
				OnInfo:        "#FFFFFF",
			},
			Border: BorderColors{
				App:     "#E2E8F0",
				Body:    "#E2E8F0",
				Sidebar: "#E2E8F0",
				Header:  "#E2E8F0",
				Card:    "#E2E8F0",
				Muted:   "#EDF2F1",
				Default: "#CBD5E1",
			},
			Outline: OutlineColors{
				App:     "#00A884",
				Body:    "#00A884",
				Sidebar: "#00A884",
				Header:  "#00A884",
				Card:    "#00A884",
				Muted:   "#94A3B8",
			},
			SVG: SVGColors{
				Fill:   "#94A3B8",
				Stroke: "#FFFFFF",
			},
			Shadow: Shadows{
				SM: "0 2px 4px 0 rgba(0,0,0,0.3)",
				MD: "0 8px 16px -4px rgba(0,0,0,0.5)",
			},
			Radius: Radius{
				SM: "6px",
				MD: "12px",
				LG: "20px",
			},
			Fonts: Fonts{
				Sans:  "ui-sans-serif, system-ui, sans-serif",
				Serif: "ui-serif, Georgia, serif",
				Mono:  "ui-monospace, SFMono-Regular, Menlo, monospace",
			},
		},
		DarkTheme: DartTheme{
			Logo: "/img/logo-light.svg",
			Background: BackgroundColors{
				App:     "#1e2428",
				Body:    "#1e2428",
				Sidebar: "#282e35",
				Header:  "#1e2328",
				Card:    "#282d35",
				Muted:   "#323743",
			},
			Text: TextColors{
				App:           "#94A3B8",
				Body:          "#94A3B8",
				BodyActive:    "#00A884",
				Sidebar:       "#94A3B8",
				SidebarActive: "#FFFFFF",
				Header:        "#94A3B8",
				HeaderActive:  "#00A884",
				Card:          "#FFFFFF",
				Muted:         "#94A3B8",
				OnBrand:       "#FFFFFF",
				OnPrimary:     "#FFFFFF",
				OnSecondary:   "#FFFFFF",
				OnSuccess:     "#FFFFFF",
				OnDanger:      "#FFFFFF",
				OnWarning:     "#28251E",
				OnInfo:        "#FFFFFF",
			},
			Border: BorderColors{
				App:     "#2D323E",
				Body:    "#2D323E",
				Sidebar: "#2D323E",
				Header:  "#2D323E",
				Card:    "#2D323E",
				Muted:   "#323743",
				Default: "#3F444E",
			},
			Outline: OutlineColors{
				App:     "#00A884",
				Body:    "#00A884",
				Sidebar: "#00A884",
				Header:  "#00A884",
				Card:    "#00A884",
				Muted:   "#94A3B8",
			},
			SVG: SVGColors{
				Fill:   "#94A3B8",
				Stroke: "#FFFFFF",
			},
			Shadow: Shadows{
				SM: "0 2px 4px 0 rgba(0,0,0,0.3)",
				MD: "0 8px 16px -4px rgba(0,0,0,0.5)",
			},
		},
	}
}

// ApplyDefaults fills every empty value of t from DefaultTheme.
func (t *Theme) ApplyDefaults() {
	def := DefaultTheme()
	fillEmptyStrings(reflect.ValueOf(t).Elem(), reflect.ValueOf(def))
}

// ApplyDefaults fills every empty theme value of c from DefaultTheme.
func (c *TenantConfig) ApplyDefaults() {
	def := DefaultTheme()
	fillEmptyStrings(reflect.ValueOf(&c.LightTheme).Elem(), reflect.ValueOf(def.LightTheme))
	fillEmptyStrings(reflect.ValueOf(&c.DarkTheme).Elem(), reflect.ValueOf(def.DarkTheme))

	if c.LightBackground == "" {
		c.LightBackground = string(c.LightTheme.Background.Body)
	}
	if c.DarkBackground == "" {
		c.DarkBackground = string(c.DarkTheme.Background.Body)
	}
}

// fillEmptyStrings walks two values of the same struct type and copies each
// string field of def into dst where dst's is empty.
func fillEmptyStrings(dst, def reflect.Value) {
	switch dst.Kind() {
	case reflect.Struct:
		for i := 0; i < dst.NumField(); i++ {
			if dst.Field(i).CanSet() {
				fillEmptyStrings(dst.Field(i), def.Field(i))
			}
		}
	case reflect.String:
		if dst.String() == "" {
			dst.SetString(def.String())
		}
	}
}

type TenantConfig struct {
	UserId            string     `json:"userId"`
	TenantId          string     `json:"tenantId"`
	Domain            string     `json:"domain"`
	TenantName        string     `json:"tenantName"`
	AppName           string     `json:"appName"`
	BaseUrl           string     `json:"baseUrl"`
	Description       string     `json:"description"`
	Language          string     `json:"language"`
	LogoUrl           string     `json:"logoUrl"`
	FaviconUrl        string     `json:"faviconUrl"`
	Type              string     `json:"type"`
	Address           string     `json:"address"`
	Email             string     `json:"email"`
	Phone             string     `json:"phone"`
	Smtp              any        `json:"smtp"`
	Sms               any        `json:"sms"`
	Whatsapp          any        `json:"whatsapp"`
	OtherConfig       any        `json:"otherConfig,omitempty"`
	LightBackground   string     `json:"lightBackground"`
	DarkBackground    string     `json:"darkBackground"`
	DefaultTheme      string     `json:"defaultTheme"`
	LightTheme        LightTheme `json:"lightTheme"`
	DarkTheme         DartTheme  `json:"darkTheme"`
	HasMembership     bool       `json:"hasMembership"`
	PublicCanRegister bool       `json:"publicCanRegister"`
	WebsiteConfig     any        `json:"websiteConfig"`
}

type Theme struct {
	LightTheme LightTheme `json:"lightTheme"`
	DarkTheme  DartTheme  `json:"darkTheme"`
}

type LightTheme struct {
	Logo       string           `json:"logo"`
	Favicon    string           `json:"favicon"`
	Semantic   SemanticColors   `json:"semantic"`
	Background BackgroundColors `json:"background"`
	Text       TextColors       `json:"text"`
	Border     BorderColors     `json:"border"`
	Outline    OutlineColors    `json:"outline"`
	SVG        SVGColors        `json:"svg"`
	Shadow     Shadows          `json:"shadow"`

	Radius Radius `json:"radius"`
	Fonts  Fonts  `json:"fonts"`
}

type DartTheme struct {
	Logo       string           `json:"logo"`
	Favicon    string           `json:"favicon"`
	Background BackgroundColors `json:"background"`
	Text       TextColors       `json:"text"`
	Border     BorderColors     `json:"border"`
	Outline    OutlineColors    `json:"outline"`
	SVG        SVGColors        `json:"svg"`
	Shadow     Shadows          `json:"shadow"`
}

type SemanticColors struct {
	Brand     HexColor `json:"brand"`
	Primary   HexColor `json:"primary"`
	Secondary HexColor `json:"secondary,omitempty"`
	Success   HexColor `json:"success"`
	Danger    HexColor `json:"danger"`
	Warning   HexColor `json:"warning"`
	Info      HexColor `json:"info"`
}

type BackgroundColors struct {
	App     HexColor `json:"app"`
	Body    HexColor `json:"body"`
	Sidebar HexColor `json:"sidebar"`
	Header  HexColor `json:"header"`
	Card    HexColor `json:"card"`
	Muted   HexColor `json:"muted"`
}

type TextColors struct {
	App           HexColor `json:"app"`
	Body          HexColor `json:"body"`
	BodyActive    HexColor `json:"body-active"`
	Sidebar       HexColor `json:"sidebar"`
	SidebarActive HexColor `json:"sidebar-active"`
	Header        HexColor `json:"header"`
	HeaderActive  HexColor `json:"header-active"`
	Card          HexColor `json:"card"`
	Muted         HexColor `json:"muted"`

	OnBrand     HexColor `json:"on-brand"`
	OnPrimary   HexColor `json:"on-primary"`
	OnSecondary HexColor `json:"on-secondary,omitempty"`
	OnSuccess   HexColor `json:"on-success"`
	OnDanger    HexColor `json:"on-danger"`
	OnWarning   HexColor `json:"on-warning"`
	OnInfo      HexColor `json:"on-info"`
}

type BorderColors struct {
	App     HexColor `json:"app"`
	Body    HexColor `json:"body"`
	Sidebar HexColor `json:"sidebar"`
	Header  HexColor `json:"header"`
	Card    HexColor `json:"card"`
	Muted   HexColor `json:"muted"`
	Default HexColor `json:"default"`
}

type OutlineColors struct {
	App     HexColor `json:"app"`
	Body    HexColor `json:"body"`
	Sidebar HexColor `json:"sidebar"`
	Header  HexColor `json:"header"`
	Card    HexColor `json:"card"`
	Muted   HexColor `json:"muted"`
}

type SVGColors struct {
	Fill   HexColor `json:"fill"`
	Stroke HexColor `json:"stroke"`
}

type Radius struct {
	SM string `json:"sm"`
	MD string `json:"md"`
	LG string `json:"lg"`
}

type Shadows struct {
	SM string `json:"sm"`
	MD string `json:"md"`
}

type Fonts struct {
	Sans  string `json:"sans"`
	Serif string `json:"serif"`
	Mono  string `json:"mono"`
}
