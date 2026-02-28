package engine

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/jtsunne/epm-go/internal/client"
	"github.com/jtsunne/epm-go/internal/model"
)

// FetchAll calls all 5 Elasticsearch endpoints concurrently, plus the optional
// allocation endpoint. If any of the 5 core endpoints fails, FetchAll returns
// the first error. Allocation failures are non-fatal (some ES versions may not
// support /_cat/allocation); on error the field is left nil/empty.
func FetchAll(ctx context.Context, c client.ESClient) (*model.Snapshot, error) {
	var (
		health     *client.ClusterHealth
		nodes      []client.NodeInfo
		nodeStats  *client.NodeStatsResponse
		indices    []client.IndexInfo
		indexStats *client.IndexStatsResponse
		allocation []client.AllocationInfo
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

	// Allocation is non-fatal and runs outside the errgroup so a slow or hung
	// /_cat/allocation call does not delay the 5 core metrics.  Uses the
	// parent ctx (not gctx) so the request is not prematurely cancelled when
	// the core requests complete.  The buffered channel prevents a goroutine
	// leak regardless of whether the result is consumed.
	allocCh := make(chan []client.AllocationInfo, 1)
	go func() {
		alloc, err := c.GetAllocation(ctx)
		if err != nil {
			allocCh <- nil
			return
		}
		allocCh <- alloc
	}()

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// Wait for allocation data or context expiry.  The goroutine above uses
	// the parent ctx, so it will complete (success or error) before ctx
	// expires; ctx.Done() acts as the outer timeout guard.
	select {
	case allocation = <-allocCh:
	case <-ctx.Done():
	}

	if health == nil || nodeStats == nil || indexStats == nil {
		return nil, fmt.Errorf("FetchAll: incomplete response (unexpected nil)")
	}

	snap := &model.Snapshot{
		Health:     *health,
		Nodes:      nodes,
		NodeStats:  *nodeStats,
		Indices:    indices,
		IndexStats: *indexStats,
		Allocation: allocation,
		FetchedAt:  time.Now(),
	}
	return snap, nil
}
