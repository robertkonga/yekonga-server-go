package yekonga

import (
	"context"
	"encoding/json"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/console"
	"github.com/robertkonga/yekonga-server-go/plugins/graphql"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/bson"
)

type FilterOperator string

// Filter operation constants
const (
	FilterEqualTo                 FilterOperator = "equalTo"
	FilterNotEqualTo              FilterOperator = "notEqualTo"
	FilterLessThan                FilterOperator = "lessThan"
	FilterNotLessThan             FilterOperator = "notLessThan"
	FilterLessThanOrEqualTo       FilterOperator = "lessThanOrEqualTo"
	FilterNotLessThanOrEqualTo    FilterOperator = "notLessThanOrEqualTo"
	FilterGreaterThan             FilterOperator = "greaterThan"
	FilterNotGreaterThan          FilterOperator = "notGreaterThan"
	FilterGreaterThanOrEqualTo    FilterOperator = "greaterThanOrEqualTo"
	FilterNotGreaterThanOrEqualTo FilterOperator = "notGreaterThanOrEqualTo"
	FilterMatchesRegex            FilterOperator = "matchesRegex"
	FilterOptions                 FilterOperator = "options"
)

type QueryContext struct {
	Data       interface{}
	Input      interface{}
	Filters    *datatype.DataMap
	Parent     interface{}
	Params     map[string]interface{}
	AccessRole string
	Route      string
}

type DataModelQuery struct {
	Model          *DataModel
	RequestContext *RequestContext
	QueryContext   QueryContext

	limit      int
	page       int
	skip       int
	where      datatype.DataMap
	orderBy    map[string]string
	selection  []string
	distinct   []string
	groupBy    []string
	groupByRaw map[string]interface{}
}

func NewDataModelQuery(model *DataModel) DataModelQuery {

	return DataModelQuery{
		Model: model,
		limit: 10,
	}
}

func (m *DataModelQuery) NewInstance() *DataModelQuery {
	return &DataModelQuery{
		Model: m.Model,
		QueryContext: QueryContext{
			Params: make(map[string]interface{}),
		},
	}
}

func (m *DataModelQuery) Where(name string, value interface{}) *DataModelQuery {
	var newValue = value
	if m.where == nil {
		m.where = make(datatype.DataMap)
	}

	if helper.Contains(m.Model.RelativeKeys, name) || name == "id" || name == "_id" {
		if name == "id" || name == "_id" {
			name = "_id"
		}

		if v, ok := value.(string); ok {
			switch v {
			case string(NULLValue):
				newValue = nil
			case string(NullValue):
				newValue = nil
			case string(nullValue):
				newValue = nil
			default:
				newValue = helper.ObjectID(v)
			}
		} else if helper.IsArray(value) {
			array := helper.ToList[interface{}](value)
			count := len(array)
			vpi := make([]bson.ObjectID, 0, count)

			for i := 0; i < count; i++ {
				vpi = append(vpi, helper.ObjectID(array[i]))
			}

			newValue = vpi
		} else if v, ok := value.(map[string]interface{}); ok {
			vp := make(map[string]interface{})

			for ki, vi := range v {
				if vii, ok := vi.(string); ok {
					switch vii {
					case string(NULLValue):
						vp[ki] = nil
					case string(NullValue):
						vp[ki] = nil
					case string(nullValue):
						vp[ki] = nil
					default:
						vp[ki] = helper.ObjectID(vii)
					}
				} else if helper.IsArray(vi) {
					array := helper.ToList[interface{}](vi)
					count := len(array)
					vpi := make([]bson.ObjectID, 0, count)

					for i := 0; i < count; i++ {
						vpi = append(vpi, helper.ObjectID(array[i]))
					}

					vp[ki] = vpi
				} else {
					switch vi {
					case string(NULLValue):
						vp[ki] = nil
					case string(NullValue):
						vp[ki] = nil
					case string(nullValue):
						vp[ki] = nil
					default:
						vp[ki] = helper.ObjectID(vi)
					}
				}
			}

			newValue = vp
		}
	}

	if w, ok := m.where[name]; ok {
		w1, ok1 := w.(map[string]interface{})
		w2, ok2 := newValue.(map[string]interface{})

		if ok1 && ok2 {
			for kii, vii := range w2 {
				w1[kii] = vii
			}
		} else {
			m.where[name] = newValue
		}
	} else {
		m.where[name] = newValue
	}

	return m
}

func (m *DataModelQuery) WhereMany(where interface{}) *DataModelQuery {
	if m.where == nil {
		m.where = make(datatype.DataMap)
	}

	if helper.IsMap(where) {
		p := helper.ToMap[interface{}](where)
		for k, v := range p {
			m.Where(k, v)
		}
	}

	return m
}

func (m *DataModelQuery) WhereAll(where interface{}) *DataModelQuery {
	return m.WhereMany(where)
}

func (m *DataModelQuery) Distinct(name string) *DataModelQuery {
	if m.distinct == nil {
		m.distinct = make([]string, 0, 3)
	}

	m.distinct = append(m.distinct, name)

	return m
}

func (m *DataModelQuery) DistinctAll(values []string) *DataModelQuery {
	if m.distinct == nil {
		m.distinct = make([]string, 0, 3)
	}

	m.distinct = append(m.distinct, values...)

	return m
}

func (m *DataModelQuery) OrderBy(name string, value string) *DataModelQuery {
	if m.orderBy == nil {
		m.orderBy = make(map[string]string)
	}

	m.orderBy[name] = value

	return m
}

func (m *DataModelQuery) OrderByAll(values []map[string]string) *DataModelQuery {
	if m.orderBy == nil {
		m.orderBy = make(map[string]string)
	}

	for _, o := range values {
		for k, v := range o {
			m.orderBy[k] = v
		}
	}

	return m
}

func (m *DataModelQuery) GroupBy(name string) *DataModelQuery {
	if m.groupBy == nil {
		m.groupBy = make([]string, 0, 3)
	}

	m.groupBy = append(m.groupBy, name)

	return m
}

func (m *DataModelQuery) GroupByRaw(key string, value interface{}) *DataModelQuery {
	if m.groupByRaw == nil {
		m.groupByRaw = make(map[string]interface{})
	}

	m.groupByRaw[key] = value

	return m
}

func (m *DataModelQuery) Page(value int) *DataModelQuery {
	m.page = value
	m.skip = (m.page - 1) * m.limit

	return m
}

func (m *DataModelQuery) Take(value int) *DataModelQuery {
	m.limit = value

	return m
}

func (m *DataModelQuery) Skip(value int) *DataModelQuery {
	m.skip = value

	return m
}

func (m *DataModelQuery) Create(data datatype.DataMap) interface{} {
	if m.Model.HasTenant && m.RequestContext != nil && m.Model.App.Config.HasTenant {
		tenantId := m.getTenantId()

		if helper.IsNotEmpty(tenantId) {
			data[TenantIDKey] = helper.ObjectID(tenantId)
		}
	}

	triggerBefore := m.runTriggerAction(BeforeCreateTriggerAllAction, data)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if helper.IsMap(triggerBefore) {
		data = helper.ToDataMap(triggerBefore)
	}

	triggerBefore = m.runTriggerAction(BeforeCreateTriggerAction, data)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if helper.IsMap(triggerBefore) {
		data = helper.ToDataMap(triggerBefore)
	}

	result, err := m.collection().create(*(m.formatInputData(data, CreateInputAction)))

	if err != nil {
		return err
	}

	triggerAfter := m.runTriggerAction(AfterCreateTriggerAllAction, result)
	if helper.IsMap(triggerAfter) {
		v := helper.ToDataMap(triggerAfter)
		result = &v
	}

	triggerAfter = m.runTriggerAction(AfterCreateTriggerAction, result)
	if helper.IsMap(triggerAfter) {
		v := helper.ToDataMap(triggerAfter)
		result = &v
	}

	m.Model.App.socketServer.Of("/").Emit("database", datatype.DataMap{
		"action": "create",
		"model":  m.Model.Name,
	}, nil)

	return result
}

func (m *DataModelQuery) Update(data datatype.DataMap, where interface{}) interface{} {
	m.WhereAll(where)
	m.addTenantId()

	triggerBefore := m.runTriggerAction(BeforeUpdateTriggerAllAction, data)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if helper.IsMap(triggerBefore) {
		data = helper.ToDataMap(triggerBefore)
	}

	triggerBefore = m.runTriggerAction(BeforeUpdateTriggerAction, data)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if helper.IsMap(triggerBefore) {
		data = helper.ToDataMap(triggerBefore)
	}

	result, err := m.collection().update(*(m.formatInputData(data, UpdateInputAction)))

	if err != nil {
		return err
	}

	triggerAfter := m.runTriggerAction(AfterUpdateTriggerAllAction, result)
	if helper.IsMap(triggerAfter) {
		v := helper.ToDataMap(triggerAfter)
		result = &v
	}

	triggerAfter = m.runTriggerAction(AfterUpdateTriggerAction, result)
	if helper.IsMap(triggerAfter) {
		v := helper.ToDataMap(triggerAfter)
		result = &v
	}

	m.Model.App.socketServer.Of("/").Emit("database", datatype.DataMap{
		"action": "update",
		"model":  m.Model.Name,
	}, nil)

	return result
}

func (m *DataModelQuery) Import(data []interface{}, uniqueKeys []string) interface{} {
	triggerBefore := m.runTriggerAction(BeforeCreateTriggerAllAction, data)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if v, ok := triggerBefore.([]interface{}); ok {
		data = v
	}

	triggerBefore = m.runTriggerAction(BeforeCreateTriggerAction, data)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if v, ok := triggerBefore.([]interface{}); ok {
		data = v
	}

	uniqueKeys = append(uniqueKeys, "_id")

	message := "FAIL"
	status := false
	deleted := 0
	ignored := 0
	imported := 0
	updated := 0
	afterData := []interface{}{}

	result := map[string]interface{}{}
	formattedCreateData := make([]datatype.DataMap, 0, len(data))
	formattedUpdateData := make([]datatype.DataMap, 0, len(data))
	// inputIds := []string{}

ParentLoop:
	for _, v := range data {
		vi, _ := helper.ConvertTo[map[string]interface{}](v)

		if vi != nil {
			if m.Model.HasTenant && m.RequestContext != nil && m.Model.App.Config.HasTenant {
				tenantId := m.getTenantId()

				if helper.IsNotEmpty(tenantId) {
					vi[TenantIDKey] = helper.ObjectID(tenantId)
				}
			}

			whereData := make(datatype.DataMap)

			for _, key := range uniqueKeys {
				if value, ok := vi[key]; ok {
					if helper.IsNotEmpty(value) {
						whereData[key] = value
					} else {
						ignored++
						continue ParentLoop // Skip if unique key value is empty
					}
				}
			}

			if len(whereData) > 0 {
				existsData := m.NewInstance().FindOne(whereData)

				if existsData != nil && (*existsData) != nil {
					// Update existing data
					vi["_id"] = (*existsData)["_id"]
					formattedUpdateData = append(formattedUpdateData, vi)
				} else {
					// Create new data
					formattedCreateData = append(formattedCreateData, *m.formatInputData(vi, ImportInputAction))
				}
			} else {
				// No unique keys found, treat as new data
				formattedCreateData = append(formattedCreateData, *m.formatInputData(vi, ImportInputAction))
			}
		}
	}

	if len(formattedCreateData) > 0 {
		createData, err := m.collection().createMany(formattedCreateData)

		if err != nil {
			console.Error("DataModelQuery.Import", err.Error())
		} else if createData != nil {
			imported = len(*createData)
			interfaceData := helper.ToList[interface{}](createData)
			afterData = append(afterData, interfaceData...)

			status = true
			message = "SUCCESS"
		}

		triggerAfter := m.runTriggerAction(AfterCreateTriggerAllAction, createData)
		if result, ok := triggerAfter.([]datatype.DataMap); ok {
			createData = &(result)
		}

		triggerAfter = m.runTriggerAction(AfterCreateTriggerAction, createData)
		if result, ok := triggerAfter.([]datatype.DataMap); ok {
			createData = &(result)
		}
	}

	if len(formattedUpdateData) > 0 {
		for _, v := range formattedUpdateData {
			vi := helper.ToMap[interface{}](v)
			vii, _ := helper.ConvertTo[datatype.DataMap](vi)

			updateData := m.NewInstance().Where("id", vii["_id"]).Update(*m.formatInputData(vii, UpdateInputAction), nil)

			if ud, ok := updateData.(*datatype.DataMap); ok && ud != nil {
				updated++
				interfaceData := helper.ToList[interface{}](updateData)
				afterData = append(afterData, interfaceData...)

				status = true
				message = "SUCCESS"
			} else {
				ignored++
			}
		}
	}

	result["message"] = message
	result["status"] = status
	result["deleted"] = deleted
	result["ignored"] = ignored
	result["imported"] = imported
	result["updated"] = updated
	result["data"] = afterData

	m.Model.App.socketServer.Of("/").Broadcast("database", datatype.DataMap{
		"action": "import",
		"model":  m.Model.Name,
	}, nil)

	return result
}

func (m *DataModelQuery) Delete(where interface{}) interface{} {
	m.WhereAll(where)
	m.addTenantId()

	triggerBefore := m.runTriggerAction(BeforeDeleteTriggerAllAction, m.where)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if v, ok := triggerBefore.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	triggerBefore = m.runTriggerAction(BeforeDeleteTriggerAction, m.where)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if v, ok := triggerBefore.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	result, err := m.collection().delete()

	if err != nil {
		return err
	}

	triggerAfter := m.runTriggerAction(AfterCreateTriggerAllAction, result)
	if v, ok := triggerAfter.(datatype.DataMap); ok {
		result = v
	}

	triggerAfter = m.runTriggerAction(AfterCreateTriggerAction, result)
	if v, ok := triggerAfter.(datatype.DataMap); ok {
		result = v
	}

	m.Model.App.socketServer.Of("/").Broadcast("database", datatype.DataMap{
		"action": "delete",
		"model":  m.Model.Name,
	}, nil)

	return result
}

func (m *DataModelQuery) Exist(where interface{}) bool {
	result := m.FindOne(where)

	return helper.IsNotEmpty(result)
}

func (m *DataModelQuery) Value(key string) interface{} {
	result := m.FindOne(nil)

	if helper.IsNotEmpty(result) {
		return (*result)[key]
	}

	return nil
}

func (m *DataModelQuery) First(where interface{}) *datatype.DataMap {
	return m.FindOne(where)
}

func (m *DataModelQuery) FindOne(where interface{}) *datatype.DataMap {
	m.WhereAll(where)
	m.addTenantId()

	triggerBefore := m.runTriggerAction(BeforeFindTriggerAllAction, m.where)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if v, ok := triggerBefore.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	triggerBefore = m.runTriggerAction(BeforeFindTriggerAction, m.where)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if v, ok := triggerBefore.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	result := m.collection().findOne()

	triggerAfter := m.runTriggerAction(AfterFindTriggerAllAction, result)
	if v, ok := triggerAfter.(datatype.DataMap); ok {
		result = &v
	}

	triggerAfter = m.runTriggerAction(AfterFindTriggerAction, result)
	if v, ok := triggerAfter.(datatype.DataMap); ok {
		result = &v
	}

	return result
}

func (m *DataModelQuery) Find(where interface{}) *[]datatype.DataMap {
	m.WhereAll(where)
	m.addTenantId()

	triggerBefore := m.runTriggerAction(BeforeFindTriggerAllAction, m.where)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if v, ok := triggerBefore.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	triggerBefore = m.runTriggerAction(BeforeFindTriggerAction, m.where)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if v, ok := triggerBefore.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	result := m.collection().find()

	triggerAfter := m.runTriggerAction(AfterFindTriggerAllAction, result)
	if v, ok := triggerAfter.([]datatype.DataMap); ok {
		result = &v
	}

	triggerAfter = m.runTriggerAction(AfterFindTriggerAction, result)
	if v, ok := triggerAfter.([]datatype.DataMap); ok {
		result = &v
	}

	return result
}

func (m *DataModelQuery) Paginate(where interface{}) *datatype.DataMap {
	m.WhereAll(where)
	m.addTenantId()

	triggerBefore := m.runTriggerAction(BeforeFindTriggerAllAction, m.where)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if v, ok := triggerBefore.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	triggerBefore = m.runTriggerAction(BeforeFindTriggerAction, m.where)
	if v, ok := triggerBefore.(bool); ok && !v {
		return nil
	} else if v, ok := triggerBefore.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	result := m.collection().pagination()

	triggerAfter := m.runTriggerAction(AfterFindTriggerAllAction, result)
	if v, ok := triggerAfter.(datatype.DataMap); ok {
		result = &v
	}

	triggerAfter = m.runTriggerAction(AfterFindTriggerAction, result)
	if v, ok := triggerAfter.(datatype.DataMap); ok {
		result = &v
	}

	return result
}

func (m *DataModelQuery) Summary(where interface{}) *datatype.DataMap {
	m.WhereAll(where)
	m.addTenantId()

	result := m.runTriggerAction(BeforeFindTriggerAllAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return nil
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	result = m.runTriggerAction(BeforeFindTriggerAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return nil
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	return m.collection().summary()
}

func (m *DataModelQuery) Count(where interface{}) int64 {
	m.WhereAll(where)
	m.addTenantId()

	result := m.runTriggerAction(BeforeFindTriggerAllAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return 0
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	result = m.runTriggerAction(BeforeFindTriggerAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return 0
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	return int64(m.collection().count())
}

func (m *DataModelQuery) Sum(target string, where interface{}) float64 {
	m.WhereAll(where)
	m.addTenantId()

	result := m.runTriggerAction(BeforeFindTriggerAllAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return 0
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	result = m.runTriggerAction(BeforeFindTriggerAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return 0
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	return float64(m.collection().sum(target))
}

func (m *DataModelQuery) Max(target string, where interface{}) interface{} {
	m.WhereAll(where)
	m.addTenantId()

	result := m.runTriggerAction(BeforeFindTriggerAllAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return nil
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	result = m.runTriggerAction(BeforeFindTriggerAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return nil
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	return m.collection().max(target)
}

func (m *DataModelQuery) Min(target string, where interface{}) interface{} {
	m.WhereAll(where)
	m.addTenantId()

	result := m.runTriggerAction(BeforeFindTriggerAllAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return nil
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	result = m.runTriggerAction(BeforeFindTriggerAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return nil
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	return m.collection().min(target)
}

func (m *DataModelQuery) Average(target string, where interface{}) float64 {
	m.WhereAll(where)
	m.addTenantId()

	result := m.runTriggerAction(BeforeFindTriggerAllAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return 0
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	result = m.runTriggerAction(BeforeFindTriggerAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return 0
	} else if v, ok := result.(datatype.DataMap); ok {
		m.WhereAll(v)
	}

	return m.collection().average(target)
}

func (m *DataModelQuery) Graph(where interface{}, p *graphql.ResolveParams) interface{} {
	m.WhereAll(where)
	m.addTenantId()

	ctx, _ := p.Context.Value(RequestContextKey).(*RequestContext)
	parent := p.Source

	var accessRole string = helper.GetValueOfString(p.Args, "accessRole")
	var route string = helper.GetValueOfString(p.Args, "route")

	if m.QueryContext.Params == nil {
		m.QueryContext.Params = make(map[string]interface{})
	}

	if pp, ok := p.Source.(graphql.ResolveParams); ok {
		localWhere := helper.ToMap[interface{}](pp.Args["where"])
		if localWhere != nil {
			m.WhereAll(localWhere)
		}

		accessRole = helper.GetValueOfString(pp.Args, "accessRole")
		route = helper.GetValueOfString(pp.Args, "route")

		if pp.Args != nil {
			for k, v := range pp.Args {
				m.QueryContext.Params[k] = v
			}
		}
	}

	// console.Log("p.Params 1", m.QueryContext.Params)
	filters := helper.ConvertToDataMap(helper.GetValueOfMap(p.Args, "where"))

	m.QueryContext.AccessRole = accessRole
	m.QueryContext.Route = route
	m.QueryContext.Filters = &filters

	if p.Args != nil {
		for k, v := range p.Args {
			m.QueryContext.Params[k] = v
		}
	}

	if ctx != nil {
		m.SetRequestContext(ctx)
	}

	if p, ok := parent.(datatype.DataMap); ok {
		m.QueryContext.Parent = &p
	}

	localWhere := helper.ToMap[interface{}](p.Args["where"])
	if localWhere != nil {
		m.WhereAll(localWhere)
	}

	localOrderBy := helper.ToMapList[string](p.Args["orderBy"])
	if localOrderBy != nil {
		m.OrderByAll(localOrderBy)
	}

	if v, ok := p.Args["limit"].(int); ok {
		m.Take(v)
	}

	if v, ok := p.Args["page"].(int); ok {
		m.Page(v)
	}

	if v, ok := p.Args["skip"].(int); ok {
		m.Skip(v)
	}

	result := m.runTriggerAction(BeforeFindTriggerAllAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return nil
	}

	result = m.runTriggerAction(BeforeFindTriggerAction, m.where)
	if v, ok := result.(bool); ok && !v {
		return nil
	}

	if where == nil {
		where = map[string]interface{}{}
	}

	var whereFilter map[string]FilterValue
	if err := json.Unmarshal(helper.ToByte(where), &whereFilter); err != nil {
		return nil
	}

	// console.Log("where", where)
	// console.Log("p.Args", p.Args)
	chart := NewChartBuilder(m)

	chartData, err := chart.BuildGraph(whereFilter, false)
	if err != nil {
		return err
	}

	return helper.ToMap[interface{}](chartData)
}

func (m *DataModelQuery) Download(where interface{}, fileType string) interface{} {
	result := make(map[string]interface{})
	data := m.Find(where)

	result["data"], _ = helper.ConvertJSONArrayToCSV(data, []string{}, "")

	return result
}

func (m *DataModelQuery) SetRequest(req *Request, res *Response) *DataModelQuery {
	m.RequestContext = &RequestContext{
		Auth:         req.Auth(),
		App:          req.App,
		Client:       req.Client(),
		TokenPayload: req.TokenPayload(),
		Request:      req,
		Response:     res,
	}

	return m
}

func (m *DataModelQuery) SetRequestContext(context *RequestContext) *DataModelQuery {
	m.RequestContext = context

	return m
}

func (m *DataModelQuery) setRequestAccessRole(value string) *DataModelQuery {
	m.QueryContext.AccessRole = value

	return m
}

func (m *DataModelQuery) setRequestRoute(value string) *DataModelQuery {
	m.QueryContext.Route = value

	return m
}

func (m *DataModelQuery) formatInputData(input datatype.DataMap, action InputAction) *datatype.DataMap {
	formatInput := datatype.DataMap{}

	switch action {
	case CreateInputAction, ImportInputAction:
		for _, k := range m.Model.ValidFields {
			var v interface{} = nil

			if k == "id" {
				if _, exist := input["_id"]; !exist {
					k = "_id"
				} else {
					v = input["_id"]
				}

				if _, exist := input["id"]; exist {
					v = input["id"]
				}
			}

			if k == "_id" {
				v = helper.ObjectID(v)
			}

			if vi, exist := input[k]; exist {
				v = vi
			} else if field, exist := m.Model.Fields[k]; exist {
				v = field.DefaultValue
			}

			formatInput[k] = m.formatInputDataField(k, v)
		}
	case UpdateInputAction:
		for _, k := range m.Model.ValidFields {
			if k != "_id" && k != "id" {
				if v, ok := input[k]; ok {
					formatInput[k] = m.formatInputDataField(k, v)
				}
			}
		}

	case ActionInputAction:
		for k, v := range input {
			formatInput[k] = m.formatInputDataField(k, v)
		}
	}

	return &formatInput
}

func (m *DataModelQuery) formatInputDataField(key string, value interface{}) interface{} {
	var v interface{}

	if helper.IsNotEmpty(value) {
		if helper.Contains(m.Model.RelativeKeys, key) || key == "id" || key == "_id" {
			v = helper.ObjectID(value)
		} else if helper.Contains(m.Model.DateFields, key) {
			v = helper.GetTimestamp(value)
		} else if helper.Contains(m.Model.NumberFields, key) {
			v = helper.ToInt(value)
		} else if helper.Contains(m.Model.FloatFields, key) {
			v = helper.ToFloat(value)
		} else {
			v = value
		}
	} else {
		v = value
	}

	return v
}

func (m *DataModelQuery) runTriggerAction(action TriggerAction, data interface{}) interface{} {
	model := m.Model
	listAll := []string{
		string(BeforeFindTriggerAllAction),
		string(AfterFindTriggerAllAction),
		string(BeforeCreateTriggerAllAction),
		string(AfterCreateTriggerAllAction),
		string(BeforeUpdateTriggerAllAction),
		string(AfterUpdateTriggerAllAction),
		string(BeforeDeleteTriggerAllAction),
		string(AfterDeleteTriggerAllAction),
	}

	switch action {
	case BeforeFindTriggerAction, BeforeFindTriggerAllAction:
		if v, ok := data.(datatype.DataMap); ok {
			m.QueryContext.Filters = &v
		}
	case AfterFindTriggerAction, AfterFindTriggerAllAction:
		m.QueryContext.Data = data
	case BeforeCreateTriggerAction, BeforeCreateTriggerAllAction:
		if v, ok := data.(datatype.DataMap); ok {
			m.QueryContext.Input = &v
		}
	case AfterCreateTriggerAction, AfterCreateTriggerAllAction:
		m.QueryContext.Data = data
	case BeforeUpdateTriggerAction, BeforeUpdateTriggerAllAction:
		if v, ok := data.(datatype.DataMap); ok {
			m.QueryContext.Input = &v
		}
	case AfterUpdateTriggerAction, AfterUpdateTriggerAllAction:
		m.QueryContext.Data = data
	case BeforeDeleteTriggerAction, BeforeDeleteTriggerAllAction:
		if v, ok := data.(datatype.DataMap); ok {
			m.QueryContext.Filters = &v
		}
	case AfterDeleteTriggerAction, AfterDeleteTriggerAllAction:
		m.QueryContext.Data = data
	}

	var result interface{}
	var err error

	if helper.Contains(listAll, string(action)) {
		result, err = model.App.triggerAllCallback(action, model, m.RequestContext, &m.QueryContext)
	} else {
		result, err = model.App.triggerCallback(model.Name, action, m.RequestContext, &m.QueryContext)
	}

	if err != nil {
		// logger.Error("runTriggerAction", err.Error())
	}

	return result
}

func (m *DataModelQuery) addTenantId() {
	if m.Model.HasTenant && m.RequestContext != nil && m.Model.App.Config.HasTenant {
		payload := m.RequestContext.TokenPayload
		tenantId := ""

		if payload != nil && helper.IsNotEmpty(payload.TenantId) {
			tenantId = payload.TenantId
		} else {
			tenantId = *m.RequestContext.Request.TenantId()
		}

		if helper.IsEmpty(tenantId) {
			tenantId = "000"
		}

		m.Where(TenantIDKey, helper.ObjectID(tenantId))
	}
}

func (m *DataModelQuery) getTenantId() string {
	tenantId := ""
	if m.Model.HasTenant && m.RequestContext != nil && m.Model.App.Config.HasTenant {
		payload := m.RequestContext.TokenPayload

		if payload != nil && helper.IsNotEmpty(payload.TenantId) {
			tenantId = payload.TenantId
		} else {
			tenantId = *m.RequestContext.Request.TenantId()
		}
	}

	return tenantId
}

func (m *DataModelQuery) collection() dataModelQueryStructure {
	ctx := context.TODO()

	switch m.Model.DatabaseType {
	case config.DBTypeMongodb:
		return &mongodbConnection{
			query:  m,
			ctx:    &ctx,
			client: m.Model.DBConnect.mongodbClient,
		}
	case config.DBTypeSql:
		return &sqlConnection{
			query:  m,
			ctx:    &ctx,
			client: m.Model.DBConnect.sqlClient,
		}
	case config.DBTypeMysql:
		return &mysqlConnection{
			query:  m,
			ctx:    &ctx,
			client: m.Model.DBConnect.mysqlClient,
		}
	}

	return &localDbConnection{
		query:  m,
		ctx:    &ctx,
		client: m.Model.DBConnect.localClient,
	}
}
