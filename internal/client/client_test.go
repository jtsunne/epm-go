package client

import (
	"context"
	"encoding/json"
	"io"
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
		_, _ = w.Write([]byte(`{"cluster_name":"test-cluster","status":"green","number_of_nodes":3,"active_shards":42,"unassigned_shards":5}`))
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

func TestGetAllocation(t *testing.T) {
	fixture := `[
		{"node":"node-1","shards":"5","disk.percent":"42"},
		{"node":"node-2","shards":"3","disk.percent":"67"}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/_cat/allocation") {
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
	alloc, err := c.GetAllocation(context.Background())
	if err != nil {
		t.Fatalf("GetAllocation: %v", err)
	}
	if len(alloc) != 2 {
		t.Fatalf("len(alloc) = %d, want 2", len(alloc))
	}
	if alloc[0].Node != "node-1" {
		t.Errorf("alloc[0].Node = %q, want %q", alloc[0].Node, "node-1")
	}
	if alloc[0].Shards != "5" {
		t.Errorf("alloc[0].Shards = %q, want %q", alloc[0].Shards, "5")
	}
	if alloc[0].DiskPercent != "42" {
		t.Errorf("alloc[0].DiskPercent = %q, want %q", alloc[0].DiskPercent, "42")
	}
	if alloc[1].Node != "node-2" {
		t.Errorf("alloc[1].Node = %q, want %q", alloc[1].Node, "node-2")
	}
	if alloc[1].Shards != "3" {
		t.Errorf("alloc[1].Shards = %q, want %q", alloc[1].Shards, "3")
	}
	if alloc[1].DiskPercent != "67" {
		t.Errorf("alloc[1].DiskPercent = %q, want %q", alloc[1].DiskPercent, "67")
	}
}

func TestDeleteIndex_Success(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"acknowledged":true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.DeleteIndex(context.Background(), []string{"my-index"})
	if err != nil {
		t.Fatalf("DeleteIndex: %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/my-index" {
		t.Errorf("path = %q, want /my-index", gotPath)
	}
}

func TestDeleteIndex_BatchPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"acknowledged":true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.DeleteIndex(context.Background(), []string{"index-a", "index-b", "index-c"})
	if err != nil {
		t.Fatalf("DeleteIndex: %v", err)
	}
	if gotPath != "/index-a,index-b,index-c" {
		t.Errorf("path = %q, want /index-a,index-b,index-c", gotPath)
	}
}

func TestDeleteIndex_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":"index not found"}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.DeleteIndex(context.Background(), []string{"missing-index"})
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error %q does not contain 404", err.Error())
	}
}

func TestDeleteIndex_EmptyNames(t *testing.T) {
	// A server that records whether it received any request.
	received := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.DeleteIndex(context.Background(), []string{})
	if err == nil {
		t.Fatal("expected error on empty names slice, got nil")
	}
	if received {
		t.Error("DeleteIndex with empty names must not send any HTTP request")
	}
}

func TestGetIndexSettings(t *testing.T) {
	fixture := `{
		"my-index": {
			"settings": {
				"index": {
					"number_of_replicas": "1",
					"refresh_interval": "5s",
					"routing": {
						"allocation": {
							"include": {"_name": "node-1,node-2", "_ip": "10.0.0.1"},
							"exclude": {"_name": "", "_ip": ""},
							"require": {"_name": "", "_ip": ""},
							"total_shards_per_node": "-1"
						}
					},
					"mapping": {
						"total_fields": {"limit": "2000"}
					},
					"blocks": {
						"read_only_allow_delete": "false"
					}
				}
			}
		}
	}`

	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fixture))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	vals, err := c.GetIndexSettings(context.Background(), "my-index")
	if err != nil {
		t.Fatalf("GetIndexSettings: %v", err)
	}

	if gotPath != "/my-index/_settings" {
		t.Errorf("path = %q, want /my-index/_settings", gotPath)
	}
	if !strings.Contains(gotQuery, "filter_path") {
		t.Errorf("filter_path missing from query: %q", gotQuery)
	}
	if vals.NumberOfReplicas != "1" {
		t.Errorf("NumberOfReplicas = %q, want %q", vals.NumberOfReplicas, "1")
	}
	if vals.RefreshInterval != "5s" {
		t.Errorf("RefreshInterval = %q, want %q", vals.RefreshInterval, "5s")
	}
	if vals.Routing.Allocation.Include.Name != "node-1,node-2" {
		t.Errorf("Routing.Allocation.Include.Name = %q, want %q", vals.Routing.Allocation.Include.Name, "node-1,node-2")
	}
	if vals.Routing.Allocation.TotalShardsPerNode != "-1" {
		t.Errorf("TotalShardsPerNode = %q, want %q", vals.Routing.Allocation.TotalShardsPerNode, "-1")
	}
	if vals.Mapping.TotalFields.Limit != "2000" {
		t.Errorf("Mapping.TotalFields.Limit = %q, want %q", vals.Mapping.TotalFields.Limit, "2000")
	}
	if vals.Blocks.ReadOnlyAllowDelete != "false" {
		t.Errorf("Blocks.ReadOnlyAllowDelete = %q, want %q", vals.Blocks.ReadOnlyAllowDelete, "false")
	}
}

func TestGetIndexSettings_EmptyName(t *testing.T) {
	received := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetIndexSettings(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty name, got nil")
	}
	if received {
		t.Error("GetIndexSettings with empty name must not send any HTTP request")
	}
}

func TestGetIndexSettings_EmptyResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	_, err := c.GetIndexSettings(context.Background(), "missing-index")
	if err == nil {
		t.Fatal("expected error for empty response, got nil")
	}
}

func TestUpdateIndexSettings_Success(t *testing.T) {
	var gotMethod, gotPath, gotContentType string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"acknowledged":true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	settings := map[string]any{
		"index.number_of_replicas": "2",
		"index.refresh_interval":   "30s",
	}
	err := c.UpdateIndexSettings(context.Background(), []string{"my-index"}, settings)
	if err != nil {
		t.Fatalf("UpdateIndexSettings: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/my-index/_settings" {
		t.Errorf("path = %q, want /my-index/_settings", gotPath)
	}
	if !strings.Contains(gotContentType, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", gotContentType)
	}

	var parsed map[string]any
	if err := json.Unmarshal(gotBody, &parsed); err != nil {
		t.Fatalf("unmarshal sent body: %v", err)
	}
	index, ok := parsed["index"].(map[string]any)
	if !ok {
		t.Fatalf("body missing index key, got: %v", parsed)
	}
	if index["number_of_replicas"] != "2" {
		t.Errorf("number_of_replicas = %v, want 2", index["number_of_replicas"])
	}
	if index["refresh_interval"] != "30s" {
		t.Errorf("refresh_interval = %v, want 30s", index["refresh_interval"])
	}
}

func TestUpdateIndexSettings_BatchPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"acknowledged":true}`))
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.UpdateIndexSettings(context.Background(), []string{"idx-a", "idx-b"}, map[string]any{
		"index.number_of_replicas": "1",
	})
	if err != nil {
		t.Fatalf("UpdateIndexSettings: %v", err)
	}
	if gotPath != "/idx-a,idx-b/_settings" {
		t.Errorf("path = %q, want /idx-a,idx-b/_settings", gotPath)
	}
}

func TestUpdateIndexSettings_NoOp(t *testing.T) {
	received := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.UpdateIndexSettings(context.Background(), []string{"my-index"}, map[string]any{})
	if err != nil {
		t.Fatalf("UpdateIndexSettings with empty map: %v", err)
	}
	if received {
		t.Error("UpdateIndexSettings with empty settings must not send any HTTP request")
	}
}

func TestUpdateIndexSettings_EmptyNames(t *testing.T) {
	received := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := newTestClient(t, srv.URL)
	err := c.UpdateIndexSettings(context.Background(), []string{}, map[string]any{"index.number_of_replicas": "1"})
	if err == nil {
		t.Fatal("expected error on empty names, got nil")
	}
	if received {
		t.Error("UpdateIndexSettings with empty names must not send any HTTP request")
	}
}

func TestBuildNestedMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]any
		wantPath []string // dotted path to check
		wantVal  any
	}{
		{
			name:     "single level",
			input:    map[string]any{"key": "val"},
			wantPath: []string{"key"},
			wantVal:  "val",
		},
		{
			name:     "two levels",
			input:    map[string]any{"index.number_of_replicas": "2"},
			wantPath: []string{"index", "number_of_replicas"},
			wantVal:  "2",
		},
		{
			name:     "deep nesting",
			input:    map[string]any{"index.routing.allocation.include._name": "node-1"},
			wantPath: []string{"index", "routing", "allocation", "include", "_name"},
			wantVal:  "node-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildNestedMap(tt.input)
			// walk the path
			var cur any = result
			for i, seg := range tt.wantPath {
				m, ok := cur.(map[string]any)
				if !ok {
					t.Fatalf("step %d: expected map, got %T", i, cur)
				}
				cur = m[seg]
			}
			if cur != tt.wantVal {
				t.Errorf("value at path = %v, want %v", cur, tt.wantVal)
			}
		})
	}
}

// TestBuildNestedMap_MultipleKeys verifies that two dotted keys sharing the
// same parent prefix are both preserved under that parent — not overwritten.
func TestBuildNestedMap_MultipleKeys(t *testing.T) {
	result := buildNestedMap(map[string]any{
		"index.number_of_replicas": "2",
		"index.refresh_interval":   "30s",
	})

	indexMap, ok := result["index"].(map[string]any)
	if !ok {
		t.Fatalf("expected result[\"index\"] to be map[string]any, got %T", result["index"])
	}
	if got := indexMap["number_of_replicas"]; got != "2" {
		t.Errorf("number_of_replicas = %v, want \"2\"", got)
	}
	if got := indexMap["refresh_interval"]; got != "30s" {
		t.Errorf("refresh_interval = %v, want \"30s\"", got)
	}
}

// TestBuildNestedMap_DeepSharedPrefix verifies that keys sharing a deep
// common prefix (e.g. routing.allocation.*) are all preserved. Previously
// the loop overwrote sibling branches at any shared intermediate node.
func TestBuildNestedMap_DeepSharedPrefix(t *testing.T) {
	result := buildNestedMap(map[string]any{
		"index.routing.allocation.include._name": "node-a",
		"index.routing.allocation.exclude._name": "node-b",
		"index.routing.allocation.require._name": "node-c",
		"index.routing.allocation.include._ip":   "1.2.3.4",
	})

	walk := func(m map[string]any, path ...string) (any, bool) {
		var cur any = m
		for _, seg := range path {
			mm, ok := cur.(map[string]any)
			if !ok {
				return nil, false
			}
			cur = mm[seg]
		}
		return cur, true
	}

	checks := []struct {
		path []string
		want string
	}{
		{[]string{"index", "routing", "allocation", "include", "_name"}, "node-a"},
		{[]string{"index", "routing", "allocation", "exclude", "_name"}, "node-b"},
		{[]string{"index", "routing", "allocation", "require", "_name"}, "node-c"},
		{[]string{"index", "routing", "allocation", "include", "_ip"}, "1.2.3.4"},
	}
	for _, c := range checks {
		got, ok := walk(result, c.path...)
		if !ok {
			t.Errorf("path %v: intermediate node missing", c.path)
			continue
		}
		if got != c.want {
			t.Errorf("path %v: got %v, want %v", c.path, got, c.want)
		}
	}
}
