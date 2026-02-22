//go:build integration

package engine_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dm/epm-go/internal/client"
	"github.com/dm/epm-go/internal/engine"
)

// esClient creates a DefaultClient from $ES_URI or skips the test if unset.
func esClient(t *testing.T) client.ESClient {
	t.Helper()
	uri := os.Getenv("ES_URI")
	if uri == "" {
		t.Skip("ES_URI not set; skipping integration test")
	}
	c, err := client.NewDefaultClient(client.ClientConfig{
		BaseURL:        uri,
		RequestTimeout: 10 * time.Second,
	})
	require.NoError(t, err)
	return c
}

// TestLiveCluster_AllEndpoints connects to $ES_URI, calls FetchAll across all
// 5 endpoints, and verifies that the returned snapshot is non-empty.
func TestLiveCluster_AllEndpoints(t *testing.T) {
	c := esClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	snap, err := engine.FetchAll(ctx, c)
	require.NoError(t, err)
	require.NotNil(t, snap)

	assert.NotEmpty(t, snap.Health.ClusterName, "cluster name should not be empty")
	assert.NotEmpty(t, snap.Health.Status, "cluster status should not be empty")
	assert.Greater(t, snap.Health.NumberOfNodes, 0, "should have at least 1 node")
	assert.NotEmpty(t, snap.NodeStats.Nodes, "node stats should be non-empty")
	assert.False(t, snap.FetchedAt.IsZero(), "fetch timestamp should be set")
}

// TestLiveCluster_MetricsNonNegative runs two consecutive polls (2 s apart) and
// verifies that all computed rates and latencies are >= 0.
func TestLiveCluster_MetricsNonNegative(t *testing.T) {
	c := esClient(t)
	ctx := context.Background()

	snap1, err := engine.FetchAll(ctx, c)
	require.NoError(t, err)
	require.NotNil(t, snap1)

	time.Sleep(2 * time.Second)

	snap2, err := engine.FetchAll(ctx, c)
	require.NoError(t, err)
	require.NotNil(t, snap2)

	elapsed := snap2.FetchedAt.Sub(snap1.FetchedAt)
	metrics := engine.CalcClusterMetrics(snap1, snap2, elapsed)

	assert.GreaterOrEqual(t, metrics.IndexingRate, 0.0, "indexing rate must be >= 0")
	assert.GreaterOrEqual(t, metrics.SearchRate, 0.0, "search rate must be >= 0")
	assert.GreaterOrEqual(t, metrics.IndexLatency, 0.0, "index latency must be >= 0")
	assert.GreaterOrEqual(t, metrics.SearchLatency, 0.0, "search latency must be >= 0")
}

// TestLiveCluster_HTTPSWithInsecure skips unless ES_URI is https://.
// Verifies that InsecureSkipVerify=true allows connecting to a self-signed cert ES.
func TestLiveCluster_HTTPSWithInsecure(t *testing.T) {
	uri := os.Getenv("ES_URI")
	if uri == "" {
		t.Skip("ES_URI not set; skipping integration test")
	}
	if !strings.HasPrefix(uri, "https://") {
		t.Skip("ES_URI is not https://; skipping TLS insecure test")
	}

	c, err := client.NewDefaultClient(client.ClientConfig{
		BaseURL:            uri,
		InsecureSkipVerify: true,
		RequestTimeout:     10 * time.Second,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	snap, err := engine.FetchAll(ctx, c)
	require.NoError(t, err)
	require.NotNil(t, snap)
	assert.NotEmpty(t, snap.Health.ClusterName, "cluster name should not be empty with insecure TLS")
}
