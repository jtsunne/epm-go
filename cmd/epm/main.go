package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/dm/epm-go/internal/client"
)

// parseESURI parses an Elasticsearch URI and returns the base URL (without credentials),
// username, and password. Returns an error if the URI is invalid or has an unsupported scheme.
func parseESURI(esURI string) (baseURL, username, password string, err error) {
	u, err := url.Parse(esURI)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid URI %q: %w", esURI, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", "", "", fmt.Errorf("unsupported scheme %q (must be http or https)", u.Scheme)
	}

	if u.Hostname() == "" {
		return "", "", "", fmt.Errorf("invalid URI %q: host is required", esURI)
	}

	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
		// Remove credentials from URL stored in config
		u.User = nil
	}

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

	if *interval <= 0 {
		fmt.Fprintln(os.Stderr, "error: --interval must be positive")
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

	cfg := client.ClientConfig{
		BaseURL:            baseURL,
		Username:           username,
		Password:           password,
		InsecureSkipVerify: *insecure,
		RequestTimeout:     10 * time.Second,
	}

	c, err := client.NewDefaultClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	health, err := c.GetClusterHealth(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get cluster health: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("cluster: %s  status: %s  nodes: %d  shards: %d  poll: %v\n",
		health.ClusterName, health.Status, health.NumberOfNodes, health.ActiveShards, *interval)
}
