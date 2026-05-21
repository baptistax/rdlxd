package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicWriteJSONWritesAndReplacesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	if err := AtomicWriteJSON(path, map[string]int{"value": 1}); err != nil {
		t.Fatalf("AtomicWriteJSON returned error: %v", err)
	}
	if err := AtomicWriteJSON(path, map[string]int{"value": 2}); err != nil {
		t.Fatalf("AtomicWriteJSON replace returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	var decoded map[string]int
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if decoded["value"] != 2 {
		t.Fatalf("value = %d, want 2", decoded["value"])
	}
}
