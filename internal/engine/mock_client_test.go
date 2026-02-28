package engine

import (
	"context"
	"errors"

	"github.com/jtsunne/epm-go/internal/client"
)

// MockESClient implements client.ESClient for testing.
type MockESClient struct {
	HealthFn              func(ctx context.Context) (*client.ClusterHealth, error)
	NodesFn               func(ctx context.Context) ([]client.NodeInfo, error)
	NodeStatsFn           func(ctx context.Context) (*client.NodeStatsResponse, error)
	IndicesFn             func(ctx context.Context) ([]client.IndexInfo, error)
	IndexStatsFn          func(ctx context.Context) (*client.IndexStatsResponse, error)
	AllocationFn          func(ctx context.Context) ([]client.AllocationInfo, error)
	DeleteIndexFn         func(ctx context.Context, names []string) error
	GetIndexSettingsFn    func(ctx context.Context, name string) (*client.IndexSettingsValues, error)
	UpdateIndexSettingsFn func(ctx context.Context, names []string, settings map[string]any) error
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

func (m *MockESClient) GetAllocation(ctx context.Context) ([]client.AllocationInfo, error) {
	if m.AllocationFn != nil {
		return m.AllocationFn(ctx)
	}
	return []client.AllocationInfo{}, nil
}

func (m *MockESClient) DeleteIndex(ctx context.Context, names []string) error {
	if m.DeleteIndexFn != nil {
		return m.DeleteIndexFn(ctx, names)
	}
	return nil
}

func (m *MockESClient) GetIndexSettings(ctx context.Context, name string) (*client.IndexSettingsValues, error) {
	if m.GetIndexSettingsFn != nil {
		return m.GetIndexSettingsFn(ctx, name)
	}
	return &client.IndexSettingsValues{NumberOfReplicas: "1", RefreshInterval: "1s"}, nil
}

func (m *MockESClient) UpdateIndexSettings(ctx context.Context, names []string, settings map[string]any) error {
	if m.UpdateIndexSettingsFn != nil {
		return m.UpdateIndexSettingsFn(ctx, names, settings)
	}
	return nil
}

func (m *MockESClient) Ping(ctx context.Context) error {
	return nil
}

func (m *MockESClient) BaseURL() string {
	return "http://mock:9200"
}

var errMockFailure = errors.New("mock failure")
