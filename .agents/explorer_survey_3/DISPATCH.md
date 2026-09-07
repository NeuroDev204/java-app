## 2026-08-09T05:40:02Z
Investigate JAR Loading, JVM main invocation, and Input forwarding in /home/neuro/Documents/Tools/java-app/.

Read:
- /home/neuro/Documents/Tools/java-app/ORIGINAL_REQUEST.md

Investigate:
1. Inspect how JVM is initialized in CGO (`pkg/jvm/jvm.go` or similar) and how classpath can be configured to load both `j2me-stubs/bin` and `KnightOnline_402.jar`.
2. Analyze how Ebitengine key and mouse input events can be captured in Go and forwarded via JNI / CGO callbacks to Java Canvas (e.g. `keyPressed`, `keyReleased`, `pointerPressed`, `pointerReleased`, `pointerDragged`).
3. Analyze thread synchronization between Ebitengine's main thread update/draw loop and JVM execution thread to prevent freezing or deadlocks.

Write report to:
/home/neuro/Documents/Tools/java-app/.agents/explorer_survey_3/analysis.md
Write handoff report to:
/home/neuro/Documents/Tools/java-app/.agents/explorer_survey_3/handoff.md
