//go:build windows

package platform

import (
	"errors"
	"os/exec"
	"testing"
)

func TestRegDeleteClassifiesCapturedOutputWithoutCallingRegistry(t *testing.T) {
	for _, message := range []string{"ERROR: The system was unable to find the specified registry key or value.", "ERROR: value was not found", "错误: 找不到指定的值"} {
		if !isRegMissingValue([]byte(message), &exec.ExitError{}) {
			t.Fatal("missing registry value must be idempotent", message)
		}
	}
	if isRegMissingValue([]byte("ERROR: Access is denied."), &exec.ExitError{}) || isRegMissingValue([]byte("was not found"), errors.New("cannot launch reg")) {
		t.Fatal("real registry or process error hidden")
	}
}
