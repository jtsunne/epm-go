package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
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

// doDelete performs a DELETE request to the given path (relative to BaseURL).
// It sets Basic Auth if credentials are configured.
// Returns an error on non-2xx status.
func (c *DefaultClient) doDelete(ctx context.Context, path string) error {
	url := strings.TrimRight(c.config.BaseURL, "/") + path

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if c.config.Username != "" || c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseBytes = 32 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("response body exceeds %d MB limit", maxResponseBytes/(1024*1024))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, truncate(body, 200))
	}

	return nil
}

// DeleteIndex deletes one or more indices by name.
// Names are joined with commas into a single DELETE /<names> request.
func (c *DefaultClient) DeleteIndex(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("DeleteIndex: names must not be empty")
	}
	escaped := make([]string, len(names))
	for i, n := range names {
		escaped[i] = url.PathEscape(n)
	}
	path := "/" + strings.Join(escaped, ",")
	if err := c.doDelete(ctx, path); err != nil {
		return fmt.Errorf("DeleteIndex: %w", err)
	}
	return nil
}
