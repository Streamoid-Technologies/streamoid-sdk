"""Self-contained configuration for the streamoid SDK.

Reads a canonical ``STREAMOID_*`` env set (architecture.md C6) so the package
has no dependency on any host service's settings object. Secrets are resolved
**by reference** (C3): a value of ``env://VAR`` reads ``VAR`` from the
environment; a bare value is used as-is (dev convenience); ``vault://...`` is
recognized as a reference but, until a real secret backend is wired, falls
back to the matching env var so nothing is hardcoded in source.

Every consuming service is expected to set ``STREAMOID_PRODUCT`` explicitly
(there is no per-copy hardcoded default here, unlike the old vendored copies —
a service that forgets to set it gets an empty ``product`` tag on its events,
which is loud and discoverable in the platform data, rather than silently
mislabeled as whichever product's default happened to get copy-pasted in).
Likewise there is no hardcoded fallback to any product's pre-existing
unified-backend env var name (e.g. catalogix's ``UNIFIED_SYSTEM_API``) — a
service relying on one maps it to ``STREAMOID_PLATFORM_API_URL`` in its own
deploy config/env, one line, rather than baking product-specific names into
the shared package.
"""

from __future__ import annotations

import logging
import os
from dataclasses import dataclass
from functools import lru_cache
from typing import Optional

logger = logging.getLogger(__name__)


def resolve_secret(ref: Optional[str]) -> Optional[str]:
    """Resolve a ``secret_ref`` (C3). Never returns inline secrets from source."""
    if not ref:
        return None
    ref = ref.strip()
    if ref.startswith("env://"):
        return os.getenv(ref[len("env://") :]) or None
    if ref.startswith("vault://"):
        # Placeholder for the real secret store. Until that exists, fall back to
        # an env var named after the last path segment so deploys still work.
        var = ref.rsplit("/", 1)[-1].upper().replace("-", "_")
        value = os.getenv(var)
        if value is None:
            logger.warning("secret_ref %s unresolved (no backend, no %s env)", ref, var)
        return value
    # Bare value — dev only.
    return ref


def _truthy(value: Optional[str], default: bool = False) -> bool:
    if value is None:
        return default
    return value.strip().lower() in {"1", "true", "yes", "on"}


@dataclass
class StreamoidConfig:
    # identity of the emitting product/runtime
    product: str
    source: str
    environment: str

    # platform data plane (L3) — unified-backend
    platform_api_url: Optional[str]
    platform_token: Optional[str]
    files_enabled: bool
    audit_enabled: bool

    # event analytics fan-out (L2)
    posthog_host: Optional[str]
    posthog_key: Optional[str]

    # MCP registry (L4)
    mcp_registry_url: Optional[str]

    # network behaviour
    timeout_s: float

    @property
    def has_platform(self) -> bool:
        return bool(self.platform_api_url)


@lru_cache(maxsize=1)
def get_config() -> StreamoidConfig:
    """Load config once from the environment.

    ``STREAMOID_PRODUCT`` should be set by every consumer; see the module
    docstring for why there is no hardcoded per-service default here.
    """
    platform_api = os.getenv("STREAMOID_PLATFORM_API_URL") or None
    return StreamoidConfig(
        product=os.getenv("STREAMOID_PRODUCT", ""),
        source=os.getenv("STREAMOID_SOURCE", "backend"),
        environment=os.getenv("STREAMOID_ENV") or os.getenv("ENVIRONMENT", "development"),
        platform_api_url=(platform_api.rstrip("/") if platform_api else None),
        platform_token=resolve_secret(os.getenv("STREAMOID_PLATFORM_TOKEN_REF")),
        files_enabled=_truthy(os.getenv("STREAMOID_FILES_ENABLED"), default=True),
        audit_enabled=_truthy(os.getenv("STREAMOID_AUDIT_ENABLED"), default=True),
        posthog_host=(os.getenv("STREAMOID_POSTHOG_HOST") or "https://ph.streamoid.com").rstrip("/"),
        posthog_key=resolve_secret(os.getenv("STREAMOID_POSTHOG_KEY_REF")),
        mcp_registry_url=(
            (os.getenv("STREAMOID_MCP_REGISTRY_URL") or platform_api or "").rstrip("/")
            or None
        ),
        timeout_s=float(os.getenv("STREAMOID_HTTP_TIMEOUT_S", "5.0")),
    )
