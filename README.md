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
| [`python/`](./python/) | Python | `streamoid` | git-tag dependency (`uv add "streamoid @ git+https://github.com/Streamoid-Technologies/streamoid-sdk.git@python-vX.Y.Z#subdirectory=python"`) — resolved straight from GitHub, no registry and no token |
| [`typescript/`](./typescript/) | TypeScript | `@streamoid/sdk` | public npm registry (`bun add @streamoid/sdk`) — no token, no `.npmrc`, same as the `@streamoid/ui`/`icons`/`settings`/`tokens` packages |
| [`go/`](./go/) | Go | `github.com/Streamoid-Technologies/streamoid-sdk/go` | plain Go module resolution (`go get .../streamoid-sdk/go@go/vX.Y.Z`) — no registry needed |

## Versioning

Each language releases independently, since they move on separate schedules. A
change to one language does not require bumping the others.

- **TypeScript** ships on merge: bump `version` in `typescript/package.json` in
  the PR, and when it lands on `develop` (the trunk — or on `main`, for a
  hotfix), [`publish-typescript.yml`](.github/workflows/publish-typescript.yml)
  publishes it because that version is not yet on the registry. Merging a PR
  that touches no version is a no-op, and re-running publishes nothing. Never
  publish by hand — a hand publish can ship a tree that exists in no commit,
  which is exactly how a stale `@streamoid/agent` bundle once reached the
  registry.
- **Python and Go** are tagged: `python-vX.Y.Z` and `go/vX.Y.Z` (the `go/`
  prefix is required by Go's nested-module tagging convention — see
  https://go.dev/ref/mod#vcs-version). Consumers pin the tag directly, so the
  tag *is* the release.

The old `ts-vX.Y.Z` tags are historical: `@streamoid-technologies/sdk@0.1.0` on
GitHub Packages was the last release under that name and that registry, and
nothing consumes it. It is superseded by `@streamoid/sdk` on npm.

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
