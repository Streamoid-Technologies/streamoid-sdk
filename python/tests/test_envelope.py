import datetime

import pytest

from streamoid.envelope import build_envelope, build_object_key


def test_build_envelope_rejects_unknown_verb():
    with pytest.raises(ValueError):
        build_envelope("not_a_real_verb", product="catalogix", source="backend")


def test_build_envelope_defaults_and_overrides():
    env = build_envelope(
        "file_uploaded",
        product="catalogix",
        source="backend",
        actor={"user_id": "u1"},
        tenant={"workspace_id": "w1"},
        correlation_id="corr-1",
        properties={"size": 10},
        ts="2026-07-13T12:00:00+00:00",
    )
    assert env["event"] == "file_uploaded"
    assert env["product"] == "catalogix"
    assert env["source"] == "backend"
    assert env["correlation_id"] == "corr-1"
    assert env["ts"] == "2026-07-13T12:00:00+00:00"


def test_build_envelope_generates_correlation_id_when_unset():
    env = build_envelope("page_viewed", product="catalogix", source="backend")
    assert env["correlation_id"]
    assert env["actor"] == {}
    assert env["tenant"] == {}
    assert env["properties"] == {}


def test_build_object_key_workspace_first():
    stamp = datetime.datetime(2026, 7, 13, tzinfo=datetime.timezone.utc)
    key = build_object_key(
        product="catalogix", workspace_id="ws1", kind="file-upload",
        filename="report.csv", now=stamp,
    )
    assert key.startswith("ws1/catalogix/file-upload/2026/07/")
    assert key.endswith("-report.csv")


def test_build_object_key_unscoped_fallback():
    key = build_object_key(product="catalogix", workspace_id="", kind="file-upload", filename="a.csv")
    assert key.startswith("unscoped/catalogix/file-upload/")


def test_build_object_key_sanitizes_filename():
    key = build_object_key(
        product="catalogix", workspace_id="w1", kind="k", filename="weird name #1?.png",
    )
    assert " " not in key and "#" not in key and "?" not in key
    assert key.endswith(".png")


def test_build_object_key_collision_free():
    stamp = datetime.datetime(2026, 7, 13, tzinfo=datetime.timezone.utc)
    k1 = build_object_key(product="catalogix", workspace_id="w1", kind="k", filename="a.csv", now=stamp)
    k2 = build_object_key(product="catalogix", workspace_id="w1", kind="k", filename="a.csv", now=stamp)
    assert k1 != k2
