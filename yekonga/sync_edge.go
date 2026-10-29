package yekonga

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/robertkonga/yekonga-server-go/config"
	"github.com/robertkonga/yekonga-server-go/datatype"
	"github.com/robertkonga/yekonga-server-go/helper"
	"github.com/robertkonga/yekonga-server-go/helper/logger"
)

// An edge syncs in rounds: download its data the first time (bootstrap),
// push its own changes, then pull the cloud's. A round runs every
// sync.intervalSeconds, and shortly after a local write. While the cloud is
// unreachable rounds fail quietly and the edge keeps working on its own
// data; the next round that gets through catches up from the cursors.

const (
	syncStatePushCursor   = "pushCursor"   // seq of the last own change the cloud has
	syncStatePullCursor   = "pullCursor"   // the cloud's seq up to which changes were pulled
	syncStateBootstrapped = "bootstrapped" // the edge has downloaded its data

	syncRequestTimeout = time.Minute
)

var syncHTTPClient = &http.Client{Timeout: syncRequestTimeout}

// SyncNow starts a sync round on an edge without waiting for the next one.
func (y *YekongaData) SyncNow() {
	if y.sync != nil && y.sync.role == config.SyncRoleEdge {
		y.sync.syncSoon()
	}
}

// SyncStatus reports an edge's connection to the cloud and how many of its
// changes are waiting to be sent, e.g. for an "offline, 12 changes pending"
// indicator. Empty when sync is off.
func (y *YekongaData) SyncStatus() datatype.DataMap {
	s := y.sync
	if s == nil {
		return datatype.DataMap{}
	}

	status := datatype.DataMap{
		"role":   string(s.role),
		"nodeId": s.nodeId,
	}

	if s.role != config.SyncRoleEdge {
		return status
	}

	s.statusMu.Lock()
	status["online"] = s.online
	status["lastError"] = s.lastError
	status["lastSyncAt"] = nil
	if !s.lastSyncAt.IsZero() {
		status["lastSyncAt"] = s.lastSyncAt
	}
	s.statusMu.Unlock()

	status["pending"] = rawQuery(y.models[syncChangeModelName]).Count(datatype.DataMap{
		"nodeId": s.nodeId,
		"seq":    datatype.DataMap{"greaterThan": float64(syncInt64(s.state(syncStatePushCursor)))},
	})
	status["openConflicts"] = 0
	if conflicts := y.models[syncConflictModelName]; conflicts != nil {
		status["openConflicts"] = rawQuery(conflicts).Count(datatype.DataMap{"status": "open"})
	}

	return status
}

// statusHandler serves SyncStatus on an edge to signed-in users:
// GET <route>/status.
func (s *syncEngine) statusHandler(req *Request, res *Response) {
	masterKey := helper.ToString(req.GetContext(string(MasterKey)))
	isMaster := masterKey != "" && masterKey == s.app.Config.MasterKey

	if !isMaster && req.Auth() == nil {
		writeSyncError(res, http.StatusUnauthorized, "Must be authorized/login")
		return
	}

	res.Json(s.app.SyncStatus())
}

// syncSoon asks the loop for a round without blocking the caller.
func (s *syncEngine) syncSoon() {
	select {
	case s.kick <- struct{}{}:
	default:
	}
}

func (s *syncEngine) interval() time.Duration {
	if s.config.IntervalSeconds > 0 {
		return time.Duration(s.config.IntervalSeconds) * time.Second
	}

	return defaultSyncInterval
}

func (s *syncEngine) loop() {
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
		case <-s.kick:
			// Let a burst of writes finish and settle, so it goes in one push.
			time.Sleep(syncSettle + time.Second)
		}

		s.round()
		timer.Reset(s.interval())
	}
}

// round runs one sync round, unless one is running already.
func (s *syncEngine) round() {
	if !s.running.TryLock() {
		return
	}
	defer s.running.Unlock()

	s.recordRound(s.syncOnce())
}

// recordRound keeps a round's outcome for SyncStatus.
func (s *syncEngine) recordRound(err error) {
	s.statusMu.Lock()
	wasOnline, neverSynced := s.online, s.lastSyncAt.IsZero()
	s.online = err == nil
	if err == nil {
		s.lastSyncAt = time.Now()
		s.lastError = ""
	} else {
		s.lastError = err.Error()
	}
	s.statusMu.Unlock()

	// Logged on a change only: an edge offline for a week would otherwise
	// log every interval.
	switch {
	case err != nil && (wasOnline || neverSynced):
		logger.Error("Sync", "cloud unreachable or refused the sync, working offline:", err.Error())
	case err == nil && !wasOnline:
		logger.Success("Sync", "connected to the cloud")
	}
}

// syncOnce is one round: bootstrap if needed, push, pull.
func (s *syncEngine) syncOnce() error {
	if s.state(syncStateBootstrapped) != true {
		if err := s.bootstrap(false); err != nil {
			return err
		}
	}

	if err := s.push(); err != nil {
		return err
	}

	if err := s.pull(); err != nil {
		return err
	}

	s.cleanup()

	return nil
}

// request calls one of the cloud's sync endpoints and decodes its JSON
// response into out.
func (s *syncEngine) request(method string, path string, query url.Values, body interface{}, out interface{}) error {
	endpoint := strings.TrimSuffix(s.config.UpstreamUrl, "/") + s.app.AppendBaseUrl(s.route()+path)
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, endpoint, reader)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set(syncNodeHeader, s.nodeId)
	req.Header.Set(syncKeyHeader, s.config.NodeKey)

	resp, err := syncHTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var failure struct {
			Error string `json:"error"`
		}
		json.Unmarshal(data, &failure)
		if failure.Error == "" {
			failure.Error = strings.TrimSpace(string(data))
		}

		return fmt.Errorf("%s %s: %d %s", method, path, resp.StatusCode, failure.Error)
	}

	return json.Unmarshal(data, out)
}

// push sends the edge's own changes the cloud doesn't have yet, oldest
// first. A change the cloud refuses is kept as a SyncConflict here too; the
// cursor moves past it either way, so one bad change can't hold up the rest.
func (s *syncEngine) push() error {
	changeModel := s.app.models[syncChangeModelName]
	batch := s.batchSize()

	for {
		cursor := syncInt64(s.state(syncStatePushCursor))

		q := rawQuery(changeModel)
		q.Where("nodeId", s.nodeId)
		q.Where("seq", datatype.DataMap{"greaterThan": float64(cursor), "lessThanOrEqualTo": float64(s.settledSeq())})
		q.OrderBy("seq", "asc").Take(batch)

		changes := decodeSyncChanges(q.Find(nil))
		if len(changes) == 0 {
			return nil
		}

		var response syncPushResponse
		if err := s.request(http.MethodPost, "/push", nil, datatype.DataMap{"changes": changes}, &response); err != nil {
			return err
		}

		s.version.observe(response.ServerTime)

		if len(response.Rejected) > 0 {
			byId := make(map[string]*syncChange, len(changes))
			for i := range changes {
				byId[changes[i].ChangeId] = &changes[i]
			}

			for _, rejected := range response.Rejected {
				logger.Error("Sync", "cloud refused a change:", rejected.Reason)
				if change := byId[rejected.ChangeId]; change != nil {
					s.recordConflict(change, change.TenantId, "refused by the cloud: "+rejected.Reason, change.Data, nil)
				}
			}
		}

		if err := s.setState(syncStatePushCursor, changes[len(changes)-1].Seq); err != nil {
			return err
		}

		if len(changes) < batch {
			return nil
		}
	}
}

// pull applies the cloud's changes since the cursor. A change refused here is
// kept as a SyncConflict and skipped.
func (s *syncEngine) pull() error {
	for {
		cursor := syncInt64(s.state(syncStatePullCursor))

		var response syncPullResponse
		query := url.Values{
			"cursor": {strconv.FormatInt(cursor, 10)},
			"limit":  {strconv.Itoa(s.batchSize())},
		}
		if err := s.request(http.MethodGet, "/pull", query, nil, &response); err != nil {
			return err
		}

		if cursor > 0 && response.OldestSeq > cursor {
			// The cloud dropped changes this edge never got (it was offline
			// longer than sync.retentionDays): download everything again.
			logger.Info("Sync", "offline longer than the cloud keeps changes, downloading all data again")
			if err := s.bootstrap(true); err != nil {
				return err
			}
			continue
		}

		s.version.observe(response.ServerTime)

		for i := range response.Changes {
			change := &response.Changes[i]
			if err := s.apply(change); err != nil {
				logger.Error("Sync", "could not apply a change from the cloud:", err.Error())
				s.recordConflict(change, change.TenantId, err.Error(), nil, nil)
			}
		}

		if err := s.setState(syncStatePullCursor, response.Cursor); err != nil {
			return err
		}

		if !response.HasMore {
			return nil
		}
	}
}

// bootstrap downloads all of the edge's data from the cloud, then pulls
// changes from where the download started. Records already here are
// overwritten with the cloud's copy, so an edge pushes before it re-runs
// this. On a re-download (prune), records here the cloud no longer has are
// deleted, except ones changed here and not pushed yet.
func (s *syncEngine) bootstrap(prune bool) error {
	var info syncInfoResponse
	if err := s.request(http.MethodGet, "/info", nil, nil, &info); err != nil {
		return err
	}

	s.version.observe(info.ServerTime)

	pending := map[string]bool{}
	if prune {
		q := rawQuery(s.app.models[syncChangeModelName])
		q.Where("nodeId", s.nodeId)
		q.Where("seq", datatype.DataMap{"greaterThan": float64(syncInt64(s.state(syncStatePushCursor)))})

		for _, change := range decodeSyncChanges(q.Find(nil)) {
			pending[change.Model+":"+change.DocumentId] = true
		}
	}

	for _, name := range info.Models {
		policy := s.policies[name]
		if policy == nil {
			continue // not synced on this edge
		}

		seen := map[string]bool{}
		after := ""

		for {
			var page syncSnapshotResponse
			query := url.Values{
				"model": {name},
				"after": {after},
				"limit": {strconv.Itoa(s.batchSize())},
			}
			if err := s.request(http.MethodGet, "/snapshot", query, nil, &page); err != nil {
				return err
			}

			for _, record := range page.Records {
				id := idString(record["id"])
				if id == "" {
					continue
				}

				seen[id] = true
				change := &syncChange{Model: name, DocumentId: id, Action: syncActionCreate, Data: record}
				if err := s.write(policy.model, change, nil, findSyncRecord(policy.model, id) != nil); err != nil {
					return fmt.Errorf("download %s %s: %w", name, id, err)
				}
			}

			if !page.HasMore || page.After == "" {
				break
			}
			after = page.After
		}

		if prune {
			s.prune(policy, seen, pending)
		}

		s.app.invalidateCaches(name)
	}

	// Changes the cloud logged while the download ran are pulled again;
	// applying one twice leaves the same result.
	if err := s.setState(syncStatePullCursor, info.Seq-syncSettle.Microseconds()); err != nil {
		return err
	}

	return s.setState(syncStateBootstrapped, true)
}

// prune deletes a model's records here that weren't in the download.
func (s *syncEngine) prune(policy *syncPolicy, seen map[string]bool, pending map[string]bool) {
	tenantId := s.edgeTenantId()
	after := ""

	for {
		q := policy.model.Query()
		switch policy.scope {
		case syncScopeTenantSelf:
			q.Where("_id", tenantId)
		case syncScopeTenant:
			if tenantId != "" {
				q.Where(TenantIDKey, tenantId)
			}
		}
		if after != "" {
			q.Where("_id", datatype.DataMap{"greaterThan": after})
		}
		q.OrderBy("_id", "asc").Take(s.batchSize())

		rows := q.collection().find()
		if rows == nil || len(*rows) == 0 {
			return
		}

		for _, row := range *rows {
			id := syncRecordId(row)
			if seen[id] || pending[policy.model.Name+":"+id] {
				continue
			}

			remove := policy.model.Query()
			remove.Where("_id", id)
			remove.collection().delete()
		}

		if len(*rows) < s.batchSize() {
			return
		}
		after = syncRecordId((*rows)[len(*rows)-1])
	}
}

var errSyncDisabled = errors.New("sync is off")

// SyncOnce runs a sync round on an edge right away and returns its error,
// e.g. from a "sync now" button or a test. Rounds don't overlap: it waits for
// one already running.
func (y *YekongaData) SyncOnce() error {
	s := y.sync
	if s == nil || s.role != config.SyncRoleEdge {
		return errSyncDisabled
	}

	s.running.Lock()
	defer s.running.Unlock()

	err := s.syncOnce()
	s.recordRound(err)

	return err
}
