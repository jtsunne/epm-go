package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jtsunne/epm-go/internal/client"
)

func TestFetchAll_AllSuccess(t *testing.T) {
	health := &client.ClusterHealth{ClusterName: "my-cluster", Status: "green", NumberOfNodes: 3}
	nodes := []client.NodeInfo{{Name: "node1", IP: "10.0.0.1", NodeRole: "master"}}
	nodeStats := &client.NodeStatsResponse{Nodes: map[string]client.NodePerformanceStats{
		"abc": {Name: "node1"},
	}}
	indices := []client.IndexInfo{{Index: "logs-000001", Pri: "1", Rep: "1"}}
	indexStats := &client.IndexStatsResponse{Indices: map[string]client.IndexStatEntry{
		"logs-000001": {},
	}}

	alloc := []client.AllocationInfo{{Node: "node1", Shards: "5", DiskPercent: "42"}}

	mc := &MockESClient{
		HealthFn:     func(_ context.Context) (*client.ClusterHealth, error) { return health, nil },
		NodesFn:      func(_ context.Context) ([]client.NodeInfo, error) { return nodes, nil },
		NodeStatsFn:  func(_ context.Context) (*client.NodeStatsResponse, error) { return nodeStats, nil },
		IndicesFn:    func(_ context.Context) ([]client.IndexInfo, error) { return indices, nil },
		IndexStatsFn: func(_ context.Context) (*client.IndexStatsResponse, error) { return indexStats, nil },
		AllocationFn: func(_ context.Context) ([]client.AllocationInfo, error) { return alloc, nil },
	}

	snap, err := FetchAll(context.Background(), mc)
	require.NoError(t, err)
	require.NotNil(t, snap)

	assert.Equal(t, "my-cluster", snap.Health.ClusterName)
	assert.Equal(t, "green", snap.Health.Status)
	assert.Equal(t, 3, snap.Health.NumberOfNodes)
	assert.Equal(t, nodes, snap.Nodes)
	assert.Equal(t, *nodeStats, snap.NodeStats)
	assert.Equal(t, indices, snap.Indices)
	assert.Equal(t, *indexStats, snap.IndexStats)
	assert.Equal(t, alloc, snap.Allocation)
	assert.False(t, snap.FetchedAt.IsZero())
}

func TestFetchAll_AllocationFailureIsNonFatal(t *testing.T) {
	mc := &MockESClient{
		AllocationFn: func(_ context.Context) ([]client.AllocationInfo, error) {
			return nil, errMockFailure
		},
	}

	snap, err := FetchAll(context.Background(), mc)
	require.NoError(t, err)
	require.NotNil(t, snap)
	assert.Nil(t, snap.Allocation)
}

func TestFetchAll_PartialFailure(t *testing.T) {
	mc := &MockESClient{
		// Health succeeds
		HealthFn: func(_ context.Context) (*client.ClusterHealth, error) {
			return &client.ClusterHealth{Status: "green"}, nil
		},
		// NodeStats fails
		NodeStatsFn: func(_ context.Context) (*client.NodeStatsResponse, error) {
			return nil, errMockFailure
		},
	}

	snap, err := FetchAll(context.Background(), mc)
	assert.Error(t, err)
	assert.Nil(t, snap)
}

func TestFetchAll_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before calling FetchAll

	mc := &MockESClient{} // all defaults succeed, but context is already done

	// With a pre-cancelled context, at least one goroutine should detect it.
	// errgroup propagates the first error; stdlib HTTP honours context.
	// Since MockESClient doesn't honour context, we test that a pre-cancelled
	// ctx passed through errgroup still causes a context error when the mock
	// explicitly checks it.
	mc.HealthFn = func(ctx context.Context) (*client.ClusterHealth, error) {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return &client.ClusterHealth{Status: "green"}, nil
	}

	snap, err := FetchAll(ctx, mc)
	assert.Error(t, err)
	assert.Nil(t, snap)
}
