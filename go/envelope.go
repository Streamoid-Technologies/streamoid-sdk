package streamoid

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EventVerbs is the governed v1 verb vocabulary (architecture.md L2). Event
// names are snake_case verbs; emitting an unknown verb is a programming
// error surfaced as an error from BuildEnvelope, not silently accepted.
var EventVerbs = map[string]bool{
	"file_uploaded":        true,
	"search_performed":     true,
	"generation_started":   true,
	"generation_completed": true,
	"automation_started":   true,
	"automation_completed": true,
	"export_performed":     true,
	"user_invited":         true,
	"settings_changed":     true,
	"page_viewed":          true,
	"item_viewed":          true,
}

// Envelope is the canonical event envelope every product emits.
type Envelope struct {
	Event         string         `json:"event"`
	Timestamp     string         `json:"ts"`
	Actor         map[string]any `json:"actor"`
	Tenant        map[string]any `json:"tenant"`
	Product       string         `json:"product"`
	Source        string         `json:"source"`
	CorrelationID string         `json:"correlation_id"`
	Properties    map[string]any `json:"properties"`
}

// EnvelopeOptions carries the optional fields of BuildEnvelope. Actor should
// carry "user_id" (+ optional "email"/"ip"); Tenant carries "workspace_id"
// (+ optional "store_uuid"). A missing tenant is not an error here -- some
// legitimate events (e.g. page_viewed before login) have no workspace -- but
// callers emitting revenue/intelligence events are expected to supply one.
type EnvelopeOptions struct {
	Actor         map[string]any
	Tenant        map[string]any
	CorrelationID string
	Properties    map[string]any
	// Timestamp overrides the envelope's ts (RFC3339 UTC). Tests only; leave
	// zero to use time.Now().UTC().
	Timestamp time.Time
}

// BuildEnvelope builds the canonical envelope every product emits
// (architecture.md §L2). Returns an error if event is not in EventVerbs.
func BuildEnvelope(event, product, source string, opts EnvelopeOptions) (Envelope, error) {
	if !EventVerbs[event] {
		return Envelope{}, fmt.Errorf("streamoid: unknown event verb %q", event)
	}

	ts := opts.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}

	correlationID := opts.CorrelationID
	if correlationID == "" {
		correlationID = uuid.NewString()
	}

	actor := opts.Actor
	if actor == nil {
		actor = map[string]any{}
	}
	tenant := opts.Tenant
	if tenant == nil {
		tenant = map[string]any{}
	}
	properties := opts.Properties
	if properties == nil {
		properties = map[string]any{}
	}

	return Envelope{
		Event:         event,
		Timestamp:     ts.Format(time.RFC3339),
		Actor:         actor,
		Tenant:        tenant,
		Product:       product,
		Source:        source,
		CorrelationID: correlationID,
		Properties:    properties,
	}, nil
}

var unsafeFilenameChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
var dashBeforeDot = regexp.MustCompile(`-+\.`)

// safeFilename sanitizes a filename for use in an object key: collapses any
// character outside [A-Za-z0-9._-] to a single dash, and tidies a dash left
// immediately before the extension dot ("a-.png" -> "a.png").
func safeFilename(filename string) string {
	base := filename
	if idx := strings.LastIndexAny(filename, `/\`); idx >= 0 {
		base = filename[idx+1:]
	}
	base = strings.TrimSpace(base)
	if base == "" {
		return "file"
	}
	safe := unsafeFilenameChars.ReplaceAllString(base, "-")
	safe = strings.Trim(safe, "-.")
	safe = dashBeforeDot.ReplaceAllString(safe, ".")
	if safe == "" {
		return "file"
	}
	return safe
}

// ObjectKeyOptions carries BuildObjectKey's parameters.
type ObjectKeyOptions struct {
	Product     string
	WorkspaceID string
	Kind        string
	Filename    string
	// Now overrides the timestamp used for the {yyyy}/{mm} path segments.
	// Tests only; leave zero to use time.Now().UTC().
	Now time.Time
}

// BuildObjectKey builds the C6 storage key:
// {workspace_id}/{product}/{kind}/{yyyy}/{mm}/{uuid}-{file}.
//
// Workspace-first: the shared bucket is organized per workspace (customer),
// with product as a sub-dir so you can tell which product produced a file.
// Time-partitioned and collision-free. WorkspaceID falls back to "unscoped"
// rather than producing a key with an empty leading segment.
func BuildObjectKey(opts ObjectKeyOptions) string {
	stamp := opts.Now
	if stamp.IsZero() {
		stamp = time.Now().UTC()
	}
	ws := strings.TrimSpace(opts.WorkspaceID)
	if ws == "" {
		ws = "unscoped"
	}
	return fmt.Sprintf(
		"%s/%s/%s/%s/%s/%s-%s",
		ws, opts.Product, opts.Kind,
		stamp.Format("2006"), stamp.Format("01"),
		uuid.New().String(), safeFilename(opts.Filename),
	)
}

// StoredFile is the uniform result of a filestore Put (architecture.md §L1).
type StoredFile struct {
	Key         string
	Bucket      string
	PublicURL   string
	Size        int64
	ContentType string
	Checksum    string
	CreatedAt   time.Time
	// FileID is populated once the central files index row is written (L3).
	FileID string
}

// ToMap renders the StoredFile as the JSON body files_index/audit expect.
func (s StoredFile) ToMap() map[string]any {
	m := map[string]any{
		"key":        s.Key,
		"bucket":     s.Bucket,
		"public_url": s.PublicURL,
		"size":       s.Size,
		"created_at": s.CreatedAt.Format(time.RFC3339),
	}
	if s.ContentType != "" {
		m["content_type"] = s.ContentType
	}
	if s.Checksum != "" {
		m["checksum"] = s.Checksum
	}
	if s.FileID != "" {
		m["file_id"] = s.FileID
	}
	return m
}
