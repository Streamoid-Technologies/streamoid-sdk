package streamoid

import (
	"strings"
	"testing"
	"time"
)

func TestBuildEnvelopeRejectsUnknownVerb(t *testing.T) {
	_, err := BuildEnvelope("not_a_real_verb", "catalogix", "backend", EnvelopeOptions{})
	if err == nil {
		t.Fatal("expected an error for an unknown event verb, got nil")
	}
}

func TestBuildEnvelopeDefaultsAndOverrides(t *testing.T) {
	fixed := time.Date(2026, 7, 13, 12, 0, 0, 0, time.UTC)
	env, err := BuildEnvelope("file_uploaded", "catalogix", "backend", EnvelopeOptions{
		Actor:         map[string]any{"user_id": "u1"},
		Tenant:        map[string]any{"workspace_id": "w1"},
		CorrelationID: "corr-1",
		Properties:    map[string]any{"size": 10},
		Timestamp:     fixed,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Event != "file_uploaded" || env.Product != "catalogix" || env.Source != "backend" {
		t.Fatalf("unexpected envelope: %+v", env)
	}
	if env.CorrelationID != "corr-1" {
		t.Fatalf("correlation_id not preserved: %+v", env)
	}
	if env.Timestamp != fixed.Format(time.RFC3339) {
		t.Fatalf("timestamp not preserved: %s", env.Timestamp)
	}
}

func TestBuildEnvelopeGeneratesCorrelationIDWhenUnset(t *testing.T) {
	env, err := BuildEnvelope("page_viewed", "catalogix", "backend", EnvelopeOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.CorrelationID == "" {
		t.Fatal("expected a generated correlation_id, got empty string")
	}
	if env.Actor == nil || env.Tenant == nil || env.Properties == nil {
		t.Fatalf("expected non-nil defaults for actor/tenant/properties, got %+v", env)
	}
}

func TestBuildObjectKeyWorkspaceFirst(t *testing.T) {
	stamp := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	key := BuildObjectKey(ObjectKeyOptions{
		Product:     "catalogix",
		WorkspaceID: "ws1",
		Kind:        "file-upload",
		Filename:    "report.csv",
		Now:         stamp,
	})
	wantPrefix := "ws1/catalogix/file-upload/2026/07/"
	if !strings.HasPrefix(key, wantPrefix) {
		t.Fatalf("key %q does not start with %q", key, wantPrefix)
	}
	if !strings.HasSuffix(key, "-report.csv") {
		t.Fatalf("key %q does not end with the sanitized filename", key)
	}
}

func TestBuildObjectKeyUnscopedFallback(t *testing.T) {
	key := BuildObjectKey(ObjectKeyOptions{Product: "catalogix", Kind: "file-upload", Filename: "a.csv"})
	if !strings.HasPrefix(key, "unscoped/catalogix/file-upload/") {
		t.Fatalf("expected unscoped fallback, got %q", key)
	}
}

func TestBuildObjectKeySanitizesFilename(t *testing.T) {
	key := BuildObjectKey(ObjectKeyOptions{Product: "catalogix", WorkspaceID: "w1", Kind: "k", Filename: "weird name #1?.png"})
	if strings.Contains(key, " ") || strings.Contains(key, "#") || strings.Contains(key, "?") {
		t.Fatalf("expected sanitized filename in key, got %q", key)
	}
	if !strings.HasSuffix(key, ".png") {
		t.Fatalf("expected extension preserved, got %q", key)
	}
}

func TestBuildObjectKeyCollisionFree(t *testing.T) {
	stamp := time.Date(2026, 7, 13, 0, 0, 0, 0, time.UTC)
	opts := ObjectKeyOptions{Product: "catalogix", WorkspaceID: "w1", Kind: "k", Filename: "a.csv", Now: stamp}
	k1 := BuildObjectKey(opts)
	k2 := BuildObjectKey(opts)
	if k1 == k2 {
		t.Fatalf("expected two calls with identical inputs to still produce distinct keys (uuid), got %q twice", k1)
	}
}
