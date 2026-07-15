package streamoid

import (
	"context"
	"sync"
)

// CaptureOptions carries Capture's optional fields (mirrors EnvelopeOptions,
// kept distinct so callers don't need to think about the envelope directly).
type CaptureOptions struct {
	Actor         map[string]any
	Tenant        map[string]any
	CorrelationID string
	Properties    map[string]any
}

// Capture emits event, fanning out to PostHog (if a key is configured) and
// the audit_events store (if AuditEnabled and a platform API URL is
// configured). Both fan-outs are best-effort: a transport failure is logged
// and swallowed, never returned. The one error Capture *does* return is a
// contract violation -- event not in EventVerbs -- which is a programming
// error and must not be silently dropped (matches the Python/TS SDKs: "the
// analytics fan-out never breaks the caller, but an unknown event verb is a
// bug, not a transient failure").
func (c *Client) Capture(ctx context.Context, event string, opts CaptureOptions) error {
	if c == nil {
		return nil
	}
	envelope, err := BuildEnvelope(event, c.cfg.Product, c.cfg.Source, EnvelopeOptions{
		Actor:         opts.Actor,
		Tenant:        opts.Tenant,
		CorrelationID: opts.CorrelationID,
		Properties:    opts.Properties,
	})
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	if c.cfg.PostHogKey != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.sendPostHog(ctx, envelope); err != nil {
				c.log.Warn("streamoid: posthog capture failed", "event", event, "error", err)
			}
		}()
	}

	if c.cfg.AuditEnabled && c.cfg.HasPlatform() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := c.WriteAuditEvent(ctx, envelope); err != nil {
				c.log.Warn("streamoid: audit_events write failed", "event", event, "error", err)
			}
		}()
	}

	wg.Wait()
	return nil
}

// CaptureBg fires Capture in the background and never blocks the caller --
// the Go analogue of the TS SDK's captureBg. Any error (including a bad
// event verb) is logged, never surfaced, since there is no synchronous
// caller left to return it to.
func (c *Client) CaptureBg(ctx context.Context, event string, opts CaptureOptions) {
	go func() {
		if err := c.Capture(ctx, event, opts); err != nil {
			c.log.Warn("streamoid: capture failed", "event", event, "error", err)
		}
	}()
}

// posthogEvent is the minimal PostHog capture-endpoint payload.
type posthogEvent struct {
	APIKey     string         `json:"api_key"`
	Event      string         `json:"event"`
	Properties map[string]any `json:"properties"`
}

func (c *Client) sendPostHog(ctx context.Context, envelope Envelope) error {
	props := map[string]any{}
	for k, v := range envelope.Properties {
		props[k] = v
	}
	props["product"] = envelope.Product
	props["source"] = envelope.Source
	props["correlation_id"] = envelope.CorrelationID
	props["tenant"] = envelope.Tenant
	props["actor"] = envelope.Actor
	if uid, ok := envelope.Actor["user_id"]; ok {
		props["distinct_id"] = uid
	} else if ws, ok := envelope.Tenant["workspace_id"]; ok {
		props["distinct_id"] = ws
	} else {
		props["distinct_id"] = "unscoped"
	}

	body := posthogEvent{
		APIKey:     c.cfg.PostHogKey,
		Event:      envelope.Event,
		Properties: props,
	}
	// postJSONNoAuth: PostHog is a third party and must never see the
	// internal platform service Bearer token postJSON would otherwise attach.
	return c.postJSONNoAuth(ctx, c.cfg.PostHogHost+"/capture/", body, nil)
}
