package streamoid

import "context"

// RecordFileOptions carries RecordFile's tenancy/classification fields.
type RecordFileOptions struct {
	WorkspaceID string
	UserID      string
	// Kind classifies the file for the audit trail (e.g. "file-upload",
	// "image-export") -- free-form, not part of EventVerbs.
	Kind string
	// EntityRef optionally links the file to a domain entity, e.g.
	// {"type": "asset-export", "id": key}.
	EntityRef map[string]any
}

type recordFileRequest struct {
	Key         string         `json:"key"`
	Bucket      string         `json:"bucket"`
	PublicURL   string         `json:"public_url"`
	Size        int64          `json:"size"`
	ContentType string         `json:"content_type,omitempty"`
	WorkspaceID string         `json:"workspace_id"`
	UserID      string         `json:"user_id,omitempty"`
	Kind        string         `json:"kind"`
	EntityRef   map[string]any `json:"entity_ref,omitempty"`
}

type recordFileResponse struct {
	Data struct {
		ID string `json:"_id"`
	} `json:"data"`
}

// RecordFile upserts stored into unified-backend's files index (POST
// /api/v1/files, upsert by storage key) and, on success, sets
// stored.FileID from the response. Best-effort: soft-skips if no platform
// API URL is configured or FilesEnabled is false; any transport/HTTP error
// is returned to the caller to log (mirroring the Python/TS SDKs' "caller
// decides whether to await or fire-and-forget" split -- RecordFile itself
// never panics or breaks the upload, it just reports the failure).
func (c *Client) RecordFile(ctx context.Context, stored *StoredFile, opts RecordFileOptions) error {
	if c == nil || !c.cfg.FilesEnabled || !c.cfg.HasPlatform() {
		return nil
	}
	req := recordFileRequest{
		Key:         stored.Key,
		Bucket:      stored.Bucket,
		PublicURL:   stored.PublicURL,
		Size:        stored.Size,
		ContentType: stored.ContentType,
		WorkspaceID: opts.WorkspaceID,
		UserID:      opts.UserID,
		Kind:        opts.Kind,
		EntityRef:   opts.EntityRef,
	}
	var resp recordFileResponse
	url := c.cfg.PlatformAPIURL + "/api/v1/files"
	if err := c.postJSON(ctx, url, req, &resp); err != nil {
		return err
	}
	if resp.Data.ID != "" {
		stored.FileID = resp.Data.ID
	}
	return nil
}
