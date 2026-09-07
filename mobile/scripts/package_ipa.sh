#!/usr/bin/env bash
set -euo pipefail

# Ensure script executes relative to mobile project directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

echo "==> Packaging iOS IPA for Sideloading..."
APP_PATH="build/ios/archive/Runner.xcarchive/Products/Applications/Runner.app"
if [ ! -d "$APP_PATH" ]; then
  APP_PATH="build/ios/iphoneos/Runner.app"
fi

if [ ! -d "$APP_PATH" ]; then
  echo "Error: Runner.app not found at $APP_PATH"
  exit 1
fi

rm -rf Payload KnightOnline.ipa
mkdir -p Payload
cp -r "$APP_PATH" Payload/
zip -qr KnightOnline.ipa Payload
rm -rf Payload
echo "==> Successfully created KnightOnline.ipa ($(du -h KnightOnline.ipa | cut -f1))"
