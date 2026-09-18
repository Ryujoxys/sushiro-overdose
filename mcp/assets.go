// Package mcpassets carries the Python adapter in every single-binary release.
package mcpassets

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
)

//go:embed pyproject.toml mcp_server/*.py docs/faq.md
var Files embed.FS

// Revision changes with the payload, keeping installed environments versioned.
func Revision() string {
	h := sha256.New()
	_ = fs.WalkDir(Files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			data, err := Files.ReadFile(path)
			if err != nil {
				return err
			}
			fmt.Fprintf(h, "%s\x00%d\x00", path, len(data))
			h.Write(data)
		}
		return nil
	})
	return fmt.Sprintf("%x", h.Sum(nil))[:20]
}
