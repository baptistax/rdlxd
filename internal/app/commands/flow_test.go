package commands

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baptistax/rdlxd/internal/download"
	"github.com/baptistax/rdlxd/internal/media"
	"github.com/baptistax/rdlxd/internal/media/resolvers"
	"github.com/baptistax/rdlxd/internal/post"
	"github.com/baptistax/rdlxd/internal/storage"
)

func TestAggregatePostStatusVariants(t *testing.T) {
	tests := []struct {
		name     string
		outcomes []assetOutcome
		want     string
	}{
		{
			name:     "downloaded",
			outcomes: []assetOutcome{{Required: true, Status: string(media.StatusDownloaded)}},
			want:     string(media.StatusDownloaded),
		},
		{
			name: "partial",
			outcomes: []assetOutcome{
				{Required: true, Status: string(media.StatusDownloaded)},
				{Required: true, Status: string(media.StatusFailed), Retryable: true},
			},
			want: string(media.StatusPartial),
		},
		{
			name:     "failed",
			outcomes: []assetOutcome{{Required: true, Status: string(media.StatusFailed)}},
			want:     string(media.StatusFailed),
		},
		{
			name:     "unsupported",
			outcomes: []assetOutcome{{Required: true, Status: string(media.StatusUnsupported)}},
			want:     string(media.StatusUnsupported),
		},
		{
			name:     "video warning",
			outcomes: []assetOutcome{{Required: true, Status: string(media.StatusDownloaded), Notes: []string{string(media.SubstatusVideoMayBeSilent)}}},
			want:     string(media.StatusPartial),
		},
		{
			name:     "preview only warning",
			outcomes: []assetOutcome{{Required: true, Status: string(media.StatusDownloaded), Notes: []string{string(media.SubstatusPreviewOnly)}}},
			want:     string(media.StatusPartial),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := aggregatePostStatus(tt.outcomes)
			if got.Status != tt.want {
				t.Fatalf("status = %s, want %s", got.Status, tt.want)
			}
		})
	}
}

func TestPostProcessorDoesNotDownloadExternalPreviewAsFinalMedia(t *testing.T) {
	layout, err := storage.InitializeLayout(t.TempDir(), "r_example")
	if err != nil {
		t.Fatalf("InitializeLayout returned error: %v", err)
	}
	state, err := storage.OpenSQLiteState(layout.StatePath)
	if err != nil {
		t.Fatalf("OpenSQLiteState returned error: %v", err)
	}
	defer state.Close()
	logger, err := storage.OpenLog(layout.LogsPath)
	if err != nil {
		t.Fatalf("OpenLog returned error: %v", err)
	}
	defer logger.Close()

	processor := postProcessor{
		layout:     layout,
		state:      state,
		logger:     logger,
		runID:      "run_test",
		pipeline:   media.NewPipeline(resolvers.RedditPreviewResolver{}, resolvers.UnsupportedPostResolver{}),
		downloader: download.NewDownloader(),
	}
	if err := processor.processPost(context.Background(), post.Post{
		ID:                  "external",
		Name:                "t3_external",
		Title:               "external video",
		URL:                 "https://redgifs.com/watch/example",
		URLOverriddenByDest: "https://redgifs.com/watch/example",
		Domain:              "redgifs.com",
		PostHint:            "rich:video",
		Preview: &post.Preview{Images: []post.PreviewImage{{
			Source: post.PreviewSource{URL: "https://external-preview.redd.it/example.jpeg?auto=webp", Width: 480, Height: 864},
		}}},
	}, postProcessOptions{IncludeNSFW: true}); err != nil {
		t.Fatalf("processPost returned error: %v", err)
	}
	summary, err := state.GetSummaryCounts()
	if err != nil {
		t.Fatalf("GetSummaryCounts returned error: %v", err)
	}
	if summary.PostsFound != 1 || summary.Unsupported != 1 || summary.Downloaded != 0 {
		t.Fatalf("summary = %+v, want unsupported external post", summary)
	}
	entries, err := os.ReadDir(layout.MediaDir)
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("media entries = %d, want none", len(entries))
	}
}

func TestPostProcessorCountsAllUnsupportedPosts(t *testing.T) {
	layout, err := storage.InitializeLayout(t.TempDir(), "r_example")
	if err != nil {
		t.Fatalf("InitializeLayout returned error: %v", err)
	}
	state, err := storage.OpenSQLiteState(layout.StatePath)
	if err != nil {
		t.Fatalf("OpenSQLiteState returned error: %v", err)
	}
	defer state.Close()
	logger, err := storage.OpenLog(layout.LogsPath)
	if err != nil {
		t.Fatalf("OpenLog returned error: %v", err)
	}
	defer logger.Close()

	processor := postProcessor{
		layout:     layout,
		state:      state,
		logger:     logger,
		runID:      "run_test",
		pipeline:   media.NewPipeline(resolvers.UnsupportedPostResolver{}),
		downloader: download.NewDownloader(),
	}
	for _, redditPost := range []post.Post{
		{ID: "one", Name: "t3_one", Title: "external one", URL: "https://example.com/page"},
		{ID: "two", Name: "t3_two", Title: "external two", URL: "https://example.org/page"},
	} {
		if err := processor.processPost(context.Background(), redditPost, postProcessOptions{IncludeNSFW: true}); err != nil {
			t.Fatalf("processPost returned error: %v", err)
		}
	}
	summary, err := state.GetSummaryCounts()
	if err != nil {
		t.Fatalf("GetSummaryCounts returned error: %v", err)
	}
	total := summary.Downloaded + summary.Partial + summary.Failed + summary.Unsupported
	if summary.PostsFound != 2 || summary.Unsupported != 2 || total != summary.PostsFound {
		t.Fatalf("summary = %+v", summary)
	}
}

func TestPostProcessorMarksExternalHTMLUnsupported(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html></html>"))
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
	defer state.Close()
	logger, err := storage.OpenLog(layout.LogsPath)
	if err != nil {
		t.Fatalf("OpenLog returned error: %v", err)
	}
	defer logger.Close()

	processor := postProcessor{
		layout:     layout,
		state:      state,
		logger:     logger,
		runID:      "run_test",
		pipeline:   media.NewPipeline(resolvers.DirectMediaURLResolver{}),
		downloader: download.NewDownloader(),
	}
	if err := processor.processPost(context.Background(), post.Post{
		ID:    "abc",
		Name:  "t3_abc",
		Title: "external page",
		URL:   server.URL + "/page",
	}, postProcessOptions{IncludeNSFW: true}); err != nil {
		t.Fatalf("processPost returned error: %v", err)
	}
	rows, err := state.ListIncompletePosts()
	if err != nil {
		t.Fatalf("ListIncompletePosts returned error: %v", err)
	}
	if len(rows) != 1 || rows[0].Status != string(media.StatusUnsupported) {
		t.Fatalf("rows = %+v, want unsupported", rows)
	}
	if !strings.Contains(rows[0].Reason, "text/html") {
		t.Fatalf("reason = %q, want content type", rows[0].Reason)
	}
}

func TestPostProcessorCreatesBlobAndMediaFile(t *testing.T) {
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
	defer state.Close()
	logger, err := storage.OpenLog(layout.LogsPath)
	if err != nil {
		t.Fatalf("OpenLog returned error: %v", err)
	}
	defer logger.Close()

	processor := postProcessor{
		layout:     layout,
		state:      state,
		logger:     logger,
		runID:      "run_test",
		pipeline:   media.NewPipeline(resolvers.DirectMediaURLResolver{}),
		downloader: download.NewDownloader(),
	}
	if err := processor.processPost(context.Background(), post.Post{
		ID:    "abc",
		Name:  "t3_abc",
		Title: "image",
		URL:   server.URL + "/image.jpg?token=abc",
	}, postProcessOptions{IncludeNSFW: true}); err != nil {
		t.Fatalf("processPost returned error: %v", err)
	}
	summary, err := state.GetSummaryCounts()
	if err != nil {
		t.Fatalf("GetSummaryCounts returned error: %v", err)
	}
	if summary.Downloaded != 1 {
		t.Fatalf("summary = %+v, want one downloaded", summary)
	}
	matches, err := filepath.Glob(filepath.Join(layout.BlobsDir, "sha256_*"))
	if err != nil {
		t.Fatalf("Glob returned error: %v", err)
	}
	if len(matches) == 0 {
		t.Fatal("expected blob file")
	}
	if _, err := os.Stat(filepath.Join(layout.MediaDir, "t3_abc_001.jpg")); err != nil {
		t.Fatalf("expected media file in media dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(layout.MetadataDir, "t3_abc", "post.json")); err != nil {
		t.Fatalf("expected post metadata in metadata dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(layout.MetadataDir, "t3_abc", "media")); !os.IsNotExist(err) {
		t.Fatalf("metadata should not contain a media directory")
	}
}

func TestPostProcessorMarksNSFWExcludedAsUnsupported(t *testing.T) {
	layout, err := storage.InitializeLayout(t.TempDir(), "r_example")
	if err != nil {
		t.Fatalf("InitializeLayout returned error: %v", err)
	}
	state, err := storage.OpenSQLiteState(layout.StatePath)
	if err != nil {
		t.Fatalf("OpenSQLiteState returned error: %v", err)
	}
	defer state.Close()
	logger, err := storage.OpenLog(layout.LogsPath)
	if err != nil {
		t.Fatalf("OpenLog returned error: %v", err)
	}
	defer logger.Close()

	processor := postProcessor{
		layout:     layout,
		state:      state,
		logger:     logger,
		runID:      "run_test",
		pipeline:   media.NewPipeline(resolvers.UnsupportedPostResolver{}),
		downloader: download.NewDownloader(),
	}
	if err := processor.processPost(context.Background(), post.Post{
		ID:     "nsfw",
		Name:   "t3_nsfw",
		Title:  "nsfw",
		URL:    "https://example.com/image.jpg",
		Over18: true,
	}, postProcessOptions{IncludeNSFW: false}); err != nil {
		t.Fatalf("processPost returned error: %v", err)
	}
	summary, err := state.GetSummaryCounts()
	if err != nil {
		t.Fatalf("GetSummaryCounts returned error: %v", err)
	}
	if summary.PostsFound != 1 || summary.Unsupported != 1 || summary.Downloaded+summary.Partial+summary.Failed+summary.Unsupported != summary.PostsFound {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}
