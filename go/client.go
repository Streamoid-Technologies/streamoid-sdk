package streamoid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

// Client is the streamoid platform client: one HTTP client + config, shared
// across Capture/RecordFile/RegisterMCPServer. Construct one per process
// (or per service instance in tests) and reuse it -- unlike the Python/TS
// SDKs there is no implicit global singleton.
//
// Deliberately self-contained (plain net/http, no retry layer): this package
// has no dependency on any host application's HTTP client conventions, so it
// can be imported by any Go module without pulling one in. This matches the
// Python/TS ports, which likewise make a single best-effort attempt per call
// -- retries would only delay a write that's already logged-and-swallowed on
// failure, never change whether the caller's own request succeeds.
type Client struct {
	cfg  Config
	http *http.Client
	log  *slog.Logger
}

// NewClient builds a streamoid Client. log may be nil, in which case
// slog.Default() is used.
func NewClient(cfg Config, log *slog.Logger) *Client {
	if log == nil {
		log = slog.Default()
	}
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: cfg.Timeout},
		log:  log,
	}
}

// Config returns the client's resolved configuration.
func (c *Client) Config() Config { return c.cfg }

// postJSON POSTs body as JSON to unified-backend (url) with the shared
// service token attached, and decodes a JSON response into out (which may
// be nil to discard the body). Only ever call this for unified-backend URLs
// -- the Bearer token must never go to a third party. Use postJSONNoAuth
// for anything else (e.g. PostHog).
func (c *Client) postJSON(ctx context.Context, url string, body any, out any) error {
	var authHeader string
	if c.cfg.PlatformToken != "" {
		authHeader = "Bearer " + c.cfg.PlatformToken
	}
	return c.doPostJSON(ctx, url, body, out, authHeader)
}

// postJSONNoAuth POSTs body as JSON to url with no Authorization header --
// for third-party endpoints (PostHog) that must never see the internal
// platform service token.
func (c *Client) postJSONNoAuth(ctx context.Context, url string, body any, out any) error {
	return c.doPostJSON(ctx, url, body, out, "")
}

func (c *Client) doPostJSON(ctx context.Context, url string, body any, out any, authHeader string) error {
	raw, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("streamoid: marshal request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("streamoid: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("streamoid: %s: %w", url, err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("streamoid: %s -> %d: %s", url, resp.StatusCode, truncate(string(respBody), 300))
	}
	if out != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, out); err != nil {
			return fmt.Errorf("streamoid: decode response from %s: %w", url, err)
		}
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
