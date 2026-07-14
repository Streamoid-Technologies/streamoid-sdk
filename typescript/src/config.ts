/**
 * Self-contained config for the streamoid TS SDK.
 *
 * Reads the canonical `STREAMOID_*` env set (C6). Secrets are resolved by
 * reference (C3): `env://VAR` reads an env var; a bare value is dev-only.
 *
 * Every consuming service is expected to set `STREAMOID_PRODUCT` explicitly
 * (there is no per-copy hardcoded default here, unlike the old vendored
 * copies) and `STREAMOID_PLATFORM_API_URL` (map your own pre-existing
 * unified-backend var to it in your own config, one line, rather than baking
 * product-specific names into this shared package).
 */

export interface StreamoidConfig {
  product: string;
  source: string;
  environment: string;
  platformApiUrl?: string;
  platformToken?: string;
  filesEnabled: boolean;
  auditEnabled: boolean;
  posthogHost?: string;
  posthogKey?: string;
  mcpRegistryUrl?: string;
  timeoutMs: number;
}

export function resolveSecret(ref?: string): string | undefined {
  if (!ref) return undefined;
  const r = ref.trim();
  if (r.startsWith("env://")) return process.env[r.slice("env://".length)] || undefined;
  if (r.startsWith("vault://")) {
    // No secret backend yet — fall back to an env var named after the segment.
    const v = r.split("/").pop()!.toUpperCase().replace(/-/g, "_");
    return process.env[v] || undefined;
  }
  return r;
}

function truthy(v: string | undefined, dflt: boolean): boolean {
  if (v === undefined) return dflt;
  return ["1", "true", "yes", "on"].includes(v.trim().toLowerCase());
}

let cached: StreamoidConfig | undefined;

export function getConfig(): StreamoidConfig {
  if (cached) return cached;
  const env = process.env;
  const platformApi = (env.STREAMOID_PLATFORM_API_URL || "").replace(/\/$/, "");
  cached = {
    product: env.STREAMOID_PRODUCT || "",
    source: env.STREAMOID_SOURCE || "backend",
    environment: env.STREAMOID_ENV || env.NODE_ENV || "development",
    platformApiUrl: platformApi || undefined,
    platformToken: resolveSecret(env.STREAMOID_PLATFORM_TOKEN_REF),
    filesEnabled: truthy(env.STREAMOID_FILES_ENABLED, true),
    auditEnabled: truthy(env.STREAMOID_AUDIT_ENABLED, true),
    posthogHost: (env.STREAMOID_POSTHOG_HOST || "https://ph.streamoid.com").replace(/\/$/, ""),
    posthogKey: resolveSecret(env.STREAMOID_POSTHOG_KEY_REF),
    mcpRegistryUrl: (env.STREAMOID_MCP_REGISTRY_URL || platformApi || "").replace(/\/$/, "") || undefined,
    timeoutMs: Number(env.STREAMOID_HTTP_TIMEOUT_MS || "5000"),
  };
  return cached;
}

/** Test hook to reset the memoized config. */
export function _resetConfigForTests(): void {
  cached = undefined;
}
