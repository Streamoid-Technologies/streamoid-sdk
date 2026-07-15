package streamoid

import (
	"context"
	"testing"
	"time"
)

func TestBootstrapSkipsRegistrationWithoutURL(t *testing.T) {
	rs, srv := newRecordingServer(t, nil)
	t.Setenv("STREAMOID_PLATFORM_API_URL", srv.URL)

	client := Bootstrap(context.Background(), BootstrapOptions{Product: "catalogix", ServiceName: "products-set"}, nil)
	if client == nil {
		t.Fatal("expected a non-nil client even without a service URL")
	}
	if client.Config().Product != "catalogix" {
		t.Fatalf("expected Product to flow through to the config, got %q", client.Config().Product)
	}
	// Registration is fire-and-forget; give the (non-existent) goroutine a
	// moment, then assert nothing was sent.
	time.Sleep(50 * time.Millisecond)
	if len(rs.all()) != 0 {
		t.Fatalf("expected no MCP registration call without a service URL, got %d requests", len(rs.all()))
	}
}

func TestBootstrapRegistersWhenURLConfigured(t *testing.T) {
	rs, srv := newRecordingServer(t, nil)
	t.Setenv("STREAMOID_PLATFORM_API_URL", srv.URL)

	client := Bootstrap(context.Background(), BootstrapOptions{
		Product:     "catalogix",
		ServiceName: "products-set",
		URL:         "http://products-set.internal:8080",
		HealthPath:  "/v1/set/test",
	}, nil)
	if client == nil {
		t.Fatal("expected a non-nil client")
	}

	deadline := time.Now().Add(2 * time.Second)
	for len(rs.all()) == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	reqs := rs.all()
	if len(reqs) != 1 || reqs[0].Path != "/api/v1/mcp/servers" {
		t.Fatalf("expected exactly 1 request to /api/v1/mcp/servers, got %+v", reqs)
	}
	if reqs[0].Body["serverId"] != "products-set" {
		t.Fatalf("expected serverId=products-set in registration payload, got %+v", reqs[0].Body)
	}
	if reqs[0].Body["owner"] != "catalogix" {
		t.Fatalf("expected owner=catalogix (from opts.Product, not a hardcoded default), got %+v", reqs[0].Body)
	}
}
