package tui

import (
	"time"

	"github.com/jtsunne/epm-go/internal/client"
	"github.com/jtsunne/epm-go/internal/model"
)

// SnapshotMsg delivers successful poll results to the TUI.
type SnapshotMsg struct {
	Snapshot        *model.Snapshot
	Metrics         model.PerformanceMetrics
	Resources       model.ClusterResources
	NodeRows        []model.NodeRow
	IndexRows       []model.IndexRow
	Recommendations []model.Recommendation
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

// DeleteResultMsg reports the outcome of a DeleteIndex operation.
type DeleteResultMsg struct {
	Names []string
	Err   error
}

// SettingsLoadedMsg delivers current index settings from ES for pre-populating the form.
// Nonce must match App.settingsNonce; stale responses from a prior session are dropped.
type SettingsLoadedMsg struct {
	Values *client.IndexSettingsValues
	Err    error
	Nonce  int
}

// SettingsResultMsg reports the outcome of an UpdateIndexSettings operation.
// Nonce must match App.settingsNonce; stale responses from a prior session are dropped.
type SettingsResultMsg struct {
	Names []string
	Err   error
	Nonce int
}
