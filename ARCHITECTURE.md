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
  └── pkg/engine/            # Orchestrator: discovery, adapter dispatch, rule evaluation, suppression
        ├── pkg/parser/      # LanguageAdapter contract plus per-language adapters
        │     ├── dart/      # flutter_test, blocTest
        │     ├── golang/    # testing, testify, gomock, gomega, quicktest
        │     ├── typescript/# jest/vitest, JSX-aware lexer
        │     ├── kotlin/    # JUnit, kotest, MockK
        │     └── swift/     # XCTest, Swift Testing
        ├── pkg/detectors/   # Universal, language-agnostic detectors (clone, naming, zeroassert)
        ├── pkg/rules/       # Polyglot domain & framework rules (dart, golang, typescript, mobile, hygiene, e2e)
        └── pkg/models/      # Pure domain models (Violation, Severity, Suppression) [Leaf package]

pkg/reporter/                # Presentation: Terminal scorecard, Azure PR logger, JSON, Progress spinner
pkg/docs/                    # Embedded rule documentation catalog (powers `explain` & `rules`)
pkg/updater/                 # Self-updating mechanism via GitHub Releases API
```

### Detailed Responsibilities

| Package | Responsibility | Invariants & Constraints |
|---|---|---|
| `cmd/testguard` | CLI commands (`explain`, `rules`, `update`), flags, wiring reporters to engine. | Coordinates I/O; never contains AST parsing or rule logic. |
| `pkg/engine` | Directory traversal (`FindTestFiles`), module identification, rule dispatch, suppression filtering. | Must remain headless; accepts `Options.ProgressFn` for event emission. |
| `pkg/parser` | Defines `LanguageAdapter` interface and `TestUnit` AST representations, plus one adapter per supported language. | Implementations must handle malformed syntax without crashing. Every discovered test file is parsed by the adapter that claims its extension; a `LocateTests` error is reported as `parse_failed`. |
| `pkg/detectors` | Language-agnostic smells: test cloning (`clone`), vague naming (`naming`), missing assertions (`zeroassert`). | Operates on generic token streams or `TestUnit` abstractions. |
| `pkg/rules` | Language- and framework-specific rules: Flutter BLoC, Go Clean Architecture and table tests, TypeScript/NestJS/React oracles, Kotlin coroutine and MockK provenance, Swift timeouts, plus shared hygiene and E2E rules. | Grouped by concern (`rules/dart`, `rules/golang`, `rules/typescript`, `rules/mobile`, `rules/hygiene`, `rules/e2e`). |
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
   File-level rules instead expose `EvaluateFile(filePath string, src string, units []parser.TestUnit)`.
3. **Register in `pkg/engine/engine.go`**:
   - Add rule field to `Engine` struct.
   - Initialize in `New()`.
   - Call during the language dispatch loop in `Run()`, keyed on `adapter.ID()`.
4. **Register in `pkg/docs/embedded.go`**:
   - Add rule metadata, rationale, remediation, and good/bad code examples to the embedded catalog so `testguard explain <rule>` works out of the box.
   - Set `Severity` to the exact severity the engine emits and `Languages` to every language the rule can fire in; the corpus parity test fails when either drifts.
5. **Add Comprehensive Tests**:
   - Unit tests covering positive match (violation triggered), negative match (valid code passes), and edge cases.
   - One **positive** and one **negative** corpus fixture under `pkg/engine/testdata/corpus/<prefix>_<scenario>/`, registered in `pkg/engine/corpus_test.go` with its exact expected findings. The corpus test fails on any false positive, false negative, severity mismatch, catalog rule without a true positive, or a catalog language without a true positive.

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
3. Register the adapter in `Engine.adapters` (`pkg/engine/engine.go`) and extend `isTestFileName` / `isSupportedLanguageExtension` in `pkg/engine/discovery.go` for its naming conventions.
4. Add its extension to `detectLanguageFromExt` in `pkg/engine/corpus_test.go` and to the `Languages` of every rule it can emit.

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
    config_applied: false;  // repository config files are not applied
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

### Known Recall Limits (Documented Misses)
Precision comes first: when a construct cannot be classified safely the engine reports nothing, so the
absences below are deliberate, reviewed misses rather than guarantees.
- **Cross-file resolution is absent.** Assertion helpers are credited one hop within the same file only
  (with the documented Go external-factory and `t.Run` delegation exceptions). Helpers imported from another
  module, a `GUARD-KT-02` dispatcher configured in a base class in another file, and a
  `GUARD-GO-UNCHECKED-WANT` row type declared in another file are all invisible.
- **Naming-convention credit is heuristic.** Where the adapter credits a helper by its name
  (`assert*`/`expect*`/`verify*`), a same-named helper that does not assert would suppress a genuine
  zero-assertion finding. This is a miss, never a manufactured finding.
- **Assertion-helper resolution is block-scoped and one hop.** A helper declared inside one `describe` block
  is not credited from a sibling block, and a locally shadowed `expect` is not treated as an assertion.
  Assertions inside nested functions are credited lexically regardless of references (see below). Helpers
  whose name matches `assert*`/`expect*`/`verify*` are credited file-wide, and imported helpers are credited
  only through those naming conventions.
- **TypeScript/JavaScript declarations are expression-tolerant, evaluation-free.** `it`/`test`/`fit`/
  `specify` declare a test when the title argument is a string/backtick literal, an identifier, a member
  expression (`c.name`), a call expression, a concatenation, or a template literal, and a function literal
  (`=>` or `function`) is supplied as the callback (second argument, or last after an options object/timeout).
  Local/shadowed `test`/`fit` functions, a `fit` utility bound by
  `import`/`require`, unknown members (`test.foo(...)`), hooks, `test.use`, `test.setTimeout` and
  function/method declarations are not tests; Playwright `test.fail('title', fn)` is not discovered
  (`it.fails` is). Dynamic titles are recorded as the original expression source, never an invented
  evaluated value: `analyzed_tests` counts declaration sites, so one `it(title, …)` inside a `forEach`/`for`
  loop is one unit regardless of runtime case count. Generic JSX elements with a balanced type-argument list
  before their attributes (`<Table<{ n: number }> … />`) are parsed; a genuinely malformed generic element
  still fails closed. Structural validation balances `()[]{}` only — `<`/`>` generics are not
  balance-checked outside JSX tags — and a regex literal containing braces or a backtick inside a template
  interpolation fails closed rather than truncating silently. Playwright implicit waits (`page.waitForURL`,
  `waitForSelector`, `waitForResponse`) are not credited as assertions, so a test whose only verification is
  such a wait is reported as zero-assertion.
- **Nested-function assertions are credited lexically, with no execution-reachability guarantee.** An
  assertion textually inside a nested function counts as verification for its test whether the function is
  invoked by name, passed as a callback, passed by reference, aliased, used as a promise executor, rendered
  as a JSX tag, or never referenced at all. Static lexical inspection cannot prove whether a framework or
  deferred call invokes a function, so TestGuard does not infer reachability. Recall gap: a genuinely
  unreferenced nested function containing assertions can hide a hollow test, and typed-return or generic
  recursive-only helpers are equally invisible; TestGuard does not claim to catch that shape. A nested
  function without any recognized assertion, and a shadowed `expect`, still produce zero-assertion findings.
- **`GUARD-UNIV-FOCUS` follows literal-title `fit` only.** A dynamic `fit(name, fn)` call (for example inside
  a `forEach`) is discovered as one analysable test but does not receive the focus warning. Minor recall gap,
  deliberately not expanded in this pass.
- **Dart arrow-body tests are not discovered.** A declaration such as `test('x', () => expect(add(1, 2), 3));`
  or a hollow `test('y', () => add(1, 2));` produces no unit, so its assertions are never inspected and no
  diagnostic is emitted; the rest of the file is still analysed and may report `complete`. This is a
  pre-existing discovery limit, not fixed in the current correction pass.
- **`GUARD-UNIV-TAUTOLOGY` inspects assert/require selectors in Go.** A bare `if got != got` comparison is
  not reported. Dart literal operands are compared as plain literals; Kotlin/Swift member paths follow the
  same-operand comparison with no invocation, so `a.id` versus `b.id` is not a tautology.
- **`GUARD-TS-CONDITIONAL-ASSERT` is lexical, not a control-flow analysis.** A branch is total when it
  asserts or ends in a statement-level `throw`, `fail(`, `expect.fail(`, `assert.fail(` or
  `expect.unreachable(`. A failure nested in an inner conditional, `try` body, call argument or function body
  does not totalise the enclosing branch, so behaviourally equivalent shapes stay reported (a miss).
- **Dart BLoC rules model the common shapes only.**
  `GUARD-UNIV-ZERO` never fires on `blocTest` units (a `blocTest` without `expect`, `verify` or `errors` is
  not reported). `GUARD-DART-SEED-ECHO` models `.having(...)` fields; a scalar seed (`seed: () => 4` with
  `expect: () => [4]`) or a whole-state echo is not reported. `GUARD-DART-MOCK-ECHO` requires a
  `Right`/`Left` result wrapper, so a direct echo or an unwrapped use case is not reported. Error
  pass-through combined with `verify(...).called(1)` is reported as a mock echo and may be a valid
  delegation test on review.
- **`GUARD-KT-01` requires a property/accessor read of the stubbed member.** A direct stub-call echo such as
  `assertEquals("alice", repo.find("a"))` is not reported.
- **`GUARD-GO-01` recognises `github.com/golang/mock/gomock`, `testify/mock` and generated `/mocks`
  packages.** It does not recognise the `go.uber.org/mock/gomock` re-module path, so a mock controller
  imported from that path is not reported even though the Go adapter can parse it.
- **The Kotlin/Swift/Dart/TypeScript adapters are lexical scanners, not full language parsers.** Unbalanced
  delimiters or an unterminated string/template/comment are reported as an adapter error only where the
  adapter has an error path; otherwise they degrade to a smaller discovery set rather than a claim of a
  complete scan. Comments, raw strings, triple-quoted strings and nested interpolation are masked before
  discovery, but a construct outside that lexical model is not guaranteed to be understood. Specific
  boundaries: Kotlin/Swift validate comments, literals and `()[]{}` only (angle-bracket generics are not
  balanced, and a Swift bare regex literal is not recognised, so an unbalanced brace inside one is read as
  code); Kotlin bare `check(cond) { … }` counts as an assertion because it throws on a false predicate, while
  `require(...)` is not credited; Swift `@Test`-attributed functions may take parameters, XCTest
  `func test…(params)` methods are not tests.

### Measured Scan Cost
Single-file TypeScript scans grow quadratically in the number of tests because each unit re-scans its token
stream. Measured on the reviewer machine (seconds per file): 2,500 tests ≈ 0.3 s, 5,000 ≈ 0.8 s,
10,000 ≈ 2.7 s, 20,000 ≈ 13 s. Realistic suites are far below this; no linear-scaling claim is made.

### Observation-First Adoption & Rollback Architecture
- **Observation-First Rollout**: Integrations in CI pipelines should run in observation-only mode (`continue-on-error: true` or non-blocking PR annotations) during baseline collection and rule calibration.
- **Promotion to Blocking**: Transitioning to a blocking gate requires explicit human owner approval and historical calibration evidence.
- **Rollback Ownership**: In the event of high false alarms or pipeline failure, the rollback procedure is to disable the blocking check while retaining report artifact collection and existing unrelated gates. Never convert missing reports or tool error exits into passing status.
- **Specification Boundary**: The v0.1.4 contract describes an unreleased release boundary.
