# 🏛️ TestGuard Architecture & Design Guidelines

This document outlines the architectural boundaries, package structure, execution data flow, and design contracts of `testguard`. It serves as the definitive reference for both human maintainers and AI coding assistants when extending or refactoring the codebase.

---

## 1. Architectural Principles

1. **Unidirectional Dependency Flow**: Dependencies flow strictly downward from the CLI presentation layer into the headless engine, which in turn depends on adapters, detectors, rules, and pure models.
2. **Headless Engine Core**: `pkg/engine` has zero direct dependencies on terminal formatting, ANSI escapes, or stdout/stderr printing. Progress and reporting are decoupled via callbacks and interfaces.
3. **Clean I/O Stream Separation**:
   - `os.Stdout`: Dedicated solely to report output (terminal scorecard, markdown, or machine-readable JSON).
   - `os.Stderr`: Used exclusively for interactive in-progress indicators (live spinner, file counter) and error messages. This guarantees `testguard --json | jq .` is always safe for Unix pipelines.
4. **Zero-Dependency Footprint**: The binary remains statically linked, sub-10MB, and lightweight (relying only on Cobra/pflag and Go standard library).
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
| `pkg/models` | Core data structures: `Violation`, `Severity`, `Suppression`. | Pure leaf package. **Must never import other `pkg/` packages**. |
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
