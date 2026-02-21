package engine

import (
	"context"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/dm/epm-go/internal/client"
	"github.com/dm/epm-go/internal/model"
)

// FetchAll calls all 5 Elasticsearch endpoints concurrently.
// If any endpoint fails, FetchAll returns the first error.
func FetchAll(ctx context.Context, c client.ESClient) (*model.Snapshot, error) {
	var (
		health     *client.ClusterHealth
		nodes      []client.NodeInfo
		nodeStats  *client.NodeStatsResponse
		indices    []client.IndexInfo
		indexStats *client.IndexStatsResponse
	)

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		health, err = c.GetClusterHealth(gctx)
		return err
	})

	g.Go(func() error {
		var err error
		nodes, err = c.GetNodes(gctx)
		return err
	})

	g.Go(func() error {
		var err error
		nodeStats, err = c.GetNodeStats(gctx)
		return err
	})

	g.Go(func() error {
		var err error
		indices, err = c.GetIndices(gctx)
		return err
	})

	g.Go(func() error {
		var err error
		indexStats, err = c.GetIndexStats(gctx)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	snap := &model.Snapshot{
		Health:     *health,
		Nodes:      nodes,
		NodeStats:  *nodeStats,
		Indices:    indices,
		IndexStats: *indexStats,
		FetchedAt:  time.Now(),
	}
	return snap, nil
}
