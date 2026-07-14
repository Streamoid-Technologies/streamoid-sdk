/**
 * Canonical event envelope, verbs, file key scheme, and StoredFile (L2 + C6).
 *
 * The TS counterpart of the streamoid Python/Go SDK ports — the same
 * cross-product contracts from docs/plan/architecture.md, so events and file
 * keys are identical across products.
 */

// L2 — governed v1 verb vocabulary. Emitting an unknown verb is a programming
// error, surfaced loudly rather than shipping a typo to analytics.
export const EVENT_VERBS = [
  "file_uploaded",
  "search_performed",
  "generation_started",
  "generation_completed",
  "automation_started",
  "automation_completed",
  "export_performed",
  "user_invited",
  "settings_changed",
  "page_viewed",
  "item_viewed",
] as const;

export type EventVerb = (typeof EVENT_VERBS)[number];

export interface Actor {
  user_id?: string;
  email?: string;
  ip?: string;
}

export interface Tenant {
  workspace_id?: string;
  store_uuid?: string;
}

export interface EventEnvelope {
  event: EventVerb;
  ts: string;
  actor: Actor;
  tenant: Tenant;
  product: string;
  source: string;
  correlation_id: string;
  properties: Record<string, unknown>;
}

export interface StoredFile {
  key: string;
  bucket?: string;
  public_url?: string;
  size?: number;
  content_type?: string | null;
  checksum?: string;
  created_at?: string;
  file_id?: string;
}

function randomId(): string {
  // crypto.randomUUID is available on Node 18+ / modern browsers.
  try {
    return globalThis.crypto.randomUUID();
  } catch {
    return `${Date.now().toString(16)}-${Math.random().toString(16).slice(2)}`;
  }
}

export function buildEnvelope(
  event: EventVerb,
  opts: {
    product: string;
    source: string;
    actor?: Actor;
    tenant?: Tenant;
    correlationId?: string;
    properties?: Record<string, unknown>;
    ts?: string;
  },
): EventEnvelope {
  if (!EVENT_VERBS.includes(event)) {
    throw new Error(
      `unknown event verb ${event}; allowed: ${EVENT_VERBS.join(", ")}`,
    );
  }
  return {
    event,
    ts: opts.ts ?? new Date().toISOString(),
    actor: opts.actor ?? {},
    tenant: opts.tenant ?? {},
    product: opts.product,
    source: opts.source,
    correlation_id: opts.correlationId ?? randomId(),
    properties: opts.properties ?? {},
  };
}

const SAFE_FILENAME = /[^A-Za-z0-9._-]+/g;

function safeFilename(filename?: string): string {
  const base = (filename ?? "").split(/[\\/]/).pop()?.trim() || "file";
  return base.replace(SAFE_FILENAME, "-").replace(/-+\./g, ".").replace(/^[-.]+|[-.]+$/g, "") || "file";
}

/**
 * C6 key scheme: `{workspace_id}/{product}/{kind}/{yyyy}/{mm}/{uuid}-{file}`.
 * Workspace-first: the shared bucket is organized per workspace (customer), with
 * `product` as a sub-dir so you can tell which product produced a file.
 */
export function buildObjectKey(opts: {
  product: string;
  workspaceId: string;
  kind: string;
  filename?: string;
  now?: Date;
}): string {
  const now = opts.now ?? new Date();
  const yyyy = String(now.getUTCFullYear());
  const mm = String(now.getUTCMonth() + 1).padStart(2, "0");
  const ws = (opts.workspaceId || "unscoped").trim() || "unscoped";
  return `${ws}/${opts.product}/${opts.kind}/${yyyy}/${mm}/${randomId().replace(/-/g, "")}-${safeFilename(opts.filename)}`;
}
