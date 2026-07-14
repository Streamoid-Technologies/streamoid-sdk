"""L2 analytics — ``capture()`` that fans one event out to PostHog + audit.

This is the single entry point products call to emit an event. It builds the
canonical envelope once and fans out to:

  * **PostHog** (product analytics, ``ph.streamoid.com``) — for dashboards.
  * **audit_events** (L3, via :mod:`streamoid.audit`) — the durable projection.

Both sinks are best-effort. Revenue/intelligence-critical events should be
emitted from the **server** (this module) rather than the browser so they are
not lossy.
"""

from __future__ import annotations

import asyncio
import logging
from typing import Any, Dict, Optional

from .audit import write_audit_event
from .config import StreamoidConfig, get_config
from .envelope import build_envelope
from .transport import post_json

logger = logging.getLogger(__name__)


class Analytics:
    """Thin client bound to a resolved config (handy for tests / DI)."""

    def __init__(self, config: Optional[StreamoidConfig] = None):
        self._config = config or get_config()

    async def capture(
        self,
        event: str,
        *,
        actor: Optional[Dict[str, Any]] = None,
        tenant: Optional[Dict[str, Any]] = None,
        properties: Optional[Dict[str, Any]] = None,
        source: Optional[str] = None,
        correlation_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        cfg = self._config
        envelope = build_envelope(
            event,
            product=cfg.product,
            source=source or cfg.source,
            actor=actor,
            tenant=tenant,
            correlation_id=correlation_id,
            properties=properties,
        )
        # Fan out concurrently; neither sink can break the caller.
        await asyncio.gather(
            self._to_posthog(envelope),
            write_audit_event(envelope, config=cfg),
            return_exceptions=True,
        )
        return envelope

    async def _to_posthog(self, envelope: Dict[str, Any]) -> bool:
        cfg = self._config
        if not cfg.posthog_key or not cfg.posthog_host:
            return False
        actor = envelope.get("actor") or {}
        tenant = envelope.get("tenant") or {}
        # PostHog capture API: distinct_id + event + properties.
        payload = {
            "api_key": cfg.posthog_key,
            "event": envelope["event"],
            "timestamp": envelope["ts"],
            "distinct_id": actor.get("user_id") or "anonymous",
            "properties": {
                **(envelope.get("properties") or {}),
                "product": envelope["product"],
                "source": envelope["source"],
                "correlation_id": envelope["correlation_id"],
                "workspace_id": tenant.get("workspace_id"),
                "store_uuid": tenant.get("store_uuid"),
                "$lib": "streamoid-sdk",
            },
        }
        resp = await post_json(
            f"{cfg.posthog_host}/capture/",
            payload,
            timeout_s=cfg.timeout_s,
            label="posthog",
        )
        return resp is not None


# Module-level convenience so call sites can `from streamoid import capture`.
async def capture(
    event: str,
    *,
    actor: Optional[Dict[str, Any]] = None,
    tenant: Optional[Dict[str, Any]] = None,
    properties: Optional[Dict[str, Any]] = None,
    source: Optional[str] = None,
    correlation_id: Optional[str] = None,
) -> Dict[str, Any]:
    return await Analytics().capture(
        event,
        actor=actor,
        tenant=tenant,
        properties=properties,
        source=source,
        correlation_id=correlation_id,
    )
