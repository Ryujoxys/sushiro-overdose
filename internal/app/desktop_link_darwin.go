//go:build desktop && darwin

package app

// Wails' CLI normally adds this framework for native save dialogs. Our release
// builds use go build directly, so keep the linker requirement in the adapter.

// #cgo LDFLAGS: -framework UniformTypeIdentifiers -mmacosx-version-min=12.0
import "C"
