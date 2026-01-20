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

type mysqlConnection struct {
	query  *DataModelQuery
	ctx    *context.Context
	client *sql.DB
	mut    sync.RWMutex
}

func newMySQLInstance(con *mysqlConnection) mysqlConnection {
	return mysqlConnection{
		ctx:    con.ctx,
		client: con.client,
		query: &DataModelQuery{
			Model:          con.query.Model,
			RequestContext: con.query.RequestContext,
			QueryContext:   con.query.QueryContext,
		},
	}
}

func (con *mysqlConnection) connect() any {
	return con.client
}

func (con *mysqlConnection) collection() any {
	return con.client
}

func (con *mysqlConnection) findOne() *datatype.DataMap {
	query := con.buildSelectQuery()
	query += " LIMIT 1"

	row := con.client.QueryRow(query)

	var result datatype.DataMap
	// For simplicity, we'll scan into a generic structure
	// In production, you'd need to dynamically handle columns
	err := row.Scan()
	if err != nil {
		if err != sql.ErrNoRows {
			logger.Error("mysqlConnection.findOne", err.Error())
		}
		return nil
	}

	if result != nil {
		result["_collection"] = con.query.Model.Collection
		result["_model"] = con.query.Model.Name
	}

	return &result
}

func (con *mysqlConnection) findAll() *[]datatype.DataMap {
	return con.find()
}

func (con *mysqlConnection) find() *[]datatype.DataMap {
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
		logger.Error("mysqlConnection.find 1", err.Error())
		return &[]datatype.DataMap{}
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		logger.Error("mysqlConnection.find 2", err.Error())
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
			logger.Error("mysqlConnection.find 3", err.Error())
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
		logger.Error("mysqlConnection.find 4", err.Error())
	}

	return &result
}

func (con *mysqlConnection) pagination() *datatype.DataMap {
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

func (con *mysqlConnection) count() int64 {
	query := fmt.Sprintf("SELECT COUNT(*) as count FROM `%s`", con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	row := con.client.QueryRow(query)

	var count int64
	err := row.Scan(&count)
	if err != nil {
		logger.Error("mysqlConnection.count", err.Error())
		return 0
	}

	return count
}

func (con *mysqlConnection) max(key string) interface{} {
	query := fmt.Sprintf("SELECT MAX(`%s`) as max_value FROM `%s`", key, con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	row := con.client.QueryRow(query)

	var result interface{}
	err := row.Scan(&result)
	if err != nil {
		logger.Error("mysqlConnection.max", err.Error())
		return nil
	}

	return result
}

func (con *mysqlConnection) min(key string) interface{} {
	query := fmt.Sprintf("SELECT MIN(`%s`) as min_value FROM `%s`", key, con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	row := con.client.QueryRow(query)

	var result interface{}
	err := row.Scan(&result)
	if err != nil {
		logger.Error("mysqlConnection.min", err.Error())
		return nil
	}

	return result
}

func (con *mysqlConnection) sum(key string) float64 {
	query := fmt.Sprintf("SELECT COALESCE(SUM(`%s`), 0) as sum_value FROM `%s`", key, con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	row := con.client.QueryRow(query)

	var result float64
	err := row.Scan(&result)
	if err != nil {
		logger.Error("mysqlConnection.sum", err.Error())
		return 0
	}

	return result
}

func (con *mysqlConnection) average(key string) float64 {
	query := fmt.Sprintf("SELECT COALESCE(AVG(`%s`), 0) as avg_value FROM `%s`", key, con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	row := con.client.QueryRow(query)

	var result float64
	err := row.Scan(&result)
	if err != nil {
		logger.Error("mysqlConnection.average", err.Error())
		return 0
	}

	return result
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

func (con *mysqlConnection) graph() *datatype.DataMap {
	return &datatype.DataMap{}
}

func (con *mysqlConnection) create(data datatype.DataMap) (*datatype.DataMap, error) {
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
		console.Log("mysqlConnection.create", err.Error())
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		console.Log("mysqlConnection.create", err.Error())
		return nil, err
	}

	// Retrieve the created record
	createdRecord := newMySQLInstance(con).query.Where("id", id).collection().findOne()
	return createdRecord, nil
}

func (con *mysqlConnection) createMany(data []datatype.DataMap) (*[]datatype.DataMap, error) {
	if len(data) == 0 {
		return nil, errors.New("no data provided")
	}

	// Build multi-row insert
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
		console.Log("mysqlConnection.createMany", err.Error())
		return nil, err
	}

	// Retrieve all created records
	result := newMySQLInstance(con).query.collection().find()
	return result, nil
}

func (con *mysqlConnection) update(data datatype.DataMap) (*datatype.DataMap, error) {
	setParts := make([]string, 0)
	values := make([]interface{}, 0)

	for k, v := range data {
		setParts = append(setParts, fmt.Sprintf("`%s` = ?", k))
		values = append(values, v)
	}

	query := fmt.Sprintf("UPDATE `%s` SET %s", con.query.Model.Collection, strings.Join(setParts, ", "))

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
		// Add where values to the values slice
		whereVals := con.getWhereValues()
		values = append(values, whereVals...)
	}

	result, err := con.client.Exec(query, values...)
	if err != nil {
		console.Log("mysqlConnection.update", err.Error())
		return nil, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return nil, err
	}

	// Retrieve the updated record
	updatedRecord := newMySQLInstance(con).query.WhereAll(*con.where()).FindOne(nil)
	return updatedRecord, nil
}

func (con *mysqlConnection) updateMany(data datatype.DataMap) (*[]datatype.DataMap, error) {
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
		console.Log("mysqlConnection.updateMany", err.Error())
		return nil, err
	}

	// Retrieve the updated records
	result := newMySQLInstance(con).query.WhereAll(*con.where()).Find(nil)
	return result, nil
}

func (con *mysqlConnection) delete() (interface{}, error) {
	if !con.hasWhere() {
		return nil, errors.New("filter is empty, not allowed to delete all records at once")
	}

	query := fmt.Sprintf("DELETE FROM `%s` WHERE %s", con.query.Model.Collection, con.whereSQL())
	whereVals := con.getWhereValues()

	result, err := con.client.Exec(query, whereVals...)
	if err != nil {
		console.Log("mysqlConnection.delete", err.Error())
		return nil, err
	}

	return result, nil
}

func (con *mysqlConnection) selection() *[]string {
	return &[]string{}
}

func (con *mysqlConnection) where() *datatype.DataMap {
	return &con.query.where
}

func (con *mysqlConnection) whereSQL() string {
	conditions := make([]string, 0)

	for k, v := range con.query.where {
		if helper.Contains([]string{"AND", "OR", "NOR"}, k) {
			// Handle complex where conditions
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

func (con *mysqlConnection) extractWhereSQL(where interface{}) string {
	// Implementation for complex where conditions
	return ""
}

func (con *mysqlConnection) extractWhereItemSQL(key string, value interface{}) string {
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

func (con *mysqlConnection) getWhereValues() []interface{} {
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

func (con *mysqlConnection) hasWhere() bool {
	return len(con.query.where) > 0
}

func (con *mysqlConnection) hasOrderBy() bool {
	return len(con.query.orderBy) > 0
}

func (con *mysqlConnection) buildSelectQuery() string {
	query := fmt.Sprintf("SELECT * FROM `%s`", con.query.Model.Collection)

	if con.hasWhere() {
		query += " WHERE " + con.whereSQL()
	}

	return query
}

func (con *mysqlConnection) orderBySQL() string {
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

func (con *mysqlConnection) page() int {
	if con.query.page < 1 {
		return 1
	}

	return con.query.page
}
