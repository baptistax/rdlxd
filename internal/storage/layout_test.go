package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitializeLayoutCreatesExpectedDirectories(t *testing.T) {
	layout, err := InitializeLayout(t.TempDir(), "r_example")
	if err != nil {
		t.Fatalf("InitializeLayout returned error: %v", err)
	}
	for _, dir := range []string{layout.SourceDir, layout.MediaDir, layout.MetadataDir, layout.ReportsDir, layout.InternalDir, layout.TempDir, layout.BlobsDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("expected directory %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
	}
	if filepath.Base(layout.InternalDir) != ".rdlxd" {
		t.Fatalf("internal dir = %s", layout.InternalDir)
	}
	if filepath.Base(layout.MediaDir) != "media" || filepath.Base(layout.MetadataDir) != "metadata" || filepath.Base(layout.ReportsDir) != "reports" {
		t.Fatalf("media/metadata/reports dirs = %s / %s / %s", layout.MediaDir, layout.MetadataDir, layout.ReportsDir)
	}
}

func TestSafeJoinBlocksPathTraversal(t *testing.T) {
	root := t.TempDir()
	if _, err := SafeJoin(root, "..", "outside"); err == nil {
		t.Fatal("expected path traversal to be blocked")
	}
	if _, err := SafeJoin(root, filepath.Join("posts", "t3_abc")); err != nil {
		t.Fatalf("expected safe path: %v", err)
	}
}

func TestPostDirBlocksTraversalBySanitizingComponent(t *testing.T) {
	layout, err := InitializeLayout(t.TempDir(), "r_example")
	if err != nil {
		t.Fatalf("InitializeLayout returned error: %v", err)
	}
	postDir, err := PostDir(layout, "../t3_abc")
	if err != nil {
		t.Fatalf("PostDir returned error: %v", err)
	}
	if filepath.Base(postDir) == ".." {
		t.Fatal("post dir kept traversal component")
	}
	if filepath.Dir(postDir) != layout.MetadataDir {
		t.Fatalf("post dir = %s, want under metadata", postDir)
	}
}
