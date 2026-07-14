/**
 * L2 event helpers for the catalogix dashboard BFF.
 *
 * Thin glue that turns an Express request into the canonical envelope's actor +
 * tenant (architecture.md §L2) and fires a best-effort, fire-and-forget event
 * via the SDK. Catalogix identifies the tenant by `store_uuid` (the Mongo
 * `storeId` in most routes) and, where present, `workspace_id`; the actor is
 * `req.user` populated by the `authorize` middleware. Emission never throws into
 * the request — telemetry is allowed to be lossy.
 */

import { captureBg, type CaptureOpts } from "./analytics";
import type { EventVerb } from "./envelope";

// Loosely typed so this module stays erasable and free of an express import.
interface ReqLike {
  user?: { _id?: unknown; email?: string };
  params?: Record<string, string | undefined>;
  query?: Record<string, unknown>;
  headers?: Record<string, unknown>;
}

/** Derive { actor, tenant } from the authenticated request. */
export function tenancyFromReq(req: ReqLike): Pick<CaptureOpts, "actor" | "tenant"> {
  const user = req?.user || {};
  const params = req?.params || {};
  const headers = (req?.headers || {}) as Record<string, string | undefined>;
  const storeUuid =
    params.storeId ||
    params.store_uuid ||
    (headers["x-store-uuid"] as string | undefined) ||
    undefined;
  const workspaceId =
    params.workspaceId ||
    (headers["x-workspace-id"] as string | undefined) ||
    undefined;
  return {
    actor: {
      user_id: user._id != null ? String(user._id) : undefined,
      email: user.email,
    },
    tenant: {
      workspace_id: workspaceId,
      store_uuid: storeUuid,
    },
  };
}

/**
 * Fire-and-forget an L2 event scoped to the request's actor/tenant. Extra
 * `properties` are merged into the envelope. Correlation id can be supplied
 * (catalogix threads `request_group_id`); otherwise the SDK generates one.
 */
export function emitEvent(
  event: EventVerb,
  req: ReqLike,
  properties: Record<string, unknown> = {},
  opts: { correlationId?: string; source?: string } = {},
): void {
  const { actor, tenant } = tenancyFromReq(req);
  captureBg(event, {
    actor,
    tenant,
    properties,
    source: opts.source ?? "backend",
    correlationId: opts.correlationId,
  });
}
