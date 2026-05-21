package download

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/baptistax/rdlxd/internal/media"
)

func TestValidateCandidateContentTypeRejectsHTML(t *testing.T) {
	err := ValidateCandidateContentType(media.MediaCandidate{MediaKind: media.MediaKindUnknown}, "text/html; charset=utf-8")
	var contentTypeErr ContentTypeError
	if !errors.As(err, &contentTypeErr) {
		t.Fatalf("expected ContentTypeError, got %v", err)
	}
	if contentTypeErr.Substatus != string(media.SubstatusExternalPageUnsupported) {
		t.Fatalf("substatus = %s", contentTypeErr.Substatus)
	}
}

func TestDownloaderRejectsHTMLContentType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html></html>"))
	}))
	defer server.Close()

	downloader := NewDownloader()
	_, err := downloader.DownloadCandidate(context.Background(), media.MediaCandidate{
		URL:       server.URL + "/page",
		MediaKind: media.MediaKindUnknown,
		Index:     1,
	}, t.TempDir())
	var contentTypeErr ContentTypeError
	if !errors.As(err, &contentTypeErr) {
		t.Fatalf("expected ContentTypeError, got %v", err)
	}
}

func TestDownloaderStoresBlobAndMaterializesFile(t *testing.T) {
	body := []byte("image bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	root := t.TempDir()
	downloader := NewDownloader()
	result, err := downloader.DownloadCandidateToStore(context.Background(), media.MediaCandidate{
		CandidateID: "candidate_image",
		URL:         server.URL + "/image.jpg?token=abc",
		MediaKind:   media.MediaKindImage,
		Index:       1,
	}, filepath.Join(root, "media"), filepath.Join(root, "temp"), filepath.Join(root, "blobs"))
	if err != nil {
		t.Fatalf("DownloadCandidateToStore returned error: %v", err)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("materialized file missing: %v", err)
	}
	if _, err := os.Stat(result.BlobPath); err != nil {
		t.Fatalf("blob file missing: %v", err)
	}
	if filepath.Base(result.Path) != "001.jpg" {
		t.Fatalf("file name = %s, want 001.jpg", filepath.Base(result.Path))
	}
}

func TestDownloaderReusesDuplicateBlob(t *testing.T) {
	body := []byte("same image")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(body)
	}))
	defer server.Close()

	root := t.TempDir()
	downloader := NewDownloader()
	_, err := downloader.DownloadCandidateToStore(context.Background(), media.MediaCandidate{
		CandidateID: "candidate_one",
		URL:         server.URL + "/one.png",
		MediaKind:   media.MediaKindImage,
		Index:       1,
	}, filepath.Join(root, "media-one"), filepath.Join(root, "temp"), filepath.Join(root, "blobs"))
	if err != nil {
		t.Fatalf("first download returned error: %v", err)
	}
	result, err := downloader.DownloadCandidateToStore(context.Background(), media.MediaCandidate{
		CandidateID: "candidate_two",
		URL:         server.URL + "/two.png",
		MediaKind:   media.MediaKindImage,
		Index:       1,
	}, filepath.Join(root, "media-two"), filepath.Join(root, "temp"), filepath.Join(root, "blobs"))
	if err != nil {
		t.Fatalf("second download returned error: %v", err)
	}
	if !result.ReusedBlob {
		t.Fatal("expected duplicate blob reuse")
	}
}
