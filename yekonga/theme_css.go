package yekonga

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/helper"
)

// themeShades are the steps of each semantic color scale; 500 is the base
// color, lower steps mix toward white and higher steps toward black.
var themeShades = []int{50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950}

// requestTheme returns the current tenant's theme, or the default theme
// when there is no tenant.
func (y *YekongaData) requestTheme(req *Request) config.Theme {
	if tenantConfig := y.GetTenantConfig(req); tenantConfig != nil {
		return config.Theme{
			LightTheme: tenantConfig.LightTheme,
			DarkTheme:  tenantConfig.DarkTheme,
		}
	}

	return config.DefaultTheme()
}

// ThemeCSS renders theme as the --yekonga-* CSS variables served by /theme.css
// when no SetCustomCSS handler provides them.
func ThemeCSS(theme config.Theme, domain string) string {
	css := ""

	css += `
:root {
	/* =========================================================
	BRAND & SEMANTIC COLORS (50–900)
	RGB values only → Tailwind adds alpha
	========================================================= */

	/* Logo */
	--yekonga-logo: url("` + helper.GetBaseUrl(theme.LightTheme.Logo, domain) + `");
	--yekonga-logo-light: url("` + helper.GetBaseUrl(theme.LightTheme.Logo, domain) + `");
	--yekonga-logo-dark: url("` + helper.GetBaseUrl(theme.DarkTheme.Logo, domain) + `");

	/* Brand */ ` + semanticColors("brand", string(theme.LightTheme.Semantic.Brand)) + `

	/* Primary */ ` + semanticColors("primary", string(theme.LightTheme.Semantic.Primary)) + `

	/* Secondary */ ` + semanticColors("secondary", string(theme.LightTheme.Semantic.Secondary)) + `

	/* Success */ ` + semanticColors("success", string(theme.LightTheme.Semantic.Success)) + `

	/* Danger */ ` + semanticColors("danger", string(theme.LightTheme.Semantic.Danger)) + `

	/* Warning */ ` + semanticColors("warning", string(theme.LightTheme.Semantic.Warning)) + `

	/* Info */ ` + semanticColors("info", string(theme.LightTheme.Semantic.Info)) + `

	/* Gray */ ` + semanticColors("gray", "#808080") + `

	/* Semantic On-Colors */
	--yekonga-on-brand: ` + GenerateRGB(string(theme.LightTheme.Text.OnBrand)) + `;
	--yekonga-on-primary: ` + GenerateRGB(string(theme.LightTheme.Text.OnPrimary)) + `;
	--yekonga-on-secondary: ` + GenerateRGB(string(theme.LightTheme.Text.OnSecondary)) + `;
	--yekonga-on-success: ` + GenerateRGB(string(theme.LightTheme.Text.OnSuccess)) + `;
	--yekonga-on-danger: ` + GenerateRGB(string(theme.LightTheme.Text.OnDanger)) + `;
	--yekonga-on-warning: ` + GenerateRGB(string(theme.LightTheme.Text.OnWarning)) + `;
	--yekonga-on-info: ` + GenerateRGB(string(theme.LightTheme.Text.OnInfo)) + `;

	--yekonga-on-brand-hex: ` + string(theme.LightTheme.Text.OnBrand) + `;
	--yekonga-on-primary-hex: ` + string(theme.LightTheme.Text.OnPrimary) + `;
	--yekonga-on-secondary-hex: ` + string(theme.LightTheme.Text.OnSecondary) + `;
	--yekonga-on-success-hex: ` + string(theme.LightTheme.Text.OnSuccess) + `;
	--yekonga-on-danger-hex: ` + string(theme.LightTheme.Text.OnDanger) + `;
	--yekonga-on-warning-hex: ` + string(theme.LightTheme.Text.OnWarning) + `;
	--yekonga-on-info-hex: ` + string(theme.LightTheme.Text.OnInfo) + `;

	/* Semantic Hex Colors */
	--yekonga-brand-hex: ` + string(theme.LightTheme.Semantic.Brand) + `;
	--yekonga-primary-hex: ` + string(theme.LightTheme.Semantic.Primary) + `;
	--yekonga-secondary-hex: ` + string(theme.LightTheme.Semantic.Secondary) + `;
	--yekonga-success-hex: ` + string(theme.LightTheme.Semantic.Success) + `;
	--yekonga-danger-hex: ` + string(theme.LightTheme.Semantic.Danger) + `;
	--yekonga-warning-hex: ` + string(theme.LightTheme.Semantic.Warning) + `;
	--yekonga-info-hex: ` + string(theme.LightTheme.Semantic.Info) + `;

	--yekonga-app-text-hex: ` + string(theme.LightTheme.Text.App) + `;
	--yekonga-body-text-hex: ` + string(theme.LightTheme.Text.App) + `;
	--yekonga-sidebar-text-hex: ` + string(theme.LightTheme.Text.Sidebar) + `;
	--yekonga-header-text-hex: ` + string(theme.LightTheme.Text.Header) + `;
	--yekonga-card-text-hex: ` + string(theme.LightTheme.Text.Card) + `;
	--yekonga-muted-text-hex: ` + string(theme.LightTheme.Text.Muted) + `;

	--yekonga-app-bg-hex: ` + string(theme.LightTheme.Background.App) + `;
	--yekonga-body-bg-hex: ` + string(theme.LightTheme.Background.Body) + `;
	--yekonga-sidebar-bg-hex: ` + string(theme.LightTheme.Background.Sidebar) + `;
	--yekonga-header-bg-hex: ` + string(theme.LightTheme.Background.Header) + `;
	--yekonga-card-bg-hex: ` + string(theme.LightTheme.Background.Card) + `;
	--yekonga-muted-bg-hex: ` + string(theme.LightTheme.Background.Muted) + `; 


	/* =========================================================
	BACKGROUNDS
	========================================================= */
	--yekonga-app-bg: ` + GenerateRGB(string(theme.LightTheme.Background.App)) + `;
	--yekonga-body-bg: ` + GenerateRGB(string(theme.LightTheme.Background.Body)) + `;
	--yekonga-sidebar-bg: ` + GenerateRGB(string(theme.LightTheme.Background.Sidebar)) + `;
	--yekonga-header-bg: ` + GenerateRGB(string(theme.LightTheme.Background.Header)) + `;
	--yekonga-card-bg: ` + GenerateRGB(string(theme.LightTheme.Background.Card)) + `;
	--yekonga-muted-bg: ` + GenerateRGB(string(theme.LightTheme.Background.Muted)) + `;


	/* =========================================================
	TEXT
	========================================================= */
	--yekonga-app-text: ` + GenerateRGB(string(theme.LightTheme.Text.App)) + `;
	--yekonga-body-text: ` + GenerateRGB(string(theme.LightTheme.Text.Body)) + `;
	--yekonga-sidebar-text: ` + GenerateRGB(string(theme.LightTheme.Text.Sidebar)) + `;
	--yekonga-header-text: ` + GenerateRGB(string(theme.LightTheme.Text.Header)) + `;
	--yekonga-card-text: ` + GenerateRGB(string(theme.LightTheme.Text.Card)) + `;
	--yekonga-muted-text: ` + GenerateRGB(string(theme.LightTheme.Text.Muted)) + `;


	/* =========================================================
	BORDERS / OUTLINES / RINGS
	========================================================= */
	/* Borders */
	--yekonga-app-border: ` + GenerateRGB(string(theme.LightTheme.Border.App)) + `;
	--yekonga-body-border: ` + GenerateRGB(string(theme.LightTheme.Border.Body)) + `;
	--yekonga-sidebar-border: ` + GenerateRGB(string(theme.LightTheme.Border.Sidebar)) + `;
	--yekonga-header-border: ` + GenerateRGB(string(theme.LightTheme.Border.Header)) + `;
	--yekonga-card-border: ` + GenerateRGB(string(theme.LightTheme.Border.Card)) + `;
	--yekonga-muted-border: ` + GenerateRGB(string(theme.LightTheme.Border.Muted)) + `;

	/* Outlines */
	--yekonga-app-outline: ` + GenerateRGB(string(theme.LightTheme.Outline.App)) + `;
	--yekonga-body-outline: ` + GenerateRGB(string(theme.LightTheme.Outline.Body)) + `;
	--yekonga-sidebar-outline: ` + GenerateRGB(string(theme.LightTheme.Outline.Sidebar)) + `;
	--yekonga-header-outline: ` + GenerateRGB(string(theme.LightTheme.Outline.Header)) + `;
	--yekonga-card-outline: ` + GenerateRGB(string(theme.LightTheme.Outline.Card)) + `;
	--yekonga-muted-outline: ` + GenerateRGB(string(theme.LightTheme.Outline.Muted)) + `;

	/* Default Border Color */
	--yekonga-border-color: ` + GenerateRGB(string(theme.LightTheme.Border.Default)) + `;


	/* =========================================================
	SVG
	========================================================= */
	--yekonga-fill-color: ` + GenerateRGB(string(theme.LightTheme.SVG.Fill)) + `;
	--yekonga-stroke-color: ` + GenerateRGB(string(theme.LightTheme.SVG.Stroke)) + `;


	/* =========================================================
	RADIUS
	========================================================= */
	--yekonga-radius-sm: ` + string(theme.LightTheme.Radius.SM) + `;
	--yekonga-radius-md: ` + string(theme.LightTheme.Radius.MD) + `;
	--yekonga-radius-lg: ` + string(theme.LightTheme.Radius.LG) + `;


	/* =========================================================
	SHADOWS
	========================================================= */
	--yekonga-shadow-sm: ` + string(theme.LightTheme.Shadow.SM) + `;
	--yekonga-shadow-md: ` + string(theme.LightTheme.Shadow.MD) + `;


	/* =========================================================
	FONTS
	========================================================= */
	--yekonga-font-sans: Inter;
	--yekonga-font-serif: Georgia;
	--yekonga-font-mono: "JetBrains Mono";
}
`

	css += `
.dark {
	/* =========================================================
	BRAND & SEMANTIC COLORS (50–900)
	RGB values only → Tailwind adds alpha
	========================================================= */

	/* Logo */
	--yekonga-logo: url("` + helper.GetBaseUrl(theme.DarkTheme.Logo, domain) + `");

	/* Semantic Hex Colors */
	--yekonga-app-text-hex: ` + string(theme.DarkTheme.Text.App) + `;
	--yekonga-body-text-hex: ` + string(theme.DarkTheme.Text.App) + `;
	--yekonga-sidebar-text-hex: ` + string(theme.DarkTheme.Text.Sidebar) + `;
	--yekonga-header-text-hex: ` + string(theme.DarkTheme.Text.Header) + `;
	--yekonga-card-text-hex: ` + string(theme.DarkTheme.Text.Card) + `;
	--yekonga-muted-text-hex: ` + string(theme.DarkTheme.Text.Muted) + `;

	--yekonga-app-bg-hex: ` + string(theme.DarkTheme.Background.App) + `;
	--yekonga-body-bg-hex: ` + string(theme.DarkTheme.Background.Body) + `;
	--yekonga-sidebar-bg-hex: ` + string(theme.DarkTheme.Background.Sidebar) + `;
	--yekonga-header-bg-hex: ` + string(theme.DarkTheme.Background.Header) + `;
	--yekonga-card-bg-hex: ` + string(theme.DarkTheme.Background.Card) + `;
	--yekonga-muted-bg-hex: ` + string(theme.DarkTheme.Background.Muted) + `;


	/* =========================================================
	BACKGROUNDS
	========================================================= */
	--yekonga-app-bg: ` + GenerateRGB(string(theme.DarkTheme.Background.App)) + `;
	--yekonga-body-bg: ` + GenerateRGB(string(theme.DarkTheme.Background.Body)) + `;
	--yekonga-sidebar-bg: ` + GenerateRGB(string(theme.DarkTheme.Background.Sidebar)) + `;
	--yekonga-header-bg: ` + GenerateRGB(string(theme.DarkTheme.Background.Header)) + `;
	--yekonga-card-bg: ` + GenerateRGB(string(theme.DarkTheme.Background.Card)) + `;
	--yekonga-muted-bg: ` + GenerateRGB(string(theme.DarkTheme.Background.Muted)) + `;


	/* =========================================================
	TEXT
	========================================================= */
	--yekonga-app-text: ` + GenerateRGB(string(theme.DarkTheme.Text.App)) + `;
	--yekonga-body-text: ` + GenerateRGB(string(theme.DarkTheme.Text.Body)) + `;
	--yekonga-sidebar-text: ` + GenerateRGB(string(theme.DarkTheme.Text.Sidebar)) + `;
	--yekonga-header-text: ` + GenerateRGB(string(theme.DarkTheme.Text.Header)) + `;
	--yekonga-card-text: ` + GenerateRGB(string(theme.DarkTheme.Text.Card)) + `;
	--yekonga-muted-text: ` + GenerateRGB(string(theme.DarkTheme.Text.Muted)) + `;


	/* =========================================================
	BORDERS / OUTLINES / RINGS
	========================================================= */
	/* Borders */
	--yekonga-app-border: ` + GenerateRGB(string(theme.DarkTheme.Border.App)) + `;
	--yekonga-body-border: ` + GenerateRGB(string(theme.DarkTheme.Border.Body)) + `;
	--yekonga-sidebar-border: ` + GenerateRGB(string(theme.DarkTheme.Border.Sidebar)) + `;
	--yekonga-header-border: ` + GenerateRGB(string(theme.DarkTheme.Border.Header)) + `;
	--yekonga-card-border: ` + GenerateRGB(string(theme.DarkTheme.Border.Card)) + `;
	--yekonga-muted-border: ` + GenerateRGB(string(theme.DarkTheme.Border.Muted)) + `;

	/* Outlines */
	--yekonga-app-outline: ` + GenerateRGB(string(theme.DarkTheme.Outline.App)) + `;
	--yekonga-body-outline: ` + GenerateRGB(string(theme.DarkTheme.Outline.Body)) + `;
	--yekonga-sidebar-outline: ` + GenerateRGB(string(theme.DarkTheme.Outline.Sidebar)) + `;
	--yekonga-header-outline: ` + GenerateRGB(string(theme.DarkTheme.Outline.Header)) + `;
	--yekonga-card-outline: ` + GenerateRGB(string(theme.DarkTheme.Outline.Card)) + `;
	--yekonga-muted-outline: ` + GenerateRGB(string(theme.DarkTheme.Outline.Muted)) + `;

	/* Default Border Color */
	--yekonga-border-color: ` + GenerateRGB(string(theme.DarkTheme.Border.Default)) + `;


	/* =========================================================
	SVG
	========================================================= */
	--yekonga-fill-color: ` + GenerateRGB(string(theme.DarkTheme.SVG.Fill)) + `;
	--yekonga-stroke-color: ` + GenerateRGB(string(theme.DarkTheme.SVG.Stroke)) + `;
}`

	return css
}

func semanticColors(name, color string) string {
	str := ""

	for _, shade := range themeShades {
		str += `
	--yekonga-` + name + `-` + strconv.Itoa(shade) + `: ` + GenerateRGB(shadeHex(color, shade)) + `;`
	}

	return str
}

// parseHex reads #rgb or #rrggbb.
func parseHex(hex string) (r, g, b int64, ok bool) {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return 0, 0, 0, false
	}

	n, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		return 0, 0, 0, false
	}

	return n >> 16 & 0xFF, n >> 8 & 0xFF, n & 0xFF, true
}

// GenerateRGB converts a hex color to "r g b", or "" when it is not one.
func GenerateRGB(hex string) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return ""
	}

	return fmt.Sprintf("%d %d %d", r, g, b)
}

// shadeHex mixes hex toward white (shade < 500) or black (shade > 500).
func shadeHex(hex string, shade int) string {
	r, g, b, ok := parseHex(hex)
	if !ok || shade == 500 {
		return hex
	}

	mix := func(c int64) int64 {
		if shade < 500 {
			return c + (255-c)*int64(500-shade)/500
		}
		return c * int64(1000-shade) / 500
	}

	return fmt.Sprintf("#%02X%02X%02X", mix(r), mix(g), mix(b))
}
