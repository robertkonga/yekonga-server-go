package yekonga

import (
	"sort"
	"strings"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/helper"
)

type DataModelFieldType string

const (
	DataModelID     DataModelFieldType = "id"
	DataModelString DataModelFieldType = "string"
	DataModelNumber DataModelFieldType = "number"
	DataModelFloat  DataModelFieldType = "float"
	DataModelDate   DataModelFieldType = "date"
	DataModelBool   DataModelFieldType = "bool"
	DataModelObject DataModelFieldType = "object"
	DataModelArray  DataModelFieldType = "array"
	DataModelFile   DataModelFieldType = "file"
)

// Define custom middleware keys
type InputAction string

const (
	CreateInputAction InputAction = "create"
	UpdateInputAction InputAction = "update"
	ImportInputAction InputAction = "import"
	ActionInputAction InputAction = "action"
)

type DataModelFieldOptions struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

type DataModelField struct {
	PrimaryKey   bool
	Name         string
	Kind         DataModelFieldType
	Required     bool
	Protected    bool
	DefaultValue interface{}
	ForeignKey   DataModelFieldForeignKey
	Options      []DataModelFieldOptions
	ID           bool
}

type DataModelFieldForeignKey struct {
	Model      *DataModel
	ModelName  string
	PrimaryKey string
	ForeignKey string
}

type DataModel struct {
	App            *YekongaData
	Config         *config.YekongaConfig
	DBConnect      *DatabaseConnections
	Name           string
	Class          string
	Collection     string
	Variable       string
	VariableSingle string
	VariablePlural string
	ForeignKey     string
	PrimaryKey     string
	PrimaryName    string
	Required       []string
	Protected      []string
	DateFields     []string
	OptionFields   []string
	FileFields     []string
	ValidFields    []string
	BooleanFields  []string
	NumberFields   []string
	FloatFields    []string
	ParentKeys     []string
	RelativeKeys   []string
	Fields         map[string]DataModelField
	ParentFields   map[string]DataModelFieldForeignKey
	ChildrenFields map[string]DataModelFieldForeignKey
	DatabaseType   config.DatabaseType
}

func NewSystemModels(config *config.YekongaConfig, database *DatabaseStructureType) map[string]*DataModel {
	var models map[string]*DataModel = map[string]*DataModel{}

	for k, v := range *database {
		model := newDataModel(config, k, v)
		models[model.Name] = model
	}

	for _, m := range models {
		for _, v := range m.ParentKeys {
			parentForeign := m.Fields[v].ForeignKey
			parentName := helper.GetParentRelativeName(
				parentForeign.ModelName,
				parentForeign.PrimaryKey,
				parentForeign.ForeignKey,
			)
			parentForeign.Model = models[parentForeign.ModelName]
			m.ParentFields[parentName] = parentForeign

			// --------------------------------

			childForeign := m.Fields[v].ForeignKey
			childName := helper.GetChildRelativeName(
				childForeign.ModelName,
				m.Name,
				childForeign.PrimaryKey,
				childForeign.ForeignKey,
			)
			childForeign.Model = models[m.Name]
			childForeign.ModelName = childForeign.Model.Name

			models[parentForeign.ModelName].ChildrenFields[childName] = childForeign
		}
	}

	return models
}

func SetSystemModelDBconnection(app *YekongaData, systemModels *map[string]*DataModel) {
	for _, m := range *systemModels {
		m.App = app
		m.DBConnect = app.dbConnect
	}
}

func SetDataGroups(models map[string]*DataModel) map[string]ResolverChartGroupData {
	values := make(map[string]ResolverChartGroupData)

	for _, v := range models {
		collection := v.Collection
		className := v.Name
		primaryKey := helper.ToVariable(helper.Singularize(collection) + "_id")
		primaryName := ""
		fields := []string{}

		for k := range v.Fields {
			fields = append(fields, k)

			if helper.IsEmpty(primaryName) {
				primaryName = k
			} else if k == "title" || k == "name" || k == "label" {
				primaryName = k
				break
			}
		}

		values[primaryKey] = ResolverChartGroupData{
			Collection:  collection,
			ClassName:   className,
			PrimaryKey:  primaryKey,
			PrimaryName: primaryName,
			Fields:      fields,
		}
	}

	return values
}

func newDataModel(config *config.YekongaConfig, collection string, fields map[string]map[string]interface{}) *DataModel {
	model := DataModel{
		Config:       config,
		DatabaseType: config.Database.Kind,
	}

	model.initialize(collection, fields)

	return &model
}

func (m *DataModel) initialize(collection string, fields map[string]map[string]interface{}) {
	count := len(fields)

	m.Name = helper.ToCamelCase(helper.Singularize(collection))
	m.Class = helper.ToCamelCase(collection)
	m.Collection = helper.ToUnderscore(helper.Pluralize(collection))
	m.Variable = helper.ToVariable(collection)
	m.PrimaryKey = "_id"
	m.PrimaryName = ""
	m.VariableSingle = helper.ToVariable(helper.Singularize(collection))
	m.VariablePlural = helper.ToVariable(helper.Pluralize(collection))
	m.Fields = make(map[string]DataModelField)
	m.DateFields = make([]string, 0, count)
	m.OptionFields = make([]string, 0, count)
	m.FileFields = make([]string, 0, count)
	m.ValidFields = make([]string, 0, count)
	m.ParentKeys = make([]string, 0, count)
	m.Required = make([]string, 0, count)
	m.Protected = make([]string, 0, count)
	m.ParentFields = make(map[string]DataModelFieldForeignKey)
	m.ChildrenFields = make(map[string]DataModelFieldForeignKey)

	hasPrimaryName := false

	for k, v := range fields {
		if k == "id" {
			continue
		}

		field := *m.getDataModelField(k, v)
		keyNames := []string{"name", "title", "label"}

		if helper.Contains(keyNames, field.Name) {
			m.PrimaryName = field.Name
			hasPrimaryName = true
		} else if hasPrimaryName &&
			strings.Contains(helper.ToUnderscore(field.Name), "name") &&
			strings.Contains(helper.ToUnderscore(field.Name), "title") {
			m.PrimaryName = field.Name
		} else if helper.IsEmpty(m.PrimaryName) && field.Name != "_id" {
			m.PrimaryName = field.Name
		}

		m.Fields[k] = field
		m.ValidFields = append(m.ValidFields, k)

		if field.Required {
			m.Required = append(m.Required, k)
		}
		if field.Protected {
			m.Protected = append(m.Protected, k)
		}

		if field.Kind == DataModelDate {
			m.DateFields = append(m.DateFields, k)
		}
		if field.Kind == DataModelFile {
			m.FileFields = append(m.FileFields, k)
		}
		if field.Kind == DataModelBool {
			m.BooleanFields = append(m.BooleanFields, k)
		}
		if field.Kind == DataModelNumber {
			m.NumberFields = append(m.NumberFields, k)
		}
		if field.Kind == DataModelFloat {
			m.FloatFields = append(m.FloatFields, k)
		}
		if len(field.Options) > 0 {
			m.OptionFields = append(m.OptionFields, k)
		}

		if helper.IsNotEmpty(field.ForeignKey.ModelName) {
			m.ParentKeys = append(m.ParentKeys, k)
			m.RelativeKeys = append(m.RelativeKeys, k)
		}
	}

	if !helper.Contains(m.ValidFields, "id") {
		k := "id"
		field := *m.getDataModelField(k, map[string]interface{}{"type": "ID", "default": nil, "required": false})

		m.Fields[k] = field
		m.ValidFields = append(m.ValidFields, k)
	}

	sort.Strings(m.ValidFields)
}

func (m *DataModel) getDataModelField(name string, field map[string]interface{}) *DataModelField {
	var primaryKey bool
	var kind DataModelFieldType = DataModelString
	var required bool
	var protected bool
	var defaultValue interface{}
	var foreignKey DataModelFieldForeignKey
	var options = make([]DataModelFieldOptions, 0, 4)

	if v, ok := field["type"]; ok {
		if vi, oki := v.(string); oki {
			vi = strings.ToLower(vi)
			// logger.Error("vi", name, "->", vi)

			switch vi {
			case "id":
				kind = DataModelID
			case "date", "time", "datetime", "timestamp":
				kind = DataModelDate
			case "bool", "boolean":
				defaultValue = false
				kind = DataModelBool
			case "float":
				defaultValue = 0
				kind = DataModelFloat
			case "int", "number":
				defaultValue = 0
				kind = DataModelNumber
			case "text", "string":
				kind = DataModelString
			case "any":
				kind = DataModelObject
			case "object":
				kind = DataModelObject
			case "array":
				defaultValue = []interface{}{}
				kind = DataModelArray
			case "url":
				kind = DataModelFile
			case "file":
				kind = DataModelFile
			}
		}
	}

	if v, ok := field["required"]; ok {
		if vi, oki := v.(bool); oki {
			required = vi
		}
	}

	if v, ok := field["protected"]; ok {
		if vi, oki := v.(bool); oki {
			protected = vi
		}
	}

	if v, ok := field["primaryKey"]; ok {
		if vi, oki := v.(bool); oki {
			primaryKey = vi
		}
	}

	if v, ok := field["defaultValue"]; ok {
		defaultValue = v
	}

	if v, ok := field["options"]; ok {
		if helper.IsArray(v) {
			vi := helper.ToList[interface{}](v)

			if len(vi) > 0 {
				for _, vii := range vi {
					options = append(options, DataModelFieldOptions{
						Value: helper.ToString(vii),
						Label: helper.ToTitle(helper.ToString(vii)),
					})
				}
			}
		} else if vi, oki := v.(map[string]string); oki {
			if len(vi) > 0 {
				for kii, vii := range vi {
					options = append(options, DataModelFieldOptions{
						Value: kii,
						Label: vii,
					})
				}
			}
		}
	}

	if v, ok := field["foreignKey"]; ok {
		if vi, oki := v.(string); oki && helper.IsNotEmpty(vi) {
			ks := strings.Split(vi, ".")
			size := len(ks)

			parentCollection := ""
			parentKey := "_id"

			switch size {
			case 2:
				parentCollection = ks[0]
				parentKey = ks[1]
			case 1:
				parentCollection = ks[0]
			}

			if parentKey == "id" {
				parentKey = "_id"
			}

			/// Remove or comment the line below in production
			// parentKey = helper.ToVariable(parentCollection + "_id")

			if helper.IsNotEmpty(parentCollection) {
				foreignKey = DataModelFieldForeignKey{
					ModelName:  helper.ToCamelCase(parentCollection),
					PrimaryKey: parentKey,
					ForeignKey: name,
				}
			}
		}
	}

	return &DataModelField{
		PrimaryKey:   primaryKey,
		Name:         name,
		Kind:         kind,
		Required:     required,
		Protected:    protected,
		DefaultValue: defaultValue,
		ForeignKey:   foreignKey,
		Options:      options,
	}
}

func (m *DataModel) Query() *DataModelQuery {
	return &DataModelQuery{
		Model: m,
		QueryContext: QueryContext{
			Params: make(map[string]interface{}),
		},
	}
}
