package model

import (
	"time"

	"github.com/dm/epm-go/internal/client"
)

// Snapshot holds the raw results of a single poll cycle across all 5 ES endpoints.
type Snapshot struct {
	Health     client.ClusterHealth
	Nodes      []client.NodeInfo
	NodeStats  client.NodeStatsResponse
	Indices    []client.IndexInfo
	IndexStats client.IndexStatsResponse
	FetchedAt  time.Time
}
