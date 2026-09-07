# Implementation Plan — J2ME Go Engine Migration

## Strategy
Follow the Project Pattern:
1. **Phase 0 — Survey & Codebase Exploration**:
   - Spawn parallel Explorer subagents (`explorer_survey_1`, `explorer_survey_2`, `explorer_survey_3`) to analyze existing codebase, `j2me-stubs`, `KnightOnline_402.jar`, `main.go`, `pkg/` structures, and existing CGO/JNI bridge code.
2. **Phase 1 — User Alignment & Architecture Design**:
   - Synthesize survey findings into a concrete design proposal.
   - Present design options, architecture, module boundaries, and trade-offs to the user and obtain explicit user alignment per AGENTS.md rules.
3. **Phase 2 — Project Decomposition & Execution**:
   - Milestone M1: Core J2ME Stubs Migration (`Image.java`, `Canvas.java` native proxies without AWT/Swing).
   - Milestone M2: Go Native Implementations & CGO Bridge (VRAM management, `*ebiten.Image` handling, drawing operations, integer ID mapping).
   - Milestone M3: JAR Loading & E2E Game Execution (Launcher `Main.java`, JNI game launch, Ebiten window rendering, keyboard/mouse input forwarding).
4. **Phase 3 — Verification & Forensic Audit**:
   - Run unit/integration tests, E2E test execution, verify against acceptance criteria, pass Forensic Auditor check.
