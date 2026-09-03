# streamoid (Go)

The Go port of the streamoid platform SDK. See the [repo root README](../README.md)
for the full picture (why this exists, the other language ports, versioning).

## Install

Go has no package registry — the module path *is* the repository URL, and
proxy.golang.org is a read-through cache of VCS rather than something you
publish to. Since this repo is private, the module is published from a
generated public mirror, [`streamoid-sdk-go`][mirror], and that is the path
consumers import:

```bash
go get github.com/Streamoid-Technologies/streamoid-sdk-go@v0.1.1
```

No `GOPRIVATE` and no credentials — it resolves through the public proxy like
any other dependency. The mirror is read-only; this directory is the source of
truth and `.github/workflows/sync-go-mirror.yml` publishes it. Cut a release by
tagging `go/vX.Y.Z` here, which creates the matching `vX.Y.Z` over there.

[mirror]: https://github.com/Streamoid-Technologies/streamoid-sdk-go

## Usage

```go
import "github.com/Streamoid-Technologies/streamoid-sdk-go"

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
