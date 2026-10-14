# 🛡️ TestGuard

**Standalone, Zero-Config Polyglot Test Quality & Architectural Boundary Guard CLI**

TestGuard inspects test suites, detects duplicate/clone tests, flags vanity test anti-patterns (zero assertions, seed echoes, uncovered mutations), and enforces architectural boundaries across **Dart/Flutter**, **Go**, and **TypeScript/JavaScript**.

---

## ⚡ Quickstart

### Native Install

**macOS & Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/HoangNguyen0403/testguard/main/install.sh | bash
```

**Windows (PowerShell):**
```powershell
irm https://raw.githubusercontent.com/HoangNguyen0403/testguard/main/install.ps1 | iex
```

**Go Install:**
```bash
go install github.com/HoangNguyen0403/testguard/cmd/testguard@latest
```

---

## 🚀 Usage

Run `testguard` inside any repository. It automatically discovers project types with **zero configuration**:

```bash
# Day-to-day: Scan only your active branch/PR changes
testguard --diff

# Full audit: Scan all test files across all modules
testguard --all

# Target a specific folder or single test file
testguard path/to/module/test/

# Explain a specific rule and remediation steps
testguard explain GUARD-DART-SEED-ECHO
```

---

## 🎯 Supported Ecosystems & Rules

| Language | Test Runners | Key Guardrails |
|---|---|---|
| **Dart / Flutter** | `flutter_test`, `bloc_test` | Traps disguised state mirrors, seed echoes, and uncovered loading mutations (`isFetching = true`). |
| **Go** | `testing`, `testify` | Clean Architecture domain purity (zero mocks in domain models) and port-only mocking. |
| **TypeScript / Web** | `jest`, `vitest` | Catches shallow mock echoes and snapshot-only test smells. |

---

## 📄 License

MIT © 2026 [Hoang Nguyen](https://github.com/HoangNguyen0403)
