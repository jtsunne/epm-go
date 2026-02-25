---
# Extend Analytics: Size-Aware Index Lifecycle Recommendations

## Overview

Add index lifecycle recommendations to the analytics screen with three new capabilities:

1. Size-aware date rollup suggestions — detect indices with time-based date suffixes (daily/weekly/monthly), use primary index size to decide whether to recommend consolidating one step (daily→weekly) or two steps directly (daily→monthly when <100 MiB/day), and emit per-group detail with estimated index and shard count impact.
2. Empty index detection — flag non-system indices with zero docs and zero storage as deletion candidates.
3. Cluster-level impact summary — one additional recommendation showing the combined effect of all rollup suggestions on total index count, total shard count, and shard:heap ratio.

A new `CategoryIndexLifecycle` category is introduced to keep these separate from `CategoryIndexConfig`.

## Context

- Files involved:
  - `internal/model/recommendation.go` — add `CategoryIndexLifecycle` constant
  - `internal/model/metrics.go` — add `PriSizeBytes int64` to `IndexRow`
  - `internal/engine/calculator.go` — populate `PriSizeBytes` from already-computed `primarySizeBytes` in `CalcIndexRows`
  - `internal/engine/calculator_test.go` — update/add test cases that check `PriSizeBytes`
  - `internal/engine/recommendations.go` — add `dateRollupRecs`, `emptyIndexRecs`, cluster impact summary, wire into `CalcRecommendations`
  - `internal/engine/recommendations_test.go` — table-driven tests for all new rules
  - `internal/tui/analytics.go` — add `CategoryIndexLifecycle` to `categories` slice and `categoryLabel`
- Related patterns: existing `heapHotspotRecs` helper (returns `[]model.Recommendation`, called from `CalcRecommendations`); package-level `regexp.MustCompile` for regex reuse.
- Dependencies: `regexp` stdlib (not yet imported in recommendations.go).

## Design Notes

**Date pattern detection** (checked in priority order — daily before monthly to avoid false monthly matches):

- Daily:   `^(.+)[.\-_](\d{4})[.\-](\d{2})[.\-](\d{2})$`
- Weekly:  `^(.+)[.\-_](\d{4})[.\-][Ww](\d{1,2})$`
- Monthly: `^(.+)[.\-_](\d{4})[.\-](\d{2})$`

System indices (prefix `.`) are skipped.

**Size-aware rollup decision for daily groups** (requires >= 7 daily indices for same base):

- Compute `avgPriSize = sum(idx.PriSizeBytes) / count` across the group
- If `avgPriSize < 100 MiB` → target = monthly (weekly would only be ~700 MB; monthly = ~3 GB — ideal shard range). Skip the intermediate weekly step.
- If `avgPriSize >= 100 MiB` → target = weekly (7 × avgPriSize ≈ 700 MB+ per week, appropriate granularity)

**Rollup decision for other granularities:**

- Weekly groups (>= 4 indices, same base): target = monthly
- Monthly groups (>= 12 indices, same base): target = yearly

**Per-group impact estimation** (included in recommendation Detail string):

- Current: N indices, avg X MB primary/index, total Y GB
- After consolidation to target period: M = ceil(N / daysInPeriod) estimated indices
- Estimated shard saving: `(N - M) × (sumTotalShards / N)` (proportional reduction assuming same average shard density)
- Estimated size per consolidated index = Y / M (total primary size divided by new index count)

**Cluster-level impact summary** (emitted from `CalcRecommendations` after all rollup recs):

- Aggregates `savedIndices` and `savedShards` returned by `dateRollupRecs`
- Shows: "Applying all rollup suggestions would consolidate A indices into B, saving ~C shards. Shard/GB heap: current X → estimated Y after rollup."
- Emitted as `SeverityNormal` / `CategoryIndexLifecycle` only when savedShards > 0 and heap data is available

**Empty index detection:**

- Non-system, DocCount == 0 AND TotalSizeBytes == 0
- Fires when >= 3 such indices exist (avoids noise from freshly-created indices)

## Development Approach

- Testing approach: regular (implement then test)
- Complete each task fully before moving to the next
- CRITICAL: every task MUST include new/updated tests
- CRITICAL: all tests must pass before starting next task

## Implementation Steps

### Task 1: Extend model and calculator

**Files:**
- Modify: `internal/model/recommendation.go`
- Modify: `internal/model/metrics.go`
- Modify: `internal/engine/calculator.go`
- Modify: `internal/engine/calculator_test.go`

- [x] Add `CategoryIndexLifecycle RecommendationCategory` constant after `CategoryHotspot` in `recommendation.go`
- [x] Add `PriSizeBytes int64` field to `IndexRow` in `metrics.go` (primary data bytes, excluding replicas)
- [x] In `CalcIndexRows` in `calculator.go`, assign `row.PriSizeBytes = primarySizeBytes` (already computed, just not stored)
- [x] Update `calculator_test.go`: add assertions that `PriSizeBytes` is populated correctly for index rows with known fixture data
- [x] Run `make test` — must pass before task 2

### Task 2: Update TUI analytics screen

**Files:**
- Modify: `internal/tui/analytics.go`

- [x] Add `"Index Lifecycle"` case for `CategoryIndexLifecycle` in `categoryLabel()` in `analytics.go`
- [x] Add `CategoryIndexLifecycle` to the `categories` slice in the analytics render path
- [x] Run `make test` — must pass before task 3

### Task 3: Implement dateRollupRecs with size-aware logic

**Files:**
- Modify: `internal/engine/recommendations.go`

- [x] Add `import "regexp"` and declare three package-level compiled regexes: `reIndexDaily`, `reIndexWeekly`, `reIndexMonthly`
- [x] Define size threshold constant `rollupThresholdMiB int64 = 100` for the daily→monthly vs daily→weekly boundary
- [x] Implement `dateRollupRecs(indexRows []model.IndexRow) (recs []model.Recommendation, savedIndices int, savedShards int)`:
  - For each non-system index, try matching daily → weekly → monthly (first match wins), group by `granularity:base`
  - Per group, compute `avgPriSize`, decide target granularity (daily groups: monthly if <100 MiB, weekly otherwise)
  - Per group meeting the count threshold, emit one `SeverityWarning` / `CategoryIndexLifecycle` recommendation with:
    - Title: e.g. `"Consolidate daily 'app-logs' indices → monthly"`
    - Detail: current N indices / avg size / total size, estimated post-consolidation count M, shard reduction estimate, estimated size per consolidated index
  - Accumulate `savedIndices` and `savedShards` across all emitted recommendations
- [x] Run `make test` — must pass before task 4

### Task 4: Implement emptyIndexRecs and wire everything into CalcRecommendations

**Files:**
- Modify: `internal/engine/recommendations.go`

- [x] Implement `emptyIndexRecs(indexRows []model.IndexRow) []model.Recommendation`:
  - Collect non-system indices where `DocCount == 0 AND TotalSizeBytes == 0`
  - If count >= 3, emit one `SeverityWarning` / `CategoryIndexLifecycle` recommendation listing first 5 names (with "... and N more" if over 5)
- [x] In `CalcRecommendations`, after existing checks, call:
  - `rollupRecs, savedIdx, savedShards := dateRollupRecs(indexRows)`
  - `result = append(result, rollupRecs...)`
  - `result = append(result, emptyIndexRecs(indexRows)...)`
  - If `savedShards > 0` and heap data available, compute new estimated shard:heap ratio and append one `SeverityNormal` / `CategoryIndexLifecycle` cluster impact summary recommendation
- [x] Run `make test` — must pass before task 5

### Task 5: Add tests

**Files:**
- Modify: `internal/engine/recommendations_test.go`

- [x] `TestDateRollupRecs_Daily_SmallSize_SuggestsMonthly` — 7 daily indices at 50 MiB primary each; expect Warning containing "monthly" (not "weekly")
- [x] `TestDateRollupRecs_Daily_LargeSize_SuggestsWeekly` — 7 daily indices at 200 MiB primary each; expect Warning containing "weekly"
- [x] `TestDateRollupRecs_Daily_BelowThreshold` — 6 daily indices, no recommendation
- [x] `TestDateRollupRecs_Weekly_AtThreshold` — 4 weekly indices, expect Warning containing "monthly"
- [x] `TestDateRollupRecs_Monthly_AtThreshold` — 12 monthly indices, expect Warning containing "yearly"
- [x] `TestDateRollupRecs_MultipleGroups` — 7 daily "app-logs" (small) and 7 daily "metrics" (large); expect 2 recs with different targets
- [x] `TestDateRollupRecs_SystemIndicesSkipped` — system index names never trigger
- [x] `TestDateRollupRecs_DailyNotConfusedWithMonthly` — YYYY.MM.DD indices not double-counted as monthly
- [x] `TestDateRollupRecs_ImpactCounts` — verify savedIndices and savedShards return values are correct
- [x] `TestEmptyIndexRecs_BelowThreshold` — 2 empty indices, no recommendation
- [x] `TestEmptyIndexRecs_AtThreshold` — 3 empty indices, expect Warning
- [x] `TestEmptyIndexRecs_SystemSkipped` — only system empty indices, no recommendation
- [x] `TestCalcRecommendations_ClusterImpactSummary` — feed CalcRecommendations with rollup-eligible indices and heap data; expect an IndexLifecycle summary recommendation containing shard count and ratio info
- [x] Run `make test` — must pass before task 6

### Task 6: Verify acceptance criteria

- [x] Manual smoke test: `make build`, connect to cluster with date-patterned indices, press `a`
- [x] `make test` — full suite passes
- [x] `make lint` — no new lint issues
- [x] Verify analytics screen shows "Index Lifecycle" section with size-aware targets
- [x] Verify cluster impact summary recommendation appears at bottom of Index Lifecycle section

### Task 7: Update documentation

- [ ] Update `CLAUDE.md`: note `CategoryIndexLifecycle` and `IndexRow.PriSizeBytes`
- [ ] Move this plan to `docs/plans/completed/`
