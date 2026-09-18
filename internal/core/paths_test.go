package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDataHomeDoesNotReplaceSystemHome(t *testing.T) {
	home, isolated := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SUSHIRO_DATA_HOME", isolated)
	if err := ValidateDataHome(); err != nil {
		t.Fatal(err)
	}
	if AppDirPath() != filepath.Join(isolated, ".sushiro") {
		t.Fatalf("unexpected data path: %s", AppDirPath())
	}
	if got, _ := os.UserHomeDir(); got != home {
		t.Fatalf("system home changed: %s", got)
	}
	t.Setenv("SUSHIRO_DATA_HOME", "")
	if HasCustomDataHome() || AppDirPath() != filepath.Join(home, ".sushiro") {
		t.Fatal("default data path is not backward compatible")
	}
}

func TestValidateDataHomeRejectsRelativePath(t *testing.T) {
	t.Setenv("SUSHIRO_DATA_HOME", "relative-preview")
	if ValidateDataHome() == nil {
		t.Fatal("relative data home accepted")
	}
}
