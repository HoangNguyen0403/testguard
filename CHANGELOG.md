# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---
## [Unreleased]

### Removed
- **Dead Subcommand**:
  - Removed dead `testguard init` command and associated `testguard.yaml` starter generation (misleading configuration documentation removed).
- **Low-Precision Rules Deleted**:
  - `GUARD-DART-UNCOVERED-MUTATION`: Deleted keyword + literal `isFetching` heuristic rule.
  - `GUARD-DART-DISGUISED-MIRROR`: Deleted rule penalizing `.having()` on mutated fields.
  - `GUARD-GO-02`: Deleted port-only mocking rule that flagged legitimate repository/adapter tests using `sqlmock`.

### Changed
- **Scope Exclusions & Deduplication**:
  - Traversal automatically skips linked git worktrees (subdirectories containing `.git` file pointing to worktrees) with reason `linked_worktree`.
  - Added `vendor/` and `third_party/` directories to default skipped directory set (`default_directory`).
- **Zero-Assertion Accuracy Overhaul**:
  - **Go**: `findExecutableAssertions` now identifies assertions in `t.Run` closures, resolves testify suite runner (`suite_runner`) units and in-file embedded suite receiver methods (`suite.Suite`), inspects one-hop same-file helpers taking `*testing.T`/`testing.TB`, and tracks gomock controller expectations with package-qualified mock constructors and mock provenance. Checked sqlmock results are recognized through their actual assertion/failure branch (`t.Fatal`/`t.Error`/`require`), not by verification call alone.
  - **Accepted Limitations**: Split-file testify suites are not analyzed (only in-file structs embedding `suite.Suite` qualify); cross-file helpers are not resolved except the documented external mock-factory and `t.Run` delegation policies.
  - **TypeScript / JavaScript**: Paren-aware raw body extraction for expression-bodied tests (`it('x', () => expect(a).toBe(1))`) and one-hop same-file helper resolution (`expectCartEmpty()`).
  - **Dart**: One-hop same-file helper resolution appending `helper:<name>` assertions; string/comment-aware paren matching for `blocTest` argument bodies preventing mid-setUp truncation.
- **Rule Refinements & Severity Adjustments**:
  - `GUARD-REACT-02`: Severity changed to Warning. Excludes delay `0` flushes, mock latency callbacks (`mockImplementation`), class bodies, and files using `useFakeTimers()`. Only evaluates non-E2E unit tests.
  - `GUARD-DART-BLOC-DUPLICATE`: Compares entire `blocTest` scenario arguments (build/setUp/seed/act/expect/skip/verify) after stripping comments and whitespace instead of comparing only act and expect blocks.
  - `GUARD-UNIV-01` / `GUARD-UNIV-02`: Exact-token duplicates consistently report as `GUARD-UNIV-01` ("Duplicate test body"), while normalized-literal variants report as `GUARD-UNIV-02` ("Similar test structure").

### Added
- **Hygiene & E2E Rules**:
  - `GUARD-UNIV-TAUTOLOGY` (Blocker): Flags trivial tautological assertions such as `expect(true).toBe(true)` or `assert.True(t, true)`.
  - `GUARD-UNIV-SKIP` (Warning): Detects permanently skipped tests (`it.skip`, `skip: true`, unconditional `t.Skip`).
  - `GUARD-E2E-01` (Warning): Catches arbitrary hard wait timers in E2E tests (`page.waitForTimeout(3000)`, `Future.delayed(2s)`).

---
## [0.1.4] - Unreleased

### Breaking Changes
- **Versioned Machine-Readable JSON Envelope (`schema_version: 1`)**:
  - Replaced the bare violation array (`[]`) emitted by `--json` and `-f json` with a versioned `ScanReport` envelope.
  - Output includes `schema_version: 1`, `tool` metadata, `scope` details, `analysis` completeness status, `findings`, `diagnostics`, `policy`, and `exit_code`.
  - Automated consumers and dashboards must verify `schema_version == 1` and cross-check process and envelope exit codes. Retained findings survive incomplete scans for diagnostic utility; however, an absence of findings can only claim clean assurance when `analysis.status == "complete"` and exit code is `0`.
- **Deterministic Exit Precedence Overhaul**:
  - Exit `0`: Nonempty analysis complete; zero blocker violations found (`complete / analyzed`).
  - Exit `1`: Nonempty analysis complete; at least one blocker violation detected (`complete / analyzed`).
  - Exit `2`: Incomplete analysis or usage error (`incomplete` or `error`). Triggered by invalid arguments, `--diff` rejection, file read/traversal/parse failures, explicit unsupported file targets, or empty test scope without `--allow-empty`.
  - Exit `3`: Explicitly not applicable (`not_applicable / empty_allowed`). Triggered when the target scope contains zero test declarations AND `--allow-empty` is supplied, with zero findings. Exit 3 is distinct from pass.
- **`--diff` / `-d` Flag Unsupported**:
  - Invoking `testguard --diff` or `testguard -d` is not implemented in 0.1.4 and is explicitly rejected with exit code 2 (`invalid_usage`). Effective scan mode is full target scan (`effective_mode: "full"`).
- **Flag Combination Rejections**:
  - Combining `--json` with `-f azure` is rejected with exit code 2 (`invalid_usage`).
  - Explicit `--output` (including empty values like `--output=""`) with `--json` or `-f azure` is rejected with exit code 2 (`invalid_usage`).

### Added
- **`--allow-empty` CLI Flag**:
  - Allows scanning targets with zero analyzable test declarations without failing the audit, returning exit code 3 (`not_applicable`).
- **Structured Diagnostics**:
  - Added `ScanReport.diagnostics` with stable machine-readable error codes: `invalid_usage`, `target_unreadable`, `traversal_failed`, `read_failed`, `parse_failed`, `unsupported_input`, `empty_scope`, `output_failed`.
- **Configuration Status Transparency**:
  - Added `scope.config_applied: false` to explicitly communicate that checked-in configuration files (such as `testguard.yaml`) are not applied by the engine.

### Adoption & Rollback Policy
- **Observation-First Rollout**:
  - Recommended CI integration pattern starts observation-only (non-blocking) to establish baseline findings and calibrate rules.
  - Promoting TestGuard to a blocking gate requires human owner approval and calibration evidence.
  - Rollback ownership: If false alarms or disruptions occur, disable the blocking gate while preserving report artifact generation and existing unrelated gates. Never convert missing evidence or errors into passing status.

### Heuristic Boundaries
- **Static Pattern Matching**:
  - TestGuard operates via static AST token and syntax heuristics; it does not execute test suites, run dynamic mutation testing engines, or evaluate runtime behavioral correctness.
- **Checkpoint Evidence Status**:
  - This entry documents the unreleased v0.1.4 specification boundary at a local documentation checkpoint. Remote CI, release publication, and pilot evaluations have not yet run or passed.

---
## [0.1.3] - 2026-10-15

### Added
- **Interactive In-Progress Scanning Notice & Spinner**:
  - Live animated terminal spinner (Unicode Braille `⠋⠙⠹...` with ASCII fallback `[-/|\]` under `NO_COLOR`) providing real-time feedback during file discovery and test AST analysis.
  - TTY-aware output rendered to `os.Stderr` using `\r\033[2K` line overwriting, ensuring `os.Stdout` remains completely unpolluted for stdout pipes and redirection.
  - Dual-phase execution tracking:
    - Discovery phase: `⠋ Discovering test files...`
    - Audit phase: `⠋ Scanning test files [current/total] • path/to/file`
  - Seamless transition: clears active progress line before scorecard rendering.
  - Automatically disabled in `--json` / `-f json` mode and non-interactive/CI environments.
- **Engine Progress Hooks**:
  - Added decoupled `ProgressFunc` callback to `engine.Options` dispatching `"discovering"`, `"discovered"`, and `"scanning"` lifecycle events.


### Fixed
- **CI Gate & Exit Code Enforcement**:
  - Enforced exit code 1 on Blocker findings across all output formats (`--json`, `-f azure`, and console).
  - Fixed `WriteJSONReport` emitting `[]` instead of `null` when no violations exist.
  - Validated target paths with `os.Stat` (errors immediately on nonexistent paths) and rejected unsupported `--format` values.
- **Polyglot Test Unit Parsing & Clone Accuracy**:
  - Extracted individual test function bodies via AST brace matching (`parser.FindMatchingBrace`) instead of passing the entire file content as `RawBody`.
  - Eliminated false-positive duplicate clone warnings (`GUARD-UNIV-01`/`02`) and multi-counting on Go, TypeScript, Kotlin, and Swift files.
  - Prevented test fixtures inside string literals from being misidentified as tests.
- **Dart AST Parsing & Location Accuracy**:
  - Added support for `testWidgets(...)` in Flutter tests.
  - Allowed apostrophes inside double-quoted test names (e.g. `test("doesn't crash", ...)`).
  - Switched `StartLine` calculation from naive substring search to exact match byte-offset binary search.
- **Target Path Deduplication**:
  - Deduplicated target test files by clean path to prevent overlapping arguments (e.g. `testguard pkg pkg/rules`) from scanning files multiple times.
- **Progress Throttling & Viewport Sizing**:
  - Decoupled `ProgressReporter.Update()` from synchronous terminal write syscalls on the engine hot path.
  - Capped line display width under 72 characters to prevent wrapping and row clutter on standard 80-column viewports.
  - Implemented rune-safe path truncation for multi-byte Unicode paths.
---

## [0.1.2] - 2026-10-15

### Added
- **Monorepo Module Breakdown**:
  - Scorecard table categorizing findings by monorepo module/service (e.g. `datasource/<org>/<service>`, `apps/<app>`, `packages/<pkg>`).
- **Top Rule Violations Ranking**:
  - Terminal summary table highlighting the most frequent rule violations across the workspace.
- **Summary Mode & Output Controls**:
  - Added `--summary` (`-s`) flag for high-level monorepo/module summaries without file dumps.
  - Added `--top` flag (default: 10) to cap individual terminal findings and prevent scroll spam.

### Fixed
- **Engine Directory Discovery**:
  - Fixed relative `.` root path skip in `FindTestFiles`.
- **Dart AST Closure Parsing**:
  - Fixed premature closure truncation on deeply nested test closures.
- **Go Port Mocking Exemption**:
  - Exempted database repository layers (`*_repository_test.go`, `/repository/`, `/repo/`, `/postgres/`, `/mysql/`, `/dao/`) from `GUARD-GO-02` port-only mocking rule.

---


## [0.1.1] - 2026-10-14

### Added
- **Expanded Polyglot Ecosystems**:
  - **Go Backend**: `GUARD-GO-02` (Port-only mocking) and `GUARD-GO-03` (Table-test duplicate input rows).
  - **NestJS Backend**: `GUARD-NEST-01` (Pass-through controller echoes) and `GUARD-NEST-02` (Unmocked database module leaks in unit tests).
  - **React / Next.js / Web**: `GUARD-REACT-01` (Shallow mock text echoes), `GUARD-REACT-02` (Arbitrary setTimeout/sleep fragility), and `GUARD-TS-01` (Massive snapshot-only test smells).
  - **Android (Kotlin)**: `GUARD-KT-01` (MockK pass-through echoes) and `GUARD-KT-02` (Unadvanced coroutine virtual time in runTest).
  - **iOS (Swift)**: `GUARD-SWIFT-01` (Excessive expectation timeouts >2.0s in local unit tests).
- **Universal Guardrails**:
  - `GUARD-UNIV-02`: Structural repetition smell (flags N>=4 repetitive test clones).
  - `GUARD-UNIV-NAME`: Vague/placeholder test name detection ('test 1', 'dummy', 'temp', 'foo').
- **Dart / Flutter Mobile**:
  - `GUARD-DART-DISGUISED-MIRROR`: Catches chains of 4+ .having() calls on state matchers.
  - `GUARD-DART-BLOC-DUPLICATE`: Detects duplicate event-state pairs in blocTest.
  - `GUARD-DART-MOCK-ECHO`: Traps 1:1 pass-through mock echoing in repositories and services.
- **In-Code Rule Suppression**:
  - Support for `// testguard:ignore [RULE-ID] reason: <explanation>` and `# testguard:ignore` to eliminate false-alert friction for legitimate edge cases.
- **CLI Commands & CLIG Ergonomics**:
  - `testguard update`: Self-updater via GitHub Releases.
  - `testguard init`: Generates starter `testguard.yaml` configuration.
  - `testguard completion`: Shell autocompletion scripts for bash, zsh, fish, and powershell.
  - Full `NO_COLOR` standard compliance (https://no-color.org).

---

## [0.1.0] - 2026-10-14

### Added
- **Core Engine**: Statically compiled, zero-dependency Go CLI for polyglot test quality audits.
- **Dart & BLoC Support**:
  - `GUARD-DART-SEED-ECHO`: Detects expected states that merely echo seed fixtures without mutation.
  - `GUARD-DART-UNCOVERED-MUTATION`: Flags async BLoC tests omitting critical loading (`isFetching: true`) state assertions.
- **Universal Detectors**:
  - `GUARD-UNIV-ZERO`: Traps zero-assertion tests across any language (`expect`, `assert`, `verify`).
  - `GUARD-UNIV-01`: AST token stream clone detection catching duplicate test executions.
- **Embedded Self-Documentation**: Offline documentation engine via `testguard explain <rule>` and `testguard rules`.
- **Developer Ergonomics**:
  - Totals-first summary header with capped output to prevent terminal spam.
  - Standard `path:line:col` clickable links for VS Code, Cursor, Android Studio, and iTerm.
  - Native Azure DevOps `##vso[task.logissue]` PR annotations.
- **Cross-Platform Installers**:
  - One-line bash installer for macOS and Linux (`install.sh`).
  - One-line PowerShell installer for Windows (`install.ps1`).
- **Comprehensive Documentation**: Claude Code-inspired `QUICKSTART.md`.
