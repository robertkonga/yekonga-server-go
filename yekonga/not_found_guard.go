package yekonga

import (
	"net/http"
	"sync"
	"time"
)

// notFoundCounter is a fixed one-second window counter: it resets whenever a
// hit lands in a different second than the one it's currently tracking.
type notFoundCounter struct {
	second int64
	count  int
}

// notFoundGuard blocks clients that flood the server with requests to
// non-existent routes — the signature of an automated path-scanning bot
// (probing for /wp-admin, /.env, etc.) rather than a real user, and a cheap
// way for an attacker to burn CPU/IO on route lookups and 404 rendering.
type notFoundGuard struct {
	mut               sync.Mutex
	counters          map[string]*notFoundCounter // client key -> current-second 404 counter
	blocked           map[string]time.Time        // client key -> unblock time
	requestsPerSecond int
	blockDuration     time.Duration
}

func newNotFoundGuard(requestsPerSecond int, blockDuration time.Duration) *notFoundGuard {
	guard := &notFoundGuard{
		counters:          make(map[string]*notFoundCounter),
		blocked:           make(map[string]time.Time),
		requestsPerSecond: requestsPerSecond,
		blockDuration:     blockDuration,
	}

	go guard.evictLoop()

	return guard
}

// isBlocked reports whether key is currently serving out a block.
func (g *notFoundGuard) isBlocked(key string) bool {
	g.mut.Lock()
	defer g.mut.Unlock()

	until, ok := g.blocked[key]
	if !ok {
		return false
	}

	if time.Now().After(until) {
		delete(g.blocked, key)
		return false
	}

	return true
}

// recordNotFound registers one 404 response for key. If key has now sent
// more than requestsPerSecond 404s within the same second, it is blocked for
// blockDuration.
func (g *notFoundGuard) recordNotFound(key string) {
	g.mut.Lock()
	defer g.mut.Unlock()

	now := time.Now()
	second := now.Unix()

	counter, ok := g.counters[key]
	if !ok || counter.second != second {
		counter = &notFoundCounter{second: second}
		g.counters[key] = counter
	}

	counter.count++

	if counter.count > g.requestsPerSecond {
		g.blocked[key] = now.Add(g.blockDuration)
		delete(g.counters, key)
	}
}

// evictLoop periodically drops stale entries so memory doesn't grow
// unbounded under a sustained flood from many distinct/spoofed source IPs.
func (g *notFoundGuard) evictLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()

		g.mut.Lock()
		for key, until := range g.blocked {
			if now.After(until) {
				delete(g.blocked, key)
			}
		}
		for key, counter := range g.counters {
			if now.Unix()-counter.second > 5 {
				delete(g.counters, key)
			}
		}
		g.mut.Unlock()
	}
}

// ensureNotFoundGuard lazily builds the guard from config, once.
func (y *YekongaData) ensureNotFoundGuard() {
	y.notFoundGuardOnce.Do(func() {
		cfg := y.Config.Security.NotFoundGuard

		requestsPerSecond := cfg.RequestsPerSecond
		if requestsPerSecond <= 0 {
			requestsPerSecond = 20
		}

		blockHours := cfg.BlockHours
		if blockHours <= 0 {
			blockHours = 24
		}

		y.notFoundGuard = newNotFoundGuard(requestsPerSecond, time.Duration(blockHours)*time.Hour)
	})
}

// checkNotFoundBlock reports whether the request may proceed. When the
// client is currently blocked it writes a 403 itself and returns false, so
// the caller should stop handling the request immediately. It is a no-op
// (always allows) when config.Security.NotFoundGuard.Enabled is false.
//
// This runs before anything else in ServeHTTP, including static file
// lookups, so a blocked source gets nothing back for the duration of the block.
func (y *YekongaData) checkNotFoundBlock(w http.ResponseWriter, r *http.Request) bool {
	if !y.Config.Security.NotFoundGuard.Enabled {
		return true
	}

	y.ensureNotFoundGuard()

	key := clientIP(r, y.Config.Security.TrustProxyHeaders)

	if !y.notFoundGuard.isBlocked(key) {
		return true
	}

	http.Error(w, "forbidden", http.StatusForbidden)

	return false
}

// recordNotFound registers one 404 response against the request's client,
// which may trip the not-found guard's block. It is a no-op when
// config.Security.NotFoundGuard.Enabled is false.
func (y *YekongaData) recordNotFound(r *http.Request) {
	if !y.Config.Security.NotFoundGuard.Enabled {
		return
	}

	y.ensureNotFoundGuard()

	key := clientIP(r, y.Config.Security.TrustProxyHeaders)

	y.notFoundGuard.recordNotFound(key)
}
