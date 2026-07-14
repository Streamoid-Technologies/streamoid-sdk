"""Canonical event envelope, event verbs, file key scheme, and StoredFile.

These are the cross-product contracts from ``architecture.md`` (L2 envelope,
C6 naming). Keeping them in one tiny module makes them trivial to keep
identical across every service that vendors this package.
"""

from __future__ import annotations

import datetime
import os
import re
import uuid
from dataclasses import dataclass, field
from typing import Any, Dict, Optional

# L2 — the governed v1 verb vocabulary. Event names are snake_case verbs and are
# CI-linted elsewhere to prevent drift; emitting an unknown verb is a programming
# error, so we surface it loudly rather than silently shipping a typo.
EVENT_VERBS = frozenset(
    {
        "file_uploaded",
        "search_performed",
        "generation_started",
        "generation_completed",
        "automation_started",
        "automation_completed",
        "export_performed",
        "user_invited",
        "settings_changed",
        "page_viewed",
        "item_viewed",
    }
)


def _utc_now_iso() -> str:
    return datetime.datetime.now(datetime.timezone.utc).isoformat()


def build_envelope(
    event: str,
    *,
    product: str,
    source: str,
    actor: Optional[Dict[str, Any]] = None,
    tenant: Optional[Dict[str, Any]] = None,
    correlation_id: Optional[str] = None,
    properties: Optional[Dict[str, Any]] = None,
    ts: Optional[str] = None,
) -> Dict[str, Any]:
    """Build the canonical envelope every product emits (architecture.md §L2).

    ``actor`` should carry ``user_id`` (+ optional ``email``/``ip``); ``tenant``
    carries ``workspace_id`` (+ optional ``store_uuid``). We do not raise on a
    missing tenant here — some legitimate events (e.g. ``page_viewed`` before
    login) have no workspace — but callers emitting revenue/intelligence events
    are expected to supply one.
    """
    if event not in EVENT_VERBS:
        raise ValueError(
            f"unknown event verb {event!r}; allowed verbs: {sorted(EVENT_VERBS)}"
        )

    return {
        "event": event,
        "ts": ts or _utc_now_iso(),
        "actor": actor or {},
        "tenant": tenant or {},
        "product": product,
        "source": source,
        "correlation_id": correlation_id or str(uuid.uuid4()),
        "properties": properties or {},
    }


_SAFE_FILENAME_RE = re.compile(r"[^A-Za-z0-9._-]+")


def _safe_filename(filename: Optional[str]) -> str:
    base = os.path.basename(filename or "").strip() or "file"
    # Collapse anything unsafe to a single dash; keep the extension readable.
    safe = _SAFE_FILENAME_RE.sub("-", base).strip("-.")
    # Tidy a dash left immediately before the extension dot ("a-.png" -> "a.png").
    safe = re.sub(r"-+\.", ".", safe)
    return safe or "file"


def build_object_key(
    *,
    product: str,
    workspace_id: str,
    kind: str,
    filename: Optional[str] = None,
    now: Optional[datetime.datetime] = None,
) -> str:
    """C6 storage key scheme: ``{workspace_id}/{product}/{kind}/{yyyy}/{mm}/{uuid}-{file}``.

    **Workspace-first**: the shared bucket is organized per workspace (customer),
    with ``product`` as a sub-dir so you can tell which product produced a file.
    Time-partitioned and collision-free. ``workspace_id`` falls back to
    ``unscoped`` rather than producing a key with an empty leading segment.
    """
    stamp = now or datetime.datetime.now(datetime.timezone.utc)
    ws = (workspace_id or "unscoped").strip() or "unscoped"
    return (
        f"{ws}/{product}/{kind}/"
        f"{stamp:%Y}/{stamp:%m}/"
        f"{uuid.uuid4().hex}-{_safe_filename(filename)}"
    )


@dataclass
class StoredFile:
    """The uniform result of a filestore ``put()`` (architecture.md §L1)."""

    key: str
    bucket: str
    public_url: str
    size: int = 0
    content_type: Optional[str] = None
    checksum: Optional[str] = None
    created_at: str = field(default_factory=_utc_now_iso)
    # populated once the central files index row is written (L3)
    file_id: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        return {
            "key": self.key,
            "bucket": self.bucket,
            "public_url": self.public_url,
            "size": self.size,
            "content_type": self.content_type,
            "checksum": self.checksum,
            "created_at": self.created_at,
            "file_id": self.file_id,
        }
