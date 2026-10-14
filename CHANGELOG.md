# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
