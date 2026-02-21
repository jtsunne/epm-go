package engine

import (
	"context"
	"errors"

	"github.com/dm/epm-go/internal/client"
)

// MockESClient implements client.ESClient for testing.
type MockESClient struct {
	HealthFn     func(ctx context.Context) (*client.ClusterHealth, error)
	NodesFn      func(ctx context.Context) ([]client.NodeInfo, error)
	NodeStatsFn  func(ctx context.Context) (*client.NodeStatsResponse, error)
	IndicesFn    func(ctx context.Context) ([]client.IndexInfo, error)
	IndexStatsFn func(ctx context.Context) (*client.IndexStatsResponse, error)
}

func (m *MockESClient) GetClusterHealth(ctx context.Context) (*client.ClusterHealth, error) {
	if m.HealthFn != nil {
		return m.HealthFn(ctx)
	}
	return &client.ClusterHealth{ClusterName: "test", Status: "green"}, nil
}

func (m *MockESClient) GetNodes(ctx context.Context) ([]client.NodeInfo, error) {
	if m.NodesFn != nil {
		return m.NodesFn(ctx)
	}
	return []client.NodeInfo{{Name: "node1", IP: "127.0.0.1", NodeRole: "master"}}, nil
}

func (m *MockESClient) GetNodeStats(ctx context.Context) (*client.NodeStatsResponse, error) {
	if m.NodeStatsFn != nil {
		return m.NodeStatsFn(ctx)
	}
	return &client.NodeStatsResponse{Nodes: map[string]client.NodePerformanceStats{}}, nil
}

func (m *MockESClient) GetIndices(ctx context.Context) ([]client.IndexInfo, error) {
	if m.IndicesFn != nil {
		return m.IndicesFn(ctx)
	}
	return []client.IndexInfo{{Index: "test-index"}}, nil
}

func (m *MockESClient) GetIndexStats(ctx context.Context) (*client.IndexStatsResponse, error) {
	if m.IndexStatsFn != nil {
		return m.IndexStatsFn(ctx)
	}
	return &client.IndexStatsResponse{Indices: map[string]client.IndexStatEntry{}}, nil
}

func (m *MockESClient) Ping(ctx context.Context) error {
	return nil
}

func (m *MockESClient) BaseURL() string {
	return "http://mock:9200"
}

// errOnce returns a function that returns err exactly once, then succeeds.
// Useful for simulating transient errors.
func errOnce(err error) func(ctx context.Context) error {
	called := false
	return func(_ context.Context) error {
		if !called {
			called = true
			return err
		}
		return nil
	}
}

var errMockFailure = errors.New("mock failure")
