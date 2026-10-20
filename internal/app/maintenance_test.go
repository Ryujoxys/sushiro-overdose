package app

import . "github.com/Ryujoxys/sushiro-overdose/internal/proxy"

import . "github.com/Ryujoxys/sushiro-overdose/internal/notify"

import . "github.com/Ryujoxys/sushiro-overdose/internal/core"

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUninstallLocalDataAllRemovesSensitiveFiles(t *testing.T) {
	setMaintenanceTestHome(t)

	removed := []string{
		LocalConfigPath(),
		liveSessionPath(),
		NotifyConfigPath(),
		FeishuConfigPath(),
		PreferencesPath(),
		StoreRegistryPath(),
		StateFilePath(),
		historyPath(),
		PidFilePath(),
		proxyStatePath(),
		filepath.Join(CertDirPath(), "ca.crt"),
		filepath.Join(CertDirPath(), "ca.key"),
	}
	for _, path := range removed {
		writeMaintenanceTestFile(t, path)
	}

	kept := []string{
		LogPath(),
		filepath.Join(CertDirPath(), "leaf.cache"),
	}
	for _, path := range kept {
		writeMaintenanceTestFile(t, path)
	}

	report := UninstallLocalData(UninstallOptions{All: true})
	if !report.OK {
		t.Fatalf("expected report OK, got %#v", report)
	}
	if len(report.Results) != len(removed) {
		t.Fatalf("expected %d results, got %d", len(removed), len(report.Results))
	}

	for _, path := range removed {
		assertMaintenancePathMissing(t, path)
	}
	for _, path := range kept {
		assertMaintenancePathExists(t, path)
	}

	statusByName := maintenanceStatusesByName(report)
	for _, name := range []string{"config", "live_session", "notify", "feishu", "preferences", "stores", "state", "history", "pid", "proxy_marker", "ca_cert", "ca_key"} {
		if statusByName[name] != MaintenanceStatusOK {
			t.Fatalf("expected %s status %q, got %q", name, MaintenanceStatusOK, statusByName[name])
		}
	}
}

func TestUninstallLocalDataPartialSelectionKeepsUnselectedFiles(t *testing.T) {
	setMaintenanceTestHome(t)

	configPath := LocalConfigPath()
	notifyPath := NotifyConfigPath()
	certPath := filepath.Join(CertDirPath(), "ca.crt")
	writeMaintenanceTestFile(t, configPath)
	writeMaintenanceTestFile(t, notifyPath)
	writeMaintenanceTestFile(t, certPath)

	report := UninstallLocalData(UninstallOptions{Config: true, Certificates: true})
	if !report.OK {
		t.Fatalf("expected report OK, got %#v", report)
	}

	assertMaintenancePathMissing(t, configPath)
	assertMaintenancePathMissing(t, certPath)
	assertMaintenancePathExists(t, notifyPath)

	statusByName := maintenanceStatusesByName(report)
	if statusByName["config"] != MaintenanceStatusOK {
		t.Fatalf("expected config removed, got %q", statusByName["config"])
	}
	if statusByName["ca_cert"] != MaintenanceStatusOK {
		t.Fatalf("expected ca_cert removed, got %q", statusByName["ca_cert"])
	}
	if statusByName["ca_key"] != MaintenanceStatusMissing {
		t.Fatalf("expected missing ca_key result, got %q", statusByName["ca_key"])
	}
	if _, ok := statusByName["notify"]; ok {
		t.Fatalf("notify was not selected but appeared in results: %#v", report.Results)
	}
}

func TestUninstallLocalDataDryRunKeepsFiles(t *testing.T) {
	setMaintenanceTestHome(t)

	configPath := LocalConfigPath()
	writeMaintenanceTestFile(t, configPath)

	report := UninstallLocalData(UninstallOptions{Config: true, DryRun: true})
	if !report.OK {
		t.Fatalf("expected report OK, got %#v", report)
	}
	assertMaintenancePathExists(t, configPath)

	statusByName := maintenanceStatusesByName(report)
	if statusByName["config"] != MaintenanceStatusWouldRemove {
		t.Fatalf("expected dry-run status %q, got %q", MaintenanceStatusWouldRemove, statusByName["config"])
	}
}

func TestUninstallLocalDataNoSelectionIsSkipped(t *testing.T) {
	setMaintenanceTestHome(t)

	report := UninstallLocalData(UninstallOptions{})
	if !report.OK {
		t.Fatalf("expected report OK, got %#v", report)
	}
	if len(report.Results) != 1 {
		t.Fatalf("expected one skipped result, got %d", len(report.Results))
	}
	if report.Results[0].Status != MaintenanceStatusSkipped {
		t.Fatalf("expected skipped status, got %q", report.Results[0].Status)
	}
}

func setMaintenanceTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func writeMaintenanceTestFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("test"), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func assertMaintenancePathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be missing, stat err=%v", path, err)
	}
}

func assertMaintenancePathExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func maintenanceStatusesByName(report MaintenanceReport) map[string]string {
	out := make(map[string]string, len(report.Results))
	for _, result := range report.Results {
		out[result.Name] = result.Status
	}
	return out
}
