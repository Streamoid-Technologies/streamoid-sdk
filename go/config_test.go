package streamoid

import "testing"

func TestResolveSecretEnvScheme(t *testing.T) {
	t.Setenv("STREAMOID_TEST_SECRET", "s3cr3t")
	if got := ResolveSecret("env://STREAMOID_TEST_SECRET"); got != "s3cr3t" {
		t.Fatalf("got %q, want %q", got, "s3cr3t")
	}
}

func TestResolveSecretEnvSchemeMissingVar(t *testing.T) {
	if got := ResolveSecret("env://STREAMOID_DOES_NOT_EXIST"); got != "" {
		t.Fatalf("expected empty string for unset env var, got %q", got)
	}
}

func TestResolveSecretVaultSchemeFallsBackToEnv(t *testing.T) {
	t.Setenv("R2_SECRET_ACCESS_KEY", "vault-fallback-value")
	got := ResolveSecret("vault://secret/data/catalogix/r2-secret-access-key")
	if got != "vault-fallback-value" {
		t.Fatalf("got %q, want %q", got, "vault-fallback-value")
	}
}

func TestResolveSecretBareValuePassthrough(t *testing.T) {
	if got := ResolveSecret("dev-only-inline-value"); got != "dev-only-inline-value" {
		t.Fatalf("bare value should be returned as-is, got %q", got)
	}
}

func TestResolveSecretEmpty(t *testing.T) {
	if got := ResolveSecret(""); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
	if got := ResolveSecret("   "); got != "" {
		t.Fatalf("expected empty string for whitespace-only ref, got %q", got)
	}
}

func TestTruthy(t *testing.T) {
	cases := []struct {
		value string
		def   bool
		want  bool
	}{
		{"", true, true},
		{"", false, false},
		{"1", false, true},
		{"true", false, true},
		{"TRUE", false, true},
		{"yes", false, true},
		{"on", false, true},
		{"0", true, false},
		{"false", true, false},
		{"garbage", true, false},
	}
	for _, c := range cases {
		if got := truthy(c.value, c.def); got != c.want {
			t.Errorf("truthy(%q, %v) = %v, want %v", c.value, c.def, got, c.want)
		}
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	for _, k := range []string{
		"STREAMOID_PRODUCT", "STREAMOID_SOURCE", "STREAMOID_ENV", "ENVIRONMENT",
		"STREAMOID_PLATFORM_API_URL", "STREAMOID_POSTHOG_HOST", "STREAMOID_MCP_REGISTRY_URL",
		"STREAMOID_HTTP_TIMEOUT_S", "STREAMOID_PLATFORM_TOKEN_REF", "STREAMOID_POSTHOG_KEY_REF",
		"STREAMOID_FILES_ENABLED", "STREAMOID_AUDIT_ENABLED",
	} {
		t.Setenv(k, "")
	}

	cfg := LoadConfig(LoadConfigOptions{DefaultProduct: "catalogix"})

	if cfg.Product != "catalogix" {
		t.Errorf("Product = %q, want catalogix", cfg.Product)
	}
	if cfg.Source != "backend" {
		t.Errorf("Source = %q, want backend (default)", cfg.Source)
	}
	if cfg.Environment != "development" {
		t.Errorf("Environment = %q, want development (default)", cfg.Environment)
	}
	if cfg.HasPlatform() {
		t.Errorf("expected HasPlatform() false with no platform URL configured")
	}
	if !cfg.FilesEnabled || !cfg.AuditEnabled {
		t.Errorf("expected FilesEnabled/AuditEnabled to default true, got %v/%v", cfg.FilesEnabled, cfg.AuditEnabled)
	}
	if cfg.PostHogHost != "https://ph.streamoid.com" {
		t.Errorf("PostHogHost = %q, want default", cfg.PostHogHost)
	}
}

func TestLoadConfigFallbackPlatformAPIURL(t *testing.T) {
	t.Setenv("STREAMOID_PLATFORM_API_URL", "")
	cfg := LoadConfig(LoadConfigOptions{FallbackPlatformAPIURL: "https://unified.example.com/"})
	if cfg.PlatformAPIURL != "https://unified.example.com" {
		t.Fatalf("PlatformAPIURL = %q, want trimmed fallback", cfg.PlatformAPIURL)
	}
	if !cfg.HasPlatform() {
		t.Fatalf("expected HasPlatform() true once a fallback URL is supplied")
	}
	if cfg.MCPRegistryURL != cfg.PlatformAPIURL {
		t.Fatalf("MCPRegistryURL should default to PlatformAPIURL, got %q vs %q", cfg.MCPRegistryURL, cfg.PlatformAPIURL)
	}
}

func TestLoadConfigExplicitPlatformAPIURLOverridesFallback(t *testing.T) {
	t.Setenv("STREAMOID_PLATFORM_API_URL", "https://explicit.example.com")
	cfg := LoadConfig(LoadConfigOptions{FallbackPlatformAPIURL: "https://fallback.example.com"})
	if cfg.PlatformAPIURL != "https://explicit.example.com" {
		t.Fatalf("explicit env var should win over fallback, got %q", cfg.PlatformAPIURL)
	}
}
