"""Best-effort async HTTP helper shared by the SDK clients.

Every platform write (event, audit row, file-index row, MCP registration) is
**non-critical to the originating request**: if the platform plane is briefly
unreachable, the user's generation/upload must still succeed. So this helper
swallows and logs transport errors and returns ``None`` instead of raising.
Callers that need delivery guarantees should layer a queue on top later; for
the vendored phase, log-on-failure matches the architecture's "frontend events
may be lossy; server events are emitted but must not take the request down".
"""

from __future__ import annotations

import logging
from typing import Any, Dict, Optional

import httpx

logger = logging.getLogger(__name__)


async def post_json(
    url: str,
    payload: Dict[str, Any],
    *,
    headers: Optional[Dict[str, str]] = None,
    timeout_s: float = 5.0,
    label: str = "platform",
) -> Optional[httpx.Response]:
    try:
        async with httpx.AsyncClient(timeout=timeout_s) as client:
            resp = await client.post(url, json=payload, headers=headers or {})
        if resp.status_code >= 400:
            logger.warning(
                "%s POST %s -> %s: %s", label, url, resp.status_code, resp.text[:300]
            )
            return None
        return resp
    except httpx.HTTPError as exc:
        logger.warning("%s POST %s failed: %s", label, url, exc)
        return None
    except Exception as exc:  # never let a telemetry write escape into the request
        logger.warning("%s POST %s unexpected error: %s", label, url, exc)
        return None


def auth_headers(token: Optional[str]) -> Dict[str, str]:
    return {"Authorization": f"Bearer {token}"} if token else {}
