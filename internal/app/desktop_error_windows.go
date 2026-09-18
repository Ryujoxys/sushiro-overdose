//go:build desktop && windows

package app

import (
	"syscall"
	"unsafe"
)

func showDesktopError(message string) {
	body, err := syscall.UTF16PtrFromString(message)
	if err != nil {
		return
	}
	title, _ := syscall.UTF16PtrFromString("无法打开寿司郎")
	syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(body)), uintptr(unsafe.Pointer(title)), 0x10)
}
