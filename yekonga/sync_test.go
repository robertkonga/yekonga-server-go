package yekonga

import (
	"reflect"
	"sync"
	"testing"

	mainConfig "github.com/robertkonga/yekonga-server-go/config"
)

func TestResolveSyncChange(t *testing.T) {
	update := func(fields ...string) syncChange {
		return syncChange{Action: syncActionUpdate, Fields: fields}
	}

	tests := []struct {
		name     string
		incoming syncChange
		later    []syncChange
		want     syncResolution
	}{
		{
			name:     "nothing later: applied as is",
			incoming: update("name", "grade"),
			want:     syncResolution{apply: true, fields: []string{"name", "grade"}},
		},
		{
			name:     "later change to other fields: both kept",
			incoming: update("name"),
			later:    []syncChange{update("grade")},
			want:     syncResolution{apply: true, fields: []string{"name"}},
		},
		{
			name:     "later change to the same field wins",
			incoming: update("name", "grade"),
			later:    []syncChange{update("grade")},
			want:     syncResolution{apply: true, fields: []string{"name"}, overlap: []string{"grade"}, conflict: true},
		},
		{
			name:     "every field changed later: skipped",
			incoming: update("grade"),
			later:    []syncChange{update("grade", "name")},
			want:     syncResolution{fields: []string{}, overlap: []string{"grade"}, conflict: true},
		},
		{
			name:     "later delete beats an update",
			incoming: update("grade"),
			later:    []syncChange{{Action: syncActionDelete}},
			want:     syncResolution{overlap: []string{"grade"}, conflict: true},
		},
		{
			name:     "a delete loses to a later update",
			incoming: syncChange{Action: syncActionDelete},
			later:    []syncChange{update("grade")},
			want:     syncResolution{conflict: true},
		},
		{
			name:     "create writes all of its record's fields",
			incoming: syncChange{Action: syncActionCreate, Data: map[string]interface{}{"id": "x", "name": "a", "grade": 1}},
			later:    []syncChange{update("name")},
			want:     syncResolution{apply: true, fields: []string{"grade"}, overlap: []string{"name"}, conflict: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveSyncChange(&tt.incoming, tt.later)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSyncClockIncreases(t *testing.T) {
	var clock syncClock
	ahead := clock.current() + 60_000_000 // a minute ahead, as if from a node with a fast clock
	clock.observe(ahead)

	if next := clock.next(); next <= ahead {
		t.Fatalf("next() = %d after observing %d; a later change must get a later version", next, ahead)
	}

	clock.observe(1) // an old value doesn't move it back
	if clock.current() <= ahead {
		t.Fatal("observe moved the clock back")
	}

	var mu sync.Mutex
	seen := map[int64]bool{}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				v := clock.next()
				mu.Lock()
				if seen[v] {
					t.Errorf("value %d handed out twice", v)
				}
				seen[v] = true
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
}

func TestBuildSyncPolicies(t *testing.T) {
	var cfg mainConfig.YekongaConfig
	cfg.HasTenant = true
	cfg.Database.Kind = mainConfig.DBTypeMongodb
	cfg.Sync.ExcludeModels = []string{"Draft"}
	cfg.Sync.Models = map[string]mainConfig.SyncModel{
		"Mark":       {Owner: syncOwnerEdge, Strategy: syncStrategyManual},
		"AuditTrail": {}, // excluded by default; listing it syncs it
	}

	models := newSystemModels(&cfg, &DatabaseStructure{
		"tenants":        {"name": {Kind: "String"}},
		"tenant_users":   {"tenantId": {Kind: "ID"}, "userId": {Kind: "ID"}},
		"users":          {"name": {Kind: "String"}},
		"students":       {"tenantId": {Kind: "ID"}, "name": {Kind: "String"}},
		"marks":          {"tenantId": {Kind: "ID"}, "score": {Kind: "Number"}},
		"locations":      {"name": {Kind: "String"}},
		"drafts":         {"tenantId": {Kind: "ID"}},
		"refresh_tokens": {"tenantId": {Kind: "ID"}},
		"audit_trails":   {"tenantId": {Kind: "ID"}},
		"sync_changes":   {"tenantId": {Kind: "ID"}},
	}, nil)

	policies := buildSyncPolicies(&cfg, models)

	check := func(name, owner, strategy, scope string) {
		t.Helper()
		p := policies[name]
		if p == nil {
			t.Fatalf("%s isn't synced", name)
		}
		if p.owner != owner || p.strategy != strategy || p.scope != scope {
			t.Errorf("%s: got owner %q strategy %q scope %q, want %q %q %q", name, p.owner, p.strategy, p.scope, owner, strategy, scope)
		}
	}

	check("Tenant", syncOwnerCloud, syncStrategyLastWriteWins, syncScopeTenantSelf)
	check("Student", syncOwnerAny, syncStrategyLastWriteWins, syncScopeTenant)
	check("Mark", syncOwnerEdge, syncStrategyManual, syncScopeTenant)
	check("User", syncOwnerAny, syncStrategyLastWriteWins, syncScopeRelation)
	check("Location", syncOwnerCloud, syncStrategyLastWriteWins, syncScopeGlobal)
	check("AuditTrail", syncOwnerAny, syncStrategyLastWriteWins, syncScopeTenant)

	if p := policies["User"]; p.throughModel != "TenantUser" || p.throughField != "userId" {
		t.Errorf("User links through %s.%s, want TenantUser.userId", p.throughModel, p.throughField)
	}

	for _, name := range []string{"Draft", "RefreshToken", "SyncChange"} {
		if policies[name] != nil {
			t.Errorf("%s shouldn't be synced", name)
		}
	}
}
