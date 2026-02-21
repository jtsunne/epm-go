package format

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{"zero", 0, "0 B"},
		{"bytes_small", 512, "512 B"},
		{"bytes_max", 1023, "1023 B"},
		{"one_kb", 1024, "1.0 KB"},
		{"one_and_half_kb", 1536, "1.5 KB"},
		{"just_under_mb", 1024*1024 - 1, "1024.0 KB"},
		{"one_mb", 1024 * 1024, "1.0 MB"},
		{"twenty_mb", 20 * 1024 * 1024, "20.0 MB"},
		{"one_gb", 1024 * 1024 * 1024, "1.0 GB"},
		{"one_and_half_gb", int64(1.5 * 1024 * 1024 * 1024), "1.5 GB"},
		{"one_tb", 1024 * 1024 * 1024 * 1024, "1.0 TB"},
		{"two_tb", 2 * 1024 * 1024 * 1024 * 1024, "2.0 TB"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatBytes(tc.input))
		})
	}
}

func TestFormatLatency(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"zero", 0, "0.00 ms"},
		{"small_ms", 2.34, "2.34 ms"},
		{"just_under_1s", 999.99, "999.99 ms"},
		{"exactly_1s", 1000, "1.00 s"},
		{"one_and_half_s", 1500, "1.50 s"},
		{"ten_s", 10000, "10.00 s"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatLatency(tc.input))
		})
	}
}

func TestFormatRate(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"zero", 0, "0 /s"},
		{"one", 1.0, "1.0 /s"},
		{"fractional", 1204.3, "1,204.3 /s"},
		{"large", 1000000.0, "1,000,000.0 /s"},
		{"small_fraction", 0.5, "0.5 /s"},
		{"hundreds", 892.1, "892.1 /s"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatRate(tc.input))
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{"zero", 0, "0"},
		{"small", 42, "42"},
		{"three_digits", 999, "999"},
		{"four_digits", 1000, "1,000"},
		{"six_digits", 123456, "123,456"},
		{"seven_digits", 1234567, "1,234,567"},
		{"nine_digits", 12345678, "12,345,678"},
		{"negative", -12345, "-12,345"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatNumber(tc.input))
		})
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  string
	}{
		{"zero", 0, "0.0%"},
		{"small", 1.5, "1.5%"},
		{"typical", 34.5, "34.5%"},
		{"hundred", 100.0, "100.0%"},
		{"fractional", 67.89, "67.9%"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, FormatPercent(tc.input))
		})
	}
}

func TestParseHumanBytes(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{"bytes", "512b", 512},
		{"kb_exact", "1kb", 1024},
		{"mb_exact", "100mb", 100 * 1024 * 1024},
		{"gb_exact", "50gb", 50 * 1024 * 1024 * 1024},
		{"gb_decimal", "20.4gb", int64(math.Round(20.4 * 1024 * 1024 * 1024))},
		{"tb_decimal", "1.5tb", int64(math.Round(1.5 * 1024 * 1024 * 1024 * 1024))},
		{"uppercase", "10GB", 10 * 1024 * 1024 * 1024},
		{"mixed_case", "5Mb", 5 * 1024 * 1024},
		{"empty", "", 0},
		{"invalid", "notanumber", 0},
		{"plain_int", "1024", 1024},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, ParseHumanBytes(tc.input))
		})
	}
}
