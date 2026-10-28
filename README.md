# 🛡️ TestGuard

**Standalone, Zero-Config Polyglot Test Quality & Architectural Boundary Guard CLI**

TestGuard inspects test suites, detects duplicate/clone tests, flags vanity test anti-patterns (zero assertions, seed echoes, uncovered mutations), and enforces architectural boundaries across **Dart/Flutter**, **Go**, and **TypeScript/JavaScript**.

---

## ⚡ Quickstart

Get up and running in 60 seconds! Read the full **[QUICKSTART Guide](QUICKSTART.md)** for detailed walkthroughs, or install below:

### macOS, Linux, and WSL:
```bash
curl -fsSL https://raw.githubusercontent.com/HoangNguyen0403/testguard/main/install.sh | bash
```

### Windows (PowerShell):
```powershell
irm https://raw.githubusercontent.com/HoangNguyen0403/testguard/main/install.ps1 | iex
```

### Go Install:
```bash
go install github.com/HoangNguyen0403/testguard/cmd/testguard@latest
```

---

## 🚀 Usage

Run `testguard` inside any repository. It automatically discovers project types with **zero configuration**:

```bash
# Full audit: Scan all test files across all modules (effective mode: full)
testguard --all

# Target a specific folder or single test file
testguard path/to/module/test/

# Allow empty scope (returns exit 3: not_applicable if no test declarations are found)
testguard --allow-empty path/to/empty/

# Output versioned machine-readable JSON report
testguard --json > audit.json

# Explain a specific rule and remediation steps
testguard explain GUARD-DART-SEED-ECHO
```

### CLI Modes & Flags

- **Full Scan Default**: Target directory or file inputs are audited in full (`effective_mode: "full"`).
- **`--all` (`-a`)**: Force full scan across all candidate test files.
- **`--allow-empty`**: Permits scanning targets with zero analyzable test declarations. Without this flag, discovering zero test declarations is considered an incomplete scan (exit `2`, `empty_scope`). With `--allow-empty`, an empty scan with no findings produces exit `3` (`not_applicable / empty_allowed`).
- **`--diff` (`-d`) Unsupported**: The `--diff` flag is not implemented and is explicitly rejected with exit `2` (`invalid_usage`). Scans operate in full target mode.
- **Limiting Scope**: Pass explicit paths, e.g. `testguard path/to/module`. Linked git worktrees and `vendor/` / `third_party/` directories are skipped automatically. Repository configuration files are not applied; `scope.config_applied` is explicitly `false`. TestGuard relies entirely on zero-config automated discovery.
- **Flag Compatibility Restrictions**:
  - **`--json` with `-f azure` is Rejected**: Combining `--json` with `-f azure` is invalid usage (exit `2`, `invalid_usage`). `--json` is compatible only with default console mode or explicit `-f json`.
  - **Explicit `--output` with `--json` / `-f azure` is Rejected**: Supplying `--output` (including empty values like `--output=""`) when running in `--json` or `-f azure` mode is invalid usage (exit `2`). `--output` is reserved strictly for console Markdown export, not a secondary file destination.

### Deterministic Exit Codes

TestGuard decouples analysis completeness from policy findings:

| Exit Code | Meaning | Analysis Status | Policy Status | Description |
|---|---|---|---|---|
| `0` | **Passed** | `complete` | `no_blocker` | Nonempty analysis completed successfully; no blocker violations found (advisories/warnings may exist). |
| `1` | **Blocker Found** | `complete` | `blocker_found` | Nonempty analysis completed successfully; at least one blocker violation detected. |
| `2` | **Incomplete / Error** | `incomplete` or `error` | `no_blocker` or `blocker_found` | Incomplete analysis or usage error: `--diff` rejected, unreadable/missing paths, traversal/parse errors, unsupported file extension targets, or empty test scope without `--allow-empty`. Retained findings survive. |
| `3` | **Not Applicable** | `not_applicable` | `no_blocker` | Empty test scope explicitly authorized via `--allow-empty` with zero findings. **Exit 3 is not passed**; it signifies the check was inapplicable. |

### Machine-Readable JSON Output (`schema_version: 1`)

When run with `--json` or `-f json`, TestGuard outputs a versioned JSON envelope (`ScanReport`) to `stdout`:

```json
{
  "schema_version": 1,
  "tool": {
    "name": "testguard",
    "version": "0.1.4",
    "commit": null,
    "built_at": null
  },
  "scope": {
    "requested_paths": ["."],
    "effective_mode": "full",
    "files": ["sample_test.go"],
    "excluded": [],
    "unsupported": [],
    "config_applied": false
  },
  "analysis": {
    "status": "complete",
    "reason": "analyzed",
    "discovered_files": 1,
    "analyzed_files": 1,
    "analyzed_tests": 1,
    "failed_files": 0
  },
  "findings": [],
  "diagnostics": [],
  "policy": {
    "blocker_count": 0,
    "status": "no_blocker"
  },
  "exit_code": 0
}
```

**Important for Consumers & Dashboards**:
1. Verify `"schema_version": 1`.
2. Cross-check the envelope's `"exit_code"` against the process exit code to ensure consistency.
3. Check `"analysis"."status"`: only when status is `"complete"` with exit code `0` can an empty findings array (`"findings": []`) be interpreted as clean test assurance.
4. **Retained Diagnostic Findings**: When analysis is `"incomplete"` or `"error"`, any findings collected prior to the failure survive in `.findings` and provide valuable diagnostic insight alongside `.diagnostics`. However, do not treat an incomplete scan as passing simply because blocker findings are absent.
### Heuristic Boundaries & Adoption Policy

- **Static Heuristics Only**: TestGuard inspects static AST token structures and syntactic patterns. It does not execute tests, invoke dynamic mutation testing harnesses, or verify runtime behavioral correctness.
- **No Production Assurance Claims**: TestGuard provides local-untrusted linting and anti-pattern detection; it does not certify production reliability or complete fault coverage.
- **Observation-First Adoption**: Teams should deploy TestGuard in CI under observation-only (non-blocking) mode. Activating blocking gates requires human owner review and calibrated baseline results. Rollback ownership is explicit: disable the blocking gate while retaining reporting artifacts; never convert missing evidence into success.
- **Unreleased Contract Boundary**: The 0.1.4 envelope and exit code semantics represent an unreleased specification boundary.

---

## 🎯 Supported Ecosystems & Rules

| Language / Scope | Test Runners | Key Guardrails |
|---|---|---|
| **Universal** | Polyglot | Zero assertions (`GUARD-UNIV-ZERO`), tautological assertions (`GUARD-UNIV-TAUTOLOGY`), committed skipped tests (`GUARD-UNIV-SKIP`), exact token duplicates (`GUARD-UNIV-01`), structural repetition (`GUARD-UNIV-02`), and placeholder test names (`GUARD-UNIV-NAME`). |
| **Dart / Flutter** | `flutter_test`, `bloc_test` | Traps seed echoes (`GUARD-DART-SEED-ECHO`), identical whole-scenario BLoC duplicates (`GUARD-DART-BLOC-DUPLICATE`), and pass-through mock echoes (`GUARD-DART-MOCK-ECHO`). |
| **Go** | `testing`, `testify` | Clean Architecture domain purity (zero mocks in domain models: `GUARD-GO-01`) and table-test duplicate input rows (`GUARD-GO-03`). |
| **TypeScript / Web** | `jest`, `vitest` | Real-timer sleeps > 0 ms in unit tests (`GUARD-REACT-02`), shallow mock echoes (`GUARD-REACT-01`), NestJS controller echoes & unmocked DB leaks, and massive snapshot-only smells (`GUARD-TS-01`). |
| **E2E / Integration** | `Playwright`, `Flutter IntegrationTest` | Fixed arbitrary hard waits in end-to-end tests (`GUARD-E2E-01`). |
| **Mobile (Android/iOS)** | `MockK`, `XCTest` | MockK pass-through echoes, unadvanced coroutines, and excessive expectation timeouts. |
---

## 📄 Documentation

- **[Quickstart Guide](QUICKSTART.md)** — Step-by-step installation, CI integration, and troubleshooting.
- **[Architecture & Design Guide](ARCHITECTURE.md)** — Core design principles, execution lifecycle, package boundaries, and extension recipes for maintainers and AI agents.
- **[Changelog](CHANGELOG.md)** — Release history and version updates.
---

## 📄 License

MIT © 2026 [Hoang Nguyen](https://github.com/HoangNguyen0403)
