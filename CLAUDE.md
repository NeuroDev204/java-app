# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A J2ME emulator that runs the Knight Online mobile game (a TeaMobi "HSO" client). The same J2ME app is run by **two independent execution engines** that share the rendering, networking, and bot packages:

1. **Desktop** — embeds a real HotSpot JVM (OpenJDK 21) in the Go binary via CGO/JNI. Java stubs in `j2me-stubs/` are thin `native` proxies whose implementations live in Go. Rendering/input via Ebitengine.
2. **Mobile** — a hand-written pure-Go JVM bytecode interpreter (`pkg/vm`, no CGO) exposed through a gomobile-bindable façade (`pkg/gamebridge`), consumed by the Flutter app in `mobile/`.

When editing shared logic, remember a change may need to hold for **both** engines.

## Build & run

```bash
# Desktop (requires OpenJDK 21 at /usr/lib/jvm/java-21-openjdk-amd64 + libjvm.so)
go build -o j2me-emulator .          # root main.go = full app (menu, JAR picker, auto-upgrade overlay)
go build -o j2mego ./cmd/j2me        # minimal reference build (R1–R4 scaffold)
./launch.sh [game.jar]               # cd's to app dir first so relative stub paths resolve

# Java stubs (compiled .class files already checked in under j2me-stubs/bin)
javac -d j2me-stubs/bin j2me-stubs/src/javax/microedition/lcdui/*.java

# Mobile native bridge (gomobile bind — pins x/mobile to a Go 1.24-compatible tag)
./mobile/scripts/bind_native.sh      # needs ANDROID_HOME for .aar, macOS for .xcframework
cd mobile && flutter run
```

The CGO include/lib paths for `libjvm` are **hardcoded** in the `import "C"` cgo comments of `main.go`, `cmd/j2me/main.go`, and `pkg/jvm/jvm.go`. If the JDK path differs, edit all three. The pure-Go engine (`pkg/vm`, `pkg/gamebridge`) has **no CGO** and must stay that way so `gomobile bind` produces clean bindings.

## Test

```bash
go test ./...                        # 47 test files; CGO packages (jvm, root, cmd/j2me) need the JDK present
go test ./pkg/vm/...                 # pure-Go interpreter, no JDK needed
go test -run TestName ./pkg/foo      # single test
```

## Architecture

- `pkg/jvm` — CGO/JNI bridge to the embedded HotSpot JVM. Registers Go functions as the `native` methods declared in the Java stubs; forwards key/pointer/text events into Java. `bridge.c` holds the C-side JNI glue (see `patch_bridge.py` for how it's generated/patched).
- `pkg/vm` — the alternative: a from-scratch JVM interpreter (`interpreter.go`, `opcodes.go`, `frame.go`, `object.go`) with its own class loading via `pkg/classfile`. Powers mobile.
- `pkg/gamebridge` — the only package `gomobile bind` sees. Deliberately minimal, CGO-free API (`LoadJAR`, `Tick`, `GetFrameBuffer`, `SendTouch*`, `SendKey*`, upgrade controls). Drives `pkg/vm`.
- `pkg/midp` — the MIDP `Graphics` surface: an `image.RGBA` double-buffer both engines draw into. Also `sockettap.go` and `font8x13.go`. This is the handoff point to the renderer.
- `pkg/gfx` — Ebitengine renderer, keypad mapping (`MapEbitenKeyToJ2ME`), desktop only.
- `pkg/network` — the TeaMobi HSO wire protocol: `protocol` (frame codec + dynamic-XOR stream `crypto.go`, stateful with separate read/send pointers), `game` (item/inventory models), `config`.
- `pkg/upgradebot` / `pkg/autoupgrade` — the auto item-upgrade bot (parse packets → apply rules → send actions). `autoupgrade` adds the desktop F3 overlay and a TCP bridge (`config_tcp.json`); `upgradebot` is the engine-agnostic core.
- `pkg/classfile`, `pkg/jar` — `.class` parsing and JAR reading for the pure-Go VM.
- `mobile/` — Flutter app. `lib/core/bridge/game_bridge.dart` calls the gomobile bindings; `lib/core/network`, `lib/features/autoupgrade` mirror the Go protocol/bot in Dart.

## Repo notes

- `.agents/` is a large archive of prior multi-agent run logs (BRIEFING/DISPATCH/handoff/progress). Not source — ignore unless asked.
- `ORIGINAL_REQUEST.md` records the original build spec and acceptance criteria across follow-ups.
- Several prebuilt binaries (`j2me-emulator`, `j2mego`, `java-app`, `knight-emulator`, `bin/j2me`) and log/PNG artifacts are checked in.
- `AGENTS.md` (root and `mobile/`) states the repo's own agent conventions: inspect before editing, no guessing at APIs/paths, don't swallow exceptions silently, run tests + `git status` before claiming done.
