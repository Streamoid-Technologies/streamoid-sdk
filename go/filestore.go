package streamoid

import "context"

// PutBytes is the host's low-level object write (the thing objectstore.R2 or
// an equivalent client already provides): write data under key and return
// its public URL. FileStore wraps this into the full C6 contract.
type PutBytes func(ctx context.Context, key string, data []byte, contentType string) (publicURL string, err error)

// FileStorePutInput is Put's full request.
type FileStorePutInput struct {
	Data        []byte
	WorkspaceID string
	Kind        string
	Filename    string
	ContentType string
	UserID      string
	EntityRef   map[string]any
	// AwaitEvent makes Put await the file_uploaded Capture instead of firing
	// it in the background (CaptureBg). Default false.
	AwaitEvent bool
}

// FileStore is the uniform storage contract (architecture.md §L1): PutObject
// for raw keyed writes (no index/event -- for call sites migrating off a
// legacy key), and Put for the full path (C6 key -> bytes -> files index ->
// file_uploaded).
type FileStore struct {
	client   *Client
	putBytes PutBytes
	bucket   func() string
}

// NewFileStore builds a FileStore over an existing low-level putBytes (e.g.
// objectstore.R2.PutObject, adapted to the PutBytes signature). bucket is
// optional (nil is fine) and only used to populate StoredFile.Bucket for the
// files-index/telemetry record.
func NewFileStore(client *Client, putBytes PutBytes, bucket func() string) *FileStore {
	return &FileStore{client: client, putBytes: putBytes, bucket: bucket}
}

// BuildKey builds a C6 key without writing -- useful for presigned flows.
func (f *FileStore) BuildKey(workspaceID, kind, filename string) string {
	return BuildObjectKey(ObjectKeyOptions{
		Product:     f.client.cfg.Product,
		WorkspaceID: workspaceID,
		Kind:        kind,
		Filename:    filename,
	})
}

// PutObject is a raw keyed write through the single client (no index/event).
func (f *FileStore) PutObject(ctx context.Context, key string, data []byte, contentType string) (string, error) {
	return f.putBytes(ctx, key, data, contentType)
}

// Put is the full contract: C6 key + bytes + files index (L3) +
// file_uploaded (L2). The index/event step is best-effort and never fails
// the call once the bytes are written -- a telemetry failure must not turn
// into an upload failure for the caller.
func (f *FileStore) Put(ctx context.Context, in FileStorePutInput) (StoredFile, error) {
	key := f.BuildKey(in.WorkspaceID, in.Kind, in.Filename)

	publicURL, err := f.putBytes(ctx, key, in.Data, in.ContentType)
	if err != nil {
		return StoredFile{}, err
	}

	bucket := ""
	if f.bucket != nil {
		bucket = f.bucket()
	}
	stored := StoredFile{
		Key:         key,
		Bucket:      bucket,
		PublicURL:   publicURL,
		Size:        int64(len(in.Data)),
		ContentType: in.ContentType,
	}

	recordAndEmit := func(ctx context.Context) {
		if err := f.client.RecordFile(ctx, &stored, RecordFileOptions{
			WorkspaceID: in.WorkspaceID,
			UserID:      in.UserID,
			Kind:        in.Kind,
			EntityRef:   in.EntityRef,
		}); err != nil {
			f.client.log.Warn("streamoid: record_file failed", "key", key, "error", err)
		}

		actor := map[string]any{}
		if in.UserID != "" {
			actor["user_id"] = in.UserID
		}
		props := map[string]any{
			"kind":         in.Kind,
			"content_type": in.ContentType,
			"size":         stored.Size,
			"file_id":      stored.FileID,
			"bucket":       bucket,
		}
		if err := f.client.Capture(ctx, "file_uploaded", CaptureOptions{
			Actor:      actor,
			Tenant:     map[string]any{"workspace_id": in.WorkspaceID},
			Properties: props,
		}); err != nil {
			f.client.log.Warn("streamoid: file_uploaded capture failed", "key", key, "error", err)
		}
	}

	if in.AwaitEvent {
		recordAndEmit(ctx)
	} else {
		// Detached from ctx cancellation: the HTTP handler's request context
		// is often canceled the moment the response is written, which would
		// otherwise abort telemetry before it has a chance to send.
		go recordAndEmit(context.Background())
	}

	return stored, nil
}
