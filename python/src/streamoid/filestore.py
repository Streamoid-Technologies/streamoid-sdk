"""L1 filestore contract.

The abstract storage contract every product implements (architecture.md §L1).
The concrete adapter that pushes bytes to R2 is host-specific and lives in the
service (artifax wraps its existing ``S3Service``); this module defines the
shared shape so all products expose the same ``put/get/presign/delete``
surface and the same :class:`~streamoid.envelope.StoredFile` result.
"""

from __future__ import annotations

import abc
from typing import Any, Dict, Optional

from .envelope import StoredFile, build_object_key


class FileStore(abc.ABC):
    """Uniform storage surface. Implementations wrap a concrete backend (R2)."""

    product: str

    @abc.abstractmethod
    async def put(
        self,
        data: bytes,
        *,
        workspace_id: str,
        kind: str,
        filename: Optional[str] = None,
        content_type: Optional[str] = None,
        user_id: Optional[str] = None,
        entity_ref: Optional[Dict[str, Any]] = None,
    ) -> StoredFile:
        """Store ``data`` under the C6 key scheme and return a StoredFile.

        Implementations are expected to (a) write the object, (b) record the
        ``files`` index row (L3), and (c) emit ``file_uploaded`` (L2).
        """

    @abc.abstractmethod
    async def put_object(
        self, key: str, data: bytes, content_type: Optional[str] = None
    ) -> str:
        """Raw keyed write through the single client; returns the public URL.

        The behavior-preserving entry used to route existing upload paths
        without the C6 key (parity with the TS ``putObject``)."""

    @abc.abstractmethod
    async def delete(self, key: str) -> bool: ...

    @abc.abstractmethod
    def public_url(self, key: str) -> str: ...

    def build_key(
        self, *, workspace_id: str, kind: str, filename: Optional[str] = None
    ) -> str:
        return build_object_key(
            product=self.product,
            workspace_id=workspace_id,
            kind=kind,
            filename=filename,
        )
