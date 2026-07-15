package streamoid

import (
	"context"
	"net/http"
	"testing"
)

func TestFileStorePutWritesBytesAndRecordsSynchronously(t *testing.T) {
	rs, srv := newRecordingServer(t, func(w http.ResponseWriter, r *recordedRequest) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"_id":"file-abc"}}`))
	})
	client := NewClient(testConfig(srv.URL), nil)

	var putKey string
	var putData []byte
	putBytes := func(ctx context.Context, key string, data []byte, contentType string) (string, error) {
		putKey = key
		putData = data
		return "https://cdn.example.com/" + key, nil
	}
	fs := NewFileStore(client, putBytes, func() string { return "shared-bucket" })

	stored, err := fs.Put(context.Background(), FileStorePutInput{
		Data:        []byte("hello"),
		WorkspaceID: "w1",
		Kind:        "file-upload",
		Filename:    "report.csv",
		ContentType: "text/csv",
		UserID:      "u1",
		AwaitEvent:  true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if putKey == "" || putKey != stored.Key {
		t.Fatalf("expected putBytes to receive the same C6 key as StoredFile.Key, got %q vs %q", putKey, stored.Key)
	}
	if string(putData) != "hello" {
		t.Fatalf("expected raw bytes passed through, got %q", putData)
	}
	if stored.Bucket != "shared-bucket" || stored.PublicURL == "" || stored.Size != 5 {
		t.Fatalf("unexpected StoredFile: %+v", stored)
	}
	if stored.FileID != "file-abc" {
		t.Fatalf("expected AwaitEvent=true to synchronously populate FileID, got %q", stored.FileID)
	}

	reqs := rs.all()
	if len(reqs) != 2 {
		t.Fatalf("expected 2 requests (files index + audit event), got %d: %+v", len(reqs), reqs)
	}
}

func TestFileStorePutObjectRawWriteHasNoTelemetry(t *testing.T) {
	rs, srv := newRecordingServer(t, nil)
	client := NewClient(testConfig(srv.URL), nil)

	putBytes := func(ctx context.Context, key string, data []byte, contentType string) (string, error) {
		return "https://cdn.example.com/" + key, nil
	}
	fs := NewFileStore(client, putBytes, nil)

	url, err := fs.PutObject(context.Background(), "some/raw/key.csv", []byte("data"), "text/csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://cdn.example.com/some/raw/key.csv" {
		t.Fatalf("unexpected url: %q", url)
	}
	if len(rs.all()) != 0 {
		t.Fatalf("expected PutObject to skip files-index/telemetry entirely, got %d requests", len(rs.all()))
	}
}

func TestFileStoreBuildKeyUsesClientProduct(t *testing.T) {
	client := NewClient(testConfig(""), nil)
	fs := NewFileStore(client, nil, nil)
	key := fs.BuildKey("w1", "file-upload", "a.csv")
	want := "w1/catalogix/file-upload/"
	if len(key) < len(want) || key[:len(want)] != want {
		t.Fatalf("key %q does not start with %q", key, want)
	}
}
