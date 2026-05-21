package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Manifest struct {
	Version string        `json:"version"`
	RunID   string        `json:"run_id"`
	Source  string        `json:"source"`
	Summary SummaryCounts `json:"summary"`
}

func WriteManifest(path string, manifest Manifest) error {
	return AtomicWriteJSON(path, manifest)
}

func AtomicWriteJSON(path string, value any) error {
	if path == "" {
		return errors.New("json path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tempName)
		}
	}()

	encoder := json.NewEncoder(temp)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return err
	}
	removeTemp = false
	return nil
}
