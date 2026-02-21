# Phase 1: Project Foundation + ES Client

## Overview

Bootstrap the `epm-go` project: Go module, directory layout, Makefile, and the Elasticsearch HTTP client with all 5 API endpoints. At the end of this phase the binary compiles, connects to ES, and prints cluster health.

The ES client is the most critical component: everything else depends on it being correct, tested, and reliable.

## Context

- New project at `/Users/dm/dev/epm-go/`
- Source of truth for API endpoints: Chrome extension at `/Users/dm/dev/elasticsearch-performance-monitoring/src/config/api.ts`
- Source of truth for ES response types: `/Users/dm/dev/elasticsearch-performance-monitoring/src/types/api.ts`
- All 5 endpoints are GET-only with `filter_path` query params and optional Basic Auth

## Development Approach

- **Testing approach**: Regular (code first, then tests)
- Complete each task fully before moving to next
- CRITICAL: every task MUST include tests
- CRITICAL: all tests must pass before starting next task

## Testing Strategy

- Unit tests via `net/http/httptest` — spin up a local HTTP server that returns fixture JSON
- Test each of the 5 endpoint methods independently
- Test error cases: non-200 response, timeout, context cancellation, invalid JSON
- Test Basic Auth header construction
- Test TLS skip-verify config is applied correctly
- Test URI parsing (user:pass extracted from URL userinfo)

## Implementation Steps

### Task 1: Go module + project structure

- [x] run `go mod init github.com/dm/epm-go` in `/Users/dm/dev/epm-go/`
- [x] create directory tree: `cmd/epm/`, `internal/client/`, `internal/model/`, `internal/engine/`, `internal/tui/`, `internal/format/`, `docs/plans/`
- [x] create `Makefile` with targets: `build`, `test`, `lint`, `run`
  - `build`: `go build -o bin/epm ./cmd/epm`
  - `test`: `go test ./...`
  - `lint`: `go vet ./...`
  - `run`: `go run ./cmd/epm`
- [x] create `.gitignore` with `bin/`, `*.test`, `*.out`
- [x] verify `go build ./...` succeeds (empty packages)

### Task 2: ES client interface + DefaultClient scaffold

- [x] create `internal/client/client.go` with `ESClient` interface:
  ```go
  type ESClient interface {
      GetClusterHealth(ctx context.Context) (*ClusterHealth, error)
      GetNodes(ctx context.Context) ([]NodeInfo, error)
      GetNodeStats(ctx context.Context) (*NodeStatsResponse, error)
      GetIndices(ctx context.Context) ([]IndexInfo, error)
      GetIndexStats(ctx context.Context) (*IndexStatsResponse, error)
      Ping(ctx context.Context) error
      BaseURL() string
  }
  ```
- [x] create `ClientConfig` struct: `BaseURL`, `Username`, `Password`, `InsecureSkipVerify bool`, `RequestTimeout time.Duration` (default 10s)
- [x] create `DefaultClient` struct implementing `ESClient` with `*http.Client` and `ClientConfig`
- [x] implement `NewDefaultClient(cfg ClientConfig) (*DefaultClient, error)` — builds `http.Transport` with `tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify}`, sets `Timeout`
- [x] implement `BaseURL() string` returning configured base URL
- [x] implement internal `doGet(ctx, path) ([]byte, error)` — sets `Accept: application/json`, adds `Authorization: Basic` header when credentials present, returns body or error on non-2xx
- [x] implement `Ping(ctx)` — calls `/_cluster/health` with 1s timeout context override

### Task 3: ES response types

- [x] create `internal/client/types.go` with all response structs ported from `src/types/api.ts`:
  - `ClusterHealth`: `ClusterName`, `Status`, `NumberOfNodes`, `ActiveShards`
  - `NodeInfo`: `NodeRole` (json:`"node.role"`), `Name`, `IP`
  - `NodeStatsResponse`: `Nodes map[string]NodePerformanceStats`
  - `NodePerformanceStats`: nested structs for `Indices` (indexing + search totals), `OS.CPU.Percent`, `JVM.Mem.HeapUsed/HeapMax`, `FS.Total.Total/Available`
  - `IndexInfo`: `Index`, `Pri`, `Rep`, `PriStoreSize` (json:`"pri.store.size"`), `StoreSize` (json:`"store.size"`), `DocsCount` (json:`"docs.count"`)
  - `IndexStatsResponse`: `Indices map[string]IndexStatEntry`
  - `IndexStatEntry`: `Primaries *IndexStatShard`, `Total *IndexStatShard`
  - `IndexStatShard`: `Indexing *struct{IndexTotal, IndexTimeInMillis}`, `Search *struct{QueryTotal, QueryTimeInMillis}`, `Store *struct{SizeInBytes}`
- [x] all fields use `omitempty` where optional (OS, JVM, FS sub-fields)
- [x] verify JSON field tags exactly match ES response field names

### Task 4: Endpoint path constants

- [x] create `internal/client/endpoints.go` with constants exactly matching `src/config/api.ts`:
  ```go
  const (
      endpointClusterHealth = "/_cluster/health?filter_path=cluster_name,status,number_of_nodes,active_shards"
      endpointNodes         = "/_cat/nodes?v&format=json&h=node.role,name,ip&s=node.role,ip"
      endpointNodeStats     = "/_nodes/stats/indices,os,jvm,fs?filter_path=nodes.*.name,nodes.*.host,nodes.*.ip,nodes.*.roles,nodes.*.indices.indexing.index_total,nodes.*.indices.indexing.index_time_in_millis,nodes.*.indices.search.query_total,nodes.*.indices.search.query_time_in_millis,nodes.*.os.cpu.percent,nodes.*.jvm.mem.heap_used_in_bytes,nodes.*.jvm.mem.heap_max_in_bytes,nodes.*.fs.total.total_in_bytes,nodes.*.fs.total.available_in_bytes"
      endpointIndices       = "/_cat/indices?v&format=json&h=index,pri,rep,pri.store.size,store.size,docs.count&s=index"
      endpointIndexStats    = "/_stats?filter_path=indices.*.primaries.indexing.index_total,indices.*.primaries.indexing.index_time_in_millis,indices.*.total.search.query_total,indices.*.total.search.query_time_in_millis,indices.*.primaries.store.size_in_bytes,indices.*.total.store.size_in_bytes"
  )
  ```

### Task 5: Implement all 5 endpoint methods

- [x] implement `GetClusterHealth(ctx)` — calls `endpointClusterHealth`, decodes JSON into `ClusterHealth`
- [x] implement `GetNodes(ctx)` — calls `endpointNodes`, decodes JSON array into `[]NodeInfo`
- [x] implement `GetNodeStats(ctx)` — calls `endpointNodeStats`, decodes into `NodeStatsResponse`
- [x] implement `GetIndices(ctx)` — calls `endpointIndices`, decodes JSON array into `[]IndexInfo`
- [x] implement `GetIndexStats(ctx)` — calls `endpointIndexStats`, decodes into `IndexStatsResponse`
- [x] all methods strip trailing slash from BaseURL before building full URL

### Task 6: Client tests with httptest

- [x] create `internal/client/client_test.go`
- [x] write `TestGetClusterHealth` — mock server returns fixture JSON, verify `Status="green"`, `NumberOfNodes=3`
- [x] write `TestGetNodes` — verify `NodeRole`, `Name`, `IP` parsed correctly
- [x] write `TestGetNodeStats` — verify nested indexing/search/OS/JVM/FS fields
- [x] write `TestGetIndices` — verify `Pri`, `Rep`, `DocsCount` with json field name `docs.count`
- [x] write `TestGetIndexStats` — verify primaries and total parsed separately
- [x] write `TestPing_Success` and `TestPing_Failure` (non-2xx response)
- [x] write `TestBasicAuth` — verify `Authorization` header sent when credentials set
- [x] write `TestHTTPError` — mock returns 401, verify error message contains status code
- [x] write `TestContextCancellation` — cancel context mid-request, verify error
- [x] run `go test ./internal/client/...` — all pass

### Task 7: CLI entry point

- [x] create `cmd/epm/main.go`:
  - parse positional arg as ES URI (`flag.Args()[0]`)
  - parse flags: `--interval 10s`, `--insecure`
  - extract username/password from `url.Parse()` userinfo
  - auto-set `InsecureSkipVerify=true` when scheme is `https` and `--insecure` flag given
  - build `ClientConfig`, create `DefaultClient`
  - call `GetClusterHealth`, print cluster name + status as plain text
  - exit with error if URI missing
- [x] usage: `epm <uri> [--interval 10s] [--insecure]`
- [x] verify `go run ./cmd/epm http://localhost:9200` prints cluster info

### Task 8: Verify acceptance criteria

- [ ] run `go test ./...` — all tests pass
- [ ] run `go vet ./...` — no issues
- [ ] run `go build -o bin/epm ./cmd/epm` — binary produced
- [ ] binary runs against real or mock ES cluster and prints health

## Technical Details

- HTTP client uses `keep-alive` connection pooling (default Transport behavior)
- `doGet` uses `req.SetBasicAuth(username, password)` — not manual header building
- `filter_path` params are embedded in the endpoint constants (not added dynamically)
- All `int64` for counters (ops, bytes) to avoid overflow on large clusters
- Pointer fields for optional nested structs (OS, JVM, FS) so missing fields decode as `nil`

## Post-Completion

- Test against real ES 6.x, 7.x, 8.x, 9.x clusters to verify JSON field compatibility
- The `_cat` APIs return strings for numeric fields (`pri`, `rep`, `docs.count`) — Phase 2 will handle parsing
