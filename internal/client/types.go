package client

// ClusterHealth represents the response from /_cluster/health.
type ClusterHealth struct {
	ClusterName      string `json:"cluster_name"`
	Status           string `json:"status"`
	NumberOfNodes    int    `json:"number_of_nodes"`
	ActiveShards     int    `json:"active_shards"`
	UnassignedShards int    `json:"unassigned_shards"`
}

// NodeInfo represents a single node entry from /_cat/nodes.
type NodeInfo struct {
	NodeRole string `json:"node.role"`
	Name     string `json:"name"`
	IP       string `json:"ip"`
}

// NodeStatsResponse represents the response from /_nodes/stats.
type NodeStatsResponse struct {
	Nodes map[string]NodePerformanceStats `json:"nodes"`
}

// NodePerformanceStats holds per-node performance data.
type NodePerformanceStats struct {
	Name    string            `json:"name"`
	Host    string            `json:"host"`
	IP      string            `json:"ip"`
	Roles   []string          `json:"roles"`
	Indices *NodeIndicesStats `json:"indices,omitempty"`
	OS      *NodeOSStats      `json:"os,omitempty"`
	JVM     *NodeJVMStats     `json:"jvm,omitempty"`
	FS      *NodeFSStats      `json:"fs,omitempty"`
}

// NodeIndicesStats holds indexing and search counters for a node.
type NodeIndicesStats struct {
	Indexing NodeIndexingStats `json:"indexing"`
	Search   NodeSearchStats   `json:"search"`
}

// NodeIndexingStats holds indexing operation counters.
type NodeIndexingStats struct {
	IndexTotal        int64 `json:"index_total"`
	IndexTimeInMillis int64 `json:"index_time_in_millis"`
}

// NodeSearchStats holds search query counters.
type NodeSearchStats struct {
	QueryTotal        int64 `json:"query_total"`
	QueryTimeInMillis int64 `json:"query_time_in_millis"`
}

// NodeOSStats holds OS-level metrics.
type NodeOSStats struct {
	CPU struct {
		Percent int `json:"percent"`
	} `json:"cpu"`
}

// NodeJVMStats holds JVM heap metrics.
type NodeJVMStats struct {
	Mem struct {
		HeapUsedInBytes int64 `json:"heap_used_in_bytes"`
		HeapMaxInBytes  int64 `json:"heap_max_in_bytes"`
	} `json:"mem"`
}

// NodeFSStats holds filesystem metrics.
type NodeFSStats struct {
	Total struct {
		TotalInBytes     int64 `json:"total_in_bytes"`
		AvailableInBytes int64 `json:"available_in_bytes"`
	} `json:"total"`
}

// IndexInfo represents a single index entry from /_cat/indices.
type IndexInfo struct {
	Index        string `json:"index"`
	Pri          string `json:"pri"`
	Rep          string `json:"rep"`
	PriStoreSize string `json:"pri.store.size"`
	StoreSize    string `json:"store.size"`
	DocsCount    string `json:"docs.count"`
}

// IndexStatsResponse represents the response from /_stats.
type IndexStatsResponse struct {
	Indices map[string]IndexStatEntry `json:"indices"`
}

// IndexStatEntry holds per-index statistics split by primaries and total.
type IndexStatEntry struct {
	Primaries *IndexStatShard `json:"primaries,omitempty"`
	Total     *IndexStatShard `json:"total,omitempty"`
}

// IndexStatShard holds shard-level statistics.
type IndexStatShard struct {
	Indexing *IndexingStats `json:"indexing,omitempty"`
	Search   *SearchStats   `json:"search,omitempty"`
	Store    *StoreStats    `json:"store,omitempty"`
}

// IndexingStats holds indexing operation counters for a shard.
type IndexingStats struct {
	IndexTotal        int64 `json:"index_total"`
	IndexTimeInMillis int64 `json:"index_time_in_millis"`
}

// SearchStats holds search query counters for a shard.
type SearchStats struct {
	QueryTotal        int64 `json:"query_total"`
	QueryTimeInMillis int64 `json:"query_time_in_millis"`
}

// StoreStats holds storage size for a shard.
type StoreStats struct {
	SizeInBytes int64 `json:"size_in_bytes"`
}

// AllocationInfo represents a single node entry from /_cat/allocation.
type AllocationInfo struct {
	Node        string `json:"node"`
	Shards      string `json:"shards"`
	DiskPercent string `json:"disk.percent"`
}

// IndexAllocationFilter holds the _name and _ip filter values for a routing allocation filter.
// All values are strings as returned by the ES settings API.
type IndexAllocationFilter struct {
	Name string `json:"_name,omitempty"`
	IP   string `json:"_ip,omitempty"`
}

// IndexAllocationSettings holds routing allocation sub-settings.
type IndexAllocationSettings struct {
	Include           IndexAllocationFilter `json:"include,omitempty"`
	Exclude           IndexAllocationFilter `json:"exclude,omitempty"`
	Require           IndexAllocationFilter `json:"require,omitempty"`
	TotalShardsPerNode string               `json:"total_shards_per_node,omitempty"`
}

// IndexRoutingSettings holds routing settings nested under index.routing.
type IndexRoutingSettings struct {
	Allocation IndexAllocationSettings `json:"allocation,omitempty"`
}

// IndexMappingTotalFields holds the total_fields limit setting.
type IndexMappingTotalFields struct {
	Limit string `json:"limit,omitempty"`
}

// IndexMappingSettings holds mapping settings nested under index.mapping.
type IndexMappingSettings struct {
	TotalFields IndexMappingTotalFields `json:"total_fields,omitempty"`
}

// IndexBlocksSettings holds block settings nested under index.blocks.
type IndexBlocksSettings struct {
	ReadOnlyAllowDelete string `json:"read_only_allow_delete,omitempty"`
}

// IndexSettingsValues holds the dynamic settings we expose for editing.
// All numeric values are strings as returned by the ES settings API.
type IndexSettingsValues struct {
	NumberOfReplicas string               `json:"number_of_replicas,omitempty"`
	RefreshInterval  string               `json:"refresh_interval,omitempty"`
	Routing          IndexRoutingSettings `json:"routing,omitempty"`
	Mapping          IndexMappingSettings `json:"mapping,omitempty"`
	Blocks           IndexBlocksSettings  `json:"blocks,omitempty"`
}
