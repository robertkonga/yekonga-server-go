package yekonga

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/bson"
)

// Sync keeps on-premises (edge) servers and the cloud server in step, both
// ways (see config.Sync). Every write through DataModelQuery to a synced
// model is logged in SyncChange with a copy of the record. An edge pushes its
// own log entries to the cloud and pulls the cloud's entries for its tenant;
// each side applies what it receives, field by field, keeping the later of
// two changes to the same field.

const (
	syncChangeModelName   = "SyncChange"
	syncNodeModelName     = "SyncNode"
	syncStateModelName    = "SyncState"
	syncConflictModelName = "SyncConflict"
	tenantUserModelName   = "TenantUser"

	// syncCloudNodeId is the nodeId of changes made on the cloud.
	syncCloudNodeId = "cloud"

	syncOwnerAny   = "any"
	syncOwnerCloud = "cloud"
	syncOwnerEdge  = "edge"

	syncStrategyLastWriteWins = "lastWriteWins"
	syncStrategyManual        = "manual"

	syncScopeTenant   = "tenant"
	syncScopeGlobal   = "global"
	syncScopeRelation = "relation"
	// syncScopeTenantSelf is the Tenant model's: an edge gets its own record.
	syncScopeTenantSelf = "tenantSelf"

	syncActionCreate = "create"
	syncActionUpdate = "update"
	syncActionDelete = "delete"

	defaultSyncRoute         = "/sync"
	defaultSyncInterval      = 30 * time.Second
	defaultSyncBatchSize     = 500
	defaultSyncRetentionDays = 30
)

// syncSettle is how old a log entry must be before it's sent. An entry gets
// its seq before it's written, so a later seq can be stored first; sending
// only settled entries keeps a cursor from passing one still being written.
// A variable for tests.
var syncSettle = 2 * time.Second

var DefaultSyncDatabaseStructure DatabaseStructure = DatabaseStructure{
	"SyncChanges": map[string]CollectionFieldConfig{
		"id":         {Kind: "ID", DefaultValue: nil, Required: false},
		"tenantId":   {Kind: "ID", DefaultValue: nil, Required: false, Index: true},
		"changeId":   {Kind: "String", DefaultValue: nil, Required: false, Unique: true},
		"seq":        {Kind: "Float", DefaultValue: nil, Required: false, Index: true},  // order on this server, for cursors (Float: Number values go through helper.ToInt, which can't hold it)
		"version":    {Kind: "Float", DefaultValue: nil, Required: false},               // when the change was made, on a hybrid clock shared by all nodes; the later version wins
		"nodeId":     {Kind: "String", DefaultValue: nil, Required: false, Index: true}, // the server the change was made on
		"model":      {Kind: "String", DefaultValue: nil, Required: false, Index: true},
		"documentId": {Kind: "String", DefaultValue: nil, Required: false, Index: true},
		"action":     {Kind: "String", DefaultValue: nil, Required: false, Options: []string{syncActionCreate, syncActionUpdate, syncActionDelete}},
		"fields":     {Kind: "Array", DefaultValue: []interface{}{}, Required: false}, // fields the change wrote; empty on a create (all of them)
		"data":       {Kind: "Any", DefaultValue: nil, Required: false},               // the record after the change; null on a delete
		"skipped":    {Kind: "Boolean", DefaultValue: false, Required: false},         // a received change that later local changes overrode entirely
		"createdAt":  {Kind: "Date", DefaultValue: "now", Required: false},
	},
	"SyncNodes": map[string]CollectionFieldConfig{
		"id":         {Kind: "ID", DefaultValue: nil, Required: false},
		"tenantId":   {Kind: "ID", DefaultValue: nil, Required: false, Index: true},
		"name":       {Kind: "String", DefaultValue: nil, Required: false},
		"keyHash":    {Kind: "String", DefaultValue: nil, Required: false, Protected: true},
		"status":     {Kind: "String", DefaultValue: "active", Required: false, Options: []string{"active", "inactive"}},
		"lastSeenAt": {Kind: "Date", DefaultValue: nil, Required: false},
		"lastPushAt": {Kind: "Date", DefaultValue: nil, Required: false},
		"lastPullAt": {Kind: "Date", DefaultValue: nil, Required: false},
		"createdAt":  {Kind: "Date", DefaultValue: "now", Required: false},
	},
	"SyncStates": map[string]CollectionFieldConfig{
		"id":        {Kind: "ID", DefaultValue: nil, Required: false},
		"name":      {Kind: "String", DefaultValue: nil, Required: false, Unique: true},
		"value":     {Kind: "Any", DefaultValue: nil, Required: false},
		"updatedAt": {Kind: "Date", DefaultValue: "now", Required: false},
	},
	"SyncConflicts": map[string]CollectionFieldConfig{
		"id":         {Kind: "ID", DefaultValue: nil, Required: false},
		"tenantId":   {Kind: "ID", DefaultValue: nil, Required: false, Index: true},
		"nodeId":     {Kind: "String", DefaultValue: nil, Required: false},
		"changeId":   {Kind: "String", DefaultValue: nil, Required: false},
		"model":      {Kind: "String", DefaultValue: nil, Required: false},
		"documentId": {Kind: "String", DefaultValue: nil, Required: false},
		"action":     {Kind: "String", DefaultValue: nil, Required: false},
		"fields":     {Kind: "Array", DefaultValue: []interface{}{}, Required: false},
		"localData":  {Kind: "Any", DefaultValue: nil, Required: false},
		"remoteData": {Kind: "Any", DefaultValue: nil, Required: false},
		"reason":     {Kind: "String", DefaultValue: nil, Required: false},
		"status":     {Kind: "String", DefaultValue: "open", Required: false, Options: []string{"open", "resolved"}},
		"createdAt":  {Kind: "Date", DefaultValue: "now", Required: false},
	},
}

// syncInternalModels are the sync engine's own; they're never synced and
// can't be reached through GraphQL or REST.
var syncInternalModels = []string{syncChangeModelName, syncNodeModelName, syncStateModelName}

// syncExcludedModels aren't synced unless config sync.models lists them:
// state that belongs to one server (sessions, OTPs, blocked IPs, sockets,
// gateway logs, its own audit trail).
var syncExcludedModels = []string{
	syncConflictModelName,
	"RefreshToken",
	"LoginAttempt",
	"UserVerification",
	"IpAccessRule",
	"SocketMessage",
	"UserDeviceCommand",
	"PaymentProviderRequest",
	"AuditTrail",
}

// syncCloudOwnedModels can only be changed on the cloud unless config
// sync.models says otherwise: the tenant itself and its billing.
var syncCloudOwnedModels = []string{
	"Tenant",
	"TenantFeature",
	"Module",
	"PricingPlan",
	"Subscription",
	"SubscriptionItem",
	"UsageRecord",
	"Invoice",
	"InvoiceItem",
	"AccountCredit",
}

// syncPolicy is how one model syncs.
type syncPolicy struct {
	model        *DataModel
	owner        string
	strategy     string
	scope        string
	throughModel string // scope relation: the model linking records to a tenant
	throughField string // and its field holding this model's ids
}

// syncChange is a SyncChange log entry as it's sent between nodes.
type syncChange struct {
	ChangeId   string                 `json:"changeId"`
	Seq        int64                  `json:"seq"`
	Version    int64                  `json:"version"`
	NodeId     string                 `json:"nodeId"`
	TenantId   string                 `json:"tenantId,omitempty"`
	Model      string                 `json:"model"`
	DocumentId string                 `json:"documentId"`
	Action     string                 `json:"action"`
	Fields     []string               `json:"fields,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// syncClock hands out increasing microsecond timestamps: the wall clock, or
// one more than the last value when the wall clock isn't ahead of it.
// observe moves it past a value seen from another node, so a change made
// after receiving one is always later than it (a hybrid logical clock).
type syncClock struct {
	last atomic.Int64
}

func (c *syncClock) next() int64 {
	for {
		last := c.last.Load()
		now := time.Now().UnixMicro()
		if now <= last {
			now = last + 1
		}

		if c.last.CompareAndSwap(last, now) {
			return now
		}
	}
}

func (c *syncClock) observe(value int64) {
	for {
		last := c.last.Load()
		if value <= last || c.last.CompareAndSwap(last, value) {
			return
		}
	}
}

// current is the clock's value now, without advancing it.
func (c *syncClock) current() int64 {
	now := time.Now().UnixMicro()
	if last := c.last.Load(); last > now {
		return last
	}

	return now
}

type syncEngine struct {
	app      *YekongaData
	config   *config.Sync
	role     config.SyncRole
	nodeId   string
	policies map[string]*syncPolicy

	seq     syncClock // orders this server's log entries; never observes other nodes, so it stays close to the wall clock
	version syncClock // stamps changes made here; observes every version received

	kick        chan struct{} // an edge's local write: sync soon
	running     sync.Mutex    // one sync round at a time
	startOnce   sync.Once
	lastCleanup time.Time

	statusMu   sync.Mutex
	online     bool
	lastSyncAt time.Time
	lastError  string

	edgeTenantsMu sync.Mutex
	edgeTenants   map[string]syncEdgeTenant // cloud: whether a tenant runs an edge, by tenant id
}

type syncEdgeTenant struct {
	hasEdge   bool
	checkedAt time.Time
}

// newSyncEngine returns the sync engine for config.sync, or nil when sync is
// off or misconfigured (the reason is logged).
func newSyncEngine(y *YekongaData) *syncEngine {
	cfg := &y.Config.Sync
	if !cfg.Enabled {
		return nil
	}

	s := &syncEngine{
		app:         y,
		config:      cfg,
		role:        cfg.Role,
		nodeId:      syncCloudNodeId,
		kick:        make(chan struct{}, 1),
		edgeTenants: map[string]syncEdgeTenant{},
	}

	switch s.role {
	case config.SyncRoleCloud:
	case config.SyncRoleEdge:
		s.nodeId = strings.TrimSpace(cfg.NodeId)
		missing := []string{}
		if strings.TrimSpace(cfg.UpstreamUrl) == "" {
			missing = append(missing, "upstreamUrl")
		}
		if s.nodeId == "" || s.nodeId == syncCloudNodeId {
			missing = append(missing, "nodeId")
		}
		if cfg.NodeKey == "" {
			missing = append(missing, "nodeKey")
		}
		if y.Config.HasTenant && strings.TrimSpace(cfg.TenantId) == "" {
			missing = append(missing, "tenantId")
		}
		if len(missing) > 0 {
			logger.Error("Sync disabled", "an edge needs sync."+strings.Join(missing, ", sync."))
			return nil
		}
	default:
		logger.Error("Sync disabled", "sync.role must be \"cloud\" or \"edge\", not \""+string(s.role)+"\"")
		return nil
	}

	s.policies = buildSyncPolicies(y.Config, y.models)
	s.protectInternalModels()

	return s
}

// buildSyncPolicies works out how each model syncs from its fields and
// config sync.models. Models that aren't synced are left out.
func buildSyncPolicies(cfg *config.YekongaConfig, models map[string]*DataModel) map[string]*syncPolicy {
	policies := map[string]*syncPolicy{}

	for name, model := range models {
		settings, listed := cfg.Sync.Models[name]

		if helper.Contains(syncInternalModels, name) || helper.Contains(cfg.Sync.ExcludeModels, name) {
			continue
		}
		if !listed && helper.Contains(syncExcludedModels, name) {
			continue
		}
		if model.DatabaseType == config.DBTypeLocal || model.DatabaseType == "" {
			// The local file store numbers its records instead of using
			// ObjectIDs, so they can't be matched across servers.
			logger.Error("Sync", name+" is on the local file database, which can't sync; use mongodb, mysql or sql")
			continue
		}

		policy := &syncPolicy{
			model:    model,
			owner:    settings.Owner,
			strategy: settings.Strategy,
			scope:    settings.Scope,
		}

		if policy.scope == "" {
			switch {
			case name == tenantModelName && cfg.HasTenant:
				policy.scope = syncScopeTenantSelf
			case model.HasTenant && cfg.HasTenant:
				policy.scope = syncScopeTenant
			case name == userModelName && cfg.HasTenant && models[tenantUserModelName] != nil:
				policy.scope = syncScopeRelation
			default:
				policy.scope = syncScopeGlobal
			}
		}

		if policy.scope == syncScopeRelation {
			through := settings.Through
			if through == "" && name == userModelName {
				through = tenantUserModelName + ".userId"
			}

			parts := strings.SplitN(through, ".", 2)
			throughModel := models[parts[0]]
			if len(parts) != 2 || throughModel == nil || !throughModel.HasTenant {
				logger.Error("Sync", name+": scope \"relation\" needs sync.models."+name+".through as <Model>.<field> of a model with tenantId")
				continue
			}

			policy.throughModel, policy.throughField = parts[0], parts[1]
		}

		if policy.owner == "" {
			switch {
			case helper.Contains(syncCloudOwnedModels, name):
				policy.owner = syncOwnerCloud
			case policy.scope == syncScopeGlobal && cfg.HasTenant:
				// Shared by every tenant: an edge changing it would change it
				// for all of them.
				policy.owner = syncOwnerCloud
			default:
				policy.owner = syncOwnerAny
			}
		}

		if policy.strategy == "" {
			policy.strategy = syncStrategyLastWriteWins
		}

		policies[name] = policy
	}

	return policies
}

// protectInternalModels keeps the sync log, nodes and state out of GraphQL
// and REST: the log holds full records, password hashes included. The sync
// engine reads and writes them skipping triggers.
func (s *syncEngine) protectInternalModels() {
	deny := func(rc *RequestContext, qc *QueryContext) (interface{}, error) {
		return false, errors.New("Protected by the sync engine")
	}

	for _, name := range syncInternalModels {
		s.app.BeforeFind(name, nil, nil, deny)
		s.app.BeforeCreate(name, nil, nil, deny)
		s.app.BeforeUpdate(name, nil, nil, deny)
		s.app.BeforeDelete(name, nil, nil, deny)
	}
}

// policyOf is the model's sync policy, or nil when it isn't synced.
func (s *syncEngine) policyOf(model *DataModel) *syncPolicy {
	if s == nil || model == nil {
		return nil
	}

	return s.policies[model.Name]
}

func (s *syncEngine) batchSize() int {
	if s.config.BatchSize > 0 {
		return s.config.BatchSize
	}

	return defaultSyncBatchSize
}

func (s *syncEngine) route() string {
	route := strings.TrimSpace(s.config.Route)
	if route == "" {
		route = defaultSyncRoute
	}

	return "/" + strings.Trim(route, "/")
}

// edgeTenantId is the tenant an edge serves; empty on the cloud.
func (s *syncEngine) edgeTenantId() string {
	if s == nil || s.role != config.SyncRoleEdge {
		return ""
	}

	return strings.TrimSpace(s.config.TenantId)
}

// settledSeq is the highest seq that's safe to send.
func (s *syncEngine) settledSeq() int64 {
	return s.seq.current() - syncSettle.Microseconds()
}

// loadClocks moves both clocks past the log's latest entry, so they keep
// increasing across restarts even if the wall clock went back.
func (s *syncEngine) loadClocks() {
	model := s.app.models[syncChangeModelName]
	if model == nil {
		return
	}

	s.seq.observe(syncInt64(rawQuery(model).Max("seq", nil)))
	s.version.observe(syncInt64(rawQuery(model).Max("version", nil)))
}

// rawQuery is a query that skips triggers and the tenant filter: for the
// sync engine's reads, which aren't made on behalf of a request.
func rawQuery(model *DataModel) *DataModelQuery {
	return model.Query().SkipTenant().SkipBeforeCommit()
}

// rawCreate writes a record without triggers, sync logging or socket events.
func rawCreate(model *DataModel, data datatype.DataMap) (*datatype.DataMap, error) {
	q := model.Query()
	return q.collection().create(*q.formatInputData(data, CreateInputAction))
}

// rawUpdate is rawCreate for an update of the record with id.
func rawUpdate(model *DataModel, id string, data datatype.DataMap) (*datatype.DataMap, error) {
	q := model.Query()
	q.Where("_id", id)
	return q.collection().update(*q.formatInputData(data, UpdateInputAction))
}

// findSyncRecord is the record with id, read without triggers.
func findSyncRecord(model *DataModel, id string) *datatype.DataMap {
	q := model.Query()
	q.Where("_id", id)
	record := q.collection().findOne()
	if helper.IsEmpty(record) {
		return nil
	}

	return record
}

// syncRecordData is the part of a record that's synced: the model's fields,
// with its id as "id".
func syncRecordData(model *DataModel, record datatype.DataMap) datatype.DataMap {
	data := make(datatype.DataMap, len(model.ValidFields))
	for _, k := range model.ValidFields {
		if v, ok := record[k]; ok {
			data[k] = v
		}
	}

	if id, ok := record["_id"]; ok {
		data["id"] = id
	}

	return data
}

// syncRecordId is a record's id as a string.
func syncRecordId(record datatype.DataMap) string {
	if id := idString(record["_id"]); id != "" {
		return id
	}

	return idString(record["id"])
}

// recordTenant is the tenant a record belongs to, for the log.
func (p *syncPolicy) recordTenant(record datatype.DataMap) string {
	switch p.scope {
	case syncScopeTenantSelf:
		return syncRecordId(record)
	case syncScopeTenant:
		return idString(record[TenantIDKey])
	}

	return ""
}

// syncWriteAllowed refuses writes to a model this side has read-only: owned
// by the cloud on an edge, or by the edge on the cloud for a tenant that
// runs one.
func (m *DataModelQuery) syncWriteAllowed() error {
	if m.Model.App == nil {
		return nil
	}

	s := m.Model.App.sync
	policy := s.policyOf(m.Model)
	if policy == nil {
		return nil
	}

	switch {
	case policy.owner == syncOwnerCloud && s.role == config.SyncRoleEdge:
		return fmt.Errorf("%s is read-only on this server: it's changed on the cloud", m.Model.Name)
	case policy.owner == syncOwnerEdge && s.role == config.SyncRoleCloud:
		if tenantId := idString(m.getTenantId()); tenantId != "" && s.tenantHasEdge(tenantId) {
			return fmt.Errorf("%s is read-only on the cloud: it's changed on the on-premises server", m.Model.Name)
		}
	}

	return nil
}

// tenantHasEdge reports whether a tenant has an active SyncNode, cached for
// a minute.
func (s *syncEngine) tenantHasEdge(tenantId string) bool {
	s.edgeTenantsMu.Lock()
	cached, ok := s.edgeTenants[tenantId]
	s.edgeTenantsMu.Unlock()

	if ok && time.Since(cached.checkedAt) < time.Minute {
		return cached.hasEdge
	}

	hasEdge := rawQuery(s.app.models[syncNodeModelName]).Count(datatype.DataMap{
		TenantIDKey: tenantId,
		"status":    "active",
	}) > 0

	s.edgeTenantsMu.Lock()
	s.edgeTenants[tenantId] = syncEdgeTenant{hasEdge: hasEdge, checkedAt: time.Now()}
	s.edgeTenantsMu.Unlock()

	return hasEdge
}

// syncEnabled reports whether writes to this model are logged for sync.
func (m *DataModelQuery) syncEnabled() bool {
	return m.Model.App != nil && m.Model.App.sync.policyOf(m.Model) != nil
}

// syncFields are the fields an update writes.
func syncFields(data datatype.DataMap) []string {
	fields := make([]string, 0, len(data))
	for k := range data {
		if k != "id" && k != "_id" {
			fields = append(fields, k)
		}
	}
	sort.Strings(fields)

	return fields
}

// recordSyncChanges logs writes to a synced model. fields are the fields an
// update wrote; nil for a create or delete.
func (m *DataModelQuery) recordSyncChanges(action string, records []datatype.DataMap, fields []string) {
	if m.Model.App == nil || len(records) == 0 {
		return
	}

	s := m.Model.App.sync
	policy := s.policyOf(m.Model)
	if policy == nil {
		return
	}

	entries := make([]datatype.DataMap, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}

		change := syncChange{
			ChangeId:   bson.NewObjectID().Hex(),
			Version:    s.version.next(),
			NodeId:     s.nodeId,
			TenantId:   policy.recordTenant(record),
			Model:      m.Model.Name,
			DocumentId: syncRecordId(record),
			Action:     action,
			Fields:     fields,
		}

		if change.DocumentId == "" {
			continue
		}

		if action != syncActionDelete {
			change.Data = syncRecordData(m.Model, record)
		}

		entries = append(entries, s.logEntry(&change, false))
	}

	if len(entries) == 0 {
		return
	}

	q := s.app.models[syncChangeModelName].Query()
	formatted := make([]datatype.DataMap, 0, len(entries))
	for _, entry := range entries {
		formatted = append(formatted, *q.formatInputData(entry, CreateInputAction))
	}

	if _, err := q.collection().createMany(formatted); err != nil {
		logger.Error("Sync", "could not log "+action+" of "+m.Model.Name, err.Error())
		return
	}

	if s.role == config.SyncRoleEdge {
		s.syncSoon()
	}
}

// logEntry is the SyncChange record for a change, with a new seq.
func (s *syncEngine) logEntry(change *syncChange, skipped bool) datatype.DataMap {
	entry := datatype.DataMap{
		"changeId":   change.ChangeId,
		"seq":        float64(s.seq.next()),
		"version":    float64(change.Version),
		"nodeId":     change.NodeId,
		"model":      change.Model,
		"documentId": change.DocumentId,
		"action":     change.Action,
		"fields":     change.Fields,
		"data":       change.Data,
		"skipped":    skipped,
	}

	if change.Fields == nil {
		entry["fields"] = []string{}
	}

	if change.TenantId != "" {
		entry["tenantId"] = change.TenantId
	}

	return entry
}

// decodeSyncChanges reads SyncChange records as syncChanges. The JSON round
// trip turns ObjectIDs, dates and nested documents into what they're sent
// as.
func decodeSyncChanges(records *[]datatype.DataMap) []syncChange {
	if records == nil {
		return []syncChange{}
	}

	changes := make([]syncChange, 0, len(*records))
	for _, record := range *records {
		var change syncChange
		if err := json.Unmarshal(helper.ToByte(record), &change); err == nil {
			changes = append(changes, change)
		}
	}

	return changes
}

// syncResolution is what to do with a received change, given the changes
// to the same record made here later than it.
type syncResolution struct {
	apply    bool
	fields   []string // the fields to write; nil writes all of them
	overlap  []string // fields changed both here, later, and by the change: the later value here is kept
	conflict bool     // the change lost to a later one here, entirely or in overlap
}

// resolveSyncChange applies last-write-wins per field: the change keeps the
// fields no later change here wrote. A delete loses to any later change, and
// a later delete here beats any change.
func resolveSyncChange(incoming *syncChange, later []syncChange) syncResolution {
	if len(later) == 0 {
		return syncResolution{apply: true, fields: incoming.Fields}
	}

	if incoming.Action == syncActionDelete {
		return syncResolution{conflict: true}
	}

	laterFields := map[string]bool{}
	for _, change := range later {
		if change.Action == syncActionDelete || change.Action == syncActionCreate {
			return syncResolution{conflict: true, overlap: incomingSyncFields(incoming)}
		}

		for _, field := range change.Fields {
			laterFields[field] = true
		}
	}

	resolution := syncResolution{fields: []string{}}
	for _, field := range incomingSyncFields(incoming) {
		if laterFields[field] {
			resolution.overlap = append(resolution.overlap, field)
		} else {
			resolution.fields = append(resolution.fields, field)
		}
	}

	resolution.apply = len(resolution.fields) > 0
	resolution.conflict = len(resolution.overlap) > 0

	return resolution
}

// incomingSyncFields are the fields a change writes: its fields, or all of
// its record's for a create.
func incomingSyncFields(change *syncChange) []string {
	if len(change.Fields) > 0 && change.Action != syncActionCreate {
		return change.Fields
	}

	fields := make([]string, 0, len(change.Data))
	for k := range change.Data {
		if k != "id" && k != "_id" {
			fields = append(fields, k)
		}
	}
	sort.Strings(fields)

	return fields
}

// apply writes a change received from another node, unless one here
// supersedes it, and logs it so it's applied once and passed on. A returned
// error means the change was refused.
func (s *syncEngine) apply(change *syncChange) error {
	model := s.app.models[change.Model]
	policy := s.policyOf(model)
	if policy == nil {
		return fmt.Errorf("%s isn't synced here", change.Model)
	}

	if change.ChangeId == "" || change.DocumentId == "" {
		return errors.New("change without changeId or documentId")
	}

	switch change.Action {
	case syncActionCreate, syncActionUpdate, syncActionDelete:
	default:
		return fmt.Errorf("unknown action %q", change.Action)
	}

	fromCloud := change.NodeId == syncCloudNodeId
	if policy.owner == syncOwnerCloud && !fromCloud {
		return fmt.Errorf("%s can only be changed on the cloud", change.Model)
	}
	if policy.owner == syncOwnerEdge && fromCloud {
		return fmt.Errorf("%s can only be changed on the on-premises server", change.Model)
	}

	changeModel := s.app.models[syncChangeModelName]
	if rawQuery(changeModel).Count(datatype.DataMap{"changeId": change.ChangeId}) > 0 {
		return nil // applied already: a retry, or a link change sent again
	}

	s.version.observe(change.Version)

	later := decodeSyncChanges(rawQuery(changeModel).Find(datatype.DataMap{
		"model":      change.Model,
		"documentId": change.DocumentId,
		"version":    datatype.DataMap{"greaterThan": float64(change.Version)},
		"skipped":    false,
	}))

	resolution := resolveSyncChange(change, later)
	current := findSyncRecord(model, change.DocumentId)

	if resolution.conflict && policy.strategy == syncStrategyManual {
		var local datatype.DataMap
		if current != nil {
			local = syncRecordData(model, *current)
		}

		s.recordConflict(change, policy.recordTenant(syncRecordDataOr(change, local)), "changed on both sides", local, resolution.overlap)
	}

	if resolution.apply {
		if err := s.write(model, change, resolution.fields, current != nil); err != nil {
			return err
		}
	}

	logged := *change
	logged.Fields = resolution.fields
	logged.Data = nil

	if logged.Fields == nil && change.Action == syncActionUpdate {
		logged.Fields = incomingSyncFields(change)
	}

	if change.Action != syncActionDelete {
		if record := findSyncRecord(model, change.DocumentId); record != nil {
			logged.Data = syncRecordData(model, *record)
			logged.TenantId = policy.recordTenant(*record)
		}
	}

	if logged.TenantId == "" && current != nil {
		logged.TenantId = policy.recordTenant(*current)
	}

	if _, err := rawCreate(changeModel, s.logEntry(&logged, !resolution.apply)); err != nil {
		return err
	}

	if resolution.apply {
		s.app.invalidateCaches(model.Name)
		s.app.socketServer.Of("/").broadcastToTenant(logged.TenantId, "database", datatype.DataMap{
			"action": change.Action,
			"model":  model.Name,
		})
	}

	return nil
}

func syncRecordDataOr(change *syncChange, local datatype.DataMap) datatype.DataMap {
	if change.Data != nil {
		return change.Data
	}

	return local
}

// write stores a received change: deletes the record, creates it from the
// change's copy when it isn't here, or sets the given fields on it.
func (s *syncEngine) write(model *DataModel, change *syncChange, fields []string, exists bool) error {
	if change.Action == syncActionDelete {
		if !exists {
			return nil
		}

		q := model.Query()
		q.Where("_id", change.DocumentId)
		_, err := q.collection().delete()
		return err
	}

	data := syncNormalize(model, change.Data)

	if !exists {
		data["id"] = change.DocumentId
		delete(data, "_id")
		_, err := rawCreate(model, data)
		return err
	}

	if fields == nil {
		fields = incomingSyncFields(change)
	}

	update := datatype.DataMap{}
	for _, field := range fields {
		if v, ok := data[field]; ok {
			update[field] = v
		}
	}

	if len(update) == 0 {
		return nil
	}

	_, err := rawUpdate(model, change.DocumentId, update)
	return err
}

// syncNormalize turns values decoded from JSON back into the types the
// model's fields are written with; formatInputData does the rest. Numbers
// arrive as float64, which helper.ToInt misreads.
func syncNormalize(model *DataModel, data map[string]interface{}) datatype.DataMap {
	kinds := model.fieldKindSets()
	normalized := make(datatype.DataMap, len(data))

	for k, v := range data {
		if f, ok := v.(float64); ok && kinds.number[k] {
			v = int(f)
		}

		normalized[k] = v
	}

	return normalized
}

// recordConflict keeps a change that was refused or lost to a local one, for
// an admin to review. Protected fields (password hashes) aren't copied.
func (s *syncEngine) recordConflict(change *syncChange, tenantId string, reason string, local datatype.DataMap, fields []string) {
	model := s.app.models[change.Model]
	data := datatype.DataMap{
		"nodeId":     change.NodeId,
		"changeId":   change.ChangeId,
		"model":      change.Model,
		"documentId": change.DocumentId,
		"action":     change.Action,
		"fields":     fields,
		"localData":  maskSyncData(model, local),
		"remoteData": maskSyncData(model, change.Data),
		"reason":     reason,
		"status":     "open",
	}

	if fields == nil {
		data["fields"] = change.Fields
	}
	if tenantId == "" {
		tenantId = change.TenantId
	}
	if tenantId != "" {
		data[TenantIDKey] = tenantId
	}

	if conflictModel := s.app.models[syncConflictModelName]; conflictModel != nil {
		if _, err := rawCreate(conflictModel, data); err != nil {
			logger.Error("Sync", "could not record conflict", err.Error())
		}
	}
}

func maskSyncData(model *DataModel, data map[string]interface{}) datatype.DataMap {
	if data == nil {
		return nil
	}

	masked := make(datatype.DataMap, len(data))
	for k, v := range data {
		if model != nil && helper.Contains(model.Protected, k) {
			continue
		}

		masked[k] = v
	}

	return masked
}

// state reads a value the sync engine keeps between rounds.
func (s *syncEngine) state(name string) interface{} {
	record := rawQuery(s.app.models[syncStateModelName]).FindOne(datatype.DataMap{"name": name})
	if helper.IsEmpty(record) {
		return nil
	}

	return (*record)["value"]
}

func (s *syncEngine) setState(name string, value interface{}) error {
	model := s.app.models[syncStateModelName]
	data := datatype.DataMap{"value": value, "updatedAt": helper.GetTimestamp(nil)}

	if record := rawQuery(model).FindOne(datatype.DataMap{"name": name}); helper.IsNotEmpty(record) {
		_, err := rawUpdate(model, syncRecordId(*record), data)
		return err
	}

	data["name"] = name
	_, err := rawCreate(model, data)
	return err
}

// syncInt64 reads a number however the database or JSON returned it.
func syncInt64(value interface{}) int64 {
	switch n := value.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case float64:
		return int64(n)
	case float32:
		return int64(n)
	case json.Number:
		if i, err := n.Int64(); err == nil {
			return i
		}
		f, _ := n.Float64()
		return int64(f)
	case string:
		if i, err := strconv.ParseInt(n, 10, 64); err == nil {
			return i
		}
		f, _ := strconv.ParseFloat(n, 64)
		return int64(f)
	}

	return 0
}

// cleanup drops log entries older than config sync.retentionDays, at most
// hourly. An edge keeps its own entries until they're pushed.
func (s *syncEngine) cleanup() {
	days := s.config.RetentionDays
	if days < 0 || time.Since(s.lastCleanup) < time.Hour {
		return
	}
	if days == 0 {
		days = defaultSyncRetentionDays
	}

	s.lastCleanup = time.Now()
	cutoff := time.Now().UTC().AddDate(0, 0, -days)
	model := s.app.models[syncChangeModelName]

	remove := func(where datatype.DataMap) {
		q := model.Query()
		q.WhereAll(where)
		q.Where("createdAt", datatype.DataMap{"lessThan": cutoff})
		q.collection().delete()
	}

	if s.role == config.SyncRoleCloud {
		remove(datatype.DataMap{})
		return
	}

	remove(datatype.DataMap{"nodeId": datatype.DataMap{"notEqualTo": s.nodeId}})
	remove(datatype.DataMap{
		"nodeId": s.nodeId,
		"seq":    datatype.DataMap{"lessThanOrEqualTo": float64(syncInt64(s.state("pushCursor")))},
	})
}

// start loads the clocks and starts the edge's sync loop or the cloud's log
// cleanup. Called by Start; safe to call again.
func (s *syncEngine) start() {
	if s == nil {
		return
	}

	s.startOnce.Do(func() {
		if s.role == config.SyncRoleEdge {
			logger.Info("Sync", "edge "+s.nodeId+" syncing with "+s.config.UpstreamUrl)
			go s.loop()
			return
		}

		logger.Info("Sync", "cloud serving edges at "+s.app.AppendBaseUrl(s.route()))
		go func() {
			for {
				s.cleanup()
				time.Sleep(time.Hour)
			}
		}()
	})
}
