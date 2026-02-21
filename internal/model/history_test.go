package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSparklineHistory_PushAndLen(t *testing.T) {
	h := NewSparklineHistory(5)
	assert.Equal(t, 0, h.Len())

	h.Push(SparklinePoint{Timestamp: time.Now(), IndexingRate: 1.0})
	assert.Equal(t, 1, h.Len())

	h.Push(SparklinePoint{Timestamp: time.Now(), IndexingRate: 2.0})
	h.Push(SparklinePoint{Timestamp: time.Now(), IndexingRate: 3.0})
	assert.Equal(t, 3, h.Len())
}

func TestSparklineHistory_OverwritesOldest(t *testing.T) {
	h := NewSparklineHistory(3)

	// Fill to capacity
	h.Push(SparklinePoint{IndexingRate: 10})
	h.Push(SparklinePoint{IndexingRate: 20})
	h.Push(SparklinePoint{IndexingRate: 30})
	require.Equal(t, 3, h.Len())

	// Push beyond capacity — oldest (10) should be overwritten
	h.Push(SparklinePoint{IndexingRate: 40})
	assert.Equal(t, 3, h.Len())

	vals := h.Values("indexingRate")
	assert.Equal(t, []float64{20, 30, 40}, vals)

	// Another push — 20 is overwritten
	h.Push(SparklinePoint{IndexingRate: 50})
	vals = h.Values("indexingRate")
	assert.Equal(t, []float64{30, 40, 50}, vals)
}

func TestSparklineHistory_Values_ChronologicalOrder(t *testing.T) {
	h := NewSparklineHistory(5)
	rates := []float64{1, 2, 3, 4, 5}
	for _, r := range rates {
		h.Push(SparklinePoint{IndexingRate: r, SearchRate: r * 10})
	}

	assert.Equal(t, []float64{1, 2, 3, 4, 5}, h.Values("indexingRate"))
	assert.Equal(t, []float64{10, 20, 30, 40, 50}, h.Values("searchRate"))
}

func TestSparklineHistory_Values_AllFields(t *testing.T) {
	h := NewSparklineHistory(2)
	h.Push(SparklinePoint{
		IndexingRate:  1.1,
		SearchRate:    2.2,
		IndexLatency:  3.3,
		SearchLatency: 4.4,
	})

	assert.Equal(t, []float64{1.1}, h.Values("indexingRate"))
	assert.Equal(t, []float64{2.2}, h.Values("searchRate"))
	assert.Equal(t, []float64{3.3}, h.Values("indexLatency"))
	assert.Equal(t, []float64{4.4}, h.Values("searchLatency"))
}

func TestSparklineHistory_Values_UnknownField(t *testing.T) {
	h := NewSparklineHistory(3)
	h.Push(SparklinePoint{IndexingRate: 5})
	// Unknown field should return zeros
	assert.Equal(t, []float64{0}, h.Values("bogusField"))
}

func TestSparklineHistory_Clear(t *testing.T) {
	h := NewSparklineHistory(4)
	h.Push(SparklinePoint{IndexingRate: 1})
	h.Push(SparklinePoint{IndexingRate: 2})
	require.Equal(t, 2, h.Len())

	h.Clear()
	assert.Equal(t, 0, h.Len())
	assert.Empty(t, h.Values("indexingRate"))

	// Should be able to push again after clear
	h.Push(SparklinePoint{IndexingRate: 99})
	assert.Equal(t, 1, h.Len())
	assert.Equal(t, []float64{99}, h.Values("indexingRate"))
}

func TestSparklineHistory_DefaultCapacity(t *testing.T) {
	h := NewSparklineHistory(0)
	for i := 0; i < 65; i++ {
		h.Push(SparklinePoint{IndexingRate: float64(i)})
	}
	// Default cap is 60, so we should have 60 entries
	assert.Equal(t, 60, h.Len())
	vals := h.Values("indexingRate")
	// Oldest kept entry is index 5 (entries 0-4 were overwritten)
	assert.Equal(t, float64(5), vals[0])
	assert.Equal(t, float64(64), vals[59])
}

func TestSparklineHistory_WrapAround(t *testing.T) {
	h := NewSparklineHistory(3)
	// Push 7 items into capacity-3 buffer
	for i := 1; i <= 7; i++ {
		h.Push(SparklinePoint{IndexingRate: float64(i)})
	}
	assert.Equal(t, 3, h.Len())
	// Should contain [5, 6, 7]
	assert.Equal(t, []float64{5, 6, 7}, h.Values("indexingRate"))
}
