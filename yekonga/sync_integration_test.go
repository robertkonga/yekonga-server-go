package yekonga

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	mainConfig "github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/bson"
)

// A cloud and an edge, each on its own MongoDB database, syncing over HTTP.
// Point it at a throwaway server; it drops its databases:
//
//	YEKONGA_TEST_MONGO_PORT=27999 go test -run TestSyncCloudEdge ./yekonga/
func TestSyncCloudEdge(t *testing.T) {
	port := os.Getenv("YEKONGA_TEST_MONGO_PORT")
	if port == "" {
		t.Skip("set YEKONGA_TEST_MONGO_PORT to run against a throwaway MongoDB server")
	}

	t.Setenv("HOME", t.TempDir())

	settle := syncSettle
	syncSettle = 0
	t.Cleanup(func() { syncSettle = settle })

	cloud := newSyncTestServer(t, port, "yekonga_sync_cloud", mainConfig.Sync{Enabled: true, Role: mainConfig.SyncRoleCloud})

	schoolA := mustCreate(t, cloud, "Tenant", datatype.DataMap{"name": "School A", "domain": "a.example.com"})
	schoolB := mustCreate(t, cloud, "Tenant", datatype.DataMap{"name": "School B", "domain": "b.example.com"})

	// Data from before the edge exists, so it comes down in the download.
	ana := mustCreate(t, cloud, "Student", datatype.DataMap{"tenantId": schoolA, "name": "Ana", "grade": 5})
	otherSchool := mustCreate(t, cloud, "Student", datatype.DataMap{"tenantId": schoolB, "name": "Ben", "grade": 6})
	teacher := mustCreate(t, cloud, "User", datatype.DataMap{"firstName": "Teacher", "password": "hash-1"})
	mustCreate(t, cloud, "TenantUser", datatype.DataMap{"tenantId": schoolA, "userId": teacher})
	parent := mustCreate(t, cloud, "User", datatype.DataMap{"firstName": "Parent"})
	mustCreate(t, cloud, "TenantUser", datatype.DataMap{"tenantId": schoolB, "userId": parent})

	nodeId, nodeKey, err := cloud.CreateSyncNode(schoolA, "School A office")
	if err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(cloud)
	t.Cleanup(upstream.Close)

	edge := newSyncTestServer(t, port, "yekonga_sync_edge", mainConfig.Sync{
		Enabled:     true,
		Role:        mainConfig.SyncRoleEdge,
		UpstreamUrl: upstream.URL,
		NodeId:      nodeId,
		NodeKey:     nodeKey,
		TenantId:    schoolA,
	})

	syncEdge := func() {
		t.Helper()
		if err := edge.SyncOnce(); err != nil {
			t.Fatal("sync:", err)
		}
	}

	t.Run("download gets the tenant's data only", func(t *testing.T) {
		syncEdge()

		if find(edge, "Tenant", schoolA) == nil || find(edge, "Tenant", schoolB) != nil {
			t.Error("the edge should have its own tenant and no other")
		}
		if find(edge, "Student", ana) == nil || find(edge, "Student", otherSchool) != nil {
			t.Error("the edge should have its tenant's students only")
		}
		if user := find(edge, "User", teacher); user == nil || (*user)["password"] != "hash-1" {
			t.Error("the edge should have its tenant's users, password hash included, for offline login")
		}
		if find(edge, "User", parent) != nil {
			t.Error("another tenant's user reached the edge")
		}
	})

	t.Run("offline changes to different fields both survive", func(t *testing.T) {
		mustUpdate(t, edge, "Student", ana, datatype.DataMap{"grade": 6})
		mustUpdate(t, cloud, "Student", ana, datatype.DataMap{"notes": "moved class"})
		newStudent := mustCreate(t, edge, "Student", datatype.DataMap{"tenantId": schoolA, "name": "Chen", "grade": 4})

		syncEdge()

		for side, y := range map[string]*YekongaData{"cloud": cloud, "edge": edge} {
			record := find(y, "Student", ana)
			if record == nil || helper.ToInt((*record)["grade"]) != 6 || (*record)["notes"] != "moved class" {
				t.Errorf("%s: got %v, want grade 6 and the cloud's notes", side, record)
			}
		}

		created := find(cloud, "Student", newStudent)
		if created == nil || idString((*created)[TenantIDKey]) != schoolA {
			t.Errorf("a student created on the edge should reach the cloud in its tenant, got %v", created)
		}
	})

	t.Run("the later change to the same field wins on both sides", func(t *testing.T) {
		mustUpdate(t, cloud, "Student", ana, datatype.DataMap{"name": "Ana (cloud)"})
		time.Sleep(time.Millisecond)
		mustUpdate(t, edge, "Student", ana, datatype.DataMap{"name": "Ana (edge)"})

		syncEdge()

		for side, y := range map[string]*YekongaData{"cloud": cloud, "edge": edge} {
			if record := find(y, "Student", ana); record == nil || (*record)["name"] != "Ana (edge)" {
				t.Errorf("%s: got %v, want the later name", side, record)
			}
		}
	})

	t.Run("deletes on the cloud reach the edge", func(t *testing.T) {
		gone := mustCreate(t, cloud, "Student", datatype.DataMap{"tenantId": schoolA, "name": "Dora"})
		syncEdge()
		if find(edge, "Student", gone) == nil {
			t.Fatal("created on the cloud, should be on the edge")
		}

		cloud.ModelQuery("Student").Delete(datatype.DataMap{"_id": gone})
		syncEdge()
		if find(edge, "Student", gone) != nil {
			t.Error("deleted on the cloud, should be gone from the edge")
		}
	})

	t.Run("a user linked to the tenant later reaches the edge", func(t *testing.T) {
		mustCreate(t, cloud, "TenantUser", datatype.DataMap{"tenantId": schoolA, "userId": parent})
		syncEdge()

		if find(edge, "User", parent) == nil {
			t.Error("the parent's user record should come with the link")
		}
	})

	t.Run("cloud-owned models are read-only on the edge", func(t *testing.T) {
		result := edge.ModelQuery("Tenant").Update(datatype.DataMap{"name": "Renamed"}, datatype.DataMap{"_id": schoolA})
		if _, isErr := result.(error); !isErr {
			t.Errorf("updating the tenant on the edge should fail, got %v", result)
		}
	})

	t.Run("an edge can't push another tenant's records", func(t *testing.T) {
		forged := syncChange{
			ChangeId:   bson.NewObjectID().Hex(),
			Version:    time.Now().UnixMicro(),
			Model:      "Student",
			DocumentId: otherSchool,
			Action:     syncActionUpdate,
			Fields:     []string{"name"},
			Data:       map[string]interface{}{"name": "hacked"},
		}

		var response syncPushResponse
		if err := edge.sync.request(http.MethodPost, "/push", nil, datatype.DataMap{"changes": []syncChange{forged}}, &response); err != nil {
			t.Fatal(err)
		}

		if len(response.Rejected) != 1 {
			t.Errorf("got %+v, want the change refused", response)
		}
		if record := find(cloud, "Student", otherSchool); record == nil || (*record)["name"] != "Ben" {
			t.Errorf("another tenant's student changed: %v", record)
		}
	})

	t.Run("wrong node key is refused", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, upstream.URL+"/sync/info", bytes.NewReader(nil))
		req.Header.Set(syncNodeHeader, nodeId)
		req.Header.Set(syncKeyHeader, "wrong")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("got %d, want 401", resp.StatusCode)
		}
	})

	t.Run("a deactivated edge is refused, and works again once reactivated", func(t *testing.T) {
		if err := cloud.SetSyncNodeActive(nodeId, false); err != nil {
			t.Fatal(err)
		}
		if err := edge.SyncOnce(); err == nil {
			t.Error("a deactivated edge's sync should fail")
		}
		if online := edge.SyncStatus()["online"]; online != false {
			t.Errorf("online = %v while refused, want false", online)
		}

		if err := cloud.SetSyncNodeActive(nodeId, true); err != nil {
			t.Fatal(err)
		}
		syncEdge()
	})

	t.Run("the edge serves its tenant on a LAN address", func(t *testing.T) {
		tenant, _ := edge.tenantByHost("192.168.1.10")
		if tenant == nil || syncRecordId(*tenant) != schoolA {
			t.Errorf("got %v, want school A", tenant)
		}
	})

	t.Run("status shows nothing pending once synced", func(t *testing.T) {
		mustUpdate(t, edge, "Student", ana, datatype.DataMap{"grade": 7})
		if pending := edge.SyncStatus()["pending"]; pending != int64(1) {
			t.Errorf("pending = %v before sync, want 1", pending)
		}

		syncEdge()
		if pending := edge.SyncStatus()["pending"]; pending != int64(0) {
			t.Errorf("pending = %v after sync, want 0", pending)
		}

		if status := edge.SyncStatus(); status["online"] != true || status["lastSyncAt"] == nil {
			encoded, _ := json.Marshal(status)
			t.Errorf("status %s, want online with a last sync time", encoded)
		}
	})
}

func newSyncTestServer(t *testing.T, port string, dbName string, sync mainConfig.Sync) *YekongaData {
	t.Helper()

	var cfg mainConfig.YekongaConfig
	cfg.AppName = dbName
	cfg.HasTenant = true
	cfg.IsAuthorizationServer = true
	cfg.Database.Kind = mainConfig.DBTypeMongodb
	cfg.Database.Host = "127.0.0.1"
	cfg.Database.Port = port
	cfg.Database.DatabaseName = dbName
	cfg.Sync = sync

	y := ServerConfig(cfg, DatabaseStructure{
		"students": {
			"tenantId": {Kind: "ID", ForeignKey: CollectionFieldConfigForeignKey{Model: "Tenant", Key: "id"}},
			"name":     {Kind: "String"},
			"grade":    {Kind: "Number"},
			"notes":    {Kind: "String"},
		},
	})

	db := y.dbConnect.mongodbClient.Database(dbName)
	db.Drop(context.Background())
	t.Cleanup(func() { db.Drop(context.Background()) })

	return y
}

func mustCreate(t *testing.T, y *YekongaData, model string, data datatype.DataMap) string {
	t.Helper()

	result := y.ModelQuery(model).Create(data)
	record, ok := result.(*datatype.DataMap)
	if !ok || record == nil {
		t.Fatalf("create %s: %v", model, result)
	}

	return syncRecordId(*record)
}

func mustUpdate(t *testing.T, y *YekongaData, model string, id string, data datatype.DataMap) {
	t.Helper()

	if result := y.ModelQuery(model).Update(data, datatype.DataMap{"_id": id}); helper.IsEmpty(result) {
		t.Fatalf("update %s %s: %v", model, id, result)
	} else if err, isErr := result.(error); isErr {
		t.Fatalf("update %s %s: %v", model, id, err)
	}
}

func find(y *YekongaData, model string, id string) *datatype.DataMap {
	return findSyncRecord(y.models[model], id)
}
