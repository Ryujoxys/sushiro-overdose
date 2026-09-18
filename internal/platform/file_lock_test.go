package platform

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestFileLockExcludesOtherDescriptorsAndProcesses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writer.lock")
	lock, err := TryFileLock(path)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if second, err := TryFileLock(path); !errors.Is(err, ErrFileLocked) {
		second.Close()
		t.Fatalf("second lock = %v", err)
	}
	child := func() error {
		cmd := exec.Command(os.Args[0], "-test.run=^TestFileLockHelper$")
		cmd.Env = append(os.Environ(), "SUSHIRO_TEST_LOCK="+path)
		return cmd.Run()
	}
	if err := child(); err == nil {
		t.Fatal("another process acquired the lock")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := LockFile(ctx, path); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	lock.Close()
	if err := child(); err != nil {
		t.Fatalf("lock not released: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("lock file must not be unlinked")
	}
}

func TestFileLockHelper(t *testing.T) {
	path := os.Getenv("SUSHIRO_TEST_LOCK")
	if path == "" {
		return
	}
	lock, err := TryFileLock(path)
	if err != nil {
		os.Exit(2)
	}
	lock.Close()
	os.Exit(0)
}
