/**
 * Tests for the vendored streamoid TS SDK. Bundled with esbuild and run on
 * node:test (the repo has no installed test deps in this environment). Mirrors
 * the artifax Python suite: pure contracts + best-effort clients with `fetch`
 * stubbed.
 */
import { test } from "node:test";
import assert from "node:assert/strict";

import { buildEnvelope, buildObjectKey, EVENT_VERBS, type StoredFile } from "../src/envelope";
import { resolveSecret, getConfig, _resetConfigForTests } from "../src/config";
import { capture } from "../src/analytics";
import { recordFile } from "../src/filesIndex";
import { registerMcpServer } from "../src/mcp";

// ── fetch stub ─────────────────────────────────────────────────────────────
type Captured = { url: string; body: any; headers: any };
function stubFetch(jsonBody: unknown = {}): Captured[] {
  const calls: Captured[] = [];
  (globalThis as any).fetch = async (url: string, init: any) => {
    calls.push({
      url,
      body: init?.body ? JSON.parse(init.body) : undefined,
      headers: init?.headers,
    });
    return {
      ok: true,
      status: 200,
      json: async () => jsonBody,
    } as any;
  };
  return calls;
}

function setEnv(env: Record<string, string | undefined>) {
  for (const [k, v] of Object.entries(env)) {
    if (v === undefined) delete process.env[k];
    else process.env[k] = v;
  }
  _resetConfigForTests();
}

// ── envelope ─────────────────────────────────────────────────────────────
test("buildEnvelope happy path", () => {
  const e = buildEnvelope("generation_started", {
    product: "photogenix",
    source: "backend",
    actor: { user_id: "u1" },
    tenant: { workspace_id: "ws1" },
    properties: { model: "flux" },
  });
  assert.equal(e.event, "generation_started");
  assert.equal(e.product, "photogenix");
  assert.deepEqual(e.actor, { user_id: "u1" });
  assert.ok(e.correlation_id);
});

test("buildEnvelope rejects unknown verb", () => {
  assert.throws(() => buildEnvelope("frobnicate" as any, { product: "p", source: "s" }));
});

test("buildEnvelope honours correlationId override", () => {
  const e = buildEnvelope("search_performed", { product: "p", source: "s", correlationId: "c-1" });
  assert.equal(e.correlation_id, "c-1");
});

test("v1 verbs present", () => {
  for (const v of ["file_uploaded", "generation_started", "generation_completed", "export_performed"]) {
    assert.ok(EVENT_VERBS.includes(v as any));
  }
});

// ── key scheme (C6) ────────────────────────────────────────────────────────
test("buildObjectKey format + sanitization", () => {
  const k = buildObjectKey({
    product: "photogenix",
    workspaceId: "ws1",
    kind: "render-source",
    filename: "My Shirt!.png",
  });
  const parts = k.split("/");
  // workspace-first: {workspace_id}/{product}/{kind}/...
  assert.equal(parts[0], "ws1");
  assert.equal(parts[1], "photogenix");
  assert.equal(parts[2], "render-source");
  assert.match(parts[3], /^\d{4}$/);
  assert.equal(parts[4].length, 2);
  assert.ok(k.endsWith("-My-Shirt.png"));
  assert.ok(!k.includes("!") && !k.includes(" "));
});

test("buildObjectKey unscoped fallback + uniqueness", () => {
  const a = buildObjectKey({ product: "p", workspaceId: "", kind: "asset" });
  assert.ok(a.startsWith("unscoped/p/asset/"));
  const b = buildObjectKey({ product: "p", workspaceId: "w", kind: "k", filename: "x.png" });
  const c = buildObjectKey({ product: "p", workspaceId: "w", kind: "k", filename: "x.png" });
  assert.notEqual(b, c);
});

// ── config / secrets (C3) ────────────────────────────────────────────────────
test("resolveSecret variants", () => {
  process.env.MY_TOKEN = "s3cr3t";
  assert.equal(resolveSecret("env://MY_TOKEN"), "s3cr3t");
  assert.equal(resolveSecret("env://NOPE"), undefined);
  assert.equal(resolveSecret("bare"), "bare");
  assert.equal(resolveSecret(undefined), undefined);
});

test("getConfig has no hardcoded product default, and overrides work", () => {
  setEnv({
    STREAMOID_PRODUCT: undefined,
    STREAMOID_PLATFORM_API_URL: undefined,
    STREAMOID_FILES_ENABLED: undefined,
  });
  let cfg = getConfig();
  // No per-copy hardcoded default (unlike the old vendored copies) -- every
  // consumer must set STREAMOID_PRODUCT explicitly. See config.ts's docstring.
  assert.equal(cfg.product, "");
  assert.equal(cfg.platformApiUrl, undefined);
  assert.equal(cfg.filesEnabled, true);

  setEnv({
    STREAMOID_PRODUCT: "artifax",
    STREAMOID_PLATFORM_API_URL: "https://unified.test/",
    STREAMOID_FILES_ENABLED: "false",
  });
  cfg = getConfig();
  assert.equal(cfg.product, "artifax");
  assert.equal(cfg.platformApiUrl, "https://unified.test");
  assert.equal(cfg.filesEnabled, false);
  setEnv({ STREAMOID_PRODUCT: undefined, STREAMOID_FILES_ENABLED: undefined });
});

// ── analytics fan-out (L2) ───────────────────────────────────────────────────
test("capture fans out to PostHog + audit", async () => {
  setEnv({
    STREAMOID_PLATFORM_API_URL: "https://platform.test",
    STREAMOID_POSTHOG_KEY_REF: "ph-key",
    STREAMOID_POSTHOG_HOST: "https://ph.test",
    STREAMOID_PLATFORM_TOKEN_REF: "svc",
  });
  const calls = stubFetch({});
  const env = await capture("generation_completed", {
    actor: { user_id: "u1" },
    tenant: { workspace_id: "ws1" },
    properties: { n: 2 },
  });
  const urls = calls.map((c) => c.url);
  assert.ok(urls.some((u) => u.includes("/capture/")));
  assert.ok(urls.some((u) => u.includes("/api/v1/audit/events")));
  const ph = calls.find((c) => c.url.includes("/capture/"))!;
  assert.equal(ph.body.event, "generation_completed");
  assert.equal(ph.body.distinct_id, "u1");
  assert.equal(ph.body.properties.workspace_id, "ws1");
  assert.equal(env.event, "generation_completed");
});

test("capture never throws on fetch failure", async () => {
  setEnv({ STREAMOID_PLATFORM_API_URL: "https://platform.test", STREAMOID_POSTHOG_KEY_REF: "k" });
  (globalThis as any).fetch = async () => {
    throw new Error("network down");
  };
  const env = await capture("file_uploaded");
  assert.equal(env.event, "file_uploaded");
});

test("capture skips PostHog without key", async () => {
  setEnv({ STREAMOID_PLATFORM_API_URL: "https://platform.test", STREAMOID_POSTHOG_KEY_REF: undefined });
  const calls = stubFetch({});
  await capture("page_viewed");
  assert.ok(!calls.some((c) => c.url.includes("/capture/")));
});

// ── files index (L3) ─────────────────────────────────────────────────────────
test("recordFile returns file_id and sets it", async () => {
  setEnv({
    STREAMOID_PRODUCT: "photogenix",
    STREAMOID_PLATFORM_API_URL: "https://platform.test",
    STREAMOID_FILES_ENABLED: undefined,
  });
  const calls = stubFetch({ data: { file_id: "file-789" } });
  const stored: StoredFile = { key: "photogenix/ws1/upload/2026/06/abc-x.png", bucket: "b", public_url: "u", size: 5 };
  const fid = await recordFile(stored, { workspaceId: "ws1", userId: "u1", kind: "upload" });
  assert.equal(fid, "file-789");
  assert.equal(stored.file_id, "file-789");
  const call = calls.find((c) => c.url.endsWith("/api/v1/files"))!;
  assert.equal(call.body.workspace_id, "ws1");
  assert.equal(call.body.product, "photogenix");
  assert.equal(call.body.key, stored.key);
});

test("recordFile disabled does not call out", async () => {
  setEnv({ STREAMOID_PLATFORM_API_URL: "https://platform.test", STREAMOID_FILES_ENABLED: "false" });
  const calls = stubFetch({});
  const fid = await recordFile({ key: "k" }, { workspaceId: "ws1", kind: "upload" });
  assert.equal(fid, undefined);
  assert.equal(calls.length, 0);
  setEnv({ STREAMOID_FILES_ENABLED: undefined });
});

// ── MCP registration (L4) ────────────────────────────────────────────────────
test("registerMcpServer posts expected payload", async () => {
  setEnv({ STREAMOID_MCP_REGISTRY_URL: "https://platform.test", STREAMOID_PLATFORM_API_URL: "https://platform.test" });
  const calls = stubFetch({ data: { server_id: "photogenix-renders" } });
  const ok = await registerMcpServer({
    serverId: "photogenix-renders",
    owner: "photogenix",
    displayName: "Photogenix Renders",
    url: "https://pg.test/api/v4/mcp",
    healthUrl: "https://pg.test/api/v4/mcp/health",
    authSecretRef: "vault://photogenix-mcp-token",
    toolSummary: [{ name: "photogenix_list_renders" }],
  });
  assert.equal(ok, true);
  const call = calls.find((c) => c.url.endsWith("/api/v1/mcp/servers"))!;
  assert.equal(call.body.server_id, "photogenix-renders");
  assert.equal(call.body.owner, "photogenix");
  assert.equal(call.body.auth.secret_ref, "vault://photogenix-mcp-token");
});

test("registerMcpServer skips without registry url", async () => {
  setEnv({
    STREAMOID_MCP_REGISTRY_URL: undefined,
    STREAMOID_PLATFORM_API_URL: undefined,
    UNIFIED_API_BASE: undefined,
    UNIFIED_BASE_URL: undefined,
  });
  const ok = await registerMcpServer({ serverId: "x", owner: "o", url: "u" });
  assert.equal(ok, false);
});
