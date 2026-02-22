package tui

import (
	"errors"
	"testing"
	"time"

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
