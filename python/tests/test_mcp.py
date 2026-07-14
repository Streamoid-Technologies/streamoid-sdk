import httpx
import pytest
import respx

from streamoid.config import StreamoidConfig
from streamoid.mcp import register_mcp_server


def make_config(**overrides) -> StreamoidConfig:
    base = dict(
        product="catalogix",
        source="backend-test",
        environment="test",
        platform_api_url="https://unified.example.com",
        platform_token="test-token",
        files_enabled=True,
        audit_enabled=True,
        posthog_host="https://posthog.example.com",
        posthog_key=None,
        mcp_registry_url="https://unified.example.com",
        timeout_s=5.0,
    )
    base.update(overrides)
    return StreamoidConfig(**base)


@pytest.mark.asyncio
async def test_register_mcp_server_soft_skips_without_registry_url():
    cfg = make_config(mcp_registry_url=None)
    ok = await register_mcp_server(server_id="svc", owner="catalogix", display_name="svc", url="https://x", config=cfg)
    assert ok is False


@pytest.mark.asyncio
async def test_register_mcp_server_posts_expected_payload():
    with respx.mock as router:
        route = router.post("https://unified.example.com/api/v1/mcp/servers").mock(
            return_value=httpx.Response(200, json={"status": "ok"})
        )
        ok = await register_mcp_server(
            server_id="catalogix-backend",
            owner="catalogix",
            display_name="catalogix backend",
            url="https://backend.internal",
            auth_secret_ref="env://STREAMOID_MCP_TOKEN",
            config=make_config(),
        )
        assert ok is True
        req = route.calls.last.request
        assert req.headers["authorization"] == "Bearer test-token"
