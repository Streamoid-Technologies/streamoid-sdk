package streamoid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func testConfig(platformURL string) Config {
	return Config{
		Product:        "catalogix",
		Source:         "backend-test",
		Environment:    "test",
		PlatformAPIURL: platformURL,
		MCPRegistryURL: platformURL,
		PlatformToken:  "test-token",
		FilesEnabled:   true,
		AuditEnabled:   true,
		Timeout:        5 * time.Second,
	}
}

// recordingServer captures every request it receives (path, auth header, body).
type recordingServer struct {
	mu       sync.Mutex
	requests []recordedRequest
	respond  func(w http.ResponseWriter, r *recordedRequest)
}

type recordedRequest struct {
	Path string
	Auth string
	Body map[string]any
}

func newRecordingServer(t *testing.T, respond func(w http.ResponseWriter, r *recordedRequest)) (*recordingServer, *httptest.Server) {
	t.Helper()
	rs := &recordingServer{respond: respond}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(req.Body).Decode(&body)
		rec := recordedRequest{Path: req.URL.Path, Auth: req.Header.Get("Authorization"), Body: body}
		rs.mu.Lock()
		rs.requests = append(rs.requests, rec)
		rs.mu.Unlock()
		if rs.respond != nil {
			rs.respond(w, &rec)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)
	return rs, srv
}

func (rs *recordingServer) all() []recordedRequest {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	out := make([]recordedRequest, len(rs.requests))
	copy(out, rs.requests)
	return out
}

func TestCaptureRejectsUnknownVerbWithoutNetworkCall(t *testing.T) {
	rs, srv := newRecordingServer(t, nil)
	client := NewClient(testConfig(srv.URL), nil)

	err := client.Capture(context.Background(), "not_a_real_verb", CaptureOptions{})
	if err == nil {
		t.Fatal("expected an error for an unknown event verb")
	}
	if len(rs.all()) != 0 {
		t.Fatalf("expected no network calls for a rejected verb, got %d", len(rs.all()))
	}
}

func TestCaptureWritesAuditEventWithAuth(t *testing.T) {
	rs, srv := newRecordingServer(t, nil)
	client := NewClient(testConfig(srv.URL), nil)

	if err := client.Capture(context.Background(), "file_uploaded", CaptureOptions{
		Tenant:     map[string]any{"workspace_id": "w1"},
		Properties: map[string]any{"key": "w1/catalogix/file-upload/2026/07/x-a.csv"},
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := rs.all()
	if len(reqs) != 1 {
		t.Fatalf("expected exactly 1 request (audit only, no posthog key configured), got %d", len(reqs))
	}
	if reqs[0].Path != "/api/v1/audit/events" {
		t.Fatalf("expected audit_events path, got %q", reqs[0].Path)
	}
	if reqs[0].Auth != "Bearer test-token" {
		t.Fatalf("expected internal platform token attached to audit write, got %q", reqs[0].Auth)
	}
}

func TestCaptureSendsPostHogWithoutInternalAuthHeader(t *testing.T) {
	rs, srv := newRecordingServer(t, nil)
	cfg := testConfig("") // no platform URL -> audit disabled, isolates the posthog call
	cfg.PostHogHost = srv.URL
	cfg.PostHogKey = "phk_test"
	client := NewClient(cfg, nil)

	if err := client.Capture(context.Background(), "page_viewed", CaptureOptions{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	reqs := rs.all()
	if len(reqs) != 1 {
		t.Fatalf("expected exactly 1 posthog request, got %d", len(reqs))
	}
	if reqs[0].Path != "/capture/" {
		t.Fatalf("expected /capture/ path, got %q", reqs[0].Path)
	}
	if reqs[0].Auth != "" {
		t.Fatalf("SECURITY: internal platform Bearer token must never be sent to PostHog, got Authorization header %q", reqs[0].Auth)
	}
	if reqs[0].Body["api_key"] != "phk_test" {
		t.Fatalf("expected posthog api_key in body, got %+v", reqs[0].Body)
	}
}

func TestCaptureSoftSkipsWhenNothingConfigured(t *testing.T) {
	client := NewClient(testConfig(""), nil)
	if err := client.Capture(context.Background(), "page_viewed", CaptureOptions{}); err != nil {
		t.Fatalf("expected Capture to soft-skip with no platform/posthog configured, got error: %v", err)
	}
}

func TestWriteAuditEventSoftSkipsWithoutPlatformURL(t *testing.T) {
	client := NewClient(testConfig(""), nil)
	env, err := BuildEnvelope("page_viewed", "catalogix", "backend", EnvelopeOptions{})
	if err != nil {
		t.Fatalf("unexpected error building envelope: %v", err)
	}
	if err := client.WriteAuditEvent(context.Background(), env); err != nil {
		t.Fatalf("expected soft-skip (nil error) with no platform URL, got %v", err)
	}
}

func TestRecordFileSetsFileID(t *testing.T) {
	_, srv := newRecordingServer(t, func(w http.ResponseWriter, r *recordedRequest) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"_id":"file-123"}}`))
	})
	client := NewClient(testConfig(srv.URL), nil)

	stored := &StoredFile{Key: "w1/catalogix/file-upload/2026/07/x-a.csv", Size: 42}
	err := client.RecordFile(context.Background(), stored, RecordFileOptions{WorkspaceID: "w1", Kind: "file-upload"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stored.FileID != "file-123" {
		t.Fatalf("expected FileID to be set from response, got %q", stored.FileID)
	}
}

func TestRecordFileSoftSkipsWhenFilesDisabled(t *testing.T) {
	rs, srv := newRecordingServer(t, nil)
	cfg := testConfig(srv.URL)
	cfg.FilesEnabled = false
	client := NewClient(cfg, nil)

	stored := &StoredFile{Key: "k"}
	if err := client.RecordFile(context.Background(), stored, RecordFileOptions{}); err != nil {
		t.Fatalf("expected soft-skip, got error: %v", err)
	}
	if len(rs.all()) != 0 {
		t.Fatalf("expected no network calls when FilesEnabled is false, got %d", len(rs.all()))
	}
}

func TestRecordFilePropagatesTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	client := NewClient(testConfig(srv.URL), nil)

	stored := &StoredFile{Key: "k"}
	if err := client.RecordFile(context.Background(), stored, RecordFileOptions{}); err == nil {
		t.Fatal("expected an error to be returned (not swallowed) on a transport/HTTP failure")
	}
}

func TestRegisterMCPServerSoftSkipsWithoutPlatformURL(t *testing.T) {
	client := NewClient(testConfig(""), nil)
	err := client.RegisterMCPServer(context.Background(), RegisterMCPServerOptions{ServerID: "svc", Owner: "catalogix"})
	if err != nil {
		t.Fatalf("expected soft-skip (nil error) with no platform URL, got %v", err)
	}
}

func TestNilClientIsNoOp(t *testing.T) {
	var client *Client
	if err := client.Capture(context.Background(), "page_viewed", CaptureOptions{}); err != nil {
		t.Fatalf("expected nil Client.Capture to no-op, got %v", err)
	}
	client.CaptureBg(context.Background(), "page_viewed", CaptureOptions{})
	if err := client.RecordFile(context.Background(), &StoredFile{}, RecordFileOptions{}); err != nil {
		t.Fatalf("expected nil Client.RecordFile to no-op, got %v", err)
	}
	if err := client.WriteAuditEvent(context.Background(), Envelope{}); err != nil {
		t.Fatalf("expected nil Client.WriteAuditEvent to no-op, got %v", err)
	}
	if err := client.RegisterMCPServer(context.Background(), RegisterMCPServerOptions{}); err != nil {
		t.Fatalf("expected nil Client.RegisterMCPServer to no-op, got %v", err)
	}
}

func TestRegisterMCPServerPostsExpectedPayload(t *testing.T) {
	rs, srv := newRecordingServer(t, nil)
	client := NewClient(testConfig(srv.URL), nil)

	err := client.RegisterMCPServer(context.Background(), RegisterMCPServerOptions{
		ServerID:      "catalogix-backend",
		Owner:         "catalogix",
		URL:           "https://backend.internal",
		Transport:     "http",
		AuthSecretRef: "env://STREAMOID_MCP_TOKEN",
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	reqs := rs.all()
	if len(reqs) != 1 || reqs[0].Path != "/api/v1/mcp/servers" {
		t.Fatalf("expected exactly 1 request to /api/v1/mcp/servers, got %+v", reqs)
	}
	if reqs[0].Auth != "Bearer test-token" {
		t.Fatalf("expected internal platform token on MCP registration, got %q", reqs[0].Auth)
	}
	auth, _ := reqs[0].Body["auth"].(map[string]any)
	if auth == nil || auth["secretRef"] != "env://STREAMOID_MCP_TOKEN" {
		t.Fatalf("expected auth.secretRef in request body, got %+v", reqs[0].Body)
	}
}
