#!/bin/bash
set -euo pipefail

# Bundle sushiro-overdose into a macOS .app and clickable .dmg
# Usage: ./scripts/bundle-macos.sh <binary-path> <version> <output-dir>

BINARY="${1:?Usage: $0 <binary> <version> <output-dir>}"
VERSION="${2:?}"
OUTPUT_DIR="${3:?}"

APP_NAME="Sushiro Overdose"
BUNDLE_ID="com.sushiro-overdose.app"
APP_DIR="${OUTPUT_DIR}/${APP_NAME}.app"
DMG_STAGING_DIR="${OUTPUT_DIR}/dmg-staging"
DMG_NAME="Sushiro-Overdose-${VERSION}-macOS.dmg"

rm -rf "${APP_DIR}"
mkdir -p "${APP_DIR}/Contents/MacOS"
mkdir -p "${APP_DIR}/Contents/Resources"

cp "${BINARY}" "${APP_DIR}/Contents/MacOS/sushiro"
chmod +x "${APP_DIR}/Contents/MacOS/sushiro"

# 应用图标：仓库里的 sushiro.icns 嵌进 .app，让 DMG 和安装后的 App 都显示 sushiro 图标。
ICON_SRC="$(cd "$(dirname "$0")/.." && pwd)/assets/icons/sushiro.icns"
if [ -f "${ICON_SRC}" ]; then
  cp "${ICON_SRC}" "${APP_DIR}/Contents/Resources/sushiro.icns"
  APP_ICON_KEY="<key>CFBundleIconFile</key><string>sushiro.icns</string>"
else
  echo "Warning: ${ICON_SRC} not found; .app will use default icon."
  APP_ICON_KEY=""
fi

cat > "${APP_DIR}/Contents/Info.plist" << PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>zh_CN</string>
    <key>CFBundleExecutable</key>
    <string>sushiro</string>
    <key>CFBundleIdentifier</key>
    <string>${BUNDLE_ID}</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleDisplayName</key>
    <string>${APP_NAME}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>LSMinimumSystemVersion</key>
    <string>12.0</string>
    <key>LSApplicationCategoryType</key>
    <string>public.app-category.utilities</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSUIElement</key>
    <false/>
    ${APP_ICON_KEY}
</dict>
</plist>
PLIST

if [ -n "${MACOS_CODESIGN_IDENTITY:-}" ]; then
  echo "Codesigning app with identity: ${MACOS_CODESIGN_IDENTITY}"
  codesign --force --deep --options runtime --timestamp --sign "${MACOS_CODESIGN_IDENTITY}" "${APP_DIR}"
else
  echo "MACOS_CODESIGN_IDENTITY is not set; creating an unsigned app."
fi

rm -rf "${DMG_STAGING_DIR}"
mkdir -p "${DMG_STAGING_DIR}"
cp -R "${APP_DIR}" "${DMG_STAGING_DIR}/"
ln -s /Applications "${DMG_STAGING_DIR}/Applications"

hdiutil create \
  -volname "${APP_NAME}" \
  -srcfolder "${DMG_STAGING_DIR}" \
  -ov \
  -format UDZO \
  "${OUTPUT_DIR}/${DMG_NAME}"

if [ -n "${MACOS_CODESIGN_IDENTITY:-}" ]; then
  echo "Codesigning DMG with identity: ${MACOS_CODESIGN_IDENTITY}"
  codesign --force --timestamp --sign "${MACOS_CODESIGN_IDENTITY}" "${OUTPUT_DIR}/${DMG_NAME}"
fi

if [ -n "${MACOS_NOTARY_APPLE_ID:-}" ] && [ -n "${MACOS_NOTARY_PASSWORD:-}" ] && [ -n "${MACOS_NOTARY_TEAM_ID:-}" ]; then
  echo "Submitting DMG for notarization"
  xcrun notarytool submit "${OUTPUT_DIR}/${DMG_NAME}" \
    --apple-id "${MACOS_NOTARY_APPLE_ID}" \
    --password "${MACOS_NOTARY_PASSWORD}" \
    --team-id "${MACOS_NOTARY_TEAM_ID}" \
    --wait
  xcrun stapler staple "${OUTPUT_DIR}/${DMG_NAME}"
else
  echo "Notarization credentials are not set; skipping notarization."
fi

rm -rf "${DMG_STAGING_DIR}"

echo "Created: ${OUTPUT_DIR}/${DMG_NAME}"
