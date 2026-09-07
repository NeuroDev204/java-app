#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

# Last golang.org/x/mobile that still builds with Go 1.24 (later tags need 1.26).
MOBILE_VER=v0.0.0-20260209203831-923679eb55af
export GOTOOLCHAIN=local
export CGO_ENABLED=1
export PATH="$PATH:$(go env GOPATH)/bin"

go get "golang.org/x/mobile@${MOBILE_VER}"
go install "golang.org/x/mobile/cmd/gomobile@${MOBILE_VER}"
go install "golang.org/x/mobile/cmd/gobind@${MOBILE_VER}"
# Skip gomobile init: it installs gobind@latest (needs Go 1.26).

if [[ -n "${ANDROID_HOME:-}${ANDROID_SDK_ROOT:-}" ]]; then
  mkdir -p mobile/android/app/libs
  # amd64 = Android emulator (sdk gphone64); arm64 = physical devices.
  gomobile bind -target=android/amd64,android/arm64 -androidapi 21 \
    -o mobile/android/app/libs/gamebridge.aar \
    ./pkg/gamebridge
fi

if [[ "$(uname -s)" == "Darwin" ]]; then
  mkdir -p mobile/ios/Frameworks
  gomobile bind -target=ios -o mobile/ios/Frameworks/KnightBridge.xcframework ./pkg/gamebridge
fi
