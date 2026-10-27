# 🏛️ TestGuard Architecture & Design Guidelines

This document outlines the architectural boundaries, package structure, execution data flow, and design contracts of `testguard`. It serves as the definitive reference for both human maintainers and AI coding assistants when extending or refactoring the codebase.

---

## 1. Architectural Principles

1. **Unidirectional Dependency Flow**: Dependencies flow strictly downward from the CLI presentation layer into the headless engine, which in turn depends on adapters, detectors, rules, and pure models.
2. **Headless Engine Core**: `pkg/engine` has zero direct dependencies on terminal formatting, ANSI escapes, or stdout/stderr printing. Progress and reporting are decoupled via callbacks and interfaces.
3. **Clean I/O Stream Separation**:
   - `os.Stdout`: Dedicated solely to report output (terminal scorecard, markdown, or machine-readable `ScanReport` JSON envelope with `schema_version: 1`).
   - `os.Stderr`: Used exclusively for interactive in-progress indicators (live spinner, file counter) and diagnostics/errors. In `--json` mode, progress indicators are silenced to guarantee `testguard --json | jq .` is safe for Unix pipelines.
5. **Why-Over-What Commenting**: Code expresses *what* and *how*; docstrings and comments strictly explain *why* (architectural invariants, domain quirks, non-obvious AST edges).

---

## 2. Dependency Hierarchy & Package Layout

```
cmd/testguard/               # CLI Entrypoint, Cobra commands, flag parsing, reporter wiring
  └── pkg/engine/            # Orchestrator: discovery, AST delegation, rule evaluation, suppression
        ├── pkg/parser/      # AST parsing contracts and language-specific adapters (e.g. Dart)
        ├── pkg/detectors/   # Universal, language-agnostic detectors (clone, naming, zeroassert)
        ├── pkg/rules/       # Polyglot domain & framework rules (dart, golang, typescript, mobile)
        └── pkg/models/      # Pure domain models (Violation, Severity, Suppression) [Leaf package]

pkg/reporter/                # Presentation: Terminal scorecard, Azure PR logger, JSON, Progress spinner
pkg/docs/                    # Embedded rule documentation catalog (powers `explain` & `rules`)
pkg/updater/                 # Self-updating mechanism via GitHub Releases API
```

### Detailed Responsibilities

| Package | Responsibility | Invariants & Constraints |
|---|---|---|
| `cmd/testguard` | CLI commands (`explain`, `rules`, `update`, `init`), flags, wiring reporters to engine. | Coordinates I/O; never contains AST parsing or rule logic. |
| `pkg/engine` | Directory traversal (`FindTestFiles`), module identification, rule dispatch, suppression filtering. | Must remain headless; accepts `Options.ProgressFn` for event emission. |
| `pkg/parser` | Defines `LanguageAdapter` interface and `TestUnit` AST representations. | Implementations must handle malformed syntax without crashing. |
| `pkg/detectors` | Language-agnostic smells: test cloning (`clone`), vague naming (`naming`), missing assertions (`zeroassert`). | Operates on generic token streams or `TestUnit` abstractions. |
| `pkg/rules` | Language- and framework-specific rules: Flutter BLoC, Go Clean Architecture ports, React timers, NestJS. | Grouped by language (`rules/dart`, `rules/golang`, `rules/typescript`, `rules/mobile`). |
| `pkg/models` | Core data structures: `Violation`, `Severity`, `Suppression`, `ScanReport`. | Pure leaf package. **Must never import other `pkg/` packages**. |
| `pkg/reporter` | Output rendering: `TerminalReporter`, `AzureReporter`, `JSONReport`, `ProgressReporter`. | Handles TTY detection, `NO_COLOR`, and line clearing (`\r\033[2K`). |
| `pkg/docs` | Built-in rule explainers (`Explain()`, `AllRules()`). | Must keep descriptions, remediation steps, and code examples synchronized with rules. |

---

## 3. Execution Lifecycle & Data Flow

```
[CLI Invocation: testguard [paths...] -a]
                     │
                     ▼
             [cmd/testguard]
     (Instantiate ProgressReporter on os.Stderr)
                     │
                     ▼
             [pkg/engine.Run]
                     │
        ┌────────────┴────────────┐
        ▼                         ▼
   Phase: "discovering"       Walk filesystem via FindTestFiles
   (Spinner ticks on stderr)  Ignore vendor, node_modules, build, hidden dirs
        │                         │
        └────────────┬────────────┘
                     ▼
            Phase: "discovered" (Test files count known)
                     │
                     ▼
       ┌─────────────────────────────┐
       │ Loop: each test file        │
       │   1. Emit "scanning" event  │
       │   2. Read content           │
       │   3. Parse suppressions     │
       │   4. Parse AST/TestUnits    │
       │   5. Run Universal Detectors│
       │   6. Run Language Rules     │
       │   7. Filter Suppressed      │
       └─────────────────────────────┘
                     │
                     ▼
             [Result Aggregated]
                     │
                     ▼
          progress.Finish()
     (Clears active line on os.Stderr)
                     │
                     ▼
           [Output Reporter]
     (Terminal scorecard, JSON, or Azure logger)
```

---

## 4. In-Progress Feedback & TTY Protocol

TestGuard provides live visual progress during both discovery and scanning without polluting redirected output:

- **TTY Detection**: `ProgressReporter` inspects whether the target writer (`os.Stderr`) is a character device (`stat.Mode() & os.ModeCharDevice != 0`) and checks `CI` / `TERM=dumb`.
- **Spinner Frames**: Uses Unicode Braille (`⠋⠙⠹...`) for modern terminals, falling back to ASCII (`[-/|\]`) when `NO_COLOR` is active.
- **Line Clearing**: Uses ANSI escape code `\r\033[2K` on interactive terminals to overwrite the active line in place and clean up entirely on exit.
- **Headless & Machine-Readable**: In `--json` mode or non-interactive environments, `ProgressReporter` is disabled or silent to maintain clean stdout pipelines.

---

## 5. Extension Guides for Maintainers & AI Agents

### Recipe A: Adding a New Rule
1. **Choose Package**: Place in `pkg/rules/<language>/` or `pkg/detectors/` (if universal).
2. **Define Rule Struct & Evaluator**:
   ```go
   type MyNewRule struct {}
   func (r *MyNewRule) Evaluate(unit parser.TestUnit) []models.Violation { ... }
   ```
3. **Register in `pkg/engine/engine.go`**:
   - Add rule field to `Engine` struct.
   - Initialize in `New()`.
   - Call during the language dispatch loop in `Run()`.
4. **Register in `pkg/docs/embedded.go`**:
   - Add rule metadata, rationale, remediation, and good/bad code examples to the embedded catalog so `testguard explain <rule>` works out of the box.
5. **Add Comprehensive Tests**:
   - Unit tests covering positive match (violation triggered), negative match (valid code passes), and edge cases.

### Recipe B: Adding a New Language Adapter
1. Create `pkg/parser/<language>/adapter.go`.
2. Implement `parser.LanguageAdapter` interface:
   ```go
   type LanguageAdapter interface {
       ID() string
       FileExtensions() []string
       LocateTests(content []byte, filePath string) ([]TestUnit, error)
       IsGenerated(filePath string, header []byte) bool
   }
   ```
4. Add the adapter to `pkg/engine/engine.go` and hook into the file loop.

---

## 6. Code Craftsmanship Invariants

- **Public Boundaries**: All exported structs, interfaces, methods, and functions require clear GoDoc comments describing purpose, parameters, and return expectations.
- **No Inline Commentary Clutter**: Internal function bodies must be self-documenting with intention-revealing names. Never include step narrations (`// Step 1: ...`), echo comments, or dead code.
- **Race Safety**: Any background goroutines (such as `ProgressReporter` tickers) must synchronize state access via `sync.Mutex` and provide explicit shutdown channels (`done`, `doneAck`) verified by `go test -race ./...`.

---

## 7. Machine-Readable ScanReport Contract & Exit Precedence

### Envelope Specification (`schema_version: 1`)

In `--json` mode, TestGuard outputs a single versioned `ScanReport` JSON object to `stdout`:

```ts
type ScanReport = {
  schema_version: 1;
  tool: {
    name: "testguard";
    version: string;
    commit: string | null;
    built_at: string | null;
  };
  scope: {
    requested_paths: string[];
    effective_mode: "full"; // diff is not implemented
    files: string[];        // sorted unique analyzed files (cwd-relative slash paths)
    excluded: { path: string; reason: "default_directory" | "generated" | "non_test" }[];
    unsupported: string[];  // explicit unsupported target file paths
    config_applied: false;  // config files (e.g. testguard.yaml) are not applied
  };
  analysis: {
    status: "complete" | "incomplete" | "not_applicable" | "error";
    reason: "analyzed" | "input_failure" | "empty_scope" | "empty_allowed" | "unsupported_scope" | "invalid_usage" | "output_failure";
    discovered_files: number;
    analyzed_files: number;
    analyzed_tests: number;
    failed_files: number;
  };
  findings: Violation[];    // retained violations; severity: Blocker, Warning, Advisory
  diagnostics: { code: string; path: string | null; message: string }[];
  policy: {
    blocker_count: number;
    status: "no_blocker" | "blocker_found";
  };
  exit_code: 0 | 1 | 2 | 3;
};
```

### Structural Invariants

1. **Completeness vs. Policy Decoupling**: `analysis.status` records execution completeness (`complete`, `incomplete`, `not_applicable`, `error`). `policy.status` records compliance based solely on retained findings (`no_blocker`, `blocker_found`).
2. **Empty Arrays**: Fields `files`, `excluded`, `unsupported`, `findings`, and `diagnostics` are serialized as `[]`, never `null`.
3. **Full Scope & Diff Rejection**: Scans operate with `effective_mode: "full"`. Supplying `--diff` or `-d` is unsupported and rejected with exit code 2 (`invalid_usage`).
4. **Unapplied Configuration**: `scope.config_applied` is explicitly `false`; repository configuration files are not parsed or applied.
5. **Stable Diagnostic Codes**: `invalid_usage`, `target_unreadable`, `traversal_failed`, `read_failed`, `parse_failed`, `unsupported_input`, `empty_scope`, `output_failed`.
6. **Flag Compatibility Constraints**:
   - `--json` with `-f azure` is rejected with exit code 2 (`invalid_usage`).
   - Explicit `--output` (including empty `--output=""`) in `--json` or `-f azure` mode is rejected with exit code 2 (`invalid_usage`).
7. **Retained Findings Survival**: Any violations discovered before an error or input failure are preserved in `.findings` for diagnostic purposes. However, an absence of findings can only be treated as clean verification when `analysis.status` is verified to be `complete` (exit 0).
### Deterministic Exit Precedence

| Precedence | Condition | `analysis.status` / `reason` | Exit Code | Findings Handling |
|---|---|---|---|---|
| 1 (Highest) | Invalid CLI usage / unsupported `--diff` / output write failure | `error / invalid_usage` or `output_failure` | `2` | Retain any collected findings |
| 2 | Traversal / file read / parser failure | `incomplete / input_failure` | `2` | Retain all collected findings |
| 3 | Explicit unsupported file target | `incomplete / unsupported_scope` | `2` | Retain all collected findings |
| 4 | Zero test declarations found; no errors; `--allow-empty` absent | `incomplete / empty_scope` | `2` | Retain findings; cannot pass |
| 5 | Zero test declarations found; `--allow-empty` present; zero findings | `not_applicable / empty_allowed` | `3` | `[]` (explicitly inapplicable) |
| 6 | Zero test declarations found with findings under `--allow-empty` | `incomplete / empty_scope` | `2` | Retain findings |
| 7 | Nonempty complete analysis with blocker violations | `complete / analyzed` | `1` | Retain findings |
| 8 (Lowest) | Nonempty complete analysis without blocker violations | `complete / analyzed` | `0` | Retain warnings/advisories |

---

## 8. Operational Boundaries, Rollout, and Heuristic Limits

### Static Heuristic Limits (No Mutation Execution)
TestGuard operates exclusively as a static AST token and syntax analyzer.
- It **does not execute** test suites, measure dynamic code coverage, run dynamic mutation engines, or evaluate test execution oracles.
- It **cannot prove** that a test suite detects real runtime faults or provides production assurance.

### Observation-First Adoption & Rollback Architecture
- **Observation-First Rollout**: Integrations in CI pipelines should run in observation-only mode (`continue-on-error: true` or non-blocking PR annotations) during baseline collection and rule calibration.
- **Promotion to Blocking**: Transitioning to a blocking gate requires explicit human owner approval and historical calibration evidence.
- **Rollback Ownership**: In the event of high false alarms or pipeline failure, the rollback procedure is to disable the blocking check while retaining report artifact collection and existing unrelated gates. Never convert missing reports or tool error exits into passing status.
- **Specification Boundary**: The v0.2.0 contract describes an unreleased breaking release boundary.
