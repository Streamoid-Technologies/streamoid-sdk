/**
 * L3 `files` index client — one metadata row per uploaded object.
 * Called after a successful R2 put(); best-effort.
 */

import { getConfig } from "./config";
import type { StoredFile } from "./envelope";
import { authHeaders, postJson } from "./transport";

export async function recordFile(
  stored: StoredFile,
  opts: {
    workspaceId: string;
    userId?: string;
    kind: string;
    entityRef?: { type?: string; id?: string };
  },
): Promise<string | undefined> {
  const cfg = getConfig();
  if (!cfg.filesEnabled || !cfg.platformApiUrl) return undefined;
  const resp = await postJson(
    `${cfg.platformApiUrl}/api/v1/files`,
    {
      workspace_id: opts.workspaceId,
      user_id: opts.userId,
      product: cfg.product,
      kind: opts.kind,
      key: stored.key,
      bucket: stored.bucket,
      public_url: stored.public_url,
      content_type: stored.content_type,
      size: stored.size,
      checksum: stored.checksum,
      entity_ref: opts.entityRef,
      created_at: stored.created_at,
    },
    { headers: authHeaders(cfg.platformToken), timeoutMs: cfg.timeoutMs, label: "files" },
  );
  if (!resp) return undefined;
  try {
    const body: any = await resp.json();
    const row = body?.data ?? body;
    const id = row?.file_id ?? row?._id;
    if (id) stored.file_id = String(id);
    return stored.file_id;
  } catch {
    return undefined;
  }
}
