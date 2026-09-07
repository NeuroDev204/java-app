## 2026-08-09T04:46:55Z
Milestone 4: JNI Callbacks (Java -> Go) & E2E Integration for the project at /home/neuro/Documents/Tools/java-app.

Your working directory for metadata/handoff is: /home/neuro/Documents/Tools/java-app/.agents/worker_m4_1
Project root: /home/neuro/Documents/Tools/java-app

MUST READ BEFORE STARTING:
- /home/neuro/Documents/Tools/java-app/ORIGINAL_REQUEST.md
- /home/neuro/Documents/Tools/java-app/.agents/orchestrator_2/PROJECT.md
- /home/neuro/Documents/Tools/java-app/.agents/orchestrator_2/handoff.md

Objective:
Implement Milestone 4 - JNI Callbacks (Java -> Go) & End-to-End Verification:
1. Create `j2me-stubs/src/Test.java` invoking `Graphics.fillRect(10, 20, 100, 50)` and `Graphics.setColor(0x00FF00)` (or `fillRect` / `setColor` methods on `Graphics`).
2. Compile `Test.java` with `javac -d j2me-stubs/bin j2me-stubs/src/Test.java`. Ensure `j2me-stubs/bin` contains the compiled `.class` files.
3. Implement CGO JNI native callback handling in Go (in `pkg/jvm` or `main.go` / JNI bridge files). Use standard JNI exports (`Java_javax_microedition_lcdui_Graphics_fillRect` or `RegisterNatives` mapping for `javax/microedition/lcdui/Graphics`) so Java native call to `fillRect` routes to Go `go_fillRect` / CGO bridge.
4. Decouple callback execution to a Go thread-safe `RenderQueue` (with `sync.Mutex`). Upon receiving callback, log the interception message to stdout: `[CGO JNI Intercept] fillRect(x=10, y=20, width=100, height=50)`.
5. Invoke `Test.main` from Go via JNI (`CallStaticVoidMethod`) after JVM startup.
6. Verify all Acceptance Criteria:
   - `go build -o j2mego` executes without errors and links against `libjvm.so`.
   - `javac -d j2me-stubs/bin j2me-stubs/src/javax/microedition/lcdui/Graphics.java` and `javac -d j2me-stubs/bin -cp j2me-stubs/bin j2me-stubs/src/Test.java` execute without errors.
   - Running `./j2mego` initializes JVM without crashing.
   - `Test.java` invoked from Go executes `fillRect`, printing `[CGO JNI Intercept] fillRect(x=10, y=20, width=100, height=50)` to standard output.
7. Run builds and tests, ensure 0 linter errors and 0 compiler warnings/errors.
8. Deliver handoff report in `/home/neuro/Documents/Tools/java-app/.agents/worker_m4_1/handoff.md`.
