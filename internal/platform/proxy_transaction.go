package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/Ryujoxys/sushiro-overdose/internal/core"
)

var proxyTransactionMu sync.Mutex

// A durable snapshot must exist before the first mutation. A failed restore
// keeps it for repair/restart rather than silently discarding the user's proxy.
func applyProxyTransaction(path string, capture func() ([]byte, error), restore func([]byte) error, steps ...func() error) error {
	proxyTransactionMu.Lock()
	defer proxyTransactionMu.Unlock()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return fmt.Errorf("已有代理恢复备份或备份不可读，请先恢复代理")
	}
	data, err := capture()
	if err != nil {
		return fmt.Errorf("读取原代理配置失败: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := core.AtomicWriteFile(path, data, 0o600); err != nil {
		return err
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return errors.Join(err, restoreProxyTransactionLocked(path, restore))
		}
	}
	return nil
}

func restoreProxyTransaction(path string, restore func([]byte) error) error {
	proxyTransactionMu.Lock()
	defer proxyTransactionMu.Unlock()
	return restoreProxyTransactionLocked(path, restore)
}

func restoreProxyTransactionLocked(path string, restore func([]byte) error) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := restore(data); err != nil {
		return fmt.Errorf("代理恢复失败，备份已保留: %w", err)
	}
	return os.Remove(path)
}
