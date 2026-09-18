package proxy

import (
	"path/filepath"
	"testing"
)

func TestCertDirectoryUsesApplicationDataHome(t *testing.T) {
	home, isolated := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SUSHIRO_DATA_HOME", isolated)
	if got := CertDirPath(); got != filepath.Join(isolated, ".sushiro-proxy") {
		t.Fatalf("certificate escaped isolation: %s", got)
	}
	t.Setenv("SUSHIRO_DATA_HOME", "")
	if got := CertDirPath(); got != filepath.Join(home, ".sushiro-proxy") {
		t.Fatalf("default certificate directory changed: %s", got)
	}
}
