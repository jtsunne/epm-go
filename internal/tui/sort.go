package tui

import (
	"sort"
	"strings"

	"github.com/jtsunne/epm-go/internal/model"
)

// sortIndexRows returns a sorted copy of rows.
// Column mapping:
//
//	0=Name, 1=PrimaryShards, 2=TotalSizeBytes, 3=AvgShardSize, 4=DocCount,
//	5=IndexingRate, 6=SearchRate, 7=IndexLatency, 8=SearchLatency
//
// col -1 means no sort (preserve order).
// Ties are broken by Name ascending.
func sortIndexRows(rows []model.IndexRow, col int, desc bool) []model.IndexRow {
	out := make([]model.IndexRow, len(rows))
	copy(out, rows)

	if col < 0 {
		return out
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		var less bool
		switch col {
		case 0:
			la, lb := strings.ToLower(a.Name), strings.ToLower(b.Name)
			if la == lb {
				return false
			}
			less = la < lb
		case 1:
			if a.PrimaryShards != b.PrimaryShards {
				less = a.PrimaryShards < b.PrimaryShards
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 2:
			if a.TotalSizeBytes != b.TotalSizeBytes {
				less = a.TotalSizeBytes < b.TotalSizeBytes
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 3:
			if a.AvgShardSize != b.AvgShardSize {
				less = a.AvgShardSize < b.AvgShardSize
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 4:
			if a.DocCount != b.DocCount {
				less = a.DocCount < b.DocCount
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 5:
			if aSentinel, bSentinel := a.IndexingRate < 0, b.IndexingRate < 0; aSentinel != bSentinel {
				return bSentinel // sentinel always last regardless of direction
			} else if a.IndexingRate != b.IndexingRate {
				less = a.IndexingRate < b.IndexingRate
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 6:
			if aSentinel, bSentinel := a.SearchRate < 0, b.SearchRate < 0; aSentinel != bSentinel {
				return bSentinel
			} else if a.SearchRate != b.SearchRate {
				less = a.SearchRate < b.SearchRate
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 7:
			if aSentinel, bSentinel := a.IndexLatency < 0, b.IndexLatency < 0; aSentinel != bSentinel {
				return bSentinel
			} else if a.IndexLatency != b.IndexLatency {
				less = a.IndexLatency < b.IndexLatency
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 8:
			if aSentinel, bSentinel := a.SearchLatency < 0, b.SearchLatency < 0; aSentinel != bSentinel {
				return bSentinel
			} else if a.SearchLatency != b.SearchLatency {
				less = a.SearchLatency < b.SearchLatency
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		default:
			la, lb := strings.ToLower(a.Name), strings.ToLower(b.Name)
			if la == lb {
				return false
			}
			less = la < lb
		}
		if desc {
			return !less
		}
		return less
	})
	return out
}

// sortNodeRows returns a sorted copy of rows.
// Column mapping:
//
//	0=Name, 1=Role, 2=IP, 3=IndexingRate, 4=SearchRate, 5=IndexLatency, 6=SearchLatency
//
// Ties are broken by Name ascending.
func sortNodeRows(rows []model.NodeRow, col int, desc bool) []model.NodeRow {
	out := make([]model.NodeRow, len(rows))
	copy(out, rows)

	if col < 0 {
		return out
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		var less bool
		switch col {
		case 0:
			la, lb := strings.ToLower(a.Name), strings.ToLower(b.Name)
			if la == lb {
				return false
			}
			less = la < lb
		case 1:
			if a.Role != b.Role {
				less = strings.ToLower(a.Role) < strings.ToLower(b.Role)
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 2:
			if a.IP != b.IP {
				less = a.IP < b.IP
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 3:
			if aSentinel, bSentinel := a.IndexingRate < 0, b.IndexingRate < 0; aSentinel != bSentinel {
				return bSentinel
			} else if a.IndexingRate != b.IndexingRate {
				less = a.IndexingRate < b.IndexingRate
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 4:
			if aSentinel, bSentinel := a.SearchRate < 0, b.SearchRate < 0; aSentinel != bSentinel {
				return bSentinel
			} else if a.SearchRate != b.SearchRate {
				less = a.SearchRate < b.SearchRate
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 5:
			if aSentinel, bSentinel := a.IndexLatency < 0, b.IndexLatency < 0; aSentinel != bSentinel {
				return bSentinel
			} else if a.IndexLatency != b.IndexLatency {
				less = a.IndexLatency < b.IndexLatency
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		case 6:
			if aSentinel, bSentinel := a.SearchLatency < 0, b.SearchLatency < 0; aSentinel != bSentinel {
				return bSentinel
			} else if a.SearchLatency != b.SearchLatency {
				less = a.SearchLatency < b.SearchLatency
			} else {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
		default:
			la, lb := strings.ToLower(a.Name), strings.ToLower(b.Name)
			if la == lb {
				return false
			}
			less = la < lb
		}
		if desc {
			return !less
		}
		return less
	})
	return out
}

// filterIndexRows returns rows whose Name contains search (case-insensitive).
// Returns all rows when search is empty.
func filterIndexRows(rows []model.IndexRow, search string) []model.IndexRow {
	if search == "" {
		return rows
	}
	lower := strings.ToLower(search)
	out := rows[:0:0]
	for _, r := range rows {
		if strings.Contains(strings.ToLower(r.Name), lower) {
			out = append(out, r)
		}
	}
	return out
}

// filterNodeRows returns rows whose Name or IP contains search (case-insensitive).
// Returns all rows when search is empty.
func filterNodeRows(rows []model.NodeRow, search string) []model.NodeRow {
	if search == "" {
		return rows
	}
	lower := strings.ToLower(search)
	out := rows[:0:0]
	for _, r := range rows {
		if strings.Contains(strings.ToLower(r.Name), lower) ||
			strings.Contains(strings.ToLower(r.IP), lower) {
			out = append(out, r)
		}
	}
	return out
}
