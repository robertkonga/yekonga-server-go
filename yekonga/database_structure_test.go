package yekonga

import (
	"testing"

	mainConfig "github.com/robertkonga/yekonga-server-go/config"
)

func TestMultipleDatabaseStructureConnections(t *testing.T) {
	var cfg mainConfig.YekongaConfig
	cfg.Database.Kind = mainConfig.DBTypeMongodb
	cfg.Database.Host = "main-host"
	cfg.Database.DatabaseName = "main"
	cfg.DatabasePrefix = "yekonga"
	cfg.IsAuthorizationServer = true
	cfg.HasTenant = true

	billing := mainConfig.DatabaseSettings{Kind: mainConfig.DBTypeMysql, Host: "billing-host", DatabaseName: "billing"}
	analytics := mainConfig.DatabaseSettings{Kind: mainConfig.DBTypeMongodb, Host: "analytics-host", DatabaseName: "analytics"}
	fields := map[string]CollectionFieldConfig{"name": {Kind: "String"}}

	structures := MultipleDatabaseStructure{
		{Collections: DatabaseStructure{"Orders": fields, "Users": fields}},
		{DatabaseName: "reports", Collections: DatabaseStructure{"Events": fields}},
		{DatabaseSetting: billing, Collections: DatabaseStructure{"Ledgers": fields}},
		{DatabaseSetting: billing, DatabaseName: "archive", Collections: DatabaseStructure{"Journals": fields}},
		{DatabaseSetting: analytics, Collections: DatabaseStructure{"Metrics": fields, "Locations": fields}},
	}

	structure, settings := structures.combine(&cfg)
	models := newSystemModels(&cfg, structure, settings)

	defaultConnect := NewDatabaseConnections(&cfg)
	y := &YekongaData{
		Config:     &cfg,
		dbConnect:  defaultConnect,
		dbConnects: map[string]*DatabaseConnections{databaseConnectionKey(&cfg, cfg.Database): defaultConnect},
	}
	SetSystemModelDBconnection(y, &models)

	tests := []struct {
		model        string
		kind         mainConfig.DatabaseType
		host         string
		databaseName string
		onDefault    bool
	}{
		{"Order", mainConfig.DBTypeMongodb, "main-host", "yekonga_app", true}, // the app's own collections
		{"Event", mainConfig.DBTypeMongodb, "main-host", "reports", true},     // same server, so the same client
		{"Ledger", mainConfig.DBTypeMysql, "billing-host", "billing", false},
		{"Journal", mainConfig.DBTypeMysql, "billing-host", "archive", false},
		{"Metric", mainConfig.DBTypeMongodb, "analytics-host", "analytics", false}, // another MongoDB server, its own client
		// Built-in collections go to their group's database on the config's server.
		{"Notification", mainConfig.DBTypeMongodb, "main-host", "yekonga_extra", true},
		{"Tenant", mainConfig.DBTypeMongodb, "main-host", "yekonga_tenants", true},
		{"User", mainConfig.DBTypeMongodb, "main-host", "yekonga_authorizations", true}, // extended by a default entry, still in its group
		{"Location", mainConfig.DBTypeMongodb, "analytics-host", "analytics", false},    // claimed by an entry with its own database
	}

	for _, tt := range tests {
		m := models[tt.model]
		if m == nil {
			t.Fatalf("model %s missing", tt.model)
		}

		if m.DatabaseType != tt.kind || m.DatabaseSetting.Host != tt.host || m.DatabaseSetting.DatabaseName != tt.databaseName {
			t.Errorf("%s: got %s %s/%s, want %s %s/%s", tt.model,
				m.DatabaseType, m.DatabaseSetting.Host, m.DatabaseSetting.DatabaseName,
				tt.kind, tt.host, tt.databaseName)
		}

		if (m.DBConnect == defaultConnect) != tt.onDefault {
			t.Errorf("%s: on default connection = %v, want %v", tt.model, m.DBConnect == defaultConnect, tt.onDefault)
		}
	}

	// Ledger and Journal are different MySQL databases, so different pools.
	if models["Ledger"].DBConnect == models["Journal"].DBConnect {
		t.Error("Ledger and Journal share a connection")
	}

	if len(y.dbConnects) != 4 {
		t.Errorf("got %d connections, want 4", len(y.dbConnects))
	}
}

// A single structure (ServerConfig) splits the collections into their groups'
// databases only when databasePrefix is set.
func TestDatabasePrefixSplitsDefaults(t *testing.T) {
	tests := []struct {
		prefix string
		want   map[string]string
	}{
		{"", map[string]string{"Order": "main", "User": "main", "Tenant": "main", "Notification": "main"}},
		{"yekonga", map[string]string{
			"Order":        "yekonga_app",
			"User":         "yekonga_authorizations",
			"Tenant":       "yekonga_tenants",
			"Notification": "yekonga_extra",
		}},
	}

	for _, tt := range tests {
		var cfg mainConfig.YekongaConfig
		cfg.Database.Kind = mainConfig.DBTypeMongodb
		cfg.Database.DatabaseName = "main"
		cfg.DatabasePrefix = tt.prefix
		cfg.IsAuthorizationServer = true
		cfg.HasTenant = true

		structures := MultipleDatabaseStructure{{Collections: DatabaseStructure{"Orders": {"name": {Kind: "String"}}}}}
		structure, settings := structures.combine(&cfg)
		models := newSystemModels(&cfg, structure, settings)

		for name, want := range tt.want {
			if got := models[name].DatabaseSetting.DatabaseName; got != want {
				t.Errorf("prefix %q, %s: database %q, want %q", tt.prefix, name, got, want)
			}
		}
	}
}

func TestLocalDatabaseDirectory(t *testing.T) {
	var cfg mainConfig.YekongaConfig
	cfg.Database.Kind = mainConfig.DBTypeLocal
	cfg.Database.DatabaseName = "app"

	tests := map[string]string{
		"":        "database",
		"app":     "database",
		"reports": "database_reports",
	}

	for name, want := range tests {
		settings := cfg.Database
		settings.DatabaseName = name
		if got := localDatabaseDirectory(&cfg, settings); got != want {
			t.Errorf("localDatabaseDirectory(%q) = %q, want %q", name, got, want)
		}
	}
}
