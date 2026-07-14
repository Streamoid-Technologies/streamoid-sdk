# @streamoid-technologies/sdk

The TypeScript port of the streamoid platform SDK. See the [repo root README](../README.md)
for the full picture (why this exists, the other language ports, versioning).

## Install

Published to GitHub Packages' npm registry. Add to your `.npmrc`:

```
@streamoid-technologies:registry=https://npm.pkg.github.com
```

Then install normally:

```bash
bun add @streamoid-technologies/sdk
```

(Requires a `read:packages`-scoped token available to npm/bun during install —
see your package manager's docs for GitHub Packages auth, e.g. `NODE_AUTH_TOKEN`.)

## Usage

```typescript
import { capture, recordFile, registerMcpServer, createFileStore } from "@streamoid-technologies/sdk";

await capture("file_uploaded", {
  actor: { user_id: uid },
  tenant: { workspace_id: ws },
  properties: { kind: "file-upload", size },
});
```

Every consuming service must set `STREAMOID_PRODUCT` in its environment — there is
no hardcoded per-service default (see `src/config.ts`'s module docstring for why).

## Build / test

```bash
bun install
bun run type-check
bun run test
bun run build
```
