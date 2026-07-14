/**
 * streamoid — the platform SDK (L1).
 *
 * Implements the shared platform contracts from docs/plan/architecture.md
 * (L2 event envelope, L3 files index + audit_events, L4 MCP self-registration).
 * Dependency-free (uses global fetch); reads its own STREAMOID_* env. Published
 * from https://github.com/Streamoid-Technologies/streamoid-sdk as
 * `@streamoid-technologies/sdk` and consumed by photogenix and the catalogix
 * dashboard BFF -- this used to be a folder vendored verbatim into each one.
 * Nothing here throws into the caller's hot path — platform writes are
 * best-effort and log-on-failure.
 */

export * from "./envelope";
export * from "./config";
export { tenancyFromReq, emitEvent } from "./events";
export { capture, captureBg, type CaptureOpts } from "./analytics";
export { recordFile } from "./filesIndex";
export { registerMcpServer, type McpRegistration } from "./mcp";
export {
  createFileStore,
  type FileStore,
  type FileStorePutInput,
  type PutBytes,
} from "./filestore";
