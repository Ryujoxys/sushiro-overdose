package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAutoStartTargetHealth(t *testing.T) {
	target := filepath.Join(t.TempDir(), "sushiro")
	if err := os.WriteFile(target, []byte("fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, target, current string
		enabled, update       bool
	}{
		{"current", target, target, true, false},
		{"moved", target, target + "-new", true, true},
		{"missing", target + "-old", target, true, true},
		{"malformed", "", target, true, true},
		{"disabled", "", target, false, false},
		{"directory", filepath.Dir(target), target, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := assessAutoStartTarget(AutoStartStatus{Enabled: test.enabled}, test.target, test.current, false, os.Stat)
			if s.NeedsUpdate != test.update || s.Enabled != test.enabled || (test.update && s.Error == "") {
				t.Fatalf("status: %+v", s)
			}
		})
	}
	info, _ := os.Stat(target)
	s := assessAutoStartTarget(AutoStartStatus{Enabled: true}, `C:\Apps\Sushiro.exe`, `c:\apps\sushiro.exe`, true, func(string) (os.FileInfo, error) { return info, nil })
	if s.NeedsUpdate {
		t.Fatalf("case-only Windows path mismatch: %+v", s)
	}
}

func TestCollectorRegistrationParsers(t *testing.T) {
	command := `"C:\Users\Test User\Sushiro 4.0.1.exe" --queue-collector-child`
	if got := collectorCommandTarget(command); got != `C:\Users\Test User\Sushiro 4.0.1.exe` {
		t.Fatal(got)
	}
	for _, command := range []string{`"C:\Apps\sushiro.exe" --daemon-child`, `"C:\Apps\sushiro.exe" --queue-collector-child extra`, ""} {
		if collectorCommandTarget(command) != "" {
			t.Fatalf("accepted %q", command)
		}
	}
	data := []byte(`<?xml version="1.0"?><plist><dict><key>Label</key><string>sampler</string><key>ProgramArguments</key><array><string>/Applications/A &amp; B.app/sushiro</string><string>--queue-collector-child</string></array></dict></plist>`)
	if got := plistCollectorTarget(data); got != "/Applications/A & B.app/sushiro" {
		t.Fatal(got)
	}
	if plistCollectorTarget([]byte(`<plist><key>ProgramArguments</key><string>not an array</string></plist>`)) != "" {
		t.Fatal("accepted malformed plist")
	}
}

func TestAutoStartRepairOnlyUpdatesExistingRegistration(t *testing.T) {
	calls := 0
	repair := func() error { calls++; return nil }
	if err := repairRegisteredAutoStart(AutoStartStatus{}, repair); err == nil || calls != 0 {
		t.Fatal("repair enabled a new registration")
	}
	if err := repairRegisteredAutoStart(AutoStartStatus{Enabled: true, NeedsUpdate: true}, repair); err != nil || calls != 1 {
		t.Fatal("existing registration was not repaired", err)
	}
}
