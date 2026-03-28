// Package store handles all JSON file I/O for stake.
// Data lives in ~/.stake/ by default (configurable via STAKE_DATA_DIR or --data-dir).
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

var (
	dataDir     string
	dataDirOnce sync.Once
)

// SetDataDir overrides the default data directory.
func SetDataDir(dir string) {
	dataDir = dir
}

// DataDir returns the data directory, creating it if needed.
func DataDir() string {
	dataDirOnce.Do(func() {
		if dataDir == "" {
			if env := os.Getenv("STAKE_DATA_DIR"); env != "" {
				dataDir = env
			} else {
				home, _ := os.UserHomeDir()
				dataDir = filepath.Join(home, ".stake")
			}
		}
		os.MkdirAll(dataDir, 0755)
	})
	return dataDir
}

// Path returns the full path to a file in the data directory.
func Path(name string) string {
	return filepath.Join(DataDir(), name)
}

// LoadJSON reads a JSON file into the target. Returns false if file doesn't exist.
func LoadJSON(name string, target any) bool {
	data, err := os.ReadFile(Path(name))
	if err != nil {
		return false
	}
	return json.Unmarshal(data, target) == nil
}

// SaveJSON writes data as JSON to a file in the data directory.
func SaveJSON(name string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(name), b, 0644)
}
