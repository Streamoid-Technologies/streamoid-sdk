from streamoid.config import get_config, resolve_secret


def test_resolve_secret_env_scheme(monkeypatch):
    monkeypatch.setenv("STREAMOID_TEST_SECRET", "s3cr3t")
    assert resolve_secret("env://STREAMOID_TEST_SECRET") == "s3cr3t"


def test_resolve_secret_env_scheme_missing_var(monkeypatch):
    monkeypatch.delenv("STREAMOID_DOES_NOT_EXIST", raising=False)
    assert resolve_secret("env://STREAMOID_DOES_NOT_EXIST") is None


def test_resolve_secret_vault_scheme_falls_back_to_env(monkeypatch):
    monkeypatch.setenv("R2_SECRET_ACCESS_KEY", "vault-fallback-value")
    assert resolve_secret("vault://secret/data/catalogix/r2-secret-access-key") == "vault-fallback-value"


def test_resolve_secret_bare_value_passthrough():
    assert resolve_secret("dev-only-inline-value") == "dev-only-inline-value"


def test_resolve_secret_empty():
    assert resolve_secret(None) is None
    assert resolve_secret("") is None


def test_get_config_has_no_hardcoded_product_default(monkeypatch):
    monkeypatch.delenv("STREAMOID_PRODUCT", raising=False)
    monkeypatch.delenv("STREAMOID_PLATFORM_API_URL", raising=False)
    get_config.cache_clear()
    cfg = get_config()
    assert cfg.product == ""
    assert cfg.has_platform is False


def test_get_config_reads_platform_api_url(monkeypatch):
    monkeypatch.setenv("STREAMOID_PRODUCT", "catalogix")
    monkeypatch.setenv("STREAMOID_PLATFORM_API_URL", "https://unified.example.com/")
    get_config.cache_clear()
    cfg = get_config()
    assert cfg.product == "catalogix"
    assert cfg.platform_api_url == "https://unified.example.com"
    assert cfg.has_platform is True
    get_config.cache_clear()
