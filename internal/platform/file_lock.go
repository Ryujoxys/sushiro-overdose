package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var ErrFileLocked = errors.New("本机文件正由另一个任务使用")

// The OS releases the lock when the descriptor closes, including after a crash.
// Keep the file in place: removing it would let two processes lock different inodes.
type FileLock struct {
	file *os.File
	once sync.Once
}

func (l *FileLock) Close() {
	if l != nil {
		l.once.Do(func() { _ = l.file.Close() })
	}
}

func TryFileLock(path string) (*FileLock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := lockFileDescriptor(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return &FileLock{file: f}, nil
}

func LockFile(ctx context.Context, path string) (*FileLock, error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		lock, err := TryFileLock(path)
		if !errors.Is(err, ErrFileLocked) {
			return lock, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(15 * time.Millisecond):
		}
	}
}
