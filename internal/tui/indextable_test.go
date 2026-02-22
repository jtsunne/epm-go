package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dm/epm-go/internal/model"
)

func TestIndexTableSetData_AppliesDefaultSort(t *testing.T) {
	m := NewIndexTable()
	rows := []model.IndexRow{
		{Name: "alpha", IndexingRate: 10.0},
		{Name: "gamma", IndexingRate: 300.0},
		{Name: "beta", IndexingRate: 50.0},
	}
	m.SetData(rows)

	require.Len(t, m.displayRows, 3)
	assert.Equal(t, "gamma", m.displayRows[0].Name, "highest IndexingRate should be first")
	assert.Equal(t, "beta", m.displayRows[1].Name)
	assert.Equal(t, "alpha", m.displayRows[2].Name, "lowest IndexingRate should be last")
}

func TestIndexTableSearch(t *testing.T) {
	m := NewIndexTable()
	m.search = "logs"
	rows := []model.IndexRow{
		{Name: "logs-2024", IndexingRate: 100.0},
		{Name: "metrics-daily", IndexingRate: 200.0},
		{Name: "logs-2023", IndexingRate: 50.0},
	}
	m.SetData(rows)

	require.Len(t, m.displayRows, 2, "only log indices should remain after filter")
	assert.Equal(t, "logs-2024", m.displayRows[0].Name, "higher rate first within filtered set")
	assert.Equal(t, "logs-2023", m.displayRows[1].Name)
}

func TestIndexTablePagination(t *testing.T) {
	m := NewIndexTable()
	rows := make([]model.IndexRow, 25)
	for i := range rows {
		rows[i] = model.IndexRow{
			Name:         fmt.Sprintf("index-%02d", i),
			IndexingRate: float64(i),
		}
	}
	m.SetData(rows)

	assert.Equal(t, 25, len(m.displayRows))
	assert.Equal(t, 3, pageCount(len(m.displayRows), m.pageSize))
}

func TestIndexTableRender_ContainsIndexName(t *testing.T) {
	m := NewIndexTable()
	rows := []model.IndexRow{
		{Name: "my-test-index", IndexingRate: 42.0},
	}
	m.SetData(rows)

	out := m.renderTable(nil)
	assert.True(t, strings.Contains(out, "my-test-index"), "rendered output should contain the index name")
}
