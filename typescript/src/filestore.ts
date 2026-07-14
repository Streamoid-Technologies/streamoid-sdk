/**
 * L1 filestore — the single storage client/contract.
 *
 * `createFileStore` turns a low-level `putBytes` (the host's R2 client) into the
 * shared contract so every product writes objects the same way:
 *   - `putObject(key, data, ct)` — raw keyed write through the one client
 *     (behavior-preserving; used to route existing upload paths).
 *   - `put({ workspaceId, kind, ... })` — the full contract: a C6-namespaced key
 *     (`{product}/{workspace_id}/{kind}/{yyyy}/{mm}/{uuid}-{file}`), the bytes,
 *     the shared `files` index row (L3), and a `file_uploaded` event (L2) — all
 *     in one place. Best-effort index/event; never throws into the caller.
 */

import { capture, captureBg } from "./analytics";
import { getConfig } from "./config";
import { buildObjectKey, type StoredFile } from "./envelope";
import { recordFile } from "./filesIndex";

export type PutBytes = (
  key: string,
  data: Uint8Array,
  contentType?: string,
) => Promise<string>; // returns the public URL

export interface FileStorePutInput {
  data: Uint8Array;
  workspaceId: string;
  kind: string;
  filename?: string;
  contentType?: string;
  userId?: string;
  entityRef?: { type?: string; id?: string };
  /** await the file_uploaded event instead of fire-and-forget (default false) */
  awaitEvent?: boolean;
}

export interface FileStore {
  /** Raw keyed write through the single client (no index/event). */
  putObject(key: string, data: Uint8Array, contentType?: string): Promise<string>;
  /** Full contract: C6 key + bytes + files index (L3) + file_uploaded (L2). */
  put(input: FileStorePutInput): Promise<StoredFile>;
  /** Build a C6 key without writing (useful for presigned flows). */
  buildKey(opts: { workspaceId: string; kind: string; filename?: string }): string;
}

export function createFileStore(opts: {
  putBytes: PutBytes;
  product?: string;
  bucket?: () => string | undefined;
}): FileStore {
  const product = () => opts.product ?? getConfig().product;

  const buildKey: FileStore["buildKey"] = ({ workspaceId, kind, filename }) =>
    buildObjectKey({ product: product(), workspaceId, kind, filename });

  const putObject: FileStore["putObject"] = (key, data, contentType) =>
    opts.putBytes(key, data, contentType);

  const put: FileStore["put"] = async (input) => {
    const key = buildKey({
      workspaceId: input.workspaceId,
      kind: input.kind,
      filename: input.filename,
    });
    const publicUrl = await opts.putBytes(key, input.data, input.contentType);
    const stored: StoredFile = {
      key,
      bucket: opts.bucket?.(),
      public_url: publicUrl,
      size: input.data.length,
      content_type: input.contentType ?? null,
    };
    await recordFile(stored, {
      workspaceId: input.workspaceId,
      userId: input.userId,
      kind: input.kind,
      entityRef: input.entityRef,
    });
    const eventOpts = {
      actor: input.userId ? { user_id: input.userId } : undefined,
      tenant: { workspace_id: input.workspaceId },
      source: "backend",
      properties: {
        kind: input.kind,
        content_type: input.contentType,
        size: input.data.length,
        file_id: stored.file_id,
        bucket: stored.bucket,
      },
    } as const;
    if (input.awaitEvent) await capture("file_uploaded", eventOpts);
    else captureBg("file_uploaded", eventOpts);
    return stored;
  };

  return { putObject, put, buildKey };
}
