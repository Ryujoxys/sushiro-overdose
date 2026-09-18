package app

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReleaseIdentityKeepsEditionSeparateFromVersion(t *testing.T) {
	previous := Version
	t.Cleanup(func() { Version = previous })
	SetVersion("4.0")
	w := httptest.NewRecorder()
	handleStatus(w, httptest.NewRequest("GET", "/api/status", nil))
	var status struct {
		Version string `json:"version"`
		Edition string `json:"edition"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Version != "4.0" || status.Edition != "lite正式版" {
		t.Fatalf("release identity: %+v", status)
	}
	if !strings.Contains(indexHTML, "<title>SUSHIRO Overdose · lite正式版</title>") {
		t.Fatal("page title is missing the edition")
	}
}
