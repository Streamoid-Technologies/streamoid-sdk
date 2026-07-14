"""L4 MCP self-registration — a service registers its own MCP server.

Each service owner maintains its own registry row (architecture.md §L4,
mcp-registry plan): on startup the service POSTs its current connection details
and tool summary to ``mcp_servers``, keyed by ``owner`` + ``server_id`` (upsert).
A tool change therefore ships with the service and is reflected agent-side on
the next health refresh — no cxo-agent code change. Credentials are passed **by
reference** (``auth.secret_ref``), never inline (C3).
"""

from __future__ import annotations

import logging
from typing import Any, Dict, List, Optional

from .config import StreamoidConfig, get_config
from .transport import auth_headers, post_json

logger = logging.getLogger(__name__)


async def register_mcp_server(
    *,
    server_id: str,
    owner: str,
    display_name: str,
    url: str,
    transport: str = "streamable-http",
    health_url: Optional[str] = None,
    auth_secret_ref: Optional[str] = None,
    auth_type: str = "bearer",
    scopes: Optional[List[str]] = None,
    tool_summary: Optional[List[Dict[str, Any]]] = None,
    version: Optional[str] = None,
    enabled: bool = True,
    config: Optional[StreamoidConfig] = None,
) -> bool:
    """Upsert this service's row in the MCP registry. Best-effort."""
    cfg = config or get_config()
    if not cfg.mcp_registry_url:
        logger.info("MCP registry URL unset; skipping self-registration for %s", server_id)
        return False
    payload = {
        "server_id": server_id,
        "owner": owner,
        "display_name": display_name,
        "url": url,
        "transport": transport,
        "health_url": health_url,
        "auth": {"type": auth_type, "secret_ref": auth_secret_ref},
        "scopes": scopes or ["workspace"],
        "tool_summary": tool_summary or [],
        "version": version,
        "enabled": enabled,
    }
    resp = await post_json(
        f"{cfg.mcp_registry_url}/api/v1/mcp/servers",
        payload,
        headers=auth_headers(cfg.platform_token),
        timeout_s=cfg.timeout_s,
        label="mcp-registry",
    )
    if resp is not None:
        logger.info("Registered MCP server %s with the platform registry", server_id)
        return True
    return False
