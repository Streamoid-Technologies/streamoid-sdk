import httpx
import pytest
import respx

from streamoid.analytics import Analytics
from streamoid.config import StreamoidConfig


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
async def test_capture_rejects_unknown_verb_without_network_call():
    with respx.mock(assert_all_called=False) as router:
        client = Analytics(make_config())
        with pytest.raises(ValueError):
            await client.capture("not_a_real_verb")
        assert len(router.calls) == 0


@pytest.mark.asyncio
async def test_capture_writes_audit_event_with_auth():
    with respx.mock as router:
        route = router.post("https://unified.example.com/api/v1/audit/events").mock(
            return_value=httpx.Response(200, json={"status": "ok"})
        )
        client = Analytics(make_config())
        await client.capture("file_uploaded", tenant={"workspace_id": "w1"})
        assert route.called
        req = route.calls.last.request
        assert req.headers["authorization"] == "Bearer test-token"


@pytest.mark.asyncio
async def test_capture_sends_posthog_without_internal_auth_header():
    with respx.mock as router:
        route = router.post("https://posthog.example.com/capture/").mock(
            return_value=httpx.Response(200, json={"status": 1})
        )
        cfg = make_config(platform_api_url=None, posthog_key="phk_test")
        client = Analytics(cfg)
        await client.capture("page_viewed")
        assert route.called
        req = route.calls.last.request
        assert "authorization" not in {k.lower() for k in req.headers.keys()}, (
            "SECURITY: internal platform Bearer token must never be sent to PostHog"
        )


@pytest.mark.asyncio
async def test_capture_soft_skips_when_nothing_configured():
    cfg = make_config(platform_api_url=None, posthog_key=None)
    client = Analytics(cfg)
    envelope = await client.capture("page_viewed")
    assert envelope["event"] == "page_viewed"
