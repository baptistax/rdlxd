package commands

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/baptistax/rdlxd/internal/storage"
)

func TestRunSmartTreatsSourceAsDownloadFlow(t *testing.T) {
	var stdout bytes.Buffer
	outDir := t.TempDir()
	if err := RunSmart([]string{"r/example", "--out", outDir, "--limit", "0", "--exclude-nsfw"}, &stdout); err != nil {
		t.Fatalf("RunSmart returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Source: r/example") {
		t.Fatalf("output does not contain source summary: %s", output)
	}
	if !strings.Contains(output, "Output: "+filepath.ToSlash(filepath.Join(outDir, "r_example"))) {
		t.Fatalf("output does not contain output path: %s", output)
	}
}

func TestRunSmartDoesNotRequireAuthWhenNoTokenExists(t *testing.T) {
	t.Setenv("REDDITDOWNLOADER_CONFIG_DIR", t.TempDir())
	var stdout bytes.Buffer
	outDir := t.TempDir()
	if err := RunSmart([]string{"r/example", "--out", outDir, "--limit", "0"}, &stdout); err != nil {
		t.Fatalf("RunSmart returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Source: r/example") {
		t.Fatalf("output does not contain source summary: %s", output)
	}
}

func TestRunSmartTreatsStateFolderAsStatusFlow(t *testing.T) {
	layout, err := storage.InitializeLayout(t.TempDir(), "r_example")
	if err != nil {
		t.Fatalf("InitializeLayout returned error: %v", err)
	}
	state, err := storage.OpenSQLiteState(layout.StatePath)
	if err != nil {
		t.Fatalf("OpenSQLiteState returned error: %v", err)
	}
	if err := state.UpsertPost(storage.PostRecord{
		PostID:    "t3_abc",
		Permalink: "https://www.reddit.com/r/example/comments/abc/title/",
		Status:    "failed",
		Reason:    "all_candidates_failed",
		Retryable: true,
	}); err != nil {
		t.Fatalf("UpsertPost returned error: %v", err)
	}
	if err := state.Close(); err != nil {
		t.Fatalf("Close returned error: %v", err)
	}

	var stdout bytes.Buffer
	if err := RunSmart([]string{layout.SourceDir}, &stdout); err != nil {
		t.Fatalf("RunSmart returned error: %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "Not fully downloaded: 1 posts") {
		t.Fatalf("output does not contain status summary: %s", output)
	}
	if !strings.Contains(output, "all_candidates_failed") {
		t.Fatalf("output does not contain failed list: %s", output)
	}
}
