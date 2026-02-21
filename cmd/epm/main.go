package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"sort"
	"time"

	"github.com/dm/epm-go/internal/client"
	"github.com/dm/epm-go/internal/engine"
	"github.com/dm/epm-go/internal/format"
	"github.com/dm/epm-go/internal/model"
)

// parseESURI parses an Elasticsearch URI and returns the base URL (without credentials),
// username, and password. Returns an error if the URI is invalid or has an unsupported scheme.
func parseESURI(esURI string) (baseURL, username, password string, err error) {
	u, err := url.Parse(esURI)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid URI: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", "", fmt.Errorf("unsupported scheme %q (must be http or https)", u.Scheme)
	}

	if u.Hostname() == "" {
		return "", "", "", fmt.Errorf("invalid URI: host is required")
	}

	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
		// Remove credentials from URL stored in config
		u.User = nil
	}

	// Strip query string and fragment so that path concatenation in doGet
	// produces valid URLs (e.g. "http://host:9200?x=1/_cluster/health" is invalid).
	// ForceQuery must also be cleared: url.Parse("http://host?") sets ForceQuery=true
	// which causes u.String() to emit a trailing "?" even when RawQuery is empty.
	u.RawQuery = ""
	u.Fragment = ""
	u.ForceQuery = false

	return u.String(), username, password, nil
}

func main() {
	var (
		interval = flag.Duration("interval", 10*time.Second, "polling interval (e.g. 10s, 30s)")
		insecure = flag.Bool("insecure", false, "skip TLS certificate verification")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: epm [--interval 10s] [--insecure] <elasticsearch-uri>\n\n")
		fmt.Fprintf(os.Stderr, "examples:\n")
		fmt.Fprintf(os.Stderr, "  epm http://localhost:9200\n")
		fmt.Fprintf(os.Stderr, "  epm --insecure https://elastic:changeme@prod.example.com:9200\n")
		fmt.Fprintf(os.Stderr, "  epm --interval 30s http://localhost:9200\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *interval < time.Second {
		fmt.Fprintln(os.Stderr, "error: --interval must be at least 1s")
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: elasticsearch URI is required")
		flag.Usage()
		os.Exit(1)
	}
	// Reject extra positional arguments. flag.Parse stops at the first
	// non-flag argument, so trailing --flags would also be silently ignored.
	if len(args) > 1 {
		extra := args[1]
		if len(extra) > 1 && extra[0] == '-' {
			fmt.Fprintf(os.Stderr, "error: flag %q must be placed before the URI\n", extra)
		} else {
			fmt.Fprintf(os.Stderr, "error: unexpected argument %q\n", extra)
		}
		flag.Usage()
		os.Exit(1)
	}

	baseURL, username, password, err := parseESURI(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	requestTimeout := *interval - 500*time.Millisecond

	cfg := client.ClientConfig{
		BaseURL:            baseURL,
		Username:           username,
		Password:           password,
		InsecureSkipVerify: *insecure,
		RequestTimeout:     requestTimeout,
	}

	c, err := client.NewDefaultClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()

	fmt.Printf("epm debug: fetching first snapshot from %s ...\n", baseURL)
	prev, err := engine.FetchAll(ctx, c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: first fetch failed: %v\n", err)
		os.Exit(1)
	}

	printClusterSummary(prev)

	waitDur := *interval
	fmt.Printf("\nwaiting %v before second fetch ...\n", waitDur)
	time.Sleep(waitDur)

	fmt.Println("fetching second snapshot ...")
	curr, err := engine.FetchAll(ctx, c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: second fetch failed: %v\n", err)
		os.Exit(1)
	}

	elapsed := curr.FetchedAt.Sub(prev.FetchedAt)
	fmt.Printf("elapsed: %v\n\n", elapsed.Round(time.Millisecond))

	metrics := engine.CalcClusterMetrics(prev, curr, elapsed)
	resources := engine.CalcClusterResources(curr)
	nodeRows := engine.CalcNodeRows(prev, curr, elapsed)
	indexRows := engine.CalcIndexRows(prev, curr, elapsed)

	printMetrics(metrics)
	printResources(resources)
	printNodeRows(nodeRows)
	printIndexRows(indexRows)
}

func printClusterSummary(snap *model.Snapshot) {
	h := snap.Health
	fmt.Printf("cluster: %s  status: %s  nodes: %d  shards: %d\n",
		h.ClusterName, h.Status, h.NumberOfNodes, h.ActiveShards)
	fmt.Printf("fetched at: %s\n", snap.FetchedAt.Format("15:04:05"))
}

func printMetrics(m model.PerformanceMetrics) {
	fmt.Println("--- cluster metrics ---")
	fmt.Printf("  indexing rate:   %s\n", format.FormatRate(m.IndexingRate))
	fmt.Printf("  search rate:     %s\n", format.FormatRate(m.SearchRate))
	fmt.Printf("  index latency:   %s\n", format.FormatLatency(m.IndexLatency))
	fmt.Printf("  search latency:  %s\n", format.FormatLatency(m.SearchLatency))
}

func printResources(r model.ClusterResources) {
	fmt.Println("--- cluster resources ---")
	fmt.Printf("  cpu:     %s\n", format.FormatPercent(r.AvgCPUPercent))
	fmt.Printf("  jvm:     %s\n", format.FormatPercent(r.AvgJVMHeapPercent))
	fmt.Printf("  storage: %s / %s (%s)\n",
		format.FormatBytes(r.StorageUsedBytes),
		format.FormatBytes(r.StorageTotalBytes),
		format.FormatPercent(r.StoragePercent))
}

func printNodeRows(rows []model.NodeRow) {
	if len(rows) == 0 {
		return
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	fmt.Println("--- nodes ---")
	fmt.Printf("  %-20s %-6s %-15s %12s %12s %10s %10s\n",
		"name", "role", "ip", "idx/s", "srch/s", "idx-lat", "srch-lat")
	for _, r := range rows {
		fmt.Printf("  %-20s %-6s %-15s %12s %12s %10s %10s\n",
			r.Name, r.Role, r.IP,
			format.FormatRate(r.IndexingRate),
			format.FormatRate(r.SearchRate),
			format.FormatLatency(r.IndexLatency),
			format.FormatLatency(r.SearchLatency))
	}
}

func printIndexRows(rows []model.IndexRow) {
	if len(rows) == 0 {
		return
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Name < rows[j].Name })
	fmt.Println("--- indices (top 10 by name) ---")
	fmt.Printf("  %-30s %6s %8s %10s %12s %12s\n",
		"name", "shards", "docs", "size", "idx/s", "srch/s")
	limit := 10
	if len(rows) < limit {
		limit = len(rows)
	}
	for _, r := range rows[:limit] {
		fmt.Printf("  %-30s %6d %8s %10s %12s %12s\n",
			r.Name, r.TotalShards,
			format.FormatNumber(r.DocCount),
			format.FormatBytes(r.TotalSizeBytes),
			format.FormatRate(r.IndexingRate),
			format.FormatRate(r.SearchRate))
	}
	if len(rows) > limit {
		fmt.Printf("  ... and %d more\n", len(rows)-limit)
	}
}
