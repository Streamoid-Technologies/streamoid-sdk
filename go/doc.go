// Package streamoid is the Go port of the streamoid platform SDK (L1),
// alongside the Python and TypeScript ports, published from
// https://github.com/Streamoid-Technologies/streamoid-sdk. It implements
// the same shared contracts from the standard architecture:
//
//   - the event envelope (L2) + the governed v1 verb list
//   - the `files` index + `audit_events` writes (L3), against unified-backend
//   - the C6 object-key scheme (workspace-first: {workspace_id}/{product}/{kind}/{yyyy}/{mm}/{uuid}-{file})
//   - secrets-by-reference (C3): a `secret_ref` of `env://VAR` resolves VAR from
//     the environment; `vault://...` falls back to an env var named after the
//     last path segment until a real secret backend exists
//   - MCP self-registration (L4)
//
// Go modules resolve directly from git tags (`go/vX.Y.Z`), so unlike the
// Python/TS ports there's no package-registry step -- any Go module can
// depend on this one with a plain `go get`:
//
//	import "github.com/Streamoid-Technologies/streamoid-sdk-go"
//
// This package is self-contained (stdlib net/http only, no dependency on any
// host application's HTTP client conventions) so it drops into any Go module
// unchanged.
//
// All platform writes are best-effort: a telemetry/audit failure must never
// break the caller's actual request. Capture, RecordFile, and
// RegisterMCPServer all swallow transport failures (logging via the supplied
// slog.Logger, or the default logger if none is given) and return nil in
// those cases; the one exception is a contract violation (an unknown event
// verb), which is a programming error and is returned as an error from
// BuildEnvelope/Capture rather than silently dropped — matching the
// Python/TS SDKs exactly.
package streamoid
