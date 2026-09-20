package platform

import (
	"errors"
	"testing"
)

func TestIsolatedDataCannotModifySystemAutoStart(t *testing.T) {
	t.Setenv("SUSHIRO_DATA_HOME", t.TempDir())
	for _, status := range []AutoStartStatus{SamplingAutoStartStatus(), MCPAutoStartStatus()} {
		if status.Supported || status.Enabled || status.Path != "" || status.Message == "" {
			t.Fatalf("system autostart exposed to isolated profile: %+v", status)
		}
	}
	for _, change := range []func() error{
		InstallSamplingAutoStart, RemoveSamplingAutoStart, RepairSamplingAutoStart, InstallMCPAutoStart, RemoveMCPAutoStart,
	} {
		if err := change(); !errors.Is(err, errIsolatedAutoStart) {
			t.Fatalf("autostart did not reject isolated environment: %v", err)
		}
	}
}
