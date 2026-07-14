"""L3 ``files`` index client — one metadata row per uploaded object.

After a successful storage ``put()``, products call :func:`record_file` to write
the shared file-metadata row (architecture.md §L3, common-file-store plan). This
is the join table the wiki/search/intelligence features enumerate ("what files
does this workspace have?"). Best-effort; the upload itself already succeeded.
"""

from __future__ import annotations

import logging
from typing import Any, Dict, Optional

from .config import StreamoidConfig, get_config
from .envelope import StoredFile
from .transport import auth_headers, post_json

logger = logging.getLogger(__name__)


async def record_file(
    stored: StoredFile,
    *,
    workspace_id: str,
    user_id: Optional[str],
    kind: str,
    entity_ref: Optional[Dict[str, Any]] = None,
    config: Optional[StreamoidConfig] = None,
) -> Optional[str]:
    """Write a ``files`` row; returns the assigned ``file_id`` (or None)."""
    cfg = config or get_config()
    if not cfg.files_enabled or not cfg.has_platform:
        return None
    payload = {
        "workspace_id": workspace_id,
        "user_id": user_id,
        "product": cfg.product,
        "kind": kind,
        "key": stored.key,
        "bucket": stored.bucket,
        "public_url": stored.public_url,
        "content_type": stored.content_type,
        "size": stored.size,
        "checksum": stored.checksum,
        "entity_ref": entity_ref,
        "created_at": stored.created_at,
    }
    resp = await post_json(
        f"{cfg.platform_api_url}/api/v1/files",
        payload,
        headers=auth_headers(cfg.platform_token),
        timeout_s=cfg.timeout_s,
        label="files",
    )
    if resp is None:
        return None
    try:
        body = resp.json()
    except ValueError:
        return None
    # Accept both the cxo envelope ({status,data}) and a flat row.
    data = body.get("data") if isinstance(body, dict) else None
    row = data if isinstance(data, dict) else body
    file_id = row.get("file_id") or row.get("_id") if isinstance(row, dict) else None
    if file_id:
        stored.file_id = file_id
    return file_id
