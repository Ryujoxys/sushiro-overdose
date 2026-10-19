package app

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testDesktopBridge(t *testing.T, handler http.HandlerFunc) *DesktopBridge {
	t.Helper()
	old := getWebCSRFToken()
	setWebCSRFToken("desktop-test-token")
	t.Cleanup(func() { setWebCSRFToken(old) })
	server := httptest.NewServer(webSecurityMiddleware(handler))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	return &DesktopBridge{backend: &webBackend{URL: server.URL, ctx: ctx}, client: localHTTPClient()}
}

func TestDesktopBridgeKeepsWebSecurity(t *testing.T) {
	calls := 0
	b := testDesktopBridge(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.String() != "/api/records/settings" || r.Method != "POST" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		writeJSON(w, map[string]bool{"included": false})
	})
	response, err := b.Request("POST", "/api/records/settings", `{"include_history":false}`, 1000)
	if err != nil || response.Status != 200 || calls != 1 {
		t.Fatalf("request failed: %+v, %v, calls=%d", response, err, calls)
	}
}

func TestDesktopBridgeRejectsUntrustedRequests(t *testing.T) {
	b := testDesktopBridge(t, func(http.ResponseWriter, *http.Request) { t.Error("invalid request reached handler") })
	for _, resource := range []string{
		"https://example.com/api/status", "//127.0.0.1/api/status", "/api/../api/status",
		"/api/%73tatus", "/api/status#fragment", "/api/status\\other", "/api/events",
		"/api/uninstall", "/api/records/export", "/", "/api/queue/ticket",
	} {
		if _, err := b.Request("GET", resource, "", 1000); err == nil {
			t.Errorf("accepted GET %q", resource)
		}
	}
	for _, method := range []string{"", "PUT", "DELETE", "get"} {
		if _, err := b.Request(method, "/api/status", "", 1000); err == nil {
			t.Errorf("accepted method %q", method)
		}
	}
	if _, err := b.Request("POST", "/api/auth/import", strings.Repeat("x", (1<<20)+1), 1000); err == nil {
		t.Fatal("accepted oversized body")
	}
}

func TestDesktopBridgeTimeoutAndRedirect(t *testing.T) {
	b := testDesktopBridge(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("redirect") == "1" {
			http.Redirect(w, r, "/api/uninstall", http.StatusTemporaryRedirect)
			return
		}
		if r.URL.Path != "/api/status" {
			t.Error("followed redirect")
		}
		<-r.Context().Done()
	})
	if _, err := b.Request("GET", "/api/status", "", 20); err == nil {
		t.Fatal("request did not time out")
	}
	response, err := b.Request("GET", "/api/status?redirect=1", "", 1000)
	if err != nil || response.Status != http.StatusTemporaryRedirect {
		t.Fatalf("redirect response: %+v %v", response, err)
	}
}

func TestDesktopExportCancelAndSave(t *testing.T) {
	calls := 0
	b := testDesktopBridge(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.String() != "/api/records/export?days=all" || r.Method != "GET" {
			t.Errorf("unexpected export: %s %s", r.Method, r.URL)
		}
		_, _ = w.Write([]byte("{\"store_id\":1012}\n"))
	})
	b.saveDialog = func(string) (string, error) { return "", nil }
	if saved, err := b.Export("/api/records/export?days=all"); saved || err != nil || calls != 0 {
		t.Fatalf("cancel had side effects: %v %v calls=%d", saved, err, calls)
	}
	for _, resource := range []string{"/api/status", "file:///tmp/data", "/api/diagnostics/bundle?file=config.json"} {
		if _, err := b.Export(resource); err == nil {
			t.Errorf("accepted invalid export %q", resource)
		}
	}
	target := filepath.Join(t.TempDir(), "records.jsonl")
	b.saveDialog = func(name string) (string, error) {
		if name != recordsExportFilename(time.Now()) {
			t.Errorf("unexpected suggested name %q", name)
		}
		return target, nil
	}
	if saved, err := b.Export("/api/records/export?days=all"); !saved || err != nil {
		t.Fatalf("save failed: %v %v", saved, err)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "{\"store_id\":1012}\n" {
		t.Fatalf("bad export: %q %v", data, err)
	}
}

func TestDesktopExportFailureKeepsExistingFile(t *testing.T) {
	b := testDesktopBridge(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "failed", http.StatusInternalServerError)
	})
	target := filepath.Join(t.TempDir(), "existing.jsonl")
	if err := os.WriteFile(target, []byte("keep me"), 0o600); err != nil {
		t.Fatal(err)
	}
	b.saveDialog = func(string) (string, error) { return target, nil }
	if saved, err := b.Export("/api/records/export"); saved || err == nil {
		t.Fatal("saved failed response")
	}
	data, _ := os.ReadFile(target)
	if string(data) != "keep me" {
		t.Fatal("overwrote existing data on failure")
	}
}

func TestLocalWebListenerHoldsPort(t *testing.T) {
	first, err := listenLocalWeb(0)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	second, err := listenLocalWeb(first.Addr().(*net.TCPAddr).Port)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	if first.Addr().String() == second.Addr().String() {
		t.Fatal("reused an occupied port")
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })}
	go server.Serve(first)
	defer server.Close()
	client := localHTTPClient()
	client.Timeout = time.Second
	resp, err := client.Get("http://" + first.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 204 {
		t.Fatal(resp.StatusCode)
	}
}

func TestDesktopCloseWaitsForExplicitWrite(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	b := testDesktopBridge(t, func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		writeJSON(w, map[string]bool{"ok": true})
	})
	done := make(chan error, 1)
	go func() {
		_, err := b.Request("POST", "/api/records/settings", "{}", 1000)
		done <- err
	}()
	<-started
	if b.prepareClose() {
		t.Error("closed during an explicit write")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !b.prepareClose() {
		t.Fatal("completed write prevented closing")
	}
	if _, err := b.Request("POST", "/api/records/settings", "{}", 1000); err == nil {
		t.Fatal("accepted a write after closing began")
	}
}

func TestDesktopBridgeUpdateCheckIsReadOnlyAndReleasePageIsFixed(t *testing.T) {
	b := testDesktopBridge(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/update" || r.Method != "GET" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		writeJSON(w, UpdateInfo{CurrentVersion: "4.0.2"})
	})
	if response, err := b.Request("GET", "/api/update", "", 1000); err != nil || response.Status != 200 {
		t.Fatalf("update check failed: %+v, %v", response, err)
	}
	if _, err := b.Request("POST", "/api/update", "{}", 1000); err == nil {
		t.Fatal("accepted POST update check")
	}
	if err := b.OpenReleasePage(); err == nil {
		t.Fatal("opened a page before the native window was ready")
	}
	var opened []string
	b.openURL = func(url string) { opened = append(opened, url) }
	if err := b.OpenReleasePage(); err != nil || len(opened) != 1 || opened[0] != releasePageURL {
		t.Fatalf("release page: %v, %v", opened, err)
	}
	if !strings.HasPrefix(releasePageURL, "https://github.com/Ryujoxys/sushiro-overdose/releases/") {
		t.Fatalf("unexpected release page: %s", releasePageURL)
	}
}

func TestDesktopBridgeImportUsesNativeDialogOnly(t *testing.T) {
	var received string
	b := testDesktopBridge(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/records/import" || r.Method != "POST" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL)
		}
		data, _ := io.ReadAll(r.Body)
		received = string(data)
		writeJSON(w, localRecordsImportResult{Imported: 1})
	})
	if _, err := b.Request("POST", "/api/records/import", "{}", 1000); err == nil {
		t.Fatal("page could post an import body directly")
	}
	if _, err := b.ImportRecords(); err == nil {
		t.Fatal("imported before the native dialog was ready")
	}
	b.openDialog = func() (string, error) { return "", nil }
	if response, err := b.ImportRecords(); err != nil || response.Status != 0 || received != "" {
		t.Fatalf("cancelled dialog: %+v %v %q", response, err, received)
	}
	source := filepath.Join(t.TempDir(), "backup.jsonl")
	if err := os.WriteFile(source, []byte("{\"store_id\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	b.openDialog = func() (string, error) { return source, nil }
	response, err := b.ImportRecords()
	if err != nil || response.Status != 200 || received != "{\"store_id\":1}\n" {
		t.Fatalf("import: %+v %v %q", response, err, received)
	}
}
