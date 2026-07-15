package streamoid

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// ResolveSecret resolves a secret_ref (C3). Never returns an inline secret
// from source: a value of "env://VAR" reads VAR from the environment;
// "vault://..." is recognized as a reference but, until a real secret
// backend exists, falls back to an env var named after the last path
// segment (upper-cased, dashes to underscores) so nothing is hardcoded in
// source. A bare value (no scheme) is returned as-is -- dev-only.
func ResolveSecret(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if v, ok := strings.CutPrefix(ref, "env://"); ok {
		return os.Getenv(v)
	}
	if _, ok := strings.CutPrefix(ref, "vault://"); ok {
		parts := strings.Split(ref, "/")
		last := parts[len(parts)-1]
		varName := strings.ToUpper(strings.ReplaceAll(last, "-", "_"))
		return os.Getenv(varName)
	}
	return ref
}

func truthy(value string, def bool) bool {
	if value == "" {
		return def
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// Config is the streamoid SDK's own configuration, read from the environment
// (falling back to existing per-service env names where the caller supplies
// one via LoadConfig's fallback params). See docs/platform-adoption-integration.md
// (streamoid-os) for the full STREAMOID_* reference.
type Config struct {
	// Product is the emitting product tag (e.g. "catalogix").
	Product string
	// Source is the runtime tag in events (e.g. "backend").
	Source string
	// Environment is "development" | "test" | "staging" | "prod" etc.
	Environment string

	// PlatformAPIURL is the unified-backend base (L3 sink). Empty disables
	// L3 writes (soft-skip, not an error).
	PlatformAPIURL string
	// PlatformToken is the resolved (not the ref) shared service token used
	// to authenticate against unified-backend's serviceAuth middleware.
	PlatformToken string
	FilesEnabled  bool
	AuditEnabled  bool

	PostHogHost string
	PostHogKey  string

	// MCPRegistryURL is the registry base (L4). Defaults to PlatformAPIURL
	// when unset.
	MCPRegistryURL string

	// Timeout bounds outbound HTTP calls to unified-backend/PostHog.
	Timeout time.Duration
}

// HasPlatform reports whether L3 writes are configured.
func (c Config) HasPlatform() bool { return c.PlatformAPIURL != "" }

// LoadConfigOptions lets a caller supply a fallback base URL for their
// existing pre-platform env var (e.g. UNIFIED_BASE_URL, UNIFIED_SYSTEM_API)
// so adoption is incremental, matching the Python/TS SDKs' behavior.
type LoadConfigOptions struct {
	DefaultProduct         string
	DefaultSource          string
	FallbackPlatformAPIURL string
}

// LoadConfig reads Config from the environment. Call once per process (it is
// cheap; callers that want a cached singleton can wrap it themselves --
// unlike Python's lru_cache/TS's module-level `cached` var, Go has no
// implicit global cache here by design, since a long-running Go service may
// legitimately want to reload after env changes in tests).
func LoadConfig(opts LoadConfigOptions) Config {
	product := os.Getenv("STREAMOID_PRODUCT")
	if product == "" {
		product = opts.DefaultProduct
	}
	source := os.Getenv("STREAMOID_SOURCE")
	if source == "" {
		source = opts.DefaultSource
		if source == "" {
			source = "backend"
		}
	}
	environment := os.Getenv("STREAMOID_ENV")
	if environment == "" {
		environment = os.Getenv("ENVIRONMENT")
	}
	if environment == "" {
		environment = "development"
	}

	platformAPIURL := os.Getenv("STREAMOID_PLATFORM_API_URL")
	if platformAPIURL == "" {
		platformAPIURL = opts.FallbackPlatformAPIURL
	}
	platformAPIURL = strings.TrimRight(platformAPIURL, "/")

	posthogHost := os.Getenv("STREAMOID_POSTHOG_HOST")
	if posthogHost == "" {
		posthogHost = "https://ph.streamoid.com"
	}
	posthogHost = strings.TrimRight(posthogHost, "/")

	mcpRegistryURL := os.Getenv("STREAMOID_MCP_REGISTRY_URL")
	if mcpRegistryURL == "" {
		mcpRegistryURL = platformAPIURL
	}
	mcpRegistryURL = strings.TrimRight(mcpRegistryURL, "/")

	timeoutS := 5.0
	if v := os.Getenv("STREAMOID_HTTP_TIMEOUT_S"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			timeoutS = parsed
		}
	}

	return Config{
		Product:        product,
		Source:         source,
		Environment:    environment,
		PlatformAPIURL: platformAPIURL,
		PlatformToken:  ResolveSecret(os.Getenv("STREAMOID_PLATFORM_TOKEN_REF")),
		FilesEnabled:   truthy(os.Getenv("STREAMOID_FILES_ENABLED"), true),
		AuditEnabled:   truthy(os.Getenv("STREAMOID_AUDIT_ENABLED"), true),
		PostHogHost:    posthogHost,
		PostHogKey:     ResolveSecret(os.Getenv("STREAMOID_POSTHOG_KEY_REF")),
		MCPRegistryURL: mcpRegistryURL,
		Timeout:        time.Duration(timeoutS * float64(time.Second)),
	}
}
