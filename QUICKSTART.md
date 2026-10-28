# TestGuard Quickstart

> Install TestGuard in your terminal and run your first test quality & boundary audit in under 60 seconds.

This quickstart covers installing the standalone `testguard` CLI on macOS, Linux, or Windows, running your first zero-config scan, inspecting rule rationales, and adding TestGuard to your CI pipelines.

---

## Before you begin

Make sure you have:
* A terminal or command prompt open.
* A code project containing unit tests (**Dart/Flutter**, **Go**, or **TypeScript/JavaScript**).
* *Note: TestGuard is 100% standalone and requires zero external runtimes or skill configurations.*

---

## Step 1: Install TestGuard

Open your terminal and run the command for your operating system:

### macOS, Linux, and WSL (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/HoangNguyen0403/testguard/main/install.sh | bash
```

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/HoangNguyen0403/testguard/main/install.ps1 | iex
```

### Go Developers (Any Platform)

```bash
go install github.com/HoangNguyen0403/testguard/cmd/testguard@latest
```

---

## Step 2: Verify your installation

In a fresh terminal window, run:

```bash
testguard --version
```

A working installation prints:
```text
testguard version 0.1.0 (commit: abc1234, built at: 2026-10-14)
```

> 💡 **Troubleshooting PATH**: If your terminal reports `testguard: command not found`, ensure `/usr/local/bin` (or `~/.local/bin` on Linux, or `%LOCALAPPDATA%\Programs\testguard` on Windows) is in your shell's `PATH`.

---

## Step 3: Run your first zero-config scan

Navigate to any repository or monorepo workspace and run:

```bash
cd /path/to/your/project
testguard
```

TestGuard automatically detects your project types (`pubspec.yaml`, `go.mod`, `package.json`) and runs all matching guardrails with **zero configuration**:

```text
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🛡️ TestGuard Audit · 42 files scanned · 0 Parse Errors · 0 Skipped
Status: PASSED · 🚨 0 Blocker(s) · ⚠️ 1 Warning(s) · 💡 2 Advisory(ies)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

test/application/account/customer_code_bloc_test.dart:497:1
  💡 [GUARD-UNIV-01] Duplicate test body: 'Customer Code Auto-Search Success' has identical execution tokens to 'Customer Code Search Success' (line 299).
  💡 Fix: Check whether this test exercises a distinct edge case or consolidate into a table-driven parameterized test.
```

---

## Step 4: Target specific modules and handle empty scopes (`--allow-empty`)

### Full Mode Execution
TestGuard executes full target audits across all discovered test files (`effective_mode: "full"`).

> ⚠️ **Note on `--diff` / `-d`**: The `--diff` flag is unsupported in TestGuard 0.1.4 and is explicitly rejected with exit code `2` (`invalid_usage`). Scans do not calculate git diffs; they analyze target files in full.

To target a specific directory or individual test file during development:

```bash
testguard test/application/account/
```

### Handling Scopes Without Tests (`--allow-empty`)
By default, discovering zero analyzable test declarations in the target scope is treated as an incomplete scan and exits with code `2` (`incomplete / empty_scope`).

When targeting directories that may legitimately contain no test files or declarations, pass `--allow-empty`:

```bash
testguard --allow-empty test/empty_module/
```

If no test declarations are found and no errors or findings occurred, TestGuard exits with code `3` (`not_applicable / empty_allowed`). Exit `3` explicitly signifies that the check was inapplicable, rather than claiming tests passed.

> ℹ️ **Limiting Scope**: To limit scan scope, pass explicit paths, e.g. `testguard path/to/module`. Linked git worktrees and `vendor/` / `third_party/` directories are skipped automatically. Repository configuration files are not applied (`config_applied: false`). TestGuard relies purely on automated zero-config discovery.

### Flag Compatibility Rules
- **`--json` with `-f azure` is Rejected**: Combining `--json` with `-f azure` is invalid usage and exits with code `2` (`invalid_usage`). Use `--json` with the default console reporter or explicit `-f json`.
- **Explicit `--output` with `--json` / `-f azure` is Rejected**: Providing `--output` (including empty strings like `--output=""`) in `--json` or `-f azure` mode is invalid usage and exits with code `2` (`invalid_usage`). The `--output` flag is reserved solely for console Markdown export.
---

## Step 5: Understand why a rule fired (`testguard explain`)

Every TestGuard rule contains embedded, offline documentation. When you see a rule violation, inspect its rationale and fix guidance:

```bash
testguard explain GUARD-DART-SEED-ECHO
```

Outputs:
```text
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
📖 Rule: GUARD-DART-SEED-ECHO — Seed Echo Assertion in BLoC Test [Warning | Dart]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📝 Description:
  Expected state properties merely echo values already established in the seed baseline.

💡 Why It Matters:
  Asserting unchanged fields over-specifies tests and breaks resistance to refactoring.

🛠️ Remediation:
  Use base state fixtures and assert only the mutated fields (delta) in state transitions.

✅ Good Pattern:
  baseState.copyWith(isFetching: true)

❌ Bad Pattern:
  isA<State>().having((s) => s.userInfo, 'userInfo', fakeUser) // when userInfo is invariant
```

To list all available rules:
```bash
testguard rules
```

---

## Step 6: Integrate with CI (Observation-First Adoption & Rollback)

### Observation-First Adoption Policy
When introducing TestGuard to an existing repository, deploy it in **observation-only (non-blocking)** mode. Do not configure TestGuard to block pull requests immediately.

1. **Observation Baseline**: Run TestGuard as an advisory step to collect baseline findings and calibrate rule applicability.
2. **Promoting to Blocking**: Converting TestGuard into a blocking PR gate requires explicit human owner approval and empirical calibration evidence.
3. **Rollback Ownership**: If false alerts or pipeline disruptions occur, rollback ownership is clear: disable the blocking gate (e.g. set `continue-on-error: true` or decouple the status check) while preserving artifact generation and existing unrelated gates. Never convert missing evidence or errors into passing status as a rollback.

### Azure DevOps Pipelines (`azure-pipelines.yml`)

```yaml
- script: |
    curl -fsSL https://raw.githubusercontent.com/HoangNguyen0403/testguard/main/install.sh | bash
    # Observation-only mode: run scan, report actual exit, let pipeline continue
    testguard -f azure
  displayName: '🛡️ TestGuard Test Quality Audit (Observation)'
  continueOnError: true
```

### GitHub Actions (`.github/workflows/testguard.yml`)

```yaml
name: Test Quality Audit (Observation)

on: [pull_request]

jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: curl -fsSL https://raw.githubusercontent.com/HoangNguyen0403/testguard/main/install.sh | bash
      # Observation-only step: captures actual scan exit code without || true masking
      - id: scan
        run: testguard --json > testguard-report.json
        continue-on-error: true
      # Explicitly validate and diagnose captured report on every run
      - name: Diagnose report capture
        if: always()
        run: |
          if [ ! -s testguard-report.json ]; then
            echo "::warning::TestGuard report is missing or empty (scan step exit: ${{ steps.scan.outcome }})"
          elif ! jq -e '.schema_version == 1' testguard-report.json > /dev/null 2>&1; then
            echo "::warning::TestGuard report contains invalid or unsupported JSON"
          else
            echo "TestGuard report captured successfully"
          fi
      # Unconditionally upload report artifact for review
      - uses: actions/upload-artifact@v4
        if: always()
        with:
          name: testguard-report
          path: testguard-report.json
```

---

## Step 7: Consuming Machine-Readable JSON (`schema_version: 1`)

TestGuard produces a single, top-level versioned `ScanReport` envelope on `stdout` when invoked with `--json` or `-f json`.

### Migration from Bare Array to Versioned Envelope

In earlier unversioned releases, TestGuard emitted a bare array of violations (`[]`). In v0.1.4, output is wrapped in a versioned envelope:

```bash
# ❌ DEPRECATED (0.1.x bare array pattern):
# jq '.[] | select(.severity=="Blocker")' audit.json

# ✅ v0.1.4 Versioned Envelope Pattern:
# 1. Verify schema version and check if analysis completed successfully
jq -e '.schema_version == 1 and .analysis.status == "complete"' audit.json > /dev/null || {
  echo "Notice: Scan incomplete, failed, or schema incompatible; inspecting diagnostics and retained findings" >&2
}

# 2. Extract blockers or inspect findings
jq '.findings[] | select(.severity == "Blocker")' audit.json
```

### Automated Pipeline Parsing & Exit Handling

When writing automation scripts, capture the process exit code safely under `errexit` (`set -e`), validate the JSON envelope, cross-check process and report exits, handle diagnostic findings, and preserve the original exit code:

```bash
#!/usr/bin/env bash
set -euo pipefail

# Capture process exit safely under errexit
if testguard --json > audit.json; then
  scan_exit=0
else
  scan_exit=$?
fi

# 1. Verify output file exists and is nonempty
if [ ! -s audit.json ]; then
  echo "Error: TestGuard produced empty or missing JSON output" >&2
  exit 2
fi

# 2. Validate JSON envelope and schema_version
if ! report_json=$(jq -e '.' audit.json 2>/dev/null); then
  echo "Error: TestGuard emitted malformed JSON" >&2
  exit 2
fi

schema_version=$(echo "$report_json" | jq -r '.schema_version // empty')
if [ "$schema_version" != "1" ]; then
  echo "Error: Unexpected schema_version (expected 1, got '$schema_version')" >&2
  exit 2
fi

# 3. Verify consistency between process exit code and report exit_code field
report_exit=$(echo "$report_json" | jq -r '.exit_code // empty')
if [ "$report_exit" != "$scan_exit" ]; then
  echo "Error: Exit code mismatch: process returned $scan_exit, report recorded $report_exit" >&2
  exit 2
fi

analysis_status=$(echo "$report_json" | jq -r '.analysis.status // "unknown"')
analysis_reason=$(echo "$report_json" | jq -r '.analysis.reason // "unknown"')

case "$scan_exit" in
  0)
    if [ "$analysis_status" != "complete" ]; then
      echo "Error: Inconsistent report: exit 0 but analysis.status is '$analysis_status'" >&2
      exit 2
    fi
    echo "Audit Complete: No blocker findings."
    ;;
  1)
    if [ "$analysis_status" != "complete" ]; then
      echo "Error: Inconsistent report: exit 1 but analysis.status is '$analysis_status'" >&2
      exit 2
    fi
    echo "Audit Complete: Blocker violations detected."
    echo "$report_json" | jq '.findings[] | select(.severity == "Blocker")'
    ;;
  2)
    echo "Audit Incomplete or Tool Error (status: $analysis_status, reason: $analysis_reason)." >&2
    echo "Diagnostics:" >&2
    echo "$report_json" | jq -r '.diagnostics[]? | "  [\(.code)] \(.path // ""): \(.message)"' >&2
    # Retained findings survive in incomplete scans and provide useful diagnostics
    retained_count=$(echo "$report_json" | jq '.findings | length')
    if [ "$retained_count" -gt 0 ]; then
      echo "Retained diagnostic findings ($retained_count):" >&2
      echo "$report_json" | jq '.findings[]' >&2
    fi
    ;;
  3)
    echo "Audit Not Applicable (status: $analysis_status, reason: $analysis_reason): Authorized empty scope (--allow-empty)."
    ;;
  *)
    echo "Unexpected scan exit code: $scan_exit" >&2
    exit 2
    ;;
esac

# Return the validated original process exit code
exit "$scan_exit"
```
---

## Essential Commands Reference

| Command | What it does | Example |
|---|---|---|
| `testguard` | Run full scan across all candidate test files | `testguard` |
| `testguard --all` (`-a`) | Explicit full suite scan | `testguard -a` |
| `testguard <path>` | Target a specific folder or test file | `testguard test/application/` |
| `testguard --allow-empty <path>` | Scan target allowing zero test declarations (exit 3) | `testguard --allow-empty src/` |
| `testguard explain <rule>` | Show self-contained rationale & fix examples | `testguard explain GUARD-UNIV-ZERO` |
| `testguard rules` | List all built-in rules across Dart, Go, and TS | `testguard rules` |
| `testguard --json` (`-f json`) | Output versioned machine-readable JSON envelope | `testguard --json > audit.json` |
| `testguard --diff` (`-d`) | **Unsupported / Rejected**: Exits with code 2 (`invalid_usage`) | `testguard -d` (rejected) |

---

## Deterministic Exit Codes

| Exit Code | Status Category | `analysis.status` | Description |
|---|---|---|---|
| `0` | **Complete / Passed** | `complete` | Nonempty analysis completed; zero blocker violations found (warnings/advisories may exist). |
| `1` | **Complete / Blocker** | `complete` | Nonempty analysis completed; at least one `Blocker` violation detected in `.findings`. |
| `2` | **Incomplete / Error** | `incomplete` / `error` | Usage error (`--diff` flag rejected), traversal failure, file read/parse error, explicit unsupported target, or empty test scope without `--allow-empty`. Retained findings survive. |
| `3` | **Not Applicable** | `not_applicable` | Empty test scope explicitly authorized via `--allow-empty` and zero findings exist. **Exit 3 is not passed**; it indicates the check is inapplicable. |

---

## Heuristic Limits & Production Assurance Boundary

- **Static Pattern Matching**: TestGuard uses static AST token analysis and pattern recognition. It does **not** execute test suites or run dynamic mutation testing.
- **No Production Assurance Claims**: TestGuard does not certify that tests kill production bugs or satisfy behavioral correctness.
- **Unreleased Contract Boundary**: The v0.1.4 JSON envelope, exit code precedence, and `--allow-empty` boundary represent an unreleased specification. Do not claim published or released status.
