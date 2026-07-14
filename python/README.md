# streamoid (Python)

The Python port of the streamoid platform SDK. See the [repo root README](../README.md)
for the full picture (why this exists, the other language ports, versioning).

## Install

GitHub Packages has no PyPI registry, so this is installed as a git-tag dependency:

```bash
uv add "streamoid @ git+https://github.com/Streamoid-Technologies/streamoid-sdk.git@python-v0.1.0#subdirectory=python"
```

## Usage

```python
from streamoid import capture, record_file, register_mcp_server, build_object_key, StoredFile

await capture(
    "file_uploaded",
    actor={"user_id": uid},
    tenant={"workspace_id": ws},
    properties={"kind": "file-upload", "size": size},
)
```

Every consuming service must set `STREAMOID_PRODUCT` in its environment — there is
no hardcoded per-service default (see `src/streamoid/config.py`'s module docstring
for why).

## Tests

```bash
uv run pytest
```
