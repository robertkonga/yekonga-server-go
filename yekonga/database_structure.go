package yekonga

import (
	"strings"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/helper"
)

type DatabaseStructure = map[string]map[string]CollectionFieldConfig

// DatabaseStructureConnection is a group of collections stored in their own
// database. Without a DatabaseSetting (no kind) the collections use the
// config's database connection; DatabaseName, when set, picks another
// database on that connection (or overrides DatabaseSetting's).
type DatabaseStructureConnection struct {
	DatabaseName    string                  `json:"databaseName,omitempty"`
	DatabaseSetting config.DatabaseSettings `json:"databaseSetting"`
	Collections     DatabaseStructure       `json:"collections"`
}

// MultipleDatabaseStructure spreads the data models over several databases.
// With config databasePrefix set, the built-in collections (authorizations,
// tenants, payments, ...) each go to their group's database,
// <databasePrefix>_<group> on the config's connection, unless an entry with
// its own database declares them.
type MultipleDatabaseStructure []DatabaseStructureConnection

// isDefault reports whether the entry uses the config's database as is.
func (s DatabaseStructureConnection) isDefault() bool {
	return s.DatabaseSetting.Kind == "" && s.DatabaseName == ""
}

// settings is the database the entry's collections use.
func (s DatabaseStructureConnection) settings(defaults config.DatabaseSettings) config.DatabaseSettings {
	settings := defaults
	if s.DatabaseSetting.Kind != "" {
		settings = s.DatabaseSetting
	}

	if s.DatabaseName != "" {
		settings.DatabaseName = s.DatabaseName
	}

	return settings
}

// groupDatabaseName is the database of a group of built-in collections.
func groupDatabaseName(cfg *config.YekongaConfig, group string) string {
	return cfg.DatabasePrefix + "_" + group
}

// combine merges every entry's collections with the built-in ones, as
// NewDatabaseStructure does for a single structure, and returns the database
// settings of each collection not on the config's database, keyed as in the
// result. With config databasePrefix set the built-in collections go to their
// group's database; an entry with its own database still takes the ones it
// declares.
func (structures MultipleDatabaseStructure) combine(cfg *config.YekongaConfig) (*DatabaseStructure, map[string]config.DatabaseSettings) {
	extra := DatabaseStructure{}
	entrySettings := map[string]config.DatabaseSettings{}

	for _, entry := range structures {
		for k, v := range entry.Collections {
			k = helper.ToCamelCase(helper.Pluralize(k))
			if existing, ok := extra[k]; ok {
				mergeExtraCollectionFields(existing, v)
			} else {
				extra[k] = databaseCollectionFieldConfigsFromMap(v)
			}

			if !entry.isDefault() {
				entrySettings[k] = entry.settings(cfg.Database)
			}
		}
	}

	structure, groups := newDatabaseStructure(extra, cfg, false)
	settings := map[string]config.DatabaseSettings{}

	if cfg.DatabasePrefix != "" {
		for k, group := range groups {
			groupSettings := cfg.Database
			groupSettings.DatabaseName = groupDatabaseName(cfg, group)
			settings[k] = groupSettings
		}
	}

	for k, s := range entrySettings {
		settings[k] = s
	}

	return structure, settings
}

type CollectionFieldConfig struct {
	PrimaryKey   bool                            `json:"primaryKey,omitempty"`
	Name         string                          `json:"name,omitempty"`
	Kind         string                          `json:"type"`
	DefaultValue interface{}                     `json:"default,omitempty"`
	Required     bool                            `json:"required,omitempty"`
	Protected    bool                            `json:"protected,omitempty"`
	Index        bool                            `json:"index,omitempty"`  // create a database index on this field
	Unique       bool                            `json:"unique,omitempty"` // create a unique index on this field
	Options      []string                        `json:"options,omitempty"`
	ForeignKey   CollectionFieldConfigForeignKey `json:"foreignKey,omitempty"`
}

type CollectionFieldConfigForeignKey struct {
	Model string `json:"model,omitempty"`
	Key   string `json:"key,omitempty"`
}

// databaseCollectionFieldConfigFromMap builds a CollectionFieldConfig from the raw
// map[string]interface{} shape used by the external database structure JSON file
// (e.g. {"type": "String", "default": nil, "required": false, "foreignKey": "Tenant.id"}).
func databaseCollectionFieldConfigFromMap(fieldData interface{}) CollectionFieldConfig {
	result := CollectionFieldConfig{}
	field := map[string]interface{}{}

	var relation interface{}
	var relationOk bool

	if v, ok := fieldData.(CollectionFieldConfig); ok {
		result = v
	} else if helper.IsMap(fieldData) {
		field = helper.ToMap[interface{}](fieldData)
		if v, ok := field["type"]; ok {
			if vi, oki := v.(string); oki {
				result.Kind = vi
			}
		}

		if v, ok := field["default"]; ok {
			result.DefaultValue = v
		} else if v, ok := field["defaultValue"]; ok {
			result.DefaultValue = v
		}

		if v, ok := field["required"]; ok {
			if vi, oki := v.(bool); oki {
				result.Required = vi
			}
		}

		if v, ok := field["protected"]; ok {
			if vi, oki := v.(bool); oki {
				result.Protected = vi
			}
		}

		if v, ok := field["index"]; ok {
			if vi, oki := v.(bool); oki {
				result.Index = vi
			}
		}

		if v, ok := field["unique"]; ok {
			if vi, oki := v.(bool); oki {
				result.Unique = vi
			}
		}

		if v, ok := field["primaryKey"]; ok {
			if vi, oki := v.(bool); oki {
				result.PrimaryKey = vi
			}
		}

		if v, ok := field["options"]; ok && helper.IsArray(v) {
			result.Options = helper.ToList[string](v)
		}

		relation, relationOk = field["foreignKey"]
		if !relationOk {
			relation, relationOk = field["relation"]
		}
		if !relationOk {
			relation, relationOk = field["source"]
		}

		if relationOk {
			if vi, oki := relation.(string); oki && helper.IsNotEmpty(vi) {
				ks := strings.SplitN(vi, ".", 2)
				foreignKey := CollectionFieldConfigForeignKey{Model: ks[0]}

				if len(ks) == 2 {
					foreignKey.Key = ks[1]
				} else {
					foreignKey.Key = "id"
				}

				result.ForeignKey = foreignKey
			}
		}
	}

	return result
}
