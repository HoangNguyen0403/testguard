# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
