package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dm/epm-go/internal/model"
)

func TestNodeTableSetData_SortByRate(t *testing.T) {
	m := NewNodeTable()
	rows := []model.NodeRow{
		{Name: "node-1", IP: "10.0.0.1", IndexingRate: 50.0},
		{Name: "node-2", IP: "10.0.0.2", IndexingRate: 200.0},
		{Name: "node-3", IP: "10.0.0.3", IndexingRate: 100.0},
	}
	m.SetData(rows)

	require.Len(t, m.displayRows, 3)
	assert.Equal(t, "node-2", m.displayRows[0].Name, "highest IndexingRate should be first")
	assert.Equal(t, "node-3", m.displayRows[1].Name)
	assert.Equal(t, "node-1", m.displayRows[2].Name, "lowest IndexingRate should be last")
}

func TestNodeTableSearch_ByIP(t *testing.T) {
	m := NewNodeTable()
	m.search = "192.168.1"
	rows := []model.NodeRow{
		{Name: "node-1", IP: "192.168.1.1", IndexingRate: 100.0},
		{Name: "node-2", IP: "10.0.0.1", IndexingRate: 200.0},
		{Name: "node-3", IP: "192.168.1.2", IndexingRate: 50.0},
	}
	m.SetData(rows)

	require.Len(t, m.displayRows, 2, "only nodes with matching IP should remain")
	assert.Equal(t, "node-1", m.displayRows[0].Name, "higher rate first within filtered set")
	assert.Equal(t, "node-3", m.displayRows[1].Name)
}
