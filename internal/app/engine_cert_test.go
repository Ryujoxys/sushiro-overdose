package app

import (
	"fmt"
	"testing"

	"github.com/Ryujoxys/sushiro-overdose/internal/platform"
)

func TestMissingKeychainIsNotAnInstallRetry(t *testing.T) {
	engine := &BookingEngine{}
	engine.classifyCertError(fmt.Errorf("install: %w", platform.ErrUserKeychainUnavailable))
	if engine.state.ErrorKind != ErrKindCertKeychain {
		t.Fatalf("unexpected error kind: %s", engine.state.ErrorKind)
	}
	engine.classifyCertError(fmt.Errorf("other install error"))
	if engine.state.ErrorKind != ErrKindCertKeychain {
		t.Fatal("specific keychain error overwritten")
	}
}
