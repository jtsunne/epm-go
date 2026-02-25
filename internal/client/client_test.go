package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// newTestClient creates a DefaultClient pointed at the given test server URL.
func newTestClient(t *testing.T, baseURL string) *DefaultClient {
	t.Helper()
	c, err := NewDefaultClient(ClientConfig{
		BaseURL:        baseURL,
		RequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewDefaultClient: %v", err)
	}
	return c
}

func TestGetClusterHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/_cluster/health") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "filter_path") {
			t.Errorf("filter_path missing from query: %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"cluster_name":"test-cluster","status":"green","number_of_nodes":3,"active_shards":42,"unassigned_shards":5,"number_of_pending_tasks":2}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	health, err := c.GetClusterHealth(context.Background())
	if err != nil {
		t.Fatalf("GetClusterHealth: %v", err)
	}
	if health.ClusterName != "test-cluster" {
		t.Errorf("ClusterName = %q, want %q", health.ClusterName, "test-cluster")
	}
	if health.Status != "green" {
		t.Errorf("Status = %q, want %q", health.Status, "green")
	}
	if health.NumberOfNodes != 3 {
		t.Errorf("NumberOfNodes = %d, want 3", health.NumberOfNodes)
	}
	if health.ActiveShards != 42 {
		t.Errorf("ActiveShards = %d, want 42", health.ActiveShards)
	}
	if health.UnassignedShards != 5 {
		t.Errorf("UnassignedShards = %d, want 5", health.UnassignedShards)
	}
	if health.NumberOfPendingTasks != 2 {
		t.Errorf("NumberOfPendingTasks = %d, want 2", health.NumberOfPendingTasks)
	}
}

func TestGetNodes(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/_cat/nodes") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "format=json") {
			t.Errorf("format=json missing from query: %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"node.role":"m","name":"node-1","ip":"10.0.0.1"},{"node.role":"d","name":"node-2","ip":"10.0.0.2"}]`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	nodes, err := c.GetNodes(context.Background())
	if err != nil {
		t.Fatalf("GetNodes: %v", err)
	}
	if len(nodes) != 2 {
		t.Fatalf("len(nodes) = %d, want 2", len(nodes))
	}
	if nodes[0].NodeRole != "m" {
		t.Errorf("nodes[0].NodeRole = %q, want %q", nodes[0].NodeRole, "m")
	}
	if nodes[0].Name != "node-1" {
		t.Errorf("nodes[0].Name = %q, want %q", nodes[0].Name, "node-1")
	}
	if nodes[0].IP != "10.0.0.1" {
		t.Errorf("nodes[0].IP = %q, want %q", nodes[0].IP, "10.0.0.1")
	}
}

func TestGetNodeStats(t *testing.T) {
	fixture := `{
		"nodes": {
			"abc123": {
				"name": "node-1",
				"host": "host1",
				"ip": "10.0.0.1",
				"roles": ["master","data"],
				"indices": {
					"indexing": {"index_total": 1000, "index_time_in_millis": 500},
					"search":   {"query_total": 2000, "query_time_in_millis": 800}
				},
				"os":  {"cpu": {"percent": 45}},
				"jvm": {"mem": {"heap_used_in_bytes": 536870912, "heap_max_in_bytes": 1073741824}},
				"fs":  {"total": {"total_in_bytes": 10737418240, "available_in_bytes": 5368709120}}
			}
		}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/_nodes/stats") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "filter_path") {
			t.Errorf("filter_path missing from query: %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	stats, err := c.GetNodeStats(context.Background())
	if err != nil {
		t.Fatalf("GetNodeStats: %v", err)
	}
	node, ok := stats.Nodes["abc123"]
	if !ok {
		t.Fatal("node abc123 not found")
	}
	if node.Name != "node-1" {
		t.Errorf("Name = %q, want %q", node.Name, "node-1")
	}
	if node.Host != "host1" {
		t.Errorf("Host = %q, want %q", node.Host, "host1")
	}
	if len(node.Roles) != 2 || node.Roles[0] != "master" || node.Roles[1] != "data" {
		t.Errorf("Roles = %v, want [master data]", node.Roles)
	}
	if node.Indices == nil {
		t.Fatal("Indices is nil")
	}
	if node.Indices.Indexing.IndexTotal != 1000 {
		t.Errorf("IndexTotal = %d, want 1000", node.Indices.Indexing.IndexTotal)
	}
	if node.Indices.Search.QueryTotal != 2000 {
		t.Errorf("QueryTotal = %d, want 2000", node.Indices.Search.QueryTotal)
	}
	if node.OS == nil || node.OS.CPU.Percent != 45 {
		t.Errorf("OS.CPU.Percent unexpected")
	}
	if node.JVM == nil || node.JVM.Mem.HeapUsedInBytes != 536870912 {
		t.Errorf("JVM.Mem.HeapUsedInBytes unexpected")
	}
	if node.FS == nil || node.FS.Total.TotalInBytes != 10737418240 {
		t.Errorf("FS.Total.TotalInBytes unexpected")
	}
}

func TestGetIndices(t *testing.T) {
	fixture := `[
		{"index":"my-index","pri":"1","rep":"1","pri.store.size":"1gb","store.size":"2gb","docs.count":"5000"}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/_cat/indices") {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "format=json") {
			t.Errorf("format=json missing from query: %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	indices, err := c.GetIndices(context.Background())
	if err != nil {
		t.Fatalf("GetIndices: %v", err)
	}
	if len(indices) != 1 {
		t.Fatalf("len(indices) = %d, want 1", len(indices))
	}
	idx := indices[0]
	if idx.Index != "my-index" {
		t.Errorf("Index = %q, want %q", idx.Index, "my-index")
	}
	if idx.Pri != "1" {
		t.Errorf("Pri = %q, want %q", idx.Pri, "1")
	}
	if idx.Rep != "1" {
		t.Errorf("Rep = %q, want %q", idx.Rep, "1")
	}
	if idx.PriStoreSize != "1gb" {
		t.Errorf("PriStoreSize = %q, want %q", idx.PriStoreSize, "1gb")
	}
	if idx.StoreSize != "2gb" {
		t.Errorf("StoreSize = %q, want %q", idx.StoreSize, "2gb")
	}
	if idx.DocsCount != "5000" {
		t.Errorf("DocsCount = %q, want %q", idx.DocsCount, "5000")
	}
}

func TestGetIndexStats(t *testing.T) {
	fixture := `{
		"indices": {
			"my-index": {
				"primaries": {
					"indexing": {"index_total": 100, "index_time_in_millis": 50},
					"store":    {"size_in_bytes": 1048576}
				},
				"total": {
					"search": {"query_total": 200, "query_time_in_millis": 80},
					"store":  {"size_in_bytes": 2097152}
				}
			}
		}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/_stats" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "filter_path") {
			t.Errorf("filter_path missing from query: %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	stats, err := c.GetIndexStats(context.Background())
	if err != nil {
		t.Fatalf("GetIndexStats: %v", err)
	}
	entry, ok := stats.Indices["my-index"]
	if !ok {
		t.Fatal("my-index not found")
	}
	if entry.Primaries == nil || entry.Primaries.Indexing == nil {
		t.Fatal("Primaries.Indexing is nil")
	}
	if entry.Primaries.Indexing.IndexTotal != 100 {
		t.Errorf("Primaries.Indexing.IndexTotal = %d, want 100", entry.Primaries.Indexing.IndexTotal)
	}
	if entry.Total == nil || entry.Total.Search == nil {
		t.Fatal("Total.Search is nil")
	}
	if entry.Total.Search.QueryTotal != 200 {
		t.Errorf("Total.Search.QueryTotal = %d, want 200", entry.Total.Search.QueryTotal)
	}
}

func TestPing_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/_cluster/health") {
			t.Errorf("Ping: unexpected path %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"green"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	if err := c.Ping(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestPing_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	if err := c.Ping(context.Background()); err == nil {
		t.Error("expected error from Ping on non-2xx, got nil")
	}
}

func TestBasicAuth(t *testing.T) {
	var gotUser, gotPass string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, _ = r.BasicAuth()
		_, _ = w.Write([]byte(`{"status":"green"}`))
	}))
	defer srv.Close()

	c, err := NewDefaultClient(ClientConfig{
		BaseURL:  srv.URL,
		Username: "elastic",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("NewDefaultClient: %v", err)
	}

	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if gotUser != "elastic" {
		t.Errorf("user = %q, want %q", gotUser, "elastic")
	}
	if gotPass != "secret" {
		t.Errorf("pass = %q, want %q", gotPass, "secret")
	}
}

func TestBasicAuthPasswordOnly(t *testing.T) {
	var gotUser, gotPass string
	var gotOK bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, gotOK = r.BasicAuth()
		_, _ = w.Write([]byte(`{"status":"green"}`))
	}))
	defer srv.Close()

	c, err := NewDefaultClient(ClientConfig{
		BaseURL:  srv.URL,
		Username: "",
		Password: "secret",
	})
	if err != nil {
		t.Fatalf("NewDefaultClient: %v", err)
	}

	if err := c.Ping(context.Background()); err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if !gotOK {
		t.Error("expected Authorization header to be sent, but it was absent")
	}
	if gotUser != "" {
		t.Errorf("user = %q, want %q", gotUser, "")
	}
	if gotPass != "secret" {
		t.Errorf("pass = %q, want %q", gotPass, "secret")
	}
}

func TestHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"missing auth"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetClusterHealth(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error %q does not contain %q", err.Error(), "401")
	}
}

func TestContextCancellation(t *testing.T) {
	started := make(chan struct{})
	var once sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() { close(started) })
		// Block until the client disconnects
		<-r.Context().Done()
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		_, err := c.GetClusterHealth(ctx)
		done <- err
	}()

	<-started
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Error("expected error after context cancellation, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for cancelled request to return")
	}
}

func TestTLSSkipVerify(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"green"}`))
	}))
	defer srv.Close()

	// Without InsecureSkipVerify, TLS handshake should fail (self-signed cert).
	c, err := NewDefaultClient(ClientConfig{
		BaseURL:        srv.URL,
		RequestTimeout: 5 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewDefaultClient: %v", err)
	}
	if err := c.Ping(context.Background()); err == nil {
		t.Error("expected TLS certificate error without InsecureSkipVerify, got nil")
	}

	// With InsecureSkipVerify=true, the request should succeed.
	c2, err := NewDefaultClient(ClientConfig{
		BaseURL:            srv.URL,
		RequestTimeout:     5 * time.Second,
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("NewDefaultClient: %v", err)
	}
	if err := c2.Ping(context.Background()); err != nil {
		t.Errorf("Ping with InsecureSkipVerify=true: %v", err)
	}
}

func TestInvalidJSONResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"broken":`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	if _, err := c.GetClusterHealth(context.Background()); err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
