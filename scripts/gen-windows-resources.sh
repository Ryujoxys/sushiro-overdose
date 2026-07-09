#!/usr/bin/env bash
# Regenerate resource_windows_{amd64,arm64}.syso for go build.
#
# Goal: click-to-run Windows EXE (-H windowsgui) must not fail with
# "side-by-side configuration is incorrect" due to bad PE resources.
#
# Policy (enforced below):
#   - Embed icon + application manifest via go-winres
#   - use-common-controls-v6 = false  (do not declare comctl32 SxS)
#   - never declare Microsoft.VC*.CRT
#   - dpi-awareness = per monitor v2, execution-level = as invoker
#
# Sources:
#   winres/winres.json
#   assets/windows/icons/icon_{16,32,48,64,128,256}.png
#   assets/windows/app.manifest   (human-readable policy; go-winres uses winres.json)
#
# Usage (from repo root):
#   ./scripts/gen-windows-resources.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="$(go env GOPATH)/bin:${PATH:-}"

if ! command -v go-winres >/dev/null 2>&1; then
  echo "installing go-winres..."
  go install github.com/tc-hib/go-winres@latest
fi

for s in 16 32 48 64 128 256; do
  f="assets/windows/icons/icon_${s}.png"
  if [[ ! -f "$f" ]]; then
    echo "missing $f" >&2
    echo "Generate with: sips -z $s $s assets/icons/sushiro-1024.png --out $f" >&2
    exit 1
  fi
done
[[ -f winres/winres.json ]] || { echo "missing winres/winres.json" >&2; exit 1; }

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

# Paths in winres.json are relative to the json file (winres/).
go-winres make \
  --in winres/winres.json \
  --arch amd64,arm64 \
  --out "$tmpdir/rsrc" \
  --product-version 0.0.0.0 \
  --file-version 0.0.0.0

for arch in amd64 arm64; do
  src="$tmpdir/rsrc_windows_${arch}.syso"
  if [[ ! -f "$src" ]]; then
    echo "go-winres did not produce $src" >&2
    ls -la "$tmpdir" >&2
    exit 1
  fi
  cp "$src" "resource_windows_${arch}.syso"
  echo "wrote resource_windows_${arch}.syso ($(wc -c <"resource_windows_${arch}.syso") bytes)"
done

python3 - <<'PY'
from pathlib import Path

def check(arch: str) -> None:
    data = Path(f"resource_windows_{arch}.syso").read_bytes()
    # go-winres embeds manifest as UTF-8 XML inside .rsrc
    text = data.decode("latin1", errors="ignore")
    required = [
        "SushiroOverdose",
        "asInvoker",
        # permonitorv2 appears in dpiAwareness value
        "permonitorv2",
    ]
    lower = text.lower()
    for r in required:
        if r.lower() not in lower and r not in text:
            raise SystemExit(f"{arch}: missing required manifest marker {r!r}")
    forbidden = [
        "Microsoft.VC",
        "Microsoft.Windows.Common-Controls",
        "common-controls",
    ]
    for f in forbidden:
        if f.lower() in lower:
            # allow comment-less accidental; these must not appear as assembly deps
            raise SystemExit(f"{arch}: forbidden SxS dependency marker {f!r}")
    if b"IHDR" not in data:
        raise SystemExit(f"{arch}: expected PNG icon data (IHDR) missing")
    print(f"ok {arch}: icon+clean manifest, size={len(data)}")

for a in ("amd64", "arm64"):
    check(a)
PY

echo
echo "Cross-compile smoke (GUI + console):"
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags "-s -w -H windowsgui -X main.Version=dev-win-gui" \
  -o "$tmpdir/Sushiro-Overdose-dev-windows-amd64.exe" .
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build \
  -ldflags "-s -w -X main.Version=dev-win-console" \
  -o "$tmpdir/sushiro-dev-windows-amd64.exe" .
ls -la "$tmpdir"/*.exe
echo "Done. Review and commit resource_windows_*.syso + assets/windows/* + winres/"
