package app

import (
	"embed"
	"strings"
)

// Web UI sources live under webui/ as plain HTML/CSS/JS (plus logo base64).
// They are embedded into the binary at build time and assembled into indexHTML
// so the local server still serves a single HTML document with the same shape
// as before (inline CSS/JS, data-URI logo, {{CSRF_TOKEN}} placeholder).
//
// Edit webui/* for UI changes — do not re-introduce a monolithic string blob here.

//go:embed webui/index.html webui/app.css webui/app.js webui/logo.b64
var webuiFS embed.FS

// indexHTML is the full document served at GET /. Built once at init from webui/.
// Tests and handleHome treat this as the canonical UI string (same contract as
// when it was a hand-maintained Go raw string).
var indexHTML string

func init() {
	indexHTML = mustAssembleIndexHTML()
}

func mustAssembleIndexHTML() string {
	html, err := webuiFS.ReadFile("webui/index.html")
	if err != nil {
		panic("webui: read index.html: " + err.Error())
	}
	css, err := webuiFS.ReadFile("webui/app.css")
	if err != nil {
		panic("webui: read app.css: " + err.Error())
	}
	js, err := webuiFS.ReadFile("webui/app.js")
	if err != nil {
		panic("webui: read app.js: " + err.Error())
	}
	logo, err := webuiFS.ReadFile("webui/logo.b64")
	if err != nil {
		panic("webui: read logo.b64: " + err.Error())
	}
	return assembleIndexHTML(string(html), string(css), string(js), strings.TrimSpace(string(logo)))
}

// assembleIndexHTML injects CSS, JS, and logo into the HTML template placeholders.
func assembleIndexHTML(html, css, js, logoBase64 string) string {
	out := html
	out = strings.ReplaceAll(out, "{{APP_CSS}}", css)
	out = strings.ReplaceAll(out, "{{APP_JS}}", js)
	out = strings.ReplaceAll(out, "{{LOGO_BASE64}}", logoBase64)
	return out
}
