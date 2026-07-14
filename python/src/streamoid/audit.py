"""L3 audit sink — durable projection of L2 events into ``audit_events``.

Writes the canonical envelope to unified-backend's append-only audit store.
This is the same envelope PostHog receives; the audit store is the durable,
queryable side (architecture.md §L3, common-log-management plan).
"""

from __future__ import annotations

import logging
from typing import Any, Dict, Optional

from .config import StreamoidConfig, get_config
from .transport import auth_headers, post_json

logger = logging.getLogger(__name__)


async def write_audit_event(
    envelope: Dict[str, Any], *, config: Optional[StreamoidConfig] = None
) -> bool:
    """Append one envelope to ``audit_events``. Best-effort; returns success."""
    cfg = config or get_config()
    if not cfg.audit_enabled or not cfg.has_platform:
        return False
    resp = await post_json(
        f"{cfg.platform_api_url}/api/v1/audit/events",
        envelope,
        headers=auth_headers(cfg.platform_token),
        timeout_s=cfg.timeout_s,
        label="audit",
    )
    return resp is not None
