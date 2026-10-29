package yekonga

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/plugins/mongo-driver/bson"
)

// The cloud's sync endpoints, under config sync.route (default /sync). An
// edge authenticates with the X-Sync-Node and X-Sync-Key headers, and only
// ever sees or changes its own tenant's records.
const (
	syncNodeHeader = "X-Sync-Node"
	syncKeyHeader  = "X-Sync-Key"

	// syncMaxPushChanges caps one push, whatever the edge's batch size.
	syncMaxPushChanges = 5000
)

// syncNodeInfo is the SyncNode an edge's request authenticated as.
type syncNodeInfo struct {
	id       string
	tenantId string
}

type syncRejection struct {
	ChangeId string `json:"changeId"`
	Reason   string `json:"reason"`
}

type syncPushResponse struct {
	Accepted   []string        `json:"accepted"`
	Rejected   []syncRejection `json:"rejected"`
	ServerTime int64           `json:"serverTime"`
}

type syncPullResponse struct {
	Changes    []syncChange `json:"changes"`
	Cursor     int64        `json:"cursor"`
	HasMore    bool         `json:"hasMore"`
	OldestSeq  int64        `json:"oldestSeq"`
	ServerTime int64        `json:"serverTime"`
}

type syncSnapshotResponse struct {
	Records []map[string]interface{} `json:"records"`
	After   string                   `json:"after"`
	HasMore bool                     `json:"hasMore"`
}

type syncInfoResponse struct {
	NodeId     string   `json:"nodeId"`
	TenantId   string   `json:"tenantId"`
	Seq        int64    `json:"seq"`
	Models     []string `json:"models"`
	ServerTime int64    `json:"serverTime"`
}

// isSyncPath reports whether path is one of the cloud's sync endpoints,
// which authenticate the edge themselves.
func (y *YekongaData) isSyncPath(path string) bool {
	if y.sync == nil || y.sync.role != config.SyncRoleCloud {
		return false
	}

	return strings.HasPrefix(path, y.AppendBaseUrl(y.sync.route())+"/")
}

func (y *YekongaData) initializeSyncRoutes() {
	s := y.sync
	if s == nil {
		return
	}

	route := s.route()

	if s.role == config.SyncRoleEdge {
		y.Get(route+"/status", s.statusHandler)
		return
	}

	y.Get(route+"/info", s.nodeHandler(s.infoHandler))
	y.Post(route+"/push", s.nodeHandler(s.pushHandler))
	y.Get(route+"/pull", s.nodeHandler(s.pullHandler))
	y.Get(route+"/snapshot", s.nodeHandler(s.snapshotHandler))
	y.Post(route+"/nodes", s.createNodeHandler)
	y.SetPublicRoute(route + "/")
}

// CreateSyncNode registers an edge server for a tenant (empty without
// tenants) and returns the nodeId and nodeKey for its config sync section.
// Only a hash of the key is stored, so it can't be shown again.
func (y *YekongaData) CreateSyncNode(tenantId string, name string) (nodeId string, nodeKey string, err error) {
	if y.sync == nil || y.sync.role != config.SyncRoleCloud {
		return "", "", errors.New("sync nodes are created on the cloud (config sync.role \"cloud\")")
	}

	if y.Config.HasTenant {
		tenant := y.models[tenantModelName]
		if tenantId == "" || tenant == nil || findSyncRecord(tenant, tenantId) == nil {
			return "", "", fmt.Errorf("tenant %q not found", tenantId)
		}
	}

	nodeKey = helper.GetRandomString(48, "hex")
	data := datatype.DataMap{
		"name":    name,
		"keyHash": helper.HashRefreshToken(nodeKey),
		"status":  "active",
	}
	if tenantId != "" {
		data[TenantIDKey] = tenantId
	}

	record, err := rawCreate(y.models[syncNodeModelName], data)
	if err != nil {
		return "", "", err
	}

	y.sync.edgeTenantsMu.Lock()
	delete(y.sync.edgeTenants, tenantId)
	y.sync.edgeTenantsMu.Unlock()

	return syncRecordId(*record), nodeKey, nil
}

// SetSyncNodeActive turns an edge's access to the cloud on or off. An
// inactive edge's requests are refused; it keeps working offline.
func (y *YekongaData) SetSyncNodeActive(nodeId string, active bool) error {
	if y.sync == nil || y.sync.role != config.SyncRoleCloud {
		return errors.New("sync nodes are managed on the cloud (config sync.role \"cloud\")")
	}

	model := y.models[syncNodeModelName]
	record := findSyncRecord(model, nodeId)
	if record == nil {
		return fmt.Errorf("sync node %q not found", nodeId)
	}

	status := "inactive"
	if active {
		status = "active"
	}

	if _, err := rawUpdate(model, nodeId, datatype.DataMap{"status": status}); err != nil {
		return err
	}

	y.sync.edgeTenantsMu.Lock()
	delete(y.sync.edgeTenants, idString((*record)[TenantIDKey]))
	y.sync.edgeTenantsMu.Unlock()

	return nil
}

// createNodeHandler is CreateSyncNode over HTTP, for the master key only:
// POST <route>/nodes {"tenantId": "...", "name": "..."}.
func (s *syncEngine) createNodeHandler(req *Request, res *Response) {
	masterKey := helper.ToString(req.GetContext(string(MasterKey)))
	if masterKey == "" || masterKey != s.app.Config.MasterKey {
		writeSyncError(res, http.StatusUnauthorized, "master key required")
		return
	}

	body := helper.ToDataMap(req.Body())
	nodeId, nodeKey, err := s.app.CreateSyncNode(helper.GetValueOfString(body, "tenantId"), helper.GetValueOfString(body, "name"))
	if err != nil {
		writeSyncError(res, http.StatusBadRequest, err.Error())
		return
	}

	res.Json(datatype.DataMap{"nodeId": nodeId, "nodeKey": nodeKey})
}

// nodeHandler authenticates the edge before calling handler.
func (s *syncEngine) nodeHandler(handler func(*syncNodeInfo, *Request, *Response)) Handler {
	return func(req *Request, res *Response) {
		node, err := s.authenticateNode(req)
		if err != nil {
			s.app.recordErrorResponse(req, http.StatusUnauthorized)
			writeSyncError(res, http.StatusUnauthorized, err.Error())
			return
		}

		handler(node, req, res)
	}
}

func (s *syncEngine) authenticateNode(req *Request) (*syncNodeInfo, error) {
	id := strings.TrimSpace(req.GetHeader(syncNodeHeader))
	key := req.GetHeader(syncKeyHeader)
	if id == "" || key == "" {
		return nil, errors.New("sync node credentials missing")
	}

	if _, err := bson.ObjectIDFromHex(id); err != nil {
		return nil, errors.New("sync node credentials invalid")
	}

	model := s.app.models[syncNodeModelName]
	record := findSyncRecord(model, id)
	if record == nil ||
		helper.GetValueOfString(record, "status") != "active" ||
		subtle.ConstantTimeCompare([]byte(helper.HashRefreshToken(key)), []byte(helper.GetValueOfString(record, "keyHash"))) != 1 {
		return nil, errors.New("sync node credentials invalid")
	}

	rawUpdate(model, id, datatype.DataMap{"lastSeenAt": helper.GetTimestamp(nil)})

	return &syncNodeInfo{id: id, tenantId: idString((*record)[TenantIDKey])}, nil
}

func writeSyncError(res *Response, status int, message string) {
	res.Status(status)
	res.Json(datatype.DataMap{"error": message})
}

func (s *syncEngine) infoHandler(node *syncNodeInfo, req *Request, res *Response) {
	models := make([]string, 0, len(s.policies))
	for name := range s.policies {
		models = append(models, name)
	}
	sort.Strings(models)

	res.Json(syncInfoResponse{
		NodeId:     node.id,
		TenantId:   node.tenantId,
		Seq:        s.seq.current(),
		Models:     models,
		ServerTime: s.version.current(),
	})
}

// syncNodeScope answers which records belong to an edge's tenant.
type syncNodeScope struct {
	s       *syncEngine
	node    *syncNodeInfo
	related map[string]map[string]bool // ids linked to the tenant, by policy throughModel.throughField
}

func (s *syncEngine) newNodeScope(node *syncNodeInfo) *syncNodeScope {
	return &syncNodeScope{s: s, node: node, related: map[string]map[string]bool{}}
}

// relatedIds are the ids of a relation-scoped model's records linked to the
// tenant, e.g. the ids of its users from TenantUser.userId.
func (sc *syncNodeScope) relatedIds(policy *syncPolicy) map[string]bool {
	key := policy.throughModel + "." + policy.throughField
	if ids, ok := sc.related[key]; ok {
		return ids
	}

	ids := map[string]bool{}
	if sc.node.tenantId != "" {
		q := sc.s.app.models[policy.throughModel].Query()
		q.Where(TenantIDKey, sc.node.tenantId)

		if rows := q.collection().find(); rows != nil {
			for _, row := range *rows {
				if id := idString(row[policy.throughField]); id != "" {
					ids[id] = true
				}
			}
		}
	}

	sc.related[key] = ids

	return ids
}

// changed forgets cached links after a change to a model linking records.
func (sc *syncNodeScope) changed(model string) {
	for key := range sc.related {
		if strings.HasPrefix(key, model+".") {
			delete(sc.related, key)
		}
	}
}

func sortedIds(ids map[string]bool) []string {
	list := make([]string, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	sort.Strings(list)

	return list
}

// checkPush refuses a pushed change to a record outside the edge's tenant,
// and pins tenant records to it.
func (sc *syncNodeScope) checkPush(change *syncChange) error {
	policy := sc.s.policies[change.Model]
	if policy == nil {
		return fmt.Errorf("%s isn't synced", change.Model)
	}

	change.NodeId = sc.node.id
	change.TenantId = sc.node.tenantId

	if sc.node.tenantId == "" {
		return nil
	}

	switch policy.scope {
	case syncScopeTenantSelf:
		if change.DocumentId != sc.node.tenantId {
			return errors.New("another tenant's record")
		}
	case syncScopeTenant:
		if change.Data != nil {
			change.Data[TenantIDKey] = sc.node.tenantId
		}

		if existing := findSyncRecord(policy.model, change.DocumentId); existing != nil && idString((*existing)[TenantIDKey]) != sc.node.tenantId {
			return errors.New("another tenant's record")
		}
	case syncScopeRelation:
		if findSyncRecord(policy.model, change.DocumentId) != nil && !sc.relatedIds(policy)[change.DocumentId] {
			return errors.New("a record not linked to this tenant")
		}
	}

	return nil
}

// pushHandler applies an edge's changes in order: POST <route>/push
// {"changes": [...]}. Refused changes are listed with the reason and kept as
// SyncConflicts.
func (s *syncEngine) pushHandler(node *syncNodeInfo, req *Request, res *Response) {
	var body struct {
		Changes []syncChange `json:"changes"`
	}

	if err := json.Unmarshal(helper.ToByte(req.Body()), &body); err != nil {
		writeSyncError(res, http.StatusBadRequest, "invalid body")
		return
	}

	if len(body.Changes) > syncMaxPushChanges {
		writeSyncError(res, http.StatusRequestEntityTooLarge, fmt.Sprintf("at most %d changes per push", syncMaxPushChanges))
		return
	}

	scope := s.newNodeScope(node)
	response := syncPushResponse{Accepted: []string{}, Rejected: []syncRejection{}}

	for i := range body.Changes {
		change := &body.Changes[i]

		err := scope.checkPush(change)
		if err == nil {
			err = s.apply(change)
		}

		if err != nil {
			response.Rejected = append(response.Rejected, syncRejection{ChangeId: change.ChangeId, Reason: err.Error()})
			s.recordConflict(change, node.tenantId, err.Error(), nil, nil)
			continue
		}

		scope.changed(change.Model)
		response.Accepted = append(response.Accepted, change.ChangeId)
	}

	rawUpdate(s.app.models[syncNodeModelName], node.id, datatype.DataMap{"lastPushAt": helper.GetTimestamp(nil)})

	response.ServerTime = s.version.current()
	res.Json(response)
}

// pullHandler sends an edge the changes for its tenant made since its
// cursor, oldest first: GET <route>/pull?cursor=<seq>&limit=<n>. Changes the
// edge made itself aren't sent back.
func (s *syncEngine) pullHandler(node *syncNodeInfo, req *Request, res *Response) {
	cursor, _ := strconv.ParseInt(req.Query("cursor"), 10, 64)
	limit := req.QueryInt("limit", s.batchSize())
	if limit <= 0 || limit > syncMaxPushChanges {
		limit = s.batchSize()
	}

	settled := s.settledSeq()
	scope := s.newNodeScope(node)
	changes, hasMore := s.changesFor(scope, cursor, settled, limit)

	response := syncPullResponse{
		Changes:   s.withLinkedRecords(scope, changes),
		Cursor:    cursor,
		HasMore:   hasMore,
		OldestSeq: syncInt64(rawQuery(s.app.models[syncChangeModelName]).Min("seq", nil)),
	}

	switch {
	case hasMore:
		response.Cursor = changes[len(changes)-1].Seq
	case settled > cursor:
		// Everything for this edge up to settled was sent, so it can move
		// on even if none of it was its tenant's.
		response.Cursor = settled
	}

	rawUpdate(s.app.models[syncNodeModelName], node.id, datatype.DataMap{"lastPullAt": helper.GetTimestamp(nil)})

	response.ServerTime = s.version.current()
	res.Json(response)
}

// changesFor are up to limit of the edge's changes with cursor < seq <=
// settled, oldest first, and whether there are more. Each kind of scope is
// its own query; together their first limit entries are the first limit
// overall.
func (s *syncEngine) changesFor(scope *syncNodeScope, cursor int64, settled int64, limit int) ([]syncChange, bool) {
	changeModel := s.app.models[syncChangeModelName]
	query := func(where datatype.DataMap) []syncChange {
		q := rawQuery(changeModel)
		q.WhereAll(where)
		q.Where("seq", datatype.DataMap{"greaterThan": float64(cursor), "lessThanOrEqualTo": float64(settled)})
		q.Where("nodeId", datatype.DataMap{"notEqualTo": scope.node.id})
		q.Where("skipped", false)
		q.OrderBy("seq", "asc").Take(limit)

		return decodeSyncChanges(q.Find(nil))
	}

	all := []syncChange{}
	hasMore := false
	collect := func(changes []syncChange) {
		all = append(all, changes...)
		hasMore = hasMore || len(changes) >= limit
	}

	if scope.node.tenantId != "" {
		collect(query(datatype.DataMap{TenantIDKey: scope.node.tenantId}))
	}

	global := []string{}
	for name, policy := range s.policies {
		switch policy.scope {
		case syncScopeGlobal:
			global = append(global, name)
		case syncScopeRelation:
			if ids := scope.relatedIds(policy); len(ids) > 0 {
				collect(query(datatype.DataMap{"model": name, "documentId": datatype.DataMap{"in": sortedIds(ids)}}))
			}
		}
	}

	if len(global) > 0 {
		sort.Strings(global)
		collect(query(datatype.DataMap{"model": datatype.DataMap{"in": global}}))
	}

	sort.SliceStable(all, func(i, j int) bool { return all[i].Seq < all[j].Seq })

	if len(all) > limit {
		all = all[:limit]
		hasMore = true
	}

	return all, hasMore
}

// withLinkedRecords puts the linked record before each change that links one
// to the tenant (a TenantUser's User): the record's earlier changes weren't
// the tenant's, so the edge never got them.
func (s *syncEngine) withLinkedRecords(scope *syncNodeScope, changes []syncChange) []syncChange {
	result := make([]syncChange, 0, len(changes))

	for i := range changes {
		change := changes[i]

		if change.Action != syncActionDelete {
			for name, policy := range s.policies {
				if policy.scope != syncScopeRelation || policy.throughModel != change.Model {
					continue
				}

				linkedId := idString(change.Data[policy.throughField])
				record := findSyncRecord(policy.model, linkedId)
				if linkedId == "" || record == nil {
					continue
				}

				result = append(result, syncChange{
					ChangeId:   "link:" + change.ChangeId + ":" + name,
					Seq:        change.Seq,
					Version:    s.latestVersion(name, linkedId),
					NodeId:     syncCloudNodeId,
					Model:      name,
					DocumentId: linkedId,
					Action:     syncActionUpdate,
					Data:       syncRecordData(policy.model, *record),
				})
			}
		}

		result = append(result, change)
	}

	return result
}

// latestVersion is the version of the last logged change to a record, or 0
// for one changed only before sync was on: any change an edge made wins over
// it.
func (s *syncEngine) latestVersion(model string, id string) int64 {
	return syncInt64(rawQuery(s.app.models[syncChangeModelName]).Max("version", datatype.DataMap{
		"model":      model,
		"documentId": id,
	}))
}

// snapshotHandler pages through a model's records for the edge's tenant, by
// id: GET <route>/snapshot?model=<name>&after=<id>&limit=<n>. A new edge
// downloads its data this way before pulling changes.
func (s *syncEngine) snapshotHandler(node *syncNodeInfo, req *Request, res *Response) {
	name := req.Query("model")
	policy := s.policies[name]
	if policy == nil {
		writeSyncError(res, http.StatusBadRequest, fmt.Sprintf("%s isn't synced", name))
		return
	}

	after := req.Query("after")
	limit := req.QueryInt("limit", s.batchSize())
	if limit <= 0 || limit > syncMaxPushChanges {
		limit = s.batchSize()
	}

	response := syncSnapshotResponse{Records: []map[string]interface{}{}}
	q := policy.model.Query()

	if node.tenantId != "" {
		switch policy.scope {
		case syncScopeTenantSelf:
			q.Where("_id", node.tenantId)
		case syncScopeTenant:
			q.Where(TenantIDKey, node.tenantId)
		case syncScopeRelation:
			ids := s.newNodeScope(node).relatedIds(policy)
			if len(ids) == 0 {
				res.Json(response)
				return
			}

			q.Where("_id", datatype.DataMap{"in": sortedIds(ids)})
		}
	}

	if after != "" {
		q.Where("_id", datatype.DataMap{"greaterThan": after})
	}

	q.OrderBy("_id", "asc").Take(limit)

	if rows := q.collection().find(); rows != nil {
		for _, row := range *rows {
			response.Records = append(response.Records, syncRecordData(policy.model, row))
		}

		if len(*rows) > 0 {
			response.After = syncRecordId((*rows)[len(*rows)-1])
		}
		response.HasMore = len(*rows) >= limit
	}

	res.Json(response)
}
