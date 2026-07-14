/**
 * Tests for the shared FileStore adapter (createFileStore). Bundled with
 * esbuild + run on node:test. `fetch` is stubbed so the index/event writes are
 * no-ops; we assert the C6 key, the single-client putBytes, and the StoredFile.
 */
import { test } from "node:test";
import assert from "node:assert/strict";

import { createFileStore } from "../src/filestore";
import { _resetConfigForTests } from "../src/config";

// No platform configured -> recordFile/captureBg short-circuit (no fetch).
function noPlatform() {
  for (const k of [
    "STREAMOID_PLATFORM_API_URL",
    "UNIFIED_API_BASE",
    "UNIFIED_BASE_URL",
    "STREAMOID_POSTHOG_KEY_REF",
  ]) {
    delete process.env[k];
  }
  process.env.STREAMOID_PRODUCT = "photogenix";
  _resetConfigForTests();
}

test("put() builds a C6 key, writes bytes once, returns StoredFile", async () => {
  noPlatform();
  const calls: { key: string; len: number; ct?: string }[] = [];
  const fs = createFileStore({
    product: "photogenix",
    bucket: () => "photogenix-exp",
    putBytes: async (key, data, ct) => {
      calls.push({ key, len: data.length, ct });
      return `https://cdn/${key}`;
    },
  });
  const data = new Uint8Array([1, 2, 3, 4]);
  const stored = await fs.put({
    data,
    workspaceId: "ws1",
    kind: "upload",
    filename: "My Shirt!.png",
    contentType: "image/png",
    userId: "u1",
  });
  assert.equal(calls.length, 1, "putBytes called exactly once");
  const key = calls[0].key;
  assert.match(key, /^ws1\/photogenix\/upload\/\d{4}\/\d{2}\/[0-9a-f]+-My-Shirt\.png$/);
  assert.equal(stored.key, key);
  assert.equal(stored.bucket, "photogenix-exp");
  assert.equal(stored.public_url, `https://cdn/${key}`);
  assert.equal(stored.size, 4);
  assert.equal(stored.content_type, "image/png");
});

test("putObject() is a raw keyed write through the single client", async () => {
  noPlatform();
  let used = "";
  const fs = createFileStore({
    putBytes: async (key) => {
      used = key;
      return `https://cdn/${key}`;
    },
  });
  const url = await fs.putObject("legacy-filename.png", new Uint8Array([0]), "image/png");
  assert.equal(used, "legacy-filename.png"); // key preserved verbatim
  assert.equal(url, "https://cdn/legacy-filename.png");
});

test("buildKey() yields a C6 key without writing", async () => {
  noPlatform();
  let wrote = false;
  const fs = createFileStore({ product: "photogenix", putBytes: async () => { wrote = true; return ""; } });
  const key = fs.buildKey({ workspaceId: "ws9", kind: "render-source", filename: "a.jpg" });
  assert.match(key, /^ws9\/photogenix\/render-source\/\d{4}\/\d{2}\/[0-9a-f]+-a\.jpg$/);
  assert.equal(wrote, false);
});
