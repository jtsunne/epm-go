package model

import (
	"time"

	"github.com/jtsunne/epm-go/internal/client"
)

// Snapshot holds the raw results of a single poll cycle across all 6 ES endpoints.
type Snapshot struct {
	Health     client.ClusterHealth
	Nodes      []client.NodeInfo
	NodeStats  client.NodeStatsResponse
	Indices    []client.IndexInfo
	IndexStats client.IndexStatsResponse
	Allocation []client.AllocationInfo
	FetchedAt  time.Time
}
