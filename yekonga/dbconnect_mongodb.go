package yekonga

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/console"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/bson"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/mongo"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/mongo/options"
)

type mongodbConnection struct {
	query  *DataModelQuery
	ctx    *context.Context
	mut    sync.RWMutex
	client *mongo.Client
}

func newMongodbInstance(con *mongodbConnection) mongodbConnection {
	return mongodbConnection{
		ctx:    con.ctx,
		client: con.client,
		query: &DataModelQuery{
			Model:          con.query.Model,
			RequestContext: con.query.RequestContext,
			QueryContext:   con.query.QueryContext,
		},
	}
}

func (con *mongodbConnection) collection() *mongo.Collection {

	// Get collection
	collection := con.client.
		Database(con.query.Model.Config.Database.DatabaseName).
		Collection(con.query.Model.Collection)

	return collection
}

func (con *mongodbConnection) findOne() *datatype.DataMap {

	var result datatype.DataMap

	opts := options.FindOne()

	if con.hasOrderBy() {
		opts = opts.SetSort(con.orderBy())
	}

	res := con.collection().FindOne(context.TODO(), con.where(), opts)
	err := res.Decode(&result)
	if err != nil {
		// logger.Error("mongodbConnection.findOne", err.Error())
	} else if result != nil {
		result["id"] = result["_id"]
		result["_collection"] = con.query.Model.Collection
		result["_model"] = con.query.Model.Name
	}

	return &result
}

func (con *mongodbConnection) findAll() *[]datatype.DataMap {
	return con.find()
}

func (con *mongodbConnection) find() *[]datatype.DataMap {
	var cursor *mongo.Cursor
	var err error

	if con.hasGroup() {
		opts := options.Aggregate()
		// Aggregation pipeline to sum the "amount" field
		pipeline := mongo.Pipeline{}

		if con.where() != nil {
			pipeline = append(pipeline, bson.D{{Key: "$match", Value: con.where()}})
		}

		pipeline = append(pipeline, bson.D{{Key: "$group", Value: con.groupBy()}})

		if con.hasOrderBy() {
			pipeline = append(pipeline, bson.D{{Key: "$sort", Value: con.orderBy()}})
		}

		if con.hasProjection() {
			pipeline = append(pipeline, bson.D{{Key: "$project", Value: con.projection()}})
		}

		if con.skip() > 0 {
			pipeline = append(pipeline, bson.D{{Key: "$skip", Value: int64(con.skip())}})
		}

		if con.limit() > 0 {
			pipeline = append(pipeline, bson.D{{Key: "$limit", Value: int64(con.limit())}})
		}

		// console.Log("mongodbConnection.find", "Pipeline: %v", pipeline)

		cursor, err = con.collection().Aggregate(context.TODO(), pipeline, opts)
	} else {

		// Find with limit and skip
		opts := options.Find()

		if con.limit() > 0 {
			opts = opts.SetLimit(int64(con.limit()))
		}

		if con.skip() > 0 {
			opts = opts.SetSkip(int64(con.skip()))
		}

		if con.hasOrderBy() {
			opts = opts.SetSort(con.orderBy())
		}

		cursor, err = con.collection().Find(context.TODO(), con.where(), opts)
	}

	if err != nil {
		logger.Error("mongodbConnection.find 1", err.Error())
	}

	defer cursor.Close(context.TODO())

	if err := cursor.Err(); err != nil {
		logger.Error("mongodbConnection.find 2", err.Error())
	} else {
		result := make([]datatype.DataMap, 0, cursor.RemainingBatchLength())

		// cursor.All(context.TODO(), &result)
		for cursor.Next(context.TODO()) {
			// To decode into a struct, use cursor.Decode()
			var data datatype.DataMap
			err := cursor.Decode(&data)
			if err != nil {
				logger.Error("mongodbConnection.find 3", err.Error())
			} else if data != nil {
				data["id"] = data["_id"]
				data["_collection"] = con.query.Model.Collection
				data["_model"] = con.query.Model.Name

				if con.hasGroup() {
					if v, ok := data["_id"].(bson.D); ok {
						for _, vi := range v {
							data[vi.Key] = vi.Value
						}
					}
				}
				result = append(result, data)
			}
		}
		// console.Log("mongodbConnection.find", "Found %d documents", result)

		return &result
	}

	return &[]datatype.DataMap{}
}

func (con *mongodbConnection) pagination() *datatype.DataMap {
	var lastPage int64
	total := con.count()
	perPage := con.limit()
	currentPage := con.page()
	from := (perPage * (currentPage - 1)) + 1
	to := perPage * (currentPage)
	remainder := total % int64(perPage)

	if remainder == 0 {
		lastPage = (total) / int64(perPage)
	} else {
		lastPage = (total + (int64(perPage) - total%int64(perPage))) / int64(perPage)
	}

	con.query.Take(perPage)

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

func (con *mongodbConnection) summary() *datatype.DataMap {

	result := datatype.DataMap{
		"count": 0,
		"sum":   0,
		"max":   0,
		"min":   0,
		"graph": datatype.DataMap{},
	}

	return &result
}

func (con *mongodbConnection) count() int64 {
	var cursor int64
	var err error

	if len(con.query.distinct) > 0 {
		opts := options.Aggregate()

		// Aggregation pipeline to sum the "amount" field
		groupId := bson.M{}
		for _, k := range con.query.distinct {
			groupId[k] = "$" + k
		}
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: con.where()}},
			{{Key: "$group", Value: bson.M{"_id": groupId, "aggregateValue": bson.M{"$sum": 1}}}},
		}

		cursorResult, err := con.collection().Aggregate(context.TODO(), pipeline, opts)
		if err != nil {
			logger.Error("mongodbConnection.count 1", err.Error())
		}
		defer cursorResult.Close(context.TODO())

		cursor = int64(cursorResult.RemainingBatchLength())
	} else {
		cursor, err = con.collection().CountDocuments(context.TODO(), con.where())
		if err != nil {
			logger.Error("mongodbConnection.count", err.Error())
		}
	}

	return cursor
}

func (con *mongodbConnection) max(key string) interface{} {
	opts := options.Aggregate()

	// Aggregation pipeline to sum the "amount" field
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: con.where()}},
		{{Key: "$group", Value: bson.M{"_id": nil, "aggregateValue": bson.M{"$max": "$" + key}}}},
	}

	cursor, err := con.collection().Aggregate(context.TODO(), pipeline, opts)
	if err != nil {
		logger.Error("mongodbConnection.max 1", err.Error())
	}
	defer cursor.Close(context.TODO())

	// Retrieve the result
	var result struct {
		AggregateValue interface{} `bson:"aggregateValue"`
	}
	if cursor.Next(context.TODO()) {
		if err := cursor.Decode(&result); err != nil {
			logger.Error("mongodbConnection.max 2", err.Error())
		}

		switch v := result.AggregateValue.(type) {
		case float64:
			return v
		case string:
			parsedTime, err := time.Parse(time.RFC3339, v) // Convert string to time.Time
			if err != nil {
				return nil
			}
			return parsedTime
		case bson.DateTime:
			return v.Time() // Convert BSON DateTime to Go time.Time
		default:
			logger.Error(fmt.Errorf("unexpected type: %s", reflect.TypeOf(result.AggregateValue)))
			return nil
		}
	} else {
		fmt.Println("No data found")
	}

	return result.AggregateValue
}

func (con *mongodbConnection) min(key string) interface{} {

	opts := options.Aggregate()

	// Aggregation pipeline to sum the "amount" field
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: con.where()}},
		{{Key: "$group", Value: bson.M{"_id": nil, "aggregateValue": bson.M{"$min": "$" + key}}}},
	}

	cursor, err := con.collection().Aggregate(context.TODO(), pipeline, opts)
	if err != nil {
		logger.Error("mongodbConnection.min 1", err.Error())
	}
	defer cursor.Close(context.TODO())

	// Retrieve the result
	var result struct {
		AggregateValue interface{} `bson:"aggregateValue"`
	}
	if cursor.Next(context.TODO()) {

		if err := cursor.Decode(&result); err != nil {
			logger.Error("mongodbConnection.min 2", err.Error())
		}

		switch v := result.AggregateValue.(type) {
		case float64:
			return v
		case string:
			parsedTime, err := time.Parse(time.RFC3339, v) // Convert string to time.Time
			if err != nil {
				return nil
			}
			return parsedTime
		case bson.DateTime:
			return v.Time() // Convert BSON DateTime to Go time.Time
		default:
			logger.Error(fmt.Errorf("unexpected type: %s", reflect.TypeOf(result.AggregateValue)))
			return nil
		}
	} else {
		fmt.Println("No data found")
	}

	return result.AggregateValue
}

func (con *mongodbConnection) sum(key string) float64 {
	// Find with limit and skip
	opts := options.Aggregate()

	// Aggregation pipeline to sum the "amount" field
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: con.where()}},
		{{Key: "$group", Value: bson.M{"_id": nil, "AggregateValue": bson.M{"$sum": "$" + key}}}},
	}

	cursor, err := con.collection().Aggregate(context.TODO(), pipeline, opts)
	if err != nil {
		logger.Error("mongodbConnection.sum 1", err.Error())
	}
	defer cursor.Close(context.TODO())

	// Retrieve the result
	var result struct {
		AggregateValue float64 `bson:"aggregateValue"`
	}
	if cursor.Next(context.TODO()) {
		if err := cursor.Decode(&result); err != nil {
			logger.Error("mongodbConnection.sum 2", err.Error())
		}
	} else {
		fmt.Println("No data found")
	}

	return result.AggregateValue
}

func (con *mongodbConnection) average(key string) float64 {
	// Find with limit and skip
	opts := options.Aggregate()

	// Aggregation pipeline to sum the "amount" field
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: con.where()}},
		{{Key: "$group", Value: bson.M{"_id": nil, "aggregateValue": bson.M{"$avg": "$" + key}}}},
	}

	cursor, err := con.collection().Aggregate(context.TODO(), pipeline, opts)
	if err != nil {
		logger.Error("mongodbConnection.average 1", err.Error())
	}
	defer cursor.Close(context.TODO())

	// Retrieve the result
	var result struct {
		AggregateValue float64 `bson:"aggregateValue"`
	}
	if cursor.Next(context.TODO()) {
		if err := cursor.Decode(&result); err != nil {
			logger.Error("mongodbConnection.average 2", err.Error())
		}
	} else {
		fmt.Println("No data found")
	}

	return result.AggregateValue
}

func (con *mongodbConnection) graph() *datatype.DataMap {
	return &datatype.DataMap{}
}

func (con *mongodbConnection) create(data datatype.DataMap) (*datatype.DataMap, error) {
	var result *datatype.DataMap
	res, err := con.collection().InsertOne(*con.ctx, data)

	if err != nil {
		console.Log("mongodbConnection.create", err.Error())
		return nil, err
	}

	if res.Acknowledged {
		result = newMongodbInstance(con).query.Where("_id", res.InsertedID).collection().findOne()
	}

	return result, nil
}

func (con *mongodbConnection) createMany(data []datatype.DataMap) (*[]datatype.DataMap, error) {
	var result *[]datatype.DataMap
	res, err := con.collection().InsertMany(*con.ctx, data)

	if err != nil {
		console.Log("mongodbConnection.createMany", err.Error())
		return nil, err
	}

	if res.Acknowledged {
		where := map[string]interface{}{"_id": map[string]interface{}{"in": res.InsertedIDs}}
		// console.Log("ssss", res.InsertedIDs)
		result = newMongodbInstance(con).query.WhereAll(where).collection().find()
	}

	return result, nil
}

func (con *mongodbConnection) update(data datatype.DataMap) (*datatype.DataMap, error) {
	var result *datatype.DataMap
	res, err := con.collection().UpdateOne(*con.ctx, con.where(), datatype.DataMap{
		"$set": data,
	})

	if err != nil {
		console.Log("mongodbConnection.update", err.Error())
		return nil, err
	}

	if res.Acknowledged && res.ModifiedCount > 0 {
		result = newMongodbInstance(con).query.WhereAll(con.where()).FindOne(nil)
	}

	return result, nil
}

func (con *mongodbConnection) updateMany(data datatype.DataMap) (*[]datatype.DataMap, error) {
	var result *[]datatype.DataMap
	res, err := con.collection().UpdateMany(*con.ctx, con.where(), datatype.DataMap{
		"$set": data,
	})

	if err != nil {
		console.Log("mongodbConnection.updateMany", err.Error())
		return nil, err
	}

	if res.Acknowledged {
		result = newMongodbInstance(con).query.WhereAll(con.where()).Find(nil)
	}

	return result, nil
}

func (con *mongodbConnection) delete() (interface{}, error) {
	where := con.where()

	if helper.IsNotEmpty(*where) {
		res, err := con.collection().DeleteMany(*con.ctx, where)
		if err != nil {
			console.Log("mongodbConnection.delete", err.Error())
			return nil, err
		}
		return res, nil
	}

	return nil, errors.New("filter is empty, not allowed to delete all document at once")
}

func (con *mongodbConnection) selection() *[]string {
	return &[]string{}
}

func (con *mongodbConnection) where() *datatype.DataMap {
	var filters = datatype.DataMap{}

	for k := range con.query.where {
		if helper.Contains([]string{"AND", "OR", "NOR"}, k) {
			vs := con.extractWhere(con.query.where)

			for ki, vi := range vs {
				filters[ki] = vi
			}
		} else {
			vs := con.extractWhereObject(con.query.where)

			for ki, vi := range vs {
				filters[ki] = vi
			}
		}
	}

	return &filters
}

func (con *mongodbConnection) extractWhere(where interface{}) datatype.DataMap {
	var filters = datatype.DataMap{}

	if whr, ok := where.(datatype.DataMap); ok {
		for k, v := range whr {
			switch k {
			case "AND":
				var newValue = []interface{}{}
				var vi = helper.ToList[datatype.DataMap](v)

				for ii := range vi {
					newValue = append(newValue, con.extractWhereObject(vi[ii]))
				}

				if len(newValue) > 0 {
					filters["$and"] = newValue
				}
			case "OR":
				var newValue = []interface{}{}
				var vi = helper.ToList[datatype.DataMap](v)

				for ii := range vi {
					newValue = append(newValue, con.extractWhereObject(vi[ii]))
				}

				if len(newValue) > 0 {
					filters["$or"] = newValue
				}
			case "NOR":
				var newValue = []interface{}{}
				var vi = helper.ToList[datatype.DataMap](v)

				for ii := range vi {
					newValue = append(newValue, con.extractWhereObject(vi[ii]))
				}

				if len(newValue) > 0 {
					filters["$nor"] = newValue
				}
			default:
				var newValue = []interface{}{}
				var vi = helper.ToList[datatype.DataMap](v)

				for ii := range vi {
					newValue = append(newValue, con.extractWhereObject(vi[ii]))
				}

				if len(newValue) > 0 {
					filters["$and"] = newValue
				}
			}
		}
	}

	return filters
}

func (con *mongodbConnection) extractWhereObject(where interface{}) datatype.DataMap {
	var filters = datatype.DataMap{}
	// var localFilter = helper.ToMap(where)

	if whr, ok := where.(datatype.DataMap); ok {
		for k, v := range whr {
			if helper.Contains([]string{"AND", "OR", "NOR"}, k) {
				vs := con.extractWhere(whr)

				for ki, vi := range vs {
					filters[ki] = vi
				}
			} else if _, ok := v.(map[string]interface{}); ok {
				vs := con.extractWhereItem(whr)

				for ki, vi := range vs {
					filters[ki] = vi
				}
			} else {
				vs := datatype.DataMap{
					k: datatype.DataMap{
						"$eq": v,
					},
				}

				for ki, vi := range vs {
					filters[ki] = vi
				}
			}
		}
	}

	return filters
}

func (con *mongodbConnection) extractWhereItem(where interface{}) datatype.DataMap {
	var filters = make(datatype.DataMap)

	if whr, ok := where.(datatype.DataMap); ok {
		for k, v := range whr {
			if vi, ok := v.(map[string]interface{}); ok {
				for ki, vii := range vi {
					innerFilter := datatype.DataMap{}
					if inf, ok := filters[k].(datatype.DataMap); ok {
						innerFilter = inf
					}

					if helper.Contains(graphqlOperations[:], ki) ||
						helper.Contains(graphqlArrayOperations[:], ki) ||
						helper.Contains(graphqlBooleanOperations[:], ki) ||
						helper.Contains(mongodbSpecialOperations[:], ki) {

						switch vii {
						case string(NULLValue):
							vii = nil
						case string(NullValue):
							vii = nil
						case string(nullValue):
							vii = nil
						}

						switch ki {
						case "equalTo", "$eq":
							innerFilter["$eq"] = vii
						case "notEqualTo", "$ne":
							innerFilter["$ne"] = vii
						case "lessThan", "$lt":
							viii := helper.ConvertCalculatedValue(vii)
							innerFilter["$lt"] = viii
						case "notLessThan", "$not_lt":
							viii := helper.ConvertCalculatedValue(vii)
							innerFilter["$not"] = datatype.DataMap{
								"$lt": viii,
							}
						case "lessThanOrEqualTo", "$lte":
							viii := helper.ConvertCalculatedValue(vii)
							innerFilter["$lte"] = viii
						case "notLessThanOrEqualTo", "$not_lte":
							viii := helper.ConvertCalculatedValue(vii)
							innerFilter["$not"] = datatype.DataMap{
								"$lte": viii,
							}
						case "greaterThan", "$gt":
							viii := helper.ConvertCalculatedValue(vii)
							innerFilter["$gt"] = viii
						case "notGreaterThan", "$not_gt":
							viii := helper.ConvertCalculatedValue(vii)
							innerFilter["$not"] = datatype.DataMap{
								"$gt": viii,
							}
						case "greaterThanOrEqualTo", "$gte":
							viii := helper.ConvertCalculatedValue(vii)
							innerFilter["$gte"] = viii
						case "notGreaterThanOrEqualTo", "$not_gte":
							viii := helper.ConvertCalculatedValue(vii)
							innerFilter["$not"] = datatype.DataMap{
								"$gte": viii,
							}
						case "in", "$in":
							innerFilter["$in"] = vii
						case "all", "$all":
							innerFilter["$all"] = vii
						case "notIn", "$nin":
							innerFilter["$nin"] = vii
						case "exists":
							if exists, ok := vii.(bool); ok {
								if exists {
									innerFilter["$exists"] = true
									innerFilter["$nin"] = []interface{}{nil}
								} else {
									filters["$or"] = []datatype.DataMap{
										map[string]any{
											k: datatype.DataMap{
												"$exists": false,
											},
										},
										map[string]any{
											k: datatype.DataMap{
												"$nin": []interface{}{nil},
											},
										},
									}
								}
							}
						case "matchesRegex":
							if _v, ok := vii.(string); ok {
								innerFilter["$regex"] = regexp.MustCompile(_v).String()
							}
						case "options":
							innerFilter["$eq"] = vii
						case "text":
							innerFilter["$eq"] = vii
						case "inQueryKey":
							innerFilter["$eq"] = vii
						case "notInQueryKey":
							innerFilter["$eq"] = vii
						default:
							innerFilter[ki] = vii
						}

						filters[k] = innerFilter
					} else {
						if viii, ok := con.query.Model.ParentFields[k]; ok {
							// if innerFilter[viii.PrimaryKey] == nil {
							// 	// innerFilter[viii.PrimaryKey] = datatype.DataMap{}
							// }

							list := con.query.Model.App.ModelQuery(viii.ModelName).WhereAll(vi).Find(nil)
							ids := helper.GetList(list, viii.PrimaryKey)

							objectIDs := []bson.ObjectID{}
							for _, v := range ids {
								objectIDs = append(objectIDs, helper.ObjectID(v))
							}

							innerFilter[viii.ForeignKey] = datatype.DataMap{
								"$in": objectIDs,
							}
							// fmt.Println("ids.p", ids)
						} else if viii, ok := con.query.Model.ChildrenFields[k]; ok {
							// if filters[viii.PrimaryKey] == nil {
							// 	filters[viii.PrimaryKey] = datatype.DataMap{}
							// }

							list := con.query.Model.App.ModelQuery(viii.ModelName).WhereAll(vi).Find(nil)
							ids := helper.GetList(list, viii.ForeignKey)
							objectIDs := []bson.ObjectID{}
							for _, v := range ids {
								objectIDs = append(objectIDs, helper.ObjectID(v))
							}

							innerFilter[viii.PrimaryKey] = datatype.DataMap{
								"$in": objectIDs,
							}
							// console.Log("ids.c", ids)
						}

						for k, v := range innerFilter {
							filters[k] = v
						}
					}
				}
			} else {
				innerFilter := datatype.DataMap{}
				if inf, ok := filters[k].(datatype.DataMap); ok {
					innerFilter = inf
				}

				if vi, ok := v.([]interface{}); ok {
					innerFilter["$in"] = vi
				} else {
					innerFilter["$eq"] = v
				}

				filters[k] = innerFilter
			}
		}
	}

	if con.query.Model.Name == "Project" || con.query.Model.Name == "Client" {
		console.Log("filters", filters)
	}

	return filters
}

func (con *mongodbConnection) hasGroup() bool {
	if len(con.query.groupBy) > 0 {
		return true
	}

	if len(con.query.groupByRaw) > 0 {
		return true
	}

	return false
}

func (con *mongodbConnection) groupBy() bson.M {
	values := bson.M{}

	if con.hasGroup() {
		subValues := bson.M{}

		for _, v := range con.query.groupBy {
			subValues[v] = "$" + v
		}

		for k, v := range con.query.groupByRaw {
			subValues[k] = v
		}

		if _, ok := subValues["_id"]; !ok {
			values["_id"] = subValues
		} else {
			values = subValues
		}
	}

	return values
}

func (con *mongodbConnection) hasOrderBy() bool {
	if len(con.query.orderBy) > 0 {
		return true
	}

	return false
}

func (con *mongodbConnection) orderBy() interface{} {
	order := make(datatype.DataMap)

	if con.query.orderBy != nil {
		for k, v := range con.query.orderBy {
			order[k] = 1
			if strings.ToLower(v) == "desc" {
				order[k] = -1
			}
		}
	}

	return order
}

func (con *mongodbConnection) hasProjection() bool {
	if len(con.query.selection) > 0 {
		return true
	}

	return true
}

func (con *mongodbConnection) projection() interface{} {
	selectFields := make(datatype.DataMap)

	selectFields["__v"] = 0 // Always include the _id field

	if len(con.query.groupBy) > 0 {
		// selectFields["__v"] = 1
		// for _, k := range con.query.groupBy {
		// 	// selectFields[k] = 1
		// }
	} else if len(con.query.selection) > 0 {
		// selectFields["__v"] = 1
		// for _, k := range con.query.selection {
		// 	selectFields[k] = 1
		// }
	}

	return selectFields
}

func (con *mongodbConnection) limit() int {
	if con.query.limit > 0 {
		return con.query.limit
	}

	return -1
}

func (con *mongodbConnection) skip() int {
	if con.query.skip > 0 {
		return con.query.skip
	}

	return con.query.limit * (con.page() - 1)
}

func (con *mongodbConnection) page() int {
	if con.query.page < 1 {
		return 1
	}

	return con.query.page
}
