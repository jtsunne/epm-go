package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClampRate(t *testing.T) {
	cases := []struct {
		name  string
		input float64
		want  float64
	}{
		{"zero", 0, 0},
		{"normal", 1000, 1000},
		{"at limit", maxRatePerSec, maxRatePerSec},
		{"above limit", maxRatePerSec + 1, 0},
		{"huge value", 1e12, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, clampRate(tc.input))
		})
	}
}

func TestClampLatency(t *testing.T) {
	cases := []struct {
		name  string
		input float64
		want  float64
	}{
		{"zero", 0, 0},
		{"normal", 5.5, 5.5},
		{"at limit", maxLatencyMs, maxLatencyMs},
		{"above limit", maxLatencyMs + 1, maxLatencyMs},
		{"huge value", 1e9, maxLatencyMs},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, clampLatency(tc.input))
		})
	}
}

func TestSafeDivide(t *testing.T) {
	cases := []struct {
		name string
		a, b float64
		want float64
	}{
		{"normal", 10, 4, 2.5},
		{"divide by zero", 5, 0, 0},
		{"zero numerator", 0, 5, 0},
		{"both zero", 0, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, safeDivide(tc.a, tc.b))
		})
	}
}

func TestMaxFloat64(t *testing.T) {
	cases := []struct {
		name string
		a, b float64
		want float64
	}{
		{"a greater", 5, 3, 5},
		{"b greater", 3, 5, 5},
		{"equal", 4, 4, 4},
		{"negative", -1, -2, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, maxFloat64(tc.a, tc.b))
		})
	}
}
