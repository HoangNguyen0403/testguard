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

## Step 4: Scan only your active branch changes (PR Diff Mode)

When developing a feature or bug fix, avoid noise from legacy files you haven't touched:

```bash
testguard --diff
```

TestGuard automatically compares your branch against `origin/develop` (or `origin/main`), reporting only violations introduced in your active diff.

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

## Step 6: Add TestGuard to your CI Pipeline

### Azure DevOps Pipelines (`azure-pipelines.yml`)

```yaml
- script: |
    curl -fsSL https://raw.githubusercontent.com/HoangNguyen0403/testguard/main/install.sh | bash
    testguard --diff -f azure
  displayName: '🛡️ TestGuard Test Quality Gate'
```

### GitHub Actions (`.github/workflows/testguard.yml`)

```yaml
name: Test Quality Gate

on: [pull_request]

jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0 # Full history for diff resolution
      - run: curl -fsSL https://raw.githubusercontent.com/HoangNguyen0403/testguard/main/install.sh | bash
      - run: testguard --diff
```

---

## Essential Commands Reference

| Command | What it does | Example |
|---|---|---|
| `testguard` | Run smart-default audit (diff on branch, all on main) | `testguard` |
| `testguard --diff` (`-d`) | Scan only files modified against base branch | `testguard -d` |
| `testguard --all` (`-a`) | Force full suite scan across all modules | `testguard -a` |
| `testguard <path>` | Target a specific folder or test file | `testguard test/application/` |
| `testguard explain <rule>` | Show self-contained rationale & fix examples | `testguard explain GUARD-UNIV-ZERO` |
| `testguard rules` | List all built-in rules across Dart, Go, and TS | `testguard rules` |
| `testguard -f json` | Output machine-readable JSON for dashboards | `testguard -f json > audit.json` |

---

## Exit Codes

| Exit Code | Meaning |
|---|---|
| `0` | **Passed**: All tests clean or only `Advisory` warnings exist. |
| `1` | **Gate Failed**: At least one `Blocker` violation detected. Merge should be blocked. |
| `2` | **Tool Error**: Invalid arguments, shallow git clone, or unreadable path. |
