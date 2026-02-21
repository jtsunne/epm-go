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

func main() {
	var (
		interval = flag.Duration("interval", 10*time.Second, "polling interval (e.g. 10s, 30s)")
		insecure = flag.Bool("insecure", false, "skip TLS certificate verification")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: epm <elasticsearch-uri> [--interval 10s] [--insecure]\n\n")
		fmt.Fprintf(os.Stderr, "examples:\n")
		fmt.Fprintf(os.Stderr, "  epm http://localhost:9200\n")
		fmt.Fprintf(os.Stderr, "  epm https://elastic:changeme@prod.example.com:9200 --insecure\n")
		fmt.Fprintf(os.Stderr, "  epm http://localhost:9200 --interval 30s\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "error: elasticsearch URI is required")
		flag.Usage()
		os.Exit(1)
	}

	esURI := args[0]
	u, err := url.Parse(esURI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: invalid URI %q: %v\n", esURI, err)
		os.Exit(1)
	}

	var username, password string
	if u.User != nil {
		username = u.User.Username()
		password, _ = u.User.Password()
		// Remove credentials from URL stored in config
		u.User = nil
	}

	skipVerify := *insecure && u.Scheme == "https"

	cfg := client.ClientConfig{
		BaseURL:            u.String(),
		Username:           username,
		Password:           password,
		InsecureSkipVerify: skipVerify,
		RequestTimeout:     *interval,
	}

	c, err := client.NewDefaultClient(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to create client: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	health, err := c.GetClusterHealth(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to get cluster health: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("cluster: %s  status: %s  nodes: %d  shards: %d\n",
		health.ClusterName, health.Status, health.NumberOfNodes, health.ActiveShards)
}
