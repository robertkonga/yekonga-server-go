# TODO — Performance review

Findings from a performance review of the `yekonga` framework (2026-09-26), ordered by priority.
Line references point to the code as of commit `359c718`; re-check them before editing.

## Measured baselines

Benchmarked with a throwaway `_test.go` in `yekonga/` (run with `go test -vet=off`, see note at the bottom).

| Hot path | Current | After fix | Gain |
|---|---|---|---|
| Route matching, 40 routes | 553 µs, 218 KB, 2,530 allocs per request | 4.3 µs, 1 alloc | ~130× |
| `helper.GetValueOf(map, "name")` | 21.6 µs, 104 allocs | 23 ns (direct lookup) | ~1000× |
| `helper.ToSlug` (runs on every trigger lookup) | 9.6 µs, 52 allocs | ~0 with its regex precompiled | — |

Route matching alone caps throughput at roughly 1,800 requests per second per core before any real work happens.

---

## P0 — Large impact, low risk

- [ ] **Precompile route regexes.** `matchRoute` calls `regexp.MustCompile` for every route on every request (`yekonga/main.go:366`). Compile once in `addRoute` and store a `*regexp.Regexp` on `Route`.
- [ ] **Hoist per-call regexes to package-level vars** in `helper/helpers.go`: `ToSlug` (:577), `ToUnderscore` (:648, :652, :678), `GetBaseUrl` (:1481), `GetMainDomain` (:1488), email/phone validators (:2007, :2025), and `helper/template.go` (:233, :237). Also `isValidParam` and `parseSegment` in `yekonga/main.go`.
- [ ] **Fix lock contention and deadlock risk on `y.mut`.**
  - `findRoute` takes an exclusive `y.mut.Lock()` on every request (`yekonga/main.go:434`). Use `RLock`; routes are effectively immutable after `Start`.
  - `triggerCallback` / `triggerAllCallback` hold `y.mut.RLock()` while user trigger code runs, including DB I/O (`yekonga/cloud_functions.go:322`, `:417`). Routing then waits on every in-flight trigger, and nested queries in triggers re-acquire the read lock, which can deadlock when a writer is waiting. Lock only to look up the function, release, then call it.
- [ ] **Cache tenant resolution.** With `HasTenant`, each request runs 3 DB queries:
  - `TenantCatchMiddleware` looks up `Tenant` by host (`yekonga/middleware.go:85`).
  - `GetTenantConfig` runs the **same** `Tenant` lookup again, then `TenantConfig`, then a JSON round-trip via `helper.ConvertTo` (`yekonga/main_other_functions.go:17`, `:32`, `:92`).
  - Add an in-memory cache keyed by host (TTL ~30–60 s), invalidated from after-create/update/delete triggers on `Tenant` and `TenantConfig`.
  - Consider the same for `UserInfoMiddleware` when `IsAuthorizationServer` is on (`yekonga/middleware.go:302`).
- [ ] **Rewrite Mongo filter building as a single pass.**
  - `where()` rebuilds the full filter once per key (`yekonga/dbconnect_mongodb.go:512`); `extractWhereObject` rescans the whole map per map-valued key (:604). Cost grows much faster than the number of keys.
  - Relation filters run a `Find` sub-query (:764, :781) inside those passes, so the sub-query repeats: ~6× per `where()` for 3 keys with one relation filter, and `Paginate` calls `where()` twice.
  - **Security:** that sub-query has no `RequestContext`, so it skips the tenant filter and scans across tenants. Pass the parent's request context through.

## P1 — GraphQL

- [ ] **Remove the wasted second parse.** `yekonga/initializer.go:239-241` parses the query, converts the AST to JSON and back, and builds `QuerySelectors`. The only consumer is a loop that calls `fmt.Sprintf` and discards the result (`yekonga/graphql.go:2295`). Delete both.
- [ ] **Cache parsed and validated documents.** Add an LRU keyed by query string (or its hash) holding the AST and validation result, so repeated queries skip `parser.Parse` and validation.
- [ ] **Fix N+1 on relation fields.** Each parent row resolves each relation with its own query (`setModelParams`, `yekonga/graphql.go:2135`). A 50-row list with 3 relations issues 150+ extra queries. Add a per-request dataloader that batches by foreign key with `$in`; the commented-out code in `loadRelatedData` was heading this way.
- [ ] **Stop using JSON round-trips for type conversion on hot paths:**
  - `helper.ToMap[...]` on resolver args (`yekonga/graphql.go:2150`, `:2214`, `:2240`).
  - `Request.Auth()` marshals and unmarshals on every call (`yekonga/request.go:339`). Store a typed `AuthPayload` once in `UserInfoMiddleware`.
  - `helper.GetMapValue` copies the whole map via reflection to read one key (`helper/helpers.go:1656`). Fast-path `datatype.DataMap` / `map[string]interface{}` with a type assertion. Also remove the leftover `key == "data.clients"` debug branch.
- [ ] **Cheaper output formatting.** `formateOutputData` calls `GetBaseUrl` (regex) and `OriginDomain` (`url.Parse`) per row per file field (`yekonga/graphql.go:2341`). Compute the domain once per request.

## P1 — Database layer

- [ ] **Propagate request context and timeouts.** `DataModelQuery.collection()` uses `context.TODO()` (`yekonga/model_query.go:1430`), and the Mongo backend uses `context.TODO()` throughout. Queries can't be cancelled when the client disconnects and have no deadline. Pass `req.HttpRequest.Context()` with a configurable timeout.
- [ ] **Create indexes.** The framework never creates any. Add index declarations to `database.json` and an `EnsureIndexes` step at startup. Minimum: `tenantId` (added to every tenant query), foreign keys, and the tenant domain fields (`domain`, `subdomain`, `customDomain`, `customSubdomain`).
- [ ] **Remove extra round trips on writes** (`yekonga/dbconnect_mongodb.go:424-473`).
  - `create`: insert then `findOne`. Build the result from the inserted data and `InsertedID`.
  - `update`: update then `findOne` (plus another `findOne` for audit in `model_query.go:391`). Use `FindOneAndUpdate` with `ReturnDocument: After`.
- [ ] **Run `Paginate`'s count and find concurrently** (`yekonga/dbconnect_mongodb.go:187`). Consider `EstimatedDocumentCount` when the filter is empty.
- [ ] **Make the Mongo pool configurable** (`maxPoolSize`, `minPoolSize`, timeouts) in `config.json` (`yekonga/dbconnect.go:134`).

## P2 — Smaller, cheap to fix

- [ ] **Pool gzip writers.** `initGzip` allocates a new compressor (hundreds of KB) per response, even for tiny bodies (`yekonga/response.go:156`). Use `sync.Pool` and skip compression under ~1 KB. `Response.Write` also calls `WriteHeader` on every call (:168, :175); only write the header once.
- [ ] **Serialize socket broadcasts once.** `EmitToClient` marshals the payload for every client (`yekonga/socket.go:353`). Every create/update/delete broadcasts to *all* connected clients across all tenants (`yekonga/model_query.go:361`, `:421`, `:487`, `:699`). Marshal once and scope to a per-tenant room.
- [ ] **Remove per-request / per-call allocations:**
  - `ignorePaths` rebuilt on every request in `TokenMiddleware` (`yekonga/middleware.go:161`). Build once at startup.
  - `listAll` rebuilt on every `runTriggerAction` call (`yekonga/model_query.go:1287`). Make it a package-level set.
  - Token payload decoded via `json.Unmarshal([]byte(helper.ToJson(...)))` (`yekonga/middleware.go:239`).
  - `helper.Contains` over `IDKeys`, `DateFields`, `NumberFields`, `FloatFields` per field in `formatInputDataField` (`yekonga/model_query.go:1263`). Precompute sets on `DataModel`.
- [ ] **Reduce synchronous logging on hot paths:** `logger.Warn` for missing triggers on every query (`yekonga/cloud_functions.go:356`) and `console.Error` on every `Abort` (`yekonga/response.go:107`), which is costly under 404-scan traffic.

## Bugs found during the review

- [ ] `Authorization` header without `"Bearer"` panics on index `[1]` (`yekonga/middleware.go:217`).
- [ ] `find()` defers `cursor.Close` on a nil cursor when the query errors (`yekonga/dbconnect_mongodb.go:144`); same pattern in `count`, `sum`, `max`, `min`, `average`.
- [ ] Distinct `count` uses `RemainingBatchLength`, which is wrong above 101 groups (`yekonga/dbconnect_mongodb.go:253`). Use a `$count` stage.
- [ ] `isStaticPath` / `handleStaticFile` write `y.staticConfig` without a lock (data race) (`yekonga/main.go:553`, `:577`). `Static()` also pre-fills the slice with `nil` entries before appending (:527).
- [ ] Catch middlewares: the loop returns after the first one, so only one ever runs (`yekonga/main.go:854`).
- [ ] `mysqlConnect` and `sqlConnect` are empty (`yekonga/dbconnect.go:180`), so those backends never open a connection.
- [ ] `Request.Auth/Client/TokenPayload/TenantId/Tenant` read `r.Context` without the lock that `SetContext` takes (`yekonga/request.go:331-394`).

## Measurement

- [ ] Add `Benchmark*` tests for routing, middleware chain, `where()` building, and a GraphQL list query with relations, so each change above is measured.
- [ ] Add a `net/http/pprof` endpoint behind the master key for profiling in staging.
- [ ] Fix the non-constant format strings in `yekonga/initializer_other_routes.go` (:525, :610, :703) so `go test` passes `go vet` without `-vet=off`.

## Suggested order

1. **Quick wins (~1–2 days):** route regex precompile, `y.mut` locking, remove the GraphQL double parse, tenant cache, gzip pooling, and the panic/nil-cursor bugs.
2. **Structural (~1 week):** request contexts and timeouts, indexes, single-pass tenant-scoped `where()`, removing JSON round-trips.
3. **GraphQL:** dataloader for relations and the parsed-document cache.
