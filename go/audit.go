package streamoid

import "context"

// WriteAuditEvent appends envelope to unified-backend's audit_events store
// (POST /api/v1/audit/events). Soft-skips (returns nil) if no platform API
// URL is configured, matching the Python/TS SDKs' "adoption is opt-in per
// environment" behavior. Exported so callers that build their own envelope
// (rare -- Capture is the normal path) can still reach L3 directly.
func (c *Client) WriteAuditEvent(ctx context.Context, envelope Envelope) error {
	if c == nil || !c.cfg.HasPlatform() {
		return nil
	}
	url := c.cfg.PlatformAPIURL + "/api/v1/audit/events"
	return c.postJSON(ctx, url, envelope, nil)
}
