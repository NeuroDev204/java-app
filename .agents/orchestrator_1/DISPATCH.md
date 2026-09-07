## 2026-08-08T22:30:46Z
<USER_REQUEST>
You are the Project Orchestrator.
Working directory: /home/neuro/Documents/Tools/java-app/.agents/orchestrator_1
Original Request path: /home/neuro/Documents/Tools/java-app/.agents/ORIGINAL_REQUEST.md
Codebase directory: /home/neuro/Documents/Tools/java-app

Task:
Complete the J2ME Java Game Emulator rendering pipeline for KnightOnline MIDlet suite in Go (`java-app`) so that full game menu graphics, UI buttons, custom bitmap font text, and gameplay sprites render cleanly.

Requirements:
- R1: Complete J2ME Canvas & MIDP UI Rendering (main menu, server selection, dialog popups, font text strings, game world sprites on offscreen canvas without black clipping).
- R2: Complete JVM Bytecode Interpreter Engine (required Java MIDP bytecodes, string manipulation methods `charAt`, `indexOf`, `valueOf`, InputStream byte reading, image decoding, RecordStore persistent storage).

Acceptance Criteria:
- Offscreen buffer `emulator_buffer.png` contains full game menu graphics, buttons, and text.
- No full-screen black overwrite occurs during `paint()` cycles.
- All unit tests pass cleanly (`LIBRARY_PATH=/tmp go test ./...`).

Please analyze the codebase, align on plan/design, dispatch specialists, track progress in `/home/neuro/Documents/Tools/java-app/.agents/orchestrator_1/progress.md`, and report completion when done.
</USER_REQUEST>
