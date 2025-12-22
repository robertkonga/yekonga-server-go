package Yekonga

import (
	"context"

	"github.com/robertkonga/yekonga-server/datatype"
)

type mysqlConnection struct {
	query  *DataModelQuery
	ctx    *context.Context
	client *interface{}
}

func (con *mysqlConnection) connect() any {
	return nil
}

func (con *mysqlConnection) collection() any {

	return nil
}

func (con *mysqlConnection) findOne() *datatype.DataMap {
	return nil
}

func (con *mysqlConnection) findAll() *[]datatype.DataMap {

	return con.find()
}

func (con *mysqlConnection) find() *[]datatype.DataMap {
	var result []datatype.DataMap

	return &result
}

func (con *mysqlConnection) pagination() *datatype.DataMap {
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

func (con *mysqlConnection) summary() *datatype.DataMap {
	result := datatype.DataMap{
		"count": 0,
		"sum":   0,
		"max":   0,
		"min":   0,
		"graph": datatype.DataMap{},
	}

	return &result
}

func (con *mysqlConnection) count() int64 {
	return 0

}

func (con *mysqlConnection) max(key string) interface{} {
	return 0
}

func (con *mysqlConnection) min(key string) interface{} {
	return 0
}

func (con *mysqlConnection) sum(key string) float64 {
	return 0
}

func (con *mysqlConnection) average(key string) float64 {
	return 0
}

func (con *mysqlConnection) graph() *datatype.DataMap {
	return &datatype.DataMap{}
}

func (con *mysqlConnection) create(data datatype.DataMap) (*datatype.DataMap, error) {
	return nil, nil
}

func (con *mysqlConnection) createMany(data []datatype.DataMap) (*[]datatype.DataMap, error) {
	var result *[]datatype.DataMap
	// res, err := con.collection().InsertMany(*con.ctx, data)

	// if err != nil {
	// 	return nil, err
	// }

	// if res.Acknowledged {
	// 	where := map[string]interface{}{"_id": map[string]interface{}{"in": res.InsertedIDs}}
	// 	result = newInstance(con).query.WhereAll(where).collection().find()
	// }

	return result, nil
}

func (con *mysqlConnection) update(data datatype.DataMap) (*datatype.DataMap, error) {
	return nil, nil

}

func (con *mysqlConnection) updateMany(data datatype.DataMap) (*[]datatype.DataMap, error) {
	return nil, nil

}

func (con *mysqlConnection) delete() (interface{}, error) {
	return nil, nil

}

func (con *mysqlConnection) selection() *[]string {
	return &[]string{}

}

func (con *mysqlConnection) where() *datatype.DataMap {
	return &datatype.DataMap{}
}

func (con *mysqlConnection) groupBy() *[]interface{} {
	return &[]interface{}{}
}

func (con *mysqlConnection) orderBy() *[]datatype.DataMap {
	return &[]datatype.DataMap{}
}

func (con *mysqlConnection) limit() int {
	if con.query.limit > 0 {
		return con.query.limit
	}

	return -1
}
func (con *mysqlConnection) skip() int {
	if con.query.skip > 0 {
		return con.query.skip
	}

	return con.query.limit * (con.query.page - 1)
}
