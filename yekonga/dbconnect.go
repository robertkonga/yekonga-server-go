package yekonga

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
	localDB "github.com/robertkonga/yekonga-server-go/plugins/database/db"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/mongo"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/mongo/options"
)

var graphqlOperations = []string{
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

var graphqlArrayOperations = []string{
	"in",
	"all",
	"notIn",
}

var graphqlBooleanOperations = []string{
	"exists",
}

var mongodbSpecialOperations = []string{
	"$type",
}

type dataModelQueryStructure interface {
	findOne() *datatype.DataMap
	findAll() *[]datatype.DataMap
	find() *[]datatype.DataMap
	pagination() *datatype.DataMap
	summary() *datatype.DataMap
	count() int64
	max(string) interface{}
	min(string) interface{}
	sum(string) float64
	average(string) float64
	graph() *datatype.DataMap

	create(data datatype.DataMap) (*datatype.DataMap, error)
	createMany(data []datatype.DataMap) (*[]datatype.DataMap, error)
	update(data datatype.DataMap) (*datatype.DataMap, error)
	updateMany(data datatype.DataMap) (*[]datatype.DataMap, error)
	delete() (interface{}, error)
}

type DatabaseConnections struct {
	config        *config.YekongaConfig
	appPath       string
	mongodbClient *mongo.Client
	localClient   *localDB.DB
	mysqlClient   *sql.DB
	sqlClient     *sql.DB
}

func NewDatabaseConnections(config *config.YekongaConfig) *DatabaseConnections {
	dc := &DatabaseConnections{
		config: config,
	}

	return dc
}

func (dc *DatabaseConnections) connect() {
	if dc.config.Database.Kind == config.DBTypeMongodb {
		dc.mongodbConnect()
	} else if dc.config.Database.Kind == config.DBTypeMysql {
		dc.mysqlConnect()
	} else if dc.config.Database.Kind == config.DBTypeSql {
		dc.sqlConnect()
	} else {
		dc.localConnect()
	}

}

func (dc *DatabaseConnections) close() {
	if dc.config.Database.Kind == config.DBTypeMongodb {
		dc.mongodbClose()
	} else if dc.config.Database.Kind == config.DBTypeMysql {
		dc.mysqlClose()
	} else if dc.config.Database.Kind == config.DBTypeSql {
		dc.sqlClose()
	} else {
		dc.localClose()
	}

}

func (dc *DatabaseConnections) mongodbConnect() {
	// // Set MongoDB URI
	srv := ""
	if dc.config.Database.Srv {
		srv = "+srv"
	}

	connectionUrl := ""
	if helper.IsEmpty(dc.config.Database.Port) || string(dc.config.Database.Port) == "80" {
		connectionUrl = fmt.Sprintf(
			"mongodb%s://%v",
			srv,
			dc.config.Database.Host,
		)
	} else {
		connectionUrl = fmt.Sprintf(
			"mongodb%s://%v:%v",
			srv,
			dc.config.Database.Host,
			dc.config.Database.Port,
		)
	}

	// logger.Info("connectionUrl", connectionUrl)
	clientOptions := options.Client().ApplyURI(connectionUrl)

	if dc.config.Database.Username != nil {
		// logger.Error(dc.config.Database)
		var username string
		var password string

		if v, ok := dc.config.Database.Username.(string); ok {
			username = v
		}

		if v, ok := dc.config.Database.Password.(string); ok {
			password = v
		}

		credential := options.Credential{
			// AuthMechanism: "PLAIN",
			Username: username,
			Password: password,
		}
		clientOptions.SetAuth(credential)
	}

	client, err := mongo.Connect(clientOptions)

	if err != nil {
		logger.Error("Could not connect to MongoDB", err, client)
	} else {
		logger.Success("Connected to MongoDB!")
	}

	dc.mongodbClient = client
}

func (dc *DatabaseConnections) localConnect() {
	dbPath := dc.appPath + string(os.PathSeparator) + "database"
	client, err := localDB.OpenDB(dbPath)

	if err != nil {
		logger.Error("Could not connect to LocalDatabase", err)
	} else {
		logger.Success("Connected to LocalDatabase!")
	}

	dc.localClient = client
}

func (dc *DatabaseConnections) mysqlConnect() {
}

func (dc *DatabaseConnections) sqlConnect() {
}

func (dc *DatabaseConnections) mongodbClose() {
	if err := dc.mongodbClient.Disconnect(context.TODO()); err != nil {
		logger.Warn("Mongodb disconnected")
	}
}

func (dc *DatabaseConnections) localClose() {
}

func (dc *DatabaseConnections) mysqlClose() {
}

func (dc *DatabaseConnections) sqlClose() {
}
