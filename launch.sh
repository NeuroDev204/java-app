#!/usr/bin/env bash
set -e

# Change to the application directory so relative assets and JNI stubs load correctly
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$DIR"

for candidate in \
    "/home/neuro/.local/share/j2me-runtime/usr/lib/jvm/java-21-openjdk-amd64/lib/server" \
    "/usr/lib/jvm/java-21-openjdk-amd64/lib/server"; do
    if [ -f "$candidate/libjvm.so" ]; then
        export LD_LIBRARY_PATH="$candidate${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
        break
    fi
done

# Execute the J2ME Go Emulator
exec "$DIR/j2me-emulator" "$@"
