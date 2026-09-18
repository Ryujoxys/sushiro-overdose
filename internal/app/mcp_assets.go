package app

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/Ryujoxys/sushiro-overdose/internal/core"
	mcpassets "github.com/Ryujoxys/sushiro-overdose/mcp"
)

var mcpAssetsMu sync.Mutex

// Path discovery is read-only. Extraction only happens on explicit enable.
func findMCPDir() string {
	return filepath.Join(core.AppDirPath(), "mcp", mcpassets.Revision())
}

func materializeMCPAssets() (string, error) {
	mcpAssetsMu.Lock()
	defer mcpAssetsMu.Unlock()
	dir := findMCPDir()
	err := fs.WalkDir(mcpassets.Files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, filepath.FromSlash(path))
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		data, err := mcpassets.Files.ReadFile(path)
		if err != nil {
			return err
		}
		if existing, err := os.ReadFile(target); err == nil && bytes.Equal(existing, data) {
			return nil
		}
		return core.AtomicWriteFile(target, data, 0o600)
	})
	return dir, err
}
