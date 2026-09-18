//go:build darwin || linux

package platform

import (
	"errors"
	"os"
	"syscall"
)

func lockFileDescriptor(f *os.File) error {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
		return ErrFileLocked
	}
	return err
}
