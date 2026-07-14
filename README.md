# streamoid-sdk

The centralized streamoid platform SDK: the shared L1 contracts (event
envelope, C6 object-key scheme, C3 secrets-by-reference, best-effort
analytics/audit `Capture`, files-index `RecordFile`, MCP self-registration)
that every Streamoid product emits/consumes on top of `unified-backend`.

Previously this was vendored verbatim into every consuming service (16
Python copies, 2 TypeScript copies) plus one already-centralized Go copy
inside catalogix's `backend/` module. This repo is the promotion described in
the platform architecture's L1 note ("start vendored, promote to a
`platform/` shared lib") — one canonical implementation per language, each
independently versioned and consumed as a real dependency instead of
copy-pasted source.

## Layout

Three independently-released language ports, each a superset-compatible
reimplementation of the same contracts (not bindings around one core):

| Dir | Language | Package | Consumed via |
|---|---|---|---|
| [`python/`](./python/) | Python | `streamoid` | git-tag dependency (`uv add "streamoid @ git+https://github.com/Streamoid-Technologies/streamoid-sdk.git@python-vX.Y.Z#subdirectory=python"`) — GitHub Packages has no PyPI registry, so this is a plain git dependency pinned to a tag |
| [`typescript/`](./typescript/) | TypeScript | `@streamoid-technologies/sdk` | GitHub Packages npm registry |
| [`go/`](./go/) | Go | `github.com/Streamoid-Technologies/streamoid-sdk/go` | plain Go module resolution (`go get .../streamoid-sdk/go@go/vX.Y.Z`) — no registry needed |

## Versioning

Each language is tagged independently since they release on separate
schedules: `python-vX.Y.Z`, `ts-vX.Y.Z`, `go/vX.Y.Z` (the `go/` prefix is
required by Go's nested-module tagging convention — see
https://go.dev/ref/mod#vcs-version). A change to one language does not
require bumping the others.

## Why not one shared implementation?

Each language port is a natural reimplementation of the same contracts in
that language's idioms (e.g. Go's `Bootstrap`/`FileStore` wrapper has no
Python/TS equivalent yet; Python/TS share async conventions Go doesn't have).
Keeping the contracts (event verbs, C6 key format, C3 secret-ref resolution)
identical across languages matters far more than sharing implementation code,
which isn't possible across Python/TS/Go anyway.

## Consumers

- **catalogix** (`Streamoid-Technologies/catalogix`): all three languages —
  ~15 Python services, the dashboard BFF (TS), and `backend/` (Go, single
  module, imports this directly rather than vendoring).
- **artifax** (`Streamoid-Technologies/artifax`): Python (`ai-services`).
- **photogenix** (`Streamoid-Technologies/photogenix_v2`): TypeScript
  (`packages/shared`, re-exported to the rest of the photogenix workspace).
