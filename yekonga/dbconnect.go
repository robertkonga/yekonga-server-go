package yekonga

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
	localDB "github.com/robertkonga/yekonga-server-go/plugins/database/db"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/mongo"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/mongo/options"
	"github.com/robertkonga/yekonga-server-go/plugins/mysql"
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
	settings      config.DatabaseSettings
	appPath       string
	mongodbClient *mongo.Client
	localClient   *localDB.DB
	mysqlClient   *sql.DB
	sqlClient     *sql.DB
}

func NewDatabaseConnections(config *config.YekongaConfig) *DatabaseConnections {
	return newDatabaseConnections(config, config.Database)
}

// newDatabaseConnections connects to settings' database instead of the
// config's.
func newDatabaseConnections(config *config.YekongaConfig, settings config.DatabaseSettings) *DatabaseConnections {
	dc := &DatabaseConnections{
		config:   config,
		settings: settings,
	}

	return dc
}

// databaseConnectionKey identifies the connection settings need; settings
// with the same key share one connection.
func databaseConnectionKey(cfg *config.YekongaConfig, settings config.DatabaseSettings) string {
	switch settings.Kind {
	case config.DBTypeMongodb:
		// One client serves every database on the server.
		settings.DatabaseName = ""
	case config.DBTypeMysql, config.DBTypeSql:
	default:
		return "local:" + localDatabaseDirectory(cfg, settings)
	}

	key, _ := json.Marshal(settings)

	return string(key)
}

// localDatabaseDirectory is the directory of a local database: "database"
// for the config's database, "database_<name>" for any other.
func localDatabaseDirectory(cfg *config.YekongaConfig, settings config.DatabaseSettings) string {
	if settings.DatabaseName == "" || settings.DatabaseName == cfg.Database.DatabaseName {
		return "database"
	}

	return "database_" + helper.ToSlug(settings.DatabaseName)
}

// databaseConnection is the connection for settings, created on first use.
// Called while the server is being set up, before any request.
func (y *YekongaData) databaseConnection(settings config.DatabaseSettings) *DatabaseConnections {
	key := databaseConnectionKey(y.Config, settings)
	if dc, ok := y.dbConnects[key]; ok {
		return dc
	}

	dc := newDatabaseConnections(y.Config, settings)
	y.dbConnects[key] = dc

	return dc
}

func (dc *DatabaseConnections) connect() {
	if dc.settings.Kind == config.DBTypeMongodb {
		dc.mongodbConnect()
	} else if dc.settings.Kind == config.DBTypeMysql {
		dc.mysqlConnect()
	} else if dc.settings.Kind == config.DBTypeSql {
		dc.sqlConnect()
	} else {
		dc.localConnect()
	}

}

func (dc *DatabaseConnections) close() {
	if dc.settings.Kind == config.DBTypeMongodb {
		dc.mongodbClose()
	} else if dc.settings.Kind == config.DBTypeMysql {
		dc.mysqlClose()
	} else if dc.settings.Kind == config.DBTypeSql {
		dc.sqlClose()
	} else {
		dc.localClose()
	}

}

func (dc *DatabaseConnections) mongodbConnect() {
	// // Set MongoDB URI
	srv := ""
	if dc.settings.Srv {
		srv = "+srv"
	}

	connectionUrl := ""
	if helper.IsEmpty(dc.settings.Port) || string(dc.settings.Port) == "80" {
		connectionUrl = fmt.Sprintf(
			"mongodb%s://%v",
			srv,
			dc.settings.Host,
		)
	} else {
		connectionUrl = fmt.Sprintf(
			"mongodb%s://%v:%v",
			srv,
			dc.settings.Host,
			dc.settings.Port,
		)
	}

	// logger.Info("connectionUrl", connectionUrl)
	clientOptions := options.Client().ApplyURI(connectionUrl)

	db := dc.settings
	if db.MaxPoolSize > 0 {
		clientOptions.SetMaxPoolSize(uint64(db.MaxPoolSize))
	}
	if db.MinPoolSize > 0 {
		clientOptions.SetMinPoolSize(uint64(db.MinPoolSize))
	}
	if db.MaxConnIdleTimeSeconds > 0 {
		clientOptions.SetMaxConnIdleTime(time.Duration(db.MaxConnIdleTimeSeconds) * time.Second)
	}
	if db.ConnectTimeoutSeconds > 0 {
		clientOptions.SetConnectTimeout(time.Duration(db.ConnectTimeoutSeconds) * time.Second)
	}
	if db.ServerSelectionTimeoutSeconds > 0 {
		clientOptions.SetServerSelectionTimeout(time.Duration(db.ServerSelectionTimeoutSeconds) * time.Second)
	}
	if db.QueryTimeoutSeconds > 0 {
		// Applies to every operation whose context has no deadline of its own.
		clientOptions.SetTimeout(time.Duration(db.QueryTimeoutSeconds) * time.Second)
	}

	if dc.settings.Username != nil {
		// logger.Error(dc.settings)
		var username string
		var password string

		if v, ok := dc.settings.Username.(string); ok {
			username = v
		}

		if v, ok := dc.settings.Password.(string); ok {
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
	dbPath := dc.appPath + string(os.PathSeparator) + localDatabaseDirectory(dc.config, dc.settings)
	client, err := localDB.OpenDB(dbPath)

	if err != nil {
		logger.Error("Could not connect to LocalDatabase", err)
	} else {
		logger.Success("Connected to LocalDatabase!")
	}

	dc.localClient = client
}

func (dc *DatabaseConnections) mysqlConnect() {
	dc.mysqlClient = dc.openSQL("MySQL")
}

func (dc *DatabaseConnections) sqlConnect() {
	dc.sqlClient = dc.openSQL("SQL")
}

// openSQL opens a pool to a MySQL-compatible server; both SQL backends write
// MySQL syntax. Pool settings reuse the database.* options the MongoDB
// connection uses.
func (dc *DatabaseConnections) openSQL(label string) *sql.DB {
	db := dc.settings

	port := string(db.Port)
	if port == "" {
		port = "3306"
	}

	cfg := mysql.NewConfig()
	cfg.Net = "tcp"
	cfg.Addr = db.Host + ":" + port
	cfg.DBName = db.DatabaseName
	cfg.ParseTime = true
	// Arguments are escaped client-side instead of preparing each statement,
	// which would cost an extra round trip per query.
	cfg.InterpolateParams = true
	if v, ok := db.Username.(string); ok {
		cfg.User = v
	}
	if v, ok := db.Password.(string); ok {
		cfg.Passwd = v
	}

	client, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		logger.Error("Could not connect to "+label, err)
		return nil
	}

	if db.MaxPoolSize > 0 {
		client.SetMaxOpenConns(db.MaxPoolSize)
	}
	if db.MinPoolSize > 0 {
		client.SetMaxIdleConns(db.MinPoolSize)
	}
	if db.MaxConnIdleTimeSeconds > 0 {
		client.SetConnMaxIdleTime(time.Duration(db.MaxConnIdleTimeSeconds) * time.Second)
	}

	timeout := 10 * time.Second
	if db.ConnectTimeoutSeconds > 0 {
		timeout = time.Duration(db.ConnectTimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// sql.Open doesn't connect; check the server is reachable now rather
	// than on the first query. The pool keeps retrying either way.
	if err := client.PingContext(ctx); err != nil {
		logger.Error("Could not connect to "+label, err)
	} else {
		logger.Success("Connected to " + label + "!")
	}

	return client
}

func (dc *DatabaseConnections) mongodbClose() {
	if err := dc.mongodbClient.Disconnect(context.TODO()); err != nil {
		logger.Warn("Mongodb disconnected")
	}
}

func (dc *DatabaseConnections) localClose() {
}

func (dc *DatabaseConnections) mysqlClose() {
	if dc.mysqlClient != nil {
		dc.mysqlClient.Close()
	}
}

func (dc *DatabaseConnections) sqlClose() {
	if dc.sqlClient != nil {
		dc.sqlClient.Close()
	}
}
