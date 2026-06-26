package yekonga

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/console"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
)

type sqlConnection struct {
	query  *DataModelQuery
	ctx    *context.Context
	client *sql.DB
	mut    sync.RWMutex
}

func newSQLInstance(con *sqlConnection) sqlConnection {
	return sqlConnection{
		ctx:    con.ctx,
		client: con.client,
		query: &DataModelQuery{
			Model:          con.query.Model,
			RequestContext: con.query.RequestContext,
			QueryContext:   con.query.QueryContext,
		},
	}
}

func (con *sqlConnection) connect() any {
	return con.client
}

func (con *sqlConnection) collection() any {
	return con.client
}

func (con *sqlConnection) findOne() *datatype.DataMap {
	query := con.buildSelectQuery()
	query += " LIMIT 1"

	row := con.client.QueryRow(query)

	var result datatype.DataMap
	err := row.Scan()
	if err != nil {
		if err != sql.ErrNoRows {
			logger.Error("sqlConnection.findOne", err.Error())
		}
		return nil
	}

	if result != nil {
		result["_collection"] = con.query.Model.Collection
		result["_model"] = con.query.Model.Name
	}

	return &result
}

func (con *sqlConnection) findAll() *[]datatype.DataMap {
	return con.find()
}

func (con *sqlConnection) find() *[]datatype.DataMap {
	query := con.buildSelectQuery()

	if con.hasOrderBy() {
		query += " " + con.orderBySQL()
	}

	if con.limit() > 0 {
		query += fmt.Sprintf(" LIMIT %d", con.limit())
	}

	if con.skip() > 0 {
		query += fmt.Sprintf(" OFFSET %d", con.skip())
	}

	rows, err := con.client.Query(query)
	if err != nil {
		logger.Error("sqlConnection.find 1", err.Error())
		return &[]datatype.DataMap{}
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		logger.Error("sqlConnection.find 2", err.Error())
		return &[]datatype.DataMap{}
	}

	result := make([]datatype.DataMap, 0)

	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			logger.Error("sqlConnection.find 3", err.Error())
			continue
		}

		data := make(datatype.DataMap)
		for i, col := range columns {
			val := values[i]
			data[col] = val
			if val == nil {
				data[col] = nil
			}
		}

		data["_collection"] = con.query.Model.Collection
		data["_model"] = con.query.Model.Name

		result = append(result, data)
	}

	if err := rows.Err(); err != nil {
		logger.Error("sqlConnection.find 4", err.Error())
	}

	return &result
}

func (con *sqlConnection) pagination() *datatype.DataMap {
	var lastPage int64
	total := con.count()
	perPage := int64(con.limit())
	if perPage <= 0 {
		perPage = 10
	}
	currentPage := int64(con.page())
	from := (perPage * (currentPage - 1)) + 1
	to := perPage * (currentPage)
	remainder := total % perPage

	if remainder == 0 {
		lastPage = (total) / perPage
	} else {
		lastPage = (total + (perPage - total%perPage)) / perPage
	}

	con.query.Take(int(perPage))

	result := datatype.DataMap{
		"total":       total,
		"perPage":     perPage,
		"currentPage": currentPage,
		"lastPage":    lastPage,
		"from":        from,
		"to":          to,
		"data":        con.find(),
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
	query := fmt.Sprintf("SELECT COUNT(*) as count FROM `%s`", con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	row := con.client.QueryRow(query)

	var count int64
	err := row.Scan(&count)
	if err != nil {
		logger.Error("sqlConnection.count", err.Error())
		return 0
	}

	return count
}

func (con *sqlConnection) max(key string) interface{} {
	query := fmt.Sprintf("SELECT MAX(`%s`) as max_value FROM `%s`", key, con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	row := con.client.QueryRow(query)

	var result interface{}
	err := row.Scan(&result)
	if err != nil {
		logger.Error("sqlConnection.max", err.Error())
		return nil
	}

	return result
}

func (con *sqlConnection) min(key string) interface{} {
	query := fmt.Sprintf("SELECT MIN(`%s`) as min_value FROM `%s`", key, con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	row := con.client.QueryRow(query)

	var result interface{}
	err := row.Scan(&result)
	if err != nil {
		logger.Error("sqlConnection.min", err.Error())
		return nil
	}

	return result
}

func (con *sqlConnection) sum(key string) float64 {
	query := fmt.Sprintf("SELECT COALESCE(SUM(`%s`), 0) as sum_value FROM `%s`", key, con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	row := con.client.QueryRow(query)

	var result float64
	err := row.Scan(&result)
	if err != nil {
		logger.Error("sqlConnection.sum", err.Error())
		return 0
	}

	return result
}

func (con *sqlConnection) average(key string) float64 {
	query := fmt.Sprintf("SELECT COALESCE(AVG(`%s`), 0) as avg_value FROM `%s`", key, con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	row := con.client.QueryRow(query)

	var result float64
	err := row.Scan(&result)
	if err != nil {
		logger.Error("sqlConnection.average", err.Error())
		return 0
	}

	return result
}

func (con *sqlConnection) graph() *datatype.DataMap {
	return &datatype.DataMap{}
}

func (con *sqlConnection) create(data datatype.DataMap) (*datatype.DataMap, error) {
	keys := make([]string, 0)
	values := make([]interface{}, 0)

	for k, v := range data {
		keys = append(keys, fmt.Sprintf("`%s`", k))
		values = append(values, v)
	}

	placeholders := make([]string, len(keys))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		con.query.Model.Collection,
		strings.Join(keys, ", "),
		strings.Join(placeholders, ", "))

	result, err := con.client.Exec(query, values...)
	if err != nil {
		console.Log("sqlConnection.create", err.Error())
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		console.Log("sqlConnection.create", err.Error())
		return nil, err
	}

	createdRecord := newSQLInstance(con).query.Where("id", id).collection().findOne()
	return createdRecord, nil
}

func (con *sqlConnection) createMany(data []datatype.DataMap) (*[]datatype.DataMap, error) {
	if len(data) == 0 {
		return nil, errors.New("no data provided")
	}

	keys := make([]string, 0)
	allValues := make([]interface{}, 0)
	placeholderRows := make([]string, 0)

	for idx, record := range data {
		if idx == 0 {
			for k := range record {
				keys = append(keys, fmt.Sprintf("`%s`", k))
			}
		}

		rowPlaceholders := make([]string, len(keys))
		for i := range rowPlaceholders {
			rowPlaceholders[i] = "?"
		}
		placeholderRows = append(placeholderRows, "("+strings.Join(rowPlaceholders, ", ")+")")

		for _, k := range keys {
			k = strings.Trim(k, "`")
			allValues = append(allValues, record[k])
		}
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES %s",
		con.query.Model.Collection,
		strings.Join(keys, ", "),
		strings.Join(placeholderRows, ", "))

	_, err := con.client.Exec(query, allValues...)
	if err != nil {
		console.Log("sqlConnection.createMany", err.Error())
		return nil, err
	}

	result := newSQLInstance(con).query.collection().find()
	return result, nil
}

func (con *sqlConnection) update(data datatype.DataMap) (*datatype.DataMap, error) {
	setParts := make([]string, 0)
	values := make([]interface{}, 0)

	for k, v := range data {
		setParts = append(setParts, fmt.Sprintf("`%s` = ?", k))
		values = append(values, v)
	}

	query := fmt.Sprintf("UPDATE `%s` SET %s", con.query.Model.Collection, strings.Join(setParts, ", "))

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
		whereVals := con.getWhereValues()
		values = append(values, whereVals...)
	}

	result, err := con.client.Exec(query, values...)
	if err != nil {
		console.Log("sqlConnection.update", err.Error())
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return nil, err
	}

	updatedRecord := newSQLInstance(con).query.WhereAll(*con.where()).FindOne(nil)
	return updatedRecord, nil
}

func (con *sqlConnection) updateMany(data datatype.DataMap) (*[]datatype.DataMap, error) {
	setParts := make([]string, 0)
	values := make([]interface{}, 0)

	for k, v := range data {
		setParts = append(setParts, fmt.Sprintf("`%s` = ?", k))
		values = append(values, v)
	}

	query := fmt.Sprintf("UPDATE `%s` SET %s", con.query.Model.Collection, strings.Join(setParts, ", "))

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
		whereVals := con.getWhereValues()
		values = append(values, whereVals...)
	}

	_, err := con.client.Exec(query, values...)
	if err != nil {
		console.Log("sqlConnection.updateMany", err.Error())
		return nil, err
	}

	result := newSQLInstance(con).query.WhereAll(*con.where()).Find(nil)
	return result, nil
}

func (con *sqlConnection) delete() (interface{}, error) {
	if !con.hasWhere() {
		return nil, errors.New("filter is empty, not allowed to delete all records at once")
	}

	query := fmt.Sprintf("DELETE FROM `%s` WHERE %s", con.query.Model.Collection, con.whereSQL())
	whereVals := con.getWhereValues()

	result, err := con.client.Exec(query, whereVals...)
	if err != nil {
		console.Log("sqlConnection.delete", err.Error())
		return nil, err
	}

	return result, nil
}

func (con *sqlConnection) selection() *[]string {
	return &[]string{}
}

func (con *sqlConnection) where() *datatype.DataMap {
	return &con.query.where
}

func (con *sqlConnection) whereSQL() string {
	conditions := make([]string, 0)

	for k, v := range con.query.where {
		if helper.Contains([]string{"AND", "OR", "NOR"}, k) {
			conditions = append(conditions, con.extractWhereSQL(con.query.where))
		} else {
			conditions = append(conditions, con.extractWhereItemSQL(k, v))
		}
	}

	if len(conditions) == 0 {
		return ""
	}

	return strings.Join(conditions, " AND ")
}

func (con *sqlConnection) extractWhereSQL(where interface{}) string {
	return ""
}

func (con *sqlConnection) extractWhereItemSQL(key string, value interface{}) string {
	if vm, ok := value.(map[string]interface{}); ok {
		for op := range vm {
			switch op {
			case "equalTo", "$eq":
				return fmt.Sprintf("`%s` = ?", key)
			case "notEqualTo", "$ne":
				return fmt.Sprintf("`%s` != ?", key)
			case "lessThan", "$lt":
				return fmt.Sprintf("`%s` < ?", key)
			case "lessThanOrEqualTo", "$lte":
				return fmt.Sprintf("`%s` <= ?", key)
			case "greaterThan", "$gt":
				return fmt.Sprintf("`%s` > ?", key)
			case "greaterThanOrEqualTo", "$gte":
				return fmt.Sprintf("`%s` >= ?", key)
			case "in", "$in":
				if vi, ok := vm[op].([]interface{}); ok {
					placeholders := make([]string, len(vi))
					for i := range placeholders {
						placeholders[i] = "?"
					}
					return fmt.Sprintf("`%s` IN (%s)", key, strings.Join(placeholders, ", "))
				}
			case "notIn", "$nin":
				if vi, ok := vm[op].([]interface{}); ok {
					placeholders := make([]string, len(vi))
					for i := range placeholders {
						placeholders[i] = "?"
					}
					return fmt.Sprintf("`%s` NOT IN (%s)", key, strings.Join(placeholders, ", "))
				}
			}
		}
	} else if vi, ok := value.([]interface{}); ok {
		placeholders := make([]string, len(vi))
		for i := range placeholders {
			placeholders[i] = "?"
		}
		return fmt.Sprintf("`%s` IN (%s)", key, strings.Join(placeholders, ", "))
	} else {
		return fmt.Sprintf("`%s` = ?", key)
	}

	return ""
}

func (con *sqlConnection) getWhereValues() []interface{} {
	values := make([]interface{}, 0)

	for _, v := range con.query.where {
		if vm, ok := v.(map[string]interface{}); ok {
			for _, val := range vm {
				if vi, ok := val.([]interface{}); ok {
					values = append(values, vi...)
				} else {
					values = append(values, val)
				}
			}
		} else if vi, ok := v.([]interface{}); ok {
			values = append(values, vi...)
		} else {
			values = append(values, v)
		}
	}

	return values
}

func (con *sqlConnection) hasWhere() bool {
	return len(con.query.where) > 0
}

func (con *sqlConnection) hasOrderBy() bool {
	return len(con.query.orderBy) > 0
}

func (con *sqlConnection) buildSelectQuery() string {
	query := fmt.Sprintf("SELECT * FROM `%s`", con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	return query
}

func (con *sqlConnection) orderBySQL() string {
	if !con.hasOrderBy() {
		return ""
	}

	parts := make([]string, 0)
	for k, v := range con.query.orderBy {
		direction := "ASC"
		if strings.ToLower(v) == "desc" {
			direction = "DESC"
		}
		parts = append(parts, fmt.Sprintf("`%s` %s", k, direction))
	}

	return "ORDER BY " + strings.Join(parts, ", ")
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

func (con *sqlConnection) page() int {
	if con.query.page < 1 {
		return 1
	}

	return con.query.page
}
