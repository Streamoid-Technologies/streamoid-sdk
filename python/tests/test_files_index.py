import httpx
import pytest
import respx

from streamoid.config import StreamoidConfig
from streamoid.envelope import StoredFile
from streamoid.files_index import record_file


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
async def test_record_file_sets_file_id():
    with respx.mock as router:
        router.post("https://unified.example.com/api/v1/files").mock(
            return_value=httpx.Response(200, json={"data": {"_id": "file-123"}})
        )
        stored = StoredFile(key="w1/catalogix/file-upload/2026/07/x-a.csv", bucket="b", public_url="", size=42)
        file_id = await record_file(stored, workspace_id="w1", user_id=None, kind="file-upload", config=make_config())
        assert file_id == "file-123"
        assert stored.file_id == "file-123"


@pytest.mark.asyncio
async def test_record_file_soft_skips_when_files_disabled():
    with respx.mock(assert_all_called=False) as router:
        stored = StoredFile(key="k", bucket="b", public_url="", size=1)
        cfg = make_config(files_enabled=False)
        file_id = await record_file(stored, workspace_id="w1", user_id=None, kind="k", config=cfg)
        assert file_id is None
        assert len(router.calls) == 0


@pytest.mark.asyncio
async def test_record_file_soft_skips_without_platform_url():
    stored = StoredFile(key="k", bucket="b", public_url="", size=1)
    cfg = make_config(platform_api_url=None)
    file_id = await record_file(stored, workspace_id="w1", user_id=None, kind="k", config=cfg)
    assert file_id is None
