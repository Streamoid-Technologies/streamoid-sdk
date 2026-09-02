# @streamoid/sdk

The TypeScript port of the streamoid platform SDK. See the [repo root README](../README.md)
for the full picture (why this exists, the other language ports, versioning).

## Install

```bash
bun add @streamoid/sdk    # or: npm i @streamoid/sdk
```

Published to the public npm registry, so there is nothing else to configure —
no scoped `.npmrc`, no `read:packages` token, no Docker build secret. It
installs like any other dependency, and like the sibling `@streamoid/ui`,
`@streamoid/icons`, `@streamoid/settings` and `@streamoid/tokens` packages.

> Previously published to GitHub Packages as `@streamoid-technologies/sdk`,
> which needed a token to install even though this repo is public. That name is
> retired; if you find a `@streamoid-technologies:registry` line in a
> consumer's `.npmrc`, it can be deleted.

## Usage

```typescript
import { capture, recordFile, registerMcpServer, createFileStore } from "@streamoid/sdk";

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
