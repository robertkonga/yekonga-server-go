package Yekonga

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/robertkonga/yekonga-server/datatype"
	"github.com/robertkonga/yekonga-server/helper"
	"github.com/robertkonga/yekonga-server/helper/logger"
	localDB "github.com/robertkonga/yekonga-server/plugins/database/db"
)

type localDbConnection struct {
	query  *DataModelQuery
	ctx    *context.Context
	client *localDB.DB
}

func (con *localDbConnection) collection() *localDB.Col {

	// Get collection
	var collection *localDB.Col

	if !con.client.ColExists(con.query.Model.Collection) {
		con.client.Create(con.query.Model.Collection)
	}

	collection = con.client.Use(con.query.Model.Collection)

	return collection
}

func (con *localDbConnection) findOne() *datatype.DataMap {
	id := 1
	doc, err := con.collection().Read(id)
	if err != nil {
		return nil
	}

	doci := datatype.DataMap(doc)

	return &(doci)
}

func (con *localDbConnection) findAll() *[]datatype.DataMap {
	var result []datatype.DataMap = make([]datatype.DataMap, 0, 10)
	var count int = 0

	con.collection().ForEachDoc(func(id int, data []byte) bool {
		var doc datatype.DataMap
		json.Unmarshal(data, &doc)
		result = append(result, doc)
		count += 1
		return true
	})

	return &result
}

func (con *localDbConnection) find() *[]datatype.DataMap {
	var ids map[int]struct{} = make(map[int]struct{})
	var result []datatype.DataMap = make([]datatype.DataMap, 0, 10)
	var where = con.conditionParams()

	localDB.EvalQuery(where, con.collection(), &ids)

	for id := range ids {
		data, err := con.collection().Read(id)
		if err != nil {
			result = append(result, data)
		}
	}

	return &result
}

func (con *localDbConnection) pagination() *datatype.DataMap {
	result := datatype.DataMap{
		"total":       0,
		"perPage":     0,
		"currentPage": 0,
		"lastPage":    0,
		"from":        0,
		"to":          0,
		"data":        []interface{}{},
	}

	return &result
}

func (con *localDbConnection) summary() *datatype.DataMap {
	result := datatype.DataMap{
		"count": 0,
		"sum":   0,
		"max":   0,
		"min":   0,
		"graph": datatype.DataMap{},
	}

	return &result
}

func (con *localDbConnection) count() int64 {
	return 0

}

func (con *localDbConnection) max(key string) interface{} {
	return 0
}

func (con *localDbConnection) min(key string) interface{} {
	return 0
}

func (con *localDbConnection) sum(key string) float64 {
	return 0
}

func (con *localDbConnection) average(key string) float64 {
	return 0
}

func (con *localDbConnection) graph() *datatype.DataMap {
	return &datatype.DataMap{}
}

func (con *localDbConnection) create(data interface{}) (*datatype.DataMap, error) {
	if v, ok := data.(datatype.DataMap); ok {
		id, err := con.collection().Insert(v)
		if err != nil {
			return nil, err
		}

		return con.query.Where("id", id).FindOne(nil), nil
	}

	return nil, errors.New("Fail")
}

func (con *localDbConnection) createMany(data []interface{}) (*[]datatype.DataMap, error) {
	var result []datatype.DataMap = []datatype.DataMap{}

	for _, d := range data {
		if v, ok := d.(datatype.DataMap); ok {
			id, err := con.collection().Insert(v)
			if err != nil {
				// return nil, err
			} else {
				result = append(result, *con.query.Where("id", id).FindOne(nil))
			}

		}
	}

	return &result, nil
}

func (con *localDbConnection) update(data interface{}) (*datatype.DataMap, error) {
	con.query.FindOne(nil)
	id := 1
	if v, ok := data.(datatype.DataMap); ok {
		err := con.collection().Update(id, v)
		if err != nil {
			return nil, err
		}

		return con.query.Where("id", id).FindOne(nil), nil
	}

	return nil, errors.New("Fail")

}

func (con *localDbConnection) updateMany(data interface{}) (*[]datatype.DataMap, error) {
	con.query.Find(nil)
	id := 1
	if v, ok := data.(datatype.DataMap); ok {
		err := con.collection().Update(id, v)
		if err != nil {
			return nil, err
		}

		return con.query.Where("id", id).Find(nil), nil
	}

	return nil, errors.New("Fail")

}

func (con *localDbConnection) delete() (interface{}, error) {
	id := 1
	err := con.collection().Delete(id)
	if err != nil {
		return nil, err
	}

	return nil, nil

}

func (con *localDbConnection) selection() *[]string {
	return &[]string{}
}

func (con *localDbConnection) conditionParams() interface{} {
	var params = datatype.DataMap{}
	var where = con.where()

	if w, ok := where.(map[string][]datatype.DataMap); ok {
		for k, v := range w {
			params[k] = v
		}
	}

	if con.limit() > 0 {
		params["limit"] = con.limit()
	}

	return params
}

func (con *localDbConnection) where() interface{} {
	var filters = map[string][]datatype.DataMap{
		"and": []datatype.DataMap{},
	}
	operations := [...]string{
		"equalTo",
		"notEqualTo",
		"lessThan",
		"notLessThan",
		"lessThanOrEqualTo",
		"notLessThanOrEqualTo",
		"greaterThan",
		"notGreaterThan",
		"greaterThanOrEqualTo",
		"notGreaterThanOrEqualTo",
		"matchesRegex",
		"options",
	}

	arrayOperations := [...]string{
		"in",
		"all",
		"notIn",
	}

	booleanOperations := [...]string{
		"exists",
	}

	// logger.Info("where", 1)
	if con.query.where != nil {
		// logger.Info("where", 2)
		for k, v := range con.query.where {
			// logger.Info("where", 3)

			if vi, ok := v.(datatype.DataMap); ok {
				// logger.Info("where", 4)
				for ki, vii := range vi {
					// logger.Info("where", 5)
					if helper.Contains(operations[:], ki) ||
						helper.Contains(arrayOperations[:], ki) ||
						helper.Contains(booleanOperations[:], ki) {
						// logger.Info("where", 6)
						switch ki {
						case "equalTo":
							filters["and"] = append(filters["and"], datatype.DataMap{
								"eq": vii,
								"in": []interface{}{k},
							})
						case "notEqualTo":
							filters["and"] = append(filters["and"], datatype.DataMap{
								"not": datatype.DataMap{
									"eq": vii,
									"in": []interface{}{k},
								},
							})
						case "lessThan":
							viii := helper.ConvertCalculatedValue(vii)

							filters["and"] = append(filters["and"], datatype.DataMap{
								"lt": viii,
								"in": []interface{}{k},
							})
						case "notLessThan":
							viii := helper.ConvertCalculatedValue(vii)

							filters["and"] = append(filters["and"], datatype.DataMap{
								"not": datatype.DataMap{
									"lt": viii,
									"in": []interface{}{k},
								},
							})
						case "lessThanOrEqualTo":
							viii := helper.ConvertCalculatedValue(vii)
							filters["and"] = append(filters["and"], datatype.DataMap{
								"lte": viii,
								"in":  []interface{}{k},
							})
						case "notLessThanOrEqualTo":
							viii := helper.ConvertCalculatedValue(vii)
							filters["and"] = append(filters["and"], datatype.DataMap{
								"not": datatype.DataMap{
									"lte": viii,
									"in":  []interface{}{k},
								},
							})
						case "greaterThan":
							viii := helper.ConvertCalculatedValue(vii)
							filters["and"] = append(filters["and"], datatype.DataMap{
								"gt": viii,
								"in": []interface{}{k},
							})
						case "notGreaterThan":
							viii := helper.ConvertCalculatedValue(vii)
							filters["and"] = append(filters["and"], datatype.DataMap{
								"not": datatype.DataMap{
									"gt": viii,
									"in": []interface{}{k},
								},
							})
						case "greaterThanOrEqualTo":
							viii := helper.ConvertCalculatedValue(vii)
							filters["and"] = append(filters["and"], datatype.DataMap{
								"gte": viii,
								"in":  []interface{}{k},
							})
						case "notGreaterThanOrEqualTo":
							viii := helper.ConvertCalculatedValue(vii)
							filters["and"] = append(filters["and"], datatype.DataMap{
								"not": datatype.DataMap{
									"gte": viii,
									"in":  []interface{}{k},
								},
							})
						case "in":
							// filters[k]["$in"] = vii
							filters["and"] = append(filters["and"], datatype.DataMap{
								"$in": vii,
								"in":  []interface{}{k},
							})
						case "all":
							// filters[k]["$all"] = vii
							filters["and"] = append(filters["and"], datatype.DataMap{
								"all": vii,
								"in":  []interface{}{k},
							})
						case "notIn":
							// filters[k]["$nin"] = vii
							filters["and"] = append(filters["and"], datatype.DataMap{
								"not": datatype.DataMap{
									"$nin": vii,
									"in":   []interface{}{k},
								},
							})
						case "exists":
							if exists, ok := vii.(bool); ok {
								if exists {
									// filters[k]["$exists"] = true
									// filters[k]["$nin"] = []interface{}{nil}
								} else {
									// filters["$or"] = []datatype.DataMap{}

									// filters["$or"] = append(filters["$or"], map[string]datatype.DataMap{
									// 	k: datatype.DataMap{
									// 		"$exists": false,
									// 	},
									// })

									// filters["$or"] = append(filters["$or"], map[string]datatype.DataMap{
									// 	k: datatype.DataMap{
									// 		"$nin": []interface{}{nil},
									// 	},
									// })
								}
							}
						case "matchesRegex":
							filters["and"] = append(filters["and"], datatype.DataMap{
								"has": vii,
								"in":  []interface{}{k},
							})
						case "options":
							filters["and"] = append(filters["and"], datatype.DataMap{
								"eq": vii,
								"in": []interface{}{k},
							})
						case "text":
							filters["and"] = append(filters["and"], datatype.DataMap{
								"eq": vii,
								"in": []interface{}{k},
							})
						case "inQueryKey":
							filters["and"] = append(filters["and"], datatype.DataMap{
								"eq": vii,
								"in": []interface{}{k},
							})
						case "notInQueryKey":
							filters["and"] = append(filters["and"], datatype.DataMap{
								"eq": vii,
								"in": []interface{}{k},
							})
						}
					} else {
						logger.Info("where", 9)

					}
				}
			} else {
				filters["and"] = append(filters["and"], datatype.DataMap{
					"eq": v,
					"in": []interface{}{k},
				})
			}
		}
	}

	return filters
}

func (con *localDbConnection) groupBy() *[]interface{} {
	return &[]interface{}{}
}

func (con *localDbConnection) orderBy() *[]datatype.DataMap {
	return &[]datatype.DataMap{}
}

func (con *localDbConnection) limit() int {
	if con.query.limit > 0 {
		return con.query.limit
	}

	return -1
}

func (con *localDbConnection) skip() int {
	if con.query.skip > 0 {
		return con.query.skip
	}

	return con.query.limit * (con.query.page - 1)
}
