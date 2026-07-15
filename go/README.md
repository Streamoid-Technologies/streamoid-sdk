# streamoid (Go)

The Go port of the streamoid platform SDK. See the [repo root README](../README.md)
for the full picture (why this exists, the other language ports, versioning).

## Install

Go modules resolve directly from git tags — no registry step needed:

```bash
go get github.com/Streamoid-Technologies/streamoid-sdk/go@go/v0.1.0
```

Since this is a private repo, your environment needs `GOPRIVATE=github.com/Streamoid-Technologies/*`
and git credentials configured for `github.com` (SSH key or an HTTPS credential helper).

## Usage

```go
import "github.com/Streamoid-Technologies/streamoid-sdk/go"

client := streamoid.NewClient(streamoid.LoadConfig(streamoid.LoadConfigOptions{
    DefaultProduct: "catalogix",
    DefaultSource:  "backend",
}), logger)

client.CaptureBg(ctx, "file_uploaded", streamoid.CaptureOptions{
    Tenant:     map[string]any{"workspace_id": ws},
    Properties: map[string]any{"kind": "file-upload", "size": size},
})
```

`streamoid.Bootstrap(ctx, opts, logger)` is a convenience wrapper that builds
the client and self-registers into the MCP registry (L4) in the background —
see `bootstrap.go`.

Unlike the Python/TS ports, `LoadConfigOptions` takes `DefaultProduct`/
`DefaultSource`/`FallbackPlatformAPIURL` explicitly from the caller rather
than hardcoding one product's defaults into the package — there was never a
per-copy vendored version of this port to generalize away from.

## Tests

```bash
go build ./... && go vet ./... && go test ./... && gofmt -l .
```
