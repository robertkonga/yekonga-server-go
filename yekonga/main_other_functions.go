package yekonga

import (
	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
)

func (y *YekongaData) GetTenantConfig(req *Request) *config.TenantConfig {
	tenantModelName := "Tenant"
	client := req.Client()
	host := client.OriginDomain()

	var tenant any

	if req.App.Config.HasTenant {
		tenant = req.App.ModelQuery(tenantModelName).SkipBeforeCommit().FindOne(datatype.DataMap{
			"domain": host,
		})

		if helper.IsEmpty(tenant) {
			tenant = req.App.ModelQuery(tenantModelName).SkipBeforeCommit().FindOne(datatype.DataMap{
				"subdomain": host,
			})
		}
	}

	if helper.IsNotEmpty(tenant) {
		tenantConfigModelName := "TenantConfig"
		tenantId := helper.GetValueOf(tenant, "id")

		tenantConfig := req.App.ModelQuery(tenantConfigModelName).SkipTenant().SkipBeforeCommit().FindOne(datatype.DataMap{
			"tenantId": tenantId,
		})
		// console.Log("tenantConfig", tenantId, tenantConfig)

		if helper.IsEmpty(tenantConfig) {
			tenantConfig = &datatype.DataMap{
				"tenantId": tenantId,
			}
		}

		if helper.IsNotEmpty(tenantConfig) {
			port := client.Port

			if helper.IsEmpty(port) || port == "80" || port == "443" {
				port = ""
			} else {
				port = ":" + port
			}

			lightTheme := helper.GetValueOfMap(tenantConfig, "lightTheme")
			darkTheme := helper.GetValueOfMap(tenantConfig, "darkTheme")
			baseUrl := client.Proto + "://" + host + port
			logoStr := helper.GetValueOfString(lightTheme, "logo")
			logoUrl := helper.GetBaseUrl(logoStr, host)
			faviconStr := helper.GetValueOfString(lightTheme, "favicon")
			faviconUrl := helper.GetBaseUrl(faviconStr, host)

			data := datatype.DataMap{
				"domain":            host,
				"tenantId":          tenantId,
				"appName":           y.Config.AppName,
				"baseUrl":           baseUrl,
				"userId":            helper.GetValueOf(tenant, "userId"),
				"tenantName":        helper.GetValueOf(tenant, "name"),
				"description":       helper.GetValueOf(tenant, "description"),
				"language":          helper.GetValueOf(tenant, "language"),
				"type":              helper.GetValueOf(tenant, "type"),
				"address":           helper.GetValueOf(tenant, "address"),
				"email":             helper.GetValueOf(tenant, "email"),
				"phone":             helper.GetValueOf(tenant, "phone"),
				"logoUrl":           logoUrl,
				"faviconUrl":        faviconUrl,
				"smtp":              helper.GetValueOf(tenantConfig, "smtp"),
				"sms":               helper.GetValueOf(tenantConfig, "sms"),
				"whatsapp":          helper.GetValueOf(tenantConfig, "whatsapp"),
				"lightTheme":        lightTheme,
				"darkTheme":         darkTheme,
				"hasMembership":     helper.GetValueOf(tenantConfig, "hasMembership"),
				"publicCanRegister": helper.GetValueOf(tenantConfig, "publicCanRegister"),
			}
			// console.Log("TenantConfig.data", data)
			// console.Log("TenantConfig.tenantConfig", tenantConfig)

			config, err := helper.ConvertTo[config.TenantConfig](data)
			if err != nil {
				return nil
			}
			return &config
		}
	}

	return nil
}
