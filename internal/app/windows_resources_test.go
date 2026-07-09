package app

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestWindowsResourceSysoHasCleanManifest guards the PE resource objects that
// get linked into Windows builds (resource_windows_{amd64,arm64}.syso).
//
// A bad or incomplete SxS dependency declaration can surface on end-user PCs as:
//
//	应用程序无法启动，因为应用程序的并行配置不正确
//
// This binary is pure Go (CGO_ENABLED=0). Resources must ship icon + a minimal
// application manifest without Microsoft.VC*.CRT or Common-Controls v6 deps.
func TestWindowsResourceSysoHasCleanManifest(t *testing.T) {
	// syso files live at module root, not under internal/app.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))

	for _, arch := range []string{"amd64", "arm64"} {
		path := filepath.Join(root, "resource_windows_"+arch+".syso")
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v (run ./scripts/gen-windows-resources.sh)", path, err)
		}
		text := string(data)
		lower := strings.ToLower(text)

		for _, need := range []string{"sushirooverdose", "asinvoker", "permonitorv2"} {
			if !strings.Contains(lower, need) {
				t.Errorf("%s: missing required manifest marker %q — regenerate with scripts/gen-windows-resources.sh", path, need)
			}
		}
		for _, bad := range []string{
			"microsoft.vc",
			"microsoft.windows.common-controls",
		} {
			if strings.Contains(lower, bad) {
				t.Errorf("%s: forbidden SxS dependency %q present", path, bad)
			}
		}
		// Icon payload should still be embedded as PNG chunks.
		if !strings.Contains(text, "IHDR") {
			t.Errorf("%s: expected PNG icon data (IHDR) missing", path)
		}
	}
}
