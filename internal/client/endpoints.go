package client

import (
	"context"
	"encoding/json"
	"fmt"
)

const (
	endpointClusterHealth = "/_cluster/health?filter_path=cluster_name,status,number_of_nodes,active_shards,unassigned_shards"
	endpointNodes         = "/_cat/nodes?v&format=json&h=node.role,name,ip&s=node.role,ip"
	endpointNodeStats     = "/_nodes/stats/indices,os,jvm,fs?filter_path=nodes.*.name,nodes.*.host,nodes.*.ip,nodes.*.roles,nodes.*.indices.indexing.index_total,nodes.*.indices.indexing.index_time_in_millis,nodes.*.indices.search.query_total,nodes.*.indices.search.query_time_in_millis,nodes.*.os.cpu.percent,nodes.*.jvm.mem.heap_used_in_bytes,nodes.*.jvm.mem.heap_max_in_bytes,nodes.*.fs.total.total_in_bytes,nodes.*.fs.total.available_in_bytes"
	endpointIndices       = "/_cat/indices?v&format=json&h=index,pri,rep,pri.store.size,store.size,docs.count&s=index"
	endpointIndexStats    = "/_stats?filter_path=indices.*.primaries.indexing.index_total,indices.*.primaries.indexing.index_time_in_millis,indices.*.total.indexing.index_total,indices.*.total.indexing.index_time_in_millis,indices.*.total.search.query_total,indices.*.total.search.query_time_in_millis,indices.*.primaries.search.query_total,indices.*.primaries.search.query_time_in_millis,indices.*.primaries.store.size_in_bytes,indices.*.total.store.size_in_bytes"
)

// GetClusterHealth fetches cluster health from /_cluster/health.
func (c *DefaultClient) GetClusterHealth(ctx context.Context) (*ClusterHealth, error) {
	body, err := c.doGet(ctx, endpointClusterHealth)
	if err != nil {
		return nil, fmt.Errorf("GetClusterHealth: %w", err)
	}

	var result ClusterHealth
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("GetClusterHealth decode: %w", err)
	}
	return &result, nil
}

// GetNodes fetches the list of nodes from /_cat/nodes.
func (c *DefaultClient) GetNodes(ctx context.Context) ([]NodeInfo, error) {
	body, err := c.doGet(ctx, endpointNodes)
	if err != nil {
		return nil, fmt.Errorf("GetNodes: %w", err)
	}

	var result []NodeInfo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("GetNodes decode: %w", err)
	}
	return result, nil
}

// GetNodeStats fetches per-node statistics from /_nodes/stats.
func (c *DefaultClient) GetNodeStats(ctx context.Context) (*NodeStatsResponse, error) {
	body, err := c.doGet(ctx, endpointNodeStats)
	if err != nil {
		return nil, fmt.Errorf("GetNodeStats: %w", err)
	}

	var result NodeStatsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("GetNodeStats decode: %w", err)
	}
	return &result, nil
}

// GetIndices fetches the list of indices from /_cat/indices.
func (c *DefaultClient) GetIndices(ctx context.Context) ([]IndexInfo, error) {
	body, err := c.doGet(ctx, endpointIndices)
	if err != nil {
		return nil, fmt.Errorf("GetIndices: %w", err)
	}

	var result []IndexInfo
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("GetIndices decode: %w", err)
	}
	return result, nil
}

// GetIndexStats fetches per-index statistics from /_stats.
func (c *DefaultClient) GetIndexStats(ctx context.Context) (*IndexStatsResponse, error) {
	body, err := c.doGet(ctx, endpointIndexStats)
	if err != nil {
		return nil, fmt.Errorf("GetIndexStats: %w", err)
	}

	var result IndexStatsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("GetIndexStats decode: %w", err)
	}
	return &result, nil
}
