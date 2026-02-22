package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestClassifyError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"nil error", nil, ""},
		{"connection refused", errors.New("dial tcp: connection refused"), "Connection refused"},
		{"401", errors.New("unexpected status 401 Unauthorized"), "Authentication failed (401)"},
		{"unauthorized lowercase", errors.New("unauthorized access"), "Authentication failed (401)"},
		{"403", errors.New("unexpected status 403 Forbidden"), "Authentication failed (403)"},
		{"forbidden lowercase", errors.New("forbidden resource"), "Authentication failed (403)"},
		{"context deadline exceeded", errors.New("context deadline exceeded"), "Timeout"},
		{"timeout", errors.New("request timeout after 5s"), "Timeout"},
		{"certificate", errors.New("x509: certificate signed by unknown authority"), "TLS error"},
		{"tls", errors.New("tls: handshake failure"), "TLS error"},
		{"x509", errors.New("x509: unknown certificate"), "TLS error"},
		{"short unknown", errors.New("some random error"), "some random error"},
		{"long unknown", errors.New("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa..."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyError(tc.err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestIsTLSError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"connection refused", errors.New("connection refused"), false},
		{"timeout", errors.New("context deadline exceeded"), false},
		{"certificate", errors.New("x509: certificate expired"), true},
		{"tls", errors.New("tls: handshake failure"), true},
		{"x509", errors.New("x509: unknown CA"), true},
		{"mixed TLS uppercase", errors.New("TLS certificate error"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isTLSError(tc.err))
		})
	}
}

func TestRetryCountdown(t *testing.T) {
	// Zero time → fallback message.
	assert.Equal(t, "Press r to retry", retryCountdown(time.Time{}))

	// Past time → "Retrying..."
	assert.Equal(t, "Retrying...", retryCountdown(time.Now().Add(-time.Second)))

	// Future time → countdown message.
	future := time.Now().Add(15 * time.Second)
	msg := retryCountdown(future)
	assert.Contains(t, msg, "Retrying in")
	assert.Contains(t, msg, "r: retry now")
}

func TestApp_FetchErrorSetsNextRetryAt(t *testing.T) {
	app := NewApp(nil, 10*time.Second)

	before := time.Now()
	newModel, _ := app.Update(FetchErrorMsg{Err: errors.New("connection refused")})
	after := time.Now()
	updated := newModel.(*App)

	assert.False(t, updated.nextRetryAt.IsZero(), "nextRetryAt should be set after FetchErrorMsg")
	assert.True(t, updated.nextRetryAt.After(before), "nextRetryAt should be in the future")
	assert.True(t, updated.nextRetryAt.Before(after.Add(5*time.Second)), "nextRetryAt within expected backoff window")
}

func TestApp_SnapshotMsgClearsNextRetryAt(t *testing.T) {
	app := NewApp(nil, 10*time.Second)

	// First put app into disconnected state with a retry scheduled.
	newModel, _ := app.Update(FetchErrorMsg{Err: errors.New("timeout")})
	app = newModel.(*App)
	assert.False(t, app.nextRetryAt.IsZero())

	// A successful snapshot should clear the retry timer.
	snap := makeFixtureSnapshot()
	newModel, _ = app.Update(makeFixtureMsg(snap))
	app = newModel.(*App)
	assert.True(t, app.nextRetryAt.IsZero(), "nextRetryAt should be cleared after successful snapshot")
}

func TestApp_CountdownTickMsg_DropsStale(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	// Manually set countdownGen to 5; a message with gen=3 is stale.
	app.countdownGen = 5
	app.connState = stateDisconnected

	newModel, cmd := app.Update(CountdownTickMsg{Gen: 3})
	_ = newModel
	assert.Nil(t, cmd, "stale CountdownTickMsg should produce no command")
}

func TestApp_CountdownTickMsg_ReschedulesWhenDisconnected(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.countdownGen = 2
	app.connState = stateDisconnected

	_, cmd := app.Update(CountdownTickMsg{Gen: 2})
	assert.NotNil(t, cmd, "current CountdownTickMsg when disconnected should reschedule")
}

func TestApp_CountdownTickMsg_DropsWhenConnected(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.countdownGen = 2
	app.connState = stateConnected

	_, cmd := app.Update(CountdownTickMsg{Gen: 2})
	assert.Nil(t, cmd, "CountdownTickMsg when connected should not reschedule")
}

func TestSanitize(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"empty string", "", ""},
		{"plain text passthrough", "hello world", "hello world"},
		{"CSI color reset stripped", "\x1b[0m", ""},
		{"CSI color sequence stripped, text preserved", "\x1b[31mred\x1b[0m", "red"},
		{"OSC terminated by BEL stripped", "\x1b]0;title\x07text", "text"},
		{"OSC terminated by ST stripped", "\x1b]0;title\x1b\\text", "text"},
		{"single char escape stripped", "\x1bA", ""},
		{"lone ESC at end stripped", "hello\x1b", "hello"},
		{"C1 control U+0084 stripped", "a\xc2\x84b", "ab"},
		{"DEL 0x7F stripped", "a\x7fb", "ab"},
		{"NUL control 0x01 stripped", "a\x01b", "ab"},
		{"mixed safe and unsafe", "hello\x1b[31m world", "hello world"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, sanitize(tc.input))
		})
	}
}

// headerLineCount returns the number of lines in a rendered header string
// (ANSI-stripped), treating a single-line result as count=1.
func headerLineCount(rendered string) int {
	stripped := stripANSI(rendered)
	return strings.Count(stripped, "\n") + 1
}

func TestRenderHeader_LongClusterNameWidth60(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 60
	app.connState = stateConnected
	app.lastUpdated = time.Date(2024, 1, 1, 14, 32, 5, 0, time.UTC)

	snap := makeFixtureSnapshot()
	snap.Health.ClusterName = "my-very-very-long-cluster-name-that-would-normally-overflow"
	snap.Health.Status = "green"
	app.current = snap

	result := renderHeader(app)
	assert.Equal(t, 1, headerLineCount(result), "header must be single line at width=60 with long cluster name")
	assert.Equal(t, 60, lipgloss.Width(result), "rendered header must fill terminal width exactly")
}

func TestRenderHeader_VeryNarrowWidth30(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 30
	app.connState = stateConnected
	app.lastUpdated = time.Date(2024, 1, 1, 14, 32, 5, 0, time.UTC)

	snap := makeFixtureSnapshot()
	snap.Health.ClusterName = "production-cluster"
	snap.Health.Status = "green"
	app.current = snap

	result := renderHeader(app)
	assert.Equal(t, 1, headerLineCount(result), "header must be single line at width=30")
	assert.Equal(t, 30, lipgloss.Width(result), "rendered header must fill terminal width exactly")
}

func TestRenderHeader_DisconnectedWidth60(t *testing.T) {
	app := NewApp(nil, 10*time.Second)
	app.width = 60
	app.connState = stateDisconnected
	app.lastError = errors.New("connection refused")
	app.nextRetryAt = time.Now().Add(15 * time.Second)

	snap := makeFixtureSnapshot()
	snap.Health.ClusterName = "prod-cluster"
	snap.Health.Status = "green"
	app.current = snap

	result := renderHeader(app)
	assert.Equal(t, 1, headerLineCount(result), "disconnected header must be single line at width=60")
	assert.Equal(t, 60, lipgloss.Width(result), "disconnected header must fill terminal width exactly")
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		name  string
		input time.Duration
		want  string
	}{
		{"5 seconds", 5 * time.Second, "5s"},
		{"30 seconds", 30 * time.Second, "30s"},
		{"59 seconds", 59 * time.Second, "59s"},
		{"60 seconds exact", 60 * time.Second, "1m"},
		{"90 seconds", 90 * time.Second, "1m30s"},
		{"120 seconds", 120 * time.Second, "2m"},
		{"300 seconds", 300 * time.Second, "5m"},
		{"150 seconds", 150 * time.Second, "2m30s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatDuration(tc.input))
		})
	}
}
