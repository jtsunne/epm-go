package main

import (
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dm/epm-go/internal/client"
	"github.com/dm/epm-go/internal/tui"
)

// version is set at build time via -ldflags="-X main.version=..."
var version = "dev"

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
		interval    = flag.Duration("interval", 10*time.Second, "polling interval, between 5s and 300s (e.g. 10s, 30s)")
		insecure    = flag.Bool("insecure", false, "skip TLS certificate verification (for self-signed certs)")
		showVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "epm %s — Elasticsearch Performance Monitor\n\n", version)
		fmt.Fprintf(os.Stderr, "usage:\n")
		fmt.Fprintf(os.Stderr, "  epm [--interval <duration>] [--insecure] [--version] <elasticsearch-uri>\n\n")
		fmt.Fprintf(os.Stderr, "examples:\n")
		fmt.Fprintf(os.Stderr, "  epm http://localhost:9200\n")
		fmt.Fprintf(os.Stderr, "  epm --insecure https://elastic:changeme@prod.example.com:9200\n")
		fmt.Fprintf(os.Stderr, "  epm --interval 30s http://localhost:9200\n")
		fmt.Fprintf(os.Stderr, "  epm --version\n\n")
		fmt.Fprintf(os.Stderr, "flags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("epm version %s\n", version)
		os.Exit(0)
	}

	const minInterval = 5 * time.Second
	const maxInterval = 300 * time.Second
	if *interval < minInterval || *interval > maxInterval {
		fmt.Fprintf(os.Stderr, "error: --interval must be between 5s and 300s (got %s)\n", *interval)
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

	// Warn when credentials are sent over unencrypted HTTP.
	if username != "" || password != "" {
		u, _ := url.Parse(baseURL)
		if u != nil && u.Scheme == "http" {
			fmt.Fprintln(os.Stderr, "warning: credentials will be sent over unencrypted HTTP; use https:// for production clusters")
		}
	}

	// Hint: connecting to https:// without --insecure may fail if the cluster
	// uses a self-signed certificate. Print once before the TUI starts.
	{
		u, _ := url.Parse(baseURL)
		if u != nil && u.Scheme == "https" && !*insecure {
			fmt.Fprintln(os.Stderr, "note: connecting to https:// — if the cluster uses a self-signed certificate, add --insecure")
		}
	}

	// Mirror the fetchCmd context timeout: interval-500ms, capped at 10s.
	// The 10s cap ensures the HTTP transport also releases promptly on quit,
	// consistent with the context cancellation guarantee in tui.fetchTimeout.
	requestTimeout := *interval - 500*time.Millisecond
	if requestTimeout > 10*time.Second {
		requestTimeout = 10 * time.Second
	}

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

	app := tui.NewApp(c, *interval)
	p := tea.NewProgram(app, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	// After the TUI exits, if the last error was a TLS error and --insecure
	// was not set, print a post-exit hint to stderr (visible in the terminal
	// after the alt screen is restored).
	if !*insecure {
		if a, ok := finalModel.(*tui.App); ok {
			if lastErr := a.LastError(); lastErr != nil {
				msg := strings.ToLower(lastErr.Error())
				if strings.Contains(msg, "certificate") || strings.Contains(msg, "tls") || strings.Contains(msg, "x509") {
					fmt.Fprintln(os.Stderr, "hint: connection failed due to a TLS error — try adding --insecure for self-signed certificates")
				}
			}
		}
	}
}
