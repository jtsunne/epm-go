package tui

import "testing"

func TestThreshold_CPU(t *testing.T) {
	cases := []struct {
		pct  float64
		want severity
	}{
		{0, severityNormal},
		{79, severityNormal},
		{80, severityNormal},   // boundary: >80 triggers warning
		{80.1, severityWarning},
		{89, severityWarning},
		{90, severityWarning},  // boundary: >90 triggers critical
		{90.1, severityCritical},
		{100, severityCritical},
	}
	for _, tc := range cases {
		got := cpuSeverity(tc.pct)
		if got != tc.want {
			t.Errorf("cpuSeverity(%v) = %v, want %v", tc.pct, got, tc.want)
		}
	}
}

func TestThreshold_JVM(t *testing.T) {
	cases := []struct {
		pct  float64
		want severity
	}{
		{0, severityNormal},
		{74, severityNormal},
		{75, severityNormal},   // boundary: >75 triggers warning
		{75.1, severityWarning},
		{84, severityWarning},
		{85, severityWarning},  // boundary: >85 triggers critical
		{85.1, severityCritical},
		{100, severityCritical},
	}
	for _, tc := range cases {
		got := jvmSeverity(tc.pct)
		if got != tc.want {
			t.Errorf("jvmSeverity(%v) = %v, want %v", tc.pct, got, tc.want)
		}
	}
}

func TestThreshold_Storage(t *testing.T) {
	cases := []struct {
		pct  float64
		want severity
	}{
		{0, severityNormal},
		{79, severityNormal},
		{80, severityNormal},   // boundary: >80 triggers warning
		{80.1, severityWarning},
		{89, severityWarning},
		{90, severityWarning},  // boundary: >90 triggers critical
		{90.1, severityCritical},
		{100, severityCritical},
	}
	for _, tc := range cases {
		got := storageSeverity(tc.pct)
		if got != tc.want {
			t.Errorf("storageSeverity(%v) = %v, want %v", tc.pct, got, tc.want)
		}
	}
}

func TestThreshold_Latency(t *testing.T) {
	searchCases := []struct {
		ms   float64
		want severity
	}{
		{0, severityNormal},
		{999, severityNormal},
		{1000, severityNormal},  // boundary: >1000 triggers critical
		{1000.1, severityCritical},
		{5000, severityCritical},
	}
	for _, tc := range searchCases {
		got := searchLatSeverity(tc.ms)
		if got != tc.want {
			t.Errorf("searchLatSeverity(%v) = %v, want %v", tc.ms, got, tc.want)
		}
	}

	indexCases := []struct {
		ms   float64
		want severity
	}{
		{0, severityNormal},
		{499, severityNormal},
		{500, severityNormal},  // boundary: >500 triggers warning
		{500.1, severityWarning},
		{2000, severityWarning},
	}
	for _, tc := range indexCases {
		got := indexLatSeverity(tc.ms)
		if got != tc.want {
			t.Errorf("indexLatSeverity(%v) = %v, want %v", tc.ms, got, tc.want)
		}
	}
}
