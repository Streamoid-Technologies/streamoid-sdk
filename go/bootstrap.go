package streamoid

import (
	"context"
	"log/slog"
	"strings"
)

// BootstrapOptions carries what a cmd/ service supplies when wiring itself
// into the platform SDK at startup (see Bootstrap). Product/Owner are
// required from the caller -- this package serves every product, so it
// cannot guess which one is calling it (see LoadConfigOptions.DefaultProduct
// for the same reasoning).
type BootstrapOptions struct {
	// Product is this service's product tag (e.g. "catalogix", "artifax"),
	// used as STREAMOID_PRODUCT's default and the MCP registry's owner.
	Product string
	// ServiceName is this process's logical name (e.g. "products-set"), used
	// as both the SDK's Source tag and the MCP registry's server id.
	ServiceName string
	// URL is this service's externally-reachable base URL, read by the
	// caller from its own env (e.g. STREAMOID_SERVICE_URL). MCP
	// self-registration is soft-skipped -- logged, not fatal -- when empty,
	// since most deployments don't have one wired yet.
	URL string
	// HealthPath is appended to URL for the registry's health check (e.g.
	// "/healthz"). Ignored when URL is empty.
	HealthPath string
}

// Bootstrap builds the shared streamoid Client for a cmd/ service and, best-
// effort, self-registers it into the MCP registry (L4) in the background.
// Call once in main(), right after config load + logger setup. Never blocks
// or fails startup -- a registry that is down, unconfigured, or has no known
// service URL yet only produces a logged message.
func Bootstrap(ctx context.Context, opts BootstrapOptions, log *slog.Logger) *Client {
	if log == nil {
		log = slog.Default()
	}
	cfg := LoadConfig(LoadConfigOptions{DefaultProduct: opts.Product, DefaultSource: opts.ServiceName})
	client := NewClient(cfg, log)

	if opts.URL == "" {
		log.Info("streamoid: no service URL configured -- skipping MCP self-registration", "service", opts.ServiceName)
		return client
	}

	healthURL := opts.URL
	if opts.HealthPath != "" {
		healthURL = strings.TrimRight(opts.URL, "/") + opts.HealthPath
	}
	go func() {
		if err := client.RegisterMCPServer(ctx, RegisterMCPServerOptions{
			ServerID:    opts.ServiceName,
			Owner:       opts.Product,
			DisplayName: opts.ServiceName,
			URL:         opts.URL,
			Transport:   "http",
			HealthURL:   healthURL,
			Enabled:     true,
		}); err != nil {
			log.Warn("streamoid: mcp self-registration failed", "service", opts.ServiceName, "error", err)
		}
	}()
	return client
}
