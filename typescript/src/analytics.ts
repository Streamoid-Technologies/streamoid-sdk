/**
 * L2 analytics — `capture()` fans one event to PostHog + the audit store.
 * Both sinks best-effort; revenue/intelligence events are emitted server-side.
 */

import { getConfig, type StreamoidConfig } from "./config";
import { buildEnvelope, type Actor, type EventEnvelope, type EventVerb, type Tenant } from "./envelope";
import { authHeaders, postJson } from "./transport";

export interface CaptureOpts {
  actor?: Actor;
  tenant?: Tenant;
  properties?: Record<string, unknown>;
  source?: string;
  correlationId?: string;
}

async function toPosthog(env: EventEnvelope, cfg: StreamoidConfig): Promise<void> {
  if (!cfg.posthogKey || !cfg.posthogHost) return;
  await postJson(
    `${cfg.posthogHost}/capture/`,
    {
      api_key: cfg.posthogKey,
      event: env.event,
      timestamp: env.ts,
      distinct_id: env.actor.user_id || "anonymous",
      properties: {
        ...env.properties,
        product: env.product,
        source: env.source,
        correlation_id: env.correlation_id,
        workspace_id: env.tenant.workspace_id,
        store_uuid: env.tenant.store_uuid,
        $lib: "streamoid-sdk",
      },
    },
    { timeoutMs: cfg.timeoutMs, label: "posthog" },
  );
}

async function toAudit(env: EventEnvelope, cfg: StreamoidConfig): Promise<void> {
  if (!cfg.auditEnabled || !cfg.platformApiUrl) return;
  await postJson(`${cfg.platformApiUrl}/api/v1/audit/events`, env, {
    headers: authHeaders(cfg.platformToken),
    timeoutMs: cfg.timeoutMs,
    label: "audit",
  });
}

export async function capture(
  event: EventVerb,
  opts: CaptureOpts = {},
): Promise<EventEnvelope> {
  const cfg = getConfig();
  const env = buildEnvelope(event, {
    product: cfg.product,
    source: opts.source ?? cfg.source,
    actor: opts.actor,
    tenant: opts.tenant,
    correlationId: opts.correlationId,
    properties: opts.properties,
  });
  await Promise.allSettled([toPosthog(env, cfg), toAudit(env, cfg)]);
  return env;
}

/** Fire-and-forget variant for hot paths that must not await telemetry. */
export function captureBg(event: EventVerb, opts: CaptureOpts = {}): void {
  void capture(event, opts).catch(() => {});
}
