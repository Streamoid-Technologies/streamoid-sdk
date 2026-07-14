"""streamoid — the platform SDK (L1).

A small, dependency-light client library that implements the shared platform
contracts from the standard architecture (``docs/plan/architecture.md`` in the
``streamoid-os`` meta-repo): the event envelope (L2), the ``files`` index and
``audit_events`` projection (L3), and MCP self-registration (L4).

It is intentionally **self-contained** — it reads its own ``STREAMOID_*``
configuration from the environment and depends only on ``httpx``. Published
from https://github.com/Streamoid-Technologies/streamoid-sdk (git-tag
dependency; see that repo's README for why not a PyPI package) and consumed
by every Python service across artifax and catalogix — this used to be a
folder vendored verbatim into each one; it is now a single real dependency.

Nothing in here raises into the caller's hot path: analytics, audit, and
file-index writes are best-effort and log-on-failure, because emitting an event
must never break a user request (architecture rule: frontend-only events may be
lossy; revenue/intelligence events are emitted server-side but still must not
take the request down).
"""

from .config import StreamoidConfig, get_config
from .envelope import (
    EVENT_VERBS,
    StoredFile,
    build_envelope,
    build_object_key,
)
from .analytics import Analytics, capture
from .audit import write_audit_event
from .files_index import record_file
from .mcp import register_mcp_server

__all__ = [
    "StreamoidConfig",
    "get_config",
    "EVENT_VERBS",
    "StoredFile",
    "build_envelope",
    "build_object_key",
    "Analytics",
    "capture",
    "write_audit_event",
    "record_file",
    "register_mcp_server",
]
