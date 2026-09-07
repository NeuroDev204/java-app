#!/usr/bin/env bash
set -e

# Change to the application directory so relative assets and JNI stubs load correctly
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

# Execute the J2ME Go Emulator
exec "$DIR/j2me-emulator" "$@"
