# Changelog

All notable changes to epm-go are documented here.

## [v0.3.0] - 2026-03-01

### Added

- **Index settings editor** (`e` key) — select one or more indices and edit their settings (replicas, refresh interval, routing allocation, field limit, read-only block) in a full-screen form. Settings are loaded from ES and only changed fields are submitted via PUT `/_settings`. Multi-index edits show which index values were pre-filled from.
- **Shards and Disk% columns in node table** — two new columns sourced from the `_cat/allocation` endpoint. Shards shows per-node shard count; Disk% shows disk utilization with yellow highlight above 80% and red above 90%. Gracefully hidden when the endpoint returns no data.
- **Index deletion with multi-select** (`space` to select, `d` to delete) — select multiple indices, trigger a confirmation overlay, and delete them via DELETE `/<names>`. Deleted indices are removed from the table immediately.

## [v0.2.0] - 2026-02-27

### Added

- **Analytics screen** (`a` key) — resource-aware cluster recommendations with severity levels (info / warning / critical) across categories: index config, index lifecycle, cluster config, and cluster resources. Recommendations are calculated from the current snapshot without extra API calls.

## [v0.1.1] - 2026-02-26

### Fixed

- Distribution: switched from Homebrew cask to Homebrew formula for CLI install.

## [v0.1.0] - 2026-02-25

### Added

- Initial release: live terminal dashboard for Elasticsearch clusters.
- Cluster health overview bar (status, nodes, indices, shards, CPU%, JVM%, storage%).
- Metric cards with sparklines: indexing rate, search rate, index latency, search latency.
- Index table and node table with sortable columns, pagination, and search.
- Alert coloring on thresholds (CPU, JVM heap, storage, latency).
- Configurable poll interval, TLS skip-verify, basic auth via flags or env vars.
