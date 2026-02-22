package tui

import (
	"time"

	"github.com/dm/epm-go/internal/model"
)

// SnapshotMsg delivers successful poll results to the TUI.
type SnapshotMsg struct {
	Snapshot  *model.Snapshot
	Metrics   model.PerformanceMetrics
	Resources model.ClusterResources
	NodeRows  []model.NodeRow
	IndexRows []model.IndexRow
}

// FetchErrorMsg signals a poll failure.
type FetchErrorMsg struct{ Err error }

// TickMsg triggers the next scheduled poll.
// Gen must match App.tickGen; stale ticks from superseded schedules are dropped.
type TickMsg struct {
	Time time.Time
	Gen  int
}

// CountdownTickMsg triggers a 1-second display refresh while disconnected so
// the "Retrying in Ns..." countdown in the header stays current.
// Gen must match App.countdownGen; stale ticks are dropped.
type CountdownTickMsg struct{ Gen int }
