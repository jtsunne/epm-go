package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ESClient defines the interface for interacting with an Elasticsearch cluster.
type ESClient interface {
	GetClusterHealth(ctx context.Context) (*ClusterHealth, error)
	GetNodes(ctx context.Context) ([]NodeInfo, error)
	GetNodeStats(ctx context.Context) (*NodeStatsResponse, error)
	GetIndices(ctx context.Context) ([]IndexInfo, error)
	GetIndexStats(ctx context.Context) (*IndexStatsResponse, error)
	Ping(ctx context.Context) error
	BaseURL() string
}

// ClientConfig holds configuration for DefaultClient.
type ClientConfig struct {
	BaseURL            string
	Username           string
	Password           string
	InsecureSkipVerify bool
	RequestTimeout     time.Duration
}

// DefaultClient implements ESClient using the standard net/http package.
type DefaultClient struct {
	http   *http.Client
	config ClientConfig
}

// NewDefaultClient constructs a DefaultClient from the given config.
// It configures TLS skip-verify and request timeout from the config.
// Returns an error if BaseURL is empty.
func NewDefaultClient(cfg ClientConfig) (*DefaultClient, error) {
	if cfg.BaseURL == "" {
		return nil, fmt.Errorf("BaseURL is required")
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 10 * time.Second
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec
	}

	return &DefaultClient{
		http: &http.Client{
			Timeout:   cfg.RequestTimeout,
			Transport: transport,
		},
		config: cfg,
	}, nil
}

// BaseURL returns the configured base URL of the Elasticsearch cluster.
func (c *DefaultClient) BaseURL() string {
	return c.config.BaseURL
}

// doGet performs a GET request to the given path (relative to BaseURL).
// It sets Accept: application/json and Basic Auth if credentials are configured.
// Returns the response body bytes or an error on non-2xx status.
func (c *DefaultClient) doGet(ctx context.Context, path string) ([]byte, error) {
	url := strings.TrimRight(c.config.BaseURL, "/") + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if c.config.Username != "" || c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseBytes = 32 * 1024 * 1024 // 32 MB — well above any real ES response
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, truncate(body, 200))
	}

	return body, nil
}

// Ping checks connectivity by calling /_cluster/health with a 1s timeout.
func (c *DefaultClient) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	_, err := c.doGet(pingCtx, endpointClusterHealth)
	return err
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
