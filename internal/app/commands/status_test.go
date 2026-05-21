package commands

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baptistax/rdlxd/internal/post"
	"github.com/baptistax/rdlxd/internal/storage"
)

func TestRunFailedReadsState(t *testing.T) {
	layout, err := storage.InitializeLayout(t.TempDir(), "r_example")
	if err != nil {
		t.Fatalf("InitializeLayout returned error: %v", err)
	}
	state, err := storage.OpenSQLiteState(layout.StatePath)
	if err != nil {
		t.Fatalf("OpenSQLiteState returned error: %v", err)
	}
	if err := state.UpsertPost(storage.PostRecord{
		PostID:    "t3_failed",
		Permalink: "https://www.reddit.com/r/example/comments/failed/title/",
		Status:    "failed",
		Substatus: "all_candidates_failed",
		Retryable: true,
	}); err != nil {
		t.Fatalf("UpsertPost returned error: %v", err)
	}
	if err := state.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := RunFailed([]string{layout.SourceDir}, &stdout); err != nil {
		t.Fatalf("RunFailed returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "all_candidates_failed") || !strings.Contains(output, "true") {
		t.Fatalf("unexpected failed output: %s", output)
	}
}

func TestRunRetryRetriesOnlyRetryableIncompletePosts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("image bytes"))
	}))
	defer server.Close()

	layout, err := storage.InitializeLayout(t.TempDir(), "r_example")
	if err != nil {
		t.Fatalf("InitializeLayout returned error: %v", err)
	}
	state, err := storage.OpenSQLiteState(layout.StatePath)
	if err != nil {
		t.Fatalf("OpenSQLiteState returned error: %v", err)
	}
	retryablePost := post.Post{Name: "t3_retry", ID: "retry", URL: server.URL + "/image.jpg", Title: "retry"}
	unsupportedPost := post.Post{Name: "t3_unsupported", ID: "unsupported", URL: server.URL + "/skip.jpg", Title: "skip"}
	if err := state.UpsertPost(storage.PostRecord{PostID: retryablePost.Name, Name: retryablePost.Name, Status: "failed", Substatus: "download_error", Retryable: true}); err != nil {
		t.Fatalf("UpsertPost retryable returned error: %v", err)
	}
	if err := state.UpsertPost(storage.PostRecord{PostID: unsupportedPost.Name, Name: unsupportedPost.Name, Status: "unsupported", Substatus: "external_page_not_supported", Retryable: true}); err != nil {
		t.Fatalf("UpsertPost unsupported returned error: %v", err)
	}
	if err := state.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}
	writeTestSidecar(t, layout, retryablePost)
	writeTestSidecar(t, layout, unsupportedPost)

	var stdout bytes.Buffer
	if err := RunRetry([]string{layout.SourceDir}, &stdout); err != nil {
		t.Fatalf("RunRetry returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Retried: 1") || !strings.Contains(output, "Skipped: 1") {
		t.Fatalf("unexpected retry output: %s", output)
	}
}

func writeTestSidecar(t *testing.T, layout storage.Layout, p post.Post) {
	t.Helper()
	postDir, err := storage.PostDir(layout, p.Name)
	if err != nil {
		t.Fatalf("PostDir returned error: %v", err)
	}
	if err := storage.AtomicWriteJSON(filepath.Join(postDir, "post.json"), postSidecar{Post: p, Status: "failed", Retryable: true}); err != nil {
		t.Fatalf("AtomicWriteJSON returned error: %v", err)
	}
}
