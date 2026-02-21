package model

// PerformanceMetrics holds cluster-level throughput and latency metrics derived
// from the delta between two consecutive snapshots.
type PerformanceMetrics struct {
	IndexingRate  float64 // ops/sec (primaries)
	SearchRate    float64 // ops/sec (total shards)
	IndexLatency  float64 // ms/op (primaries)
	SearchLatency float64 // ms/op (total shards)
}

// ClusterResources holds cluster-wide resource utilisation averages/totals.
type ClusterResources struct {
	AvgCPUPercent     float64
	AvgJVMHeapPercent float64
	StorageUsedBytes  int64
	StorageTotalBytes int64
	StoragePercent    float64
}

// NodeRow holds display-ready data for a single row in the node table.
type NodeRow struct {
	ID           string
	Name         string
	Role         string
	IP           string
	IndexingRate float64 // ops/sec
	SearchRate   float64 // ops/sec
	IndexLatency float64 // ms/op
	SearchLatency float64 // ms/op
}

// IndexRow holds display-ready data for a single row in the index table.
type IndexRow struct {
	Name          string
	PrimaryShards int
	TotalShards   int
	TotalSizeBytes int64
	AvgShardSize  int64
	DocCount      int64
	IndexingRate  float64 // ops/sec (primaries)
	SearchRate    float64 // ops/sec (total)
	IndexLatency  float64 // ms/op (primaries)
	SearchLatency float64 // ms/op (total)
}
