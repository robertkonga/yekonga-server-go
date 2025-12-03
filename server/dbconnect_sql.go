package Yekonga

import (
	"context"

	"github.com/robertkonga/yekonga-server/datatype"
)

type sqlConnection struct {
	query  *DataModelQuery
	ctx    *context.Context
	client *interface{}
}

func (con *sqlConnection) connect() any {
	return nil
}

func (con *sqlConnection) collection() any {

	return nil
}

func (con *sqlConnection) findOne() *datatype.DataMap {
	return nil
}

func (con *sqlConnection) findAll() *[]datatype.DataMap {

	return con.find()
}

func (con *sqlConnection) find() *[]datatype.DataMap {
	var result []datatype.DataMap

	return &result
}

func (con *sqlConnection) pagination() *datatype.DataMap {
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

func (con *sqlConnection) summary() *datatype.DataMap {
	result := datatype.DataMap{
		"count": 0,
		"sum":   0,
		"max":   0,
		"min":   0,
		"graph": datatype.DataMap{},
	}

	return &result
}

func (con *sqlConnection) count() int64 {
	return 0
}

func (con *sqlConnection) max(key string) interface{} {
	return 0
}

func (con *sqlConnection) min(key string) interface{} {
	return 0
}

func (con *sqlConnection) sum(key string) float64 {
	return 0
}

func (con *sqlConnection) average(key string) float64 {
	return 0
}

func (con *sqlConnection) graph() *datatype.DataMap {
	return &datatype.DataMap{}
}

func (con *sqlConnection) create(data interface{}) (*datatype.DataMap, error) {
	return nil, nil
}

func (con *sqlConnection) createMany(data []interface{}) (*[]datatype.DataMap, error) {
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

func (con *sqlConnection) update(data interface{}) (*datatype.DataMap, error) {
	return nil, nil

}

func (con *sqlConnection) updateMany(data interface{}) (*[]datatype.DataMap, error) {
	return nil, nil

}

func (con *sqlConnection) delete() (interface{}, error) {
	return nil, nil

}

func (con *sqlConnection) selection() *[]string {
	return &[]string{}

}

func (con *sqlConnection) where() *datatype.DataMap {
	return &datatype.DataMap{}
}

func (con *sqlConnection) groupBy() *[]interface{} {
	return &[]interface{}{}
}

func (con *sqlConnection) orderBy() *[]datatype.DataMap {
	return &[]datatype.DataMap{}
}

func (con *sqlConnection) limit() int {
	if con.query.limit > 0 {
		return con.query.limit
	}

	return -1
}
func (con *sqlConnection) skip() int {
	if con.query.skip > 0 {
		return con.query.skip
	}

	return con.query.limit * (con.query.page - 1)
}
