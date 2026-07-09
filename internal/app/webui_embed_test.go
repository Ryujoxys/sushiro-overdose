package app

import (
	"strings"
	"testing"
)

// TestAssembleIndexHTMLFromEmbeddedSources drives the real shipped assembly path:
// read webui/* via webuiFS (same as init), assemble, assert the document is the
// real UI shell (not an empty stub). This fails if embed paths break or
// placeholders are missing.
func TestAssembleIndexHTMLFromEmbeddedSources(t *testing.T) {
	html, err := webuiFS.ReadFile("webui/index.html")
	if err != nil {
		t.Fatalf("read index.html: %v", err)
	}
	css, err := webuiFS.ReadFile("webui/app.css")
	if err != nil {
		t.Fatalf("read app.css: %v", err)
	}
	js, err := webuiFS.ReadFile("webui/app.js")
	if err != nil {
		t.Fatalf("read app.js: %v", err)
	}
	logo, err := webuiFS.ReadFile("webui/logo.b64")
	if err != nil {
		t.Fatalf("read logo.b64: %v", err)
	}

	got := assembleIndexHTML(string(html), string(css), string(js), strings.TrimSpace(string(logo)))
	if got != indexHTML {
		t.Fatalf("assembleIndexHTML != package indexHTML (init path diverged): len(got)=%d len(indexHTML)=%d", len(got), len(indexHTML))
	}
	for _, needle := range []string{
		"<!DOCTYPE html>",
		`meta name="sushiro-csrf" content="{{CSRF_TOKEN}}"`,
		"<style>",
		"</style>",
		"<script>",
		"</script>",
		"function go(",
		"init();",
		"</html>",
	} {
		if !strings.Contains(got, needle) {
			t.Errorf("assembled UI missing %q", needle)
		}
	}
	// Placeholders must be consumed (except CSRF which is injected at request time).
	if strings.Contains(got, "{{APP_CSS}}") || strings.Contains(got, "{{APP_JS}}") || strings.Contains(got, "{{LOGO_BASE64}}") {
		t.Fatal("assembled UI still contains unexpanded APP/LOGO placeholders")
	}
	if !strings.Contains(got, "{{CSRF_TOKEN}}") {
		t.Fatal("assembled UI must keep {{CSRF_TOKEN}} for handleHome injection")
	}
	// Logo must appear as data URI material (real base64, not placeholder).
	if !strings.Contains(got, "data:image/png;base64,iVBOR") {
		t.Fatal("assembled UI missing real PNG logo data URI")
	}
}
