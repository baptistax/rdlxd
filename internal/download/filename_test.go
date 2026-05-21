package download

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestFilenameSanitization(t *testing.T) {
	name := SanitizeFileName("../bad:name?.jpg")
	if strings.Contains(name, "..") || strings.ContainsAny(name, `/\:?*`) {
		t.Fatalf("unsafe sanitized name: %s", name)
	}
	if name == "" {
		t.Fatal("name is empty")
	}
}

func TestSafeFinalPathStaysInsideRoot(t *testing.T) {
	root := t.TempDir()
	path, err := SafeFinalPath(root, "../image.jpg")
	if err != nil {
		t.Fatalf("SafeFinalPath returned error: %v", err)
	}
	if filepath.Dir(path) != root {
		t.Fatalf("path escaped root: %s", path)
	}
}
