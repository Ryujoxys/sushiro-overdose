package platform

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestProxyTransactionRollsBackEveryPartialFailure(t *testing.T) {
	for failAt := 0; failAt < 5; failAt++ {
		t.Run(string(rune('0'+failAt)), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "backup.json")
			value, restoreCount := "original-user-proxy", 0
			capture := func() ([]byte, error) { return []byte(value), nil }
			restore := func(data []byte) error { value = string(data); restoreCount++; return nil }
			var steps []func() error
			for i := 0; i < 5; i++ {
				steps = append(steps, func() error {
					if _, err := os.Stat(path); err != nil {
						t.Fatal("mutation before backup")
					}
					value = "changed"
					if i == failAt {
						return errors.New("injected registry failure")
					}
					return nil
				})
			}
			if err := applyProxyTransaction(path, capture, restore, steps...); err == nil {
				t.Fatal("expected failure")
			}
			if value != "original-user-proxy" || restoreCount != 1 {
				t.Fatalf("rollback: %q, %d", value, restoreCount)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("successful rollback left backup")
			}
		})
	}
}

func TestProxyTransactionKeepsFailedRestoreForRepair(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.json")
	value := "original"
	capture := func() ([]byte, error) { return []byte(value), nil }
	fail := func() error { value = "changed"; return errors.New("write failed") }
	if err := applyProxyTransaction(path, capture, func([]byte) error { return errors.New("restore failed") }, fail); err == nil {
		t.Fatal("expected failure")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "original" {
		t.Fatalf("lost original backup: %s %v", data, err)
	}
	if err := applyProxyTransaction(path, capture, func([]byte) error { return nil }); err == nil {
		t.Fatal("replaced outstanding backup")
	}
	if err := restoreProxyTransaction(path, func(data []byte) error { value = string(data); return nil }); err != nil {
		t.Fatal(err)
	}
	if value != "original" {
		t.Fatal(value)
	}
}

func TestProxyTransactionNoMutationWithoutSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.json")
	mutated := false
	err := applyProxyTransaction(path, func() ([]byte, error) { return nil, errors.New("snapshot denied") }, func([]byte) error { t.Fatal("restore without snapshot"); return nil }, func() error { mutated = true; return nil })
	if err == nil || mutated {
		t.Fatal("must fail before mutation")
	}
}

func TestProxyTransactionRestoresAfterSuccessfulRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "backup.json")
	value := "original"
	restore := func(data []byte) error { value = string(data); return nil }
	if err := applyProxyTransaction(path, func() ([]byte, error) { return []byte(value), nil }, restore, func() error { value = "temporary"; return nil }); err != nil {
		t.Fatal(err)
	}
	if value != "temporary" {
		t.Fatal(value)
	}
	if err := restoreProxyTransaction(path, restore); err != nil {
		t.Fatal(err)
	}
	if value != "original" {
		t.Fatal(value)
	}
}
