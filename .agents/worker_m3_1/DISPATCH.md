## 2026-08-09T04:43:52Z
You are Worker 3 (teamwork_preview_worker) assigned to Milestone 3: JVM Embedding via CGO/JNI.
Your working directory is: /home/neuro/Documents/Tools/java-app/.agents/worker_m3_1
Project root directory: /home/neuro/Documents/Tools/java-app
Original request file: /home/neuro/Documents/Tools/java-app/ORIGINAL_REQUEST.md
Scope document: /home/neuro/Documents/Tools/java-app/.agents/orchestrator_2/PROJECT.md

Scope & Tasks for Milestone 3:
1. Implement JVM initialization in Go via CGO using standard JNI header `<jni.h>`:
   - Create package `pkg/jvm` (`pkg/jvm/jvm.go`) or CGO wrapper in `main.go`.
   - Configure CGO CFLAGS and LDFLAGS:
     - `#cgo CFLAGS: -I/usr/lib/jvm/java-21-openjdk-amd64/include -I/usr/lib/jvm/java-21-openjdk-amd64/include/linux`
     - `#cgo LDFLAGS: -L/usr/lib/jvm/java-21-openjdk-amd64/lib/server -Wl,-rpath,/usr/lib/jvm/java-21-openjdk-amd64/lib/server -ljvm`
   - Implement JVM startup routine calling `JNI_CreateJavaVM` with classpath option `-Djava.class.path=./j2me-stubs/bin`.
   - Call `runtime.LockOSThread()` before JNI calls to ensure OS thread stability.
2. Integrate JVM startup into application lifecycle:
   - Call JVM initialization in `main()` before starting the Ebitengine graphics loop.
   - Print confirmation message: `JVM initialized successfully!`.
3. Verification:
   - Run build command: `go build -o j2mego`
   - Verify dynamic linking with `ldd ./j2mego` confirming `libjvm.so` is linked.
   - Run `./j2mego` (or headless test) verifying JVM initializes without crashing (Requirement R2 & Acceptance Criteria).
   - Run unit tests: `go test ./...`
