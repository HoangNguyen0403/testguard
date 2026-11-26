# 🛡️ TestGuard

**Standalone, Zero-Config Polyglot Test Quality & Architectural Boundary Guard CLI**

TestGuard inspects test suites, detects duplicate/clone tests, flags vanity test anti-patterns (zero assertions, seed echoes, mock echoes, weak oracles, skipped or focused tests), and enforces architectural boundaries across **Dart/Flutter**, **Go**, **TypeScript/JavaScript**, **Kotlin**, and **Swift**.

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

### Go Install (pinned to the current release):
```bash
go install github.com/HoangNguyen0403/testguard/cmd/testguard@v0.1.7
```

---

## 🚀 Usage

Run `testguard` inside any repository. It automatically discovers project types with **zero configuration**:

```bash
# Full audit: Scan all test files across all modules (effective mode: full)
testguard --all

# Diff audit: Scan only git modified/untracked test files against base branch/ref
testguard --diff --base origin/develop

# Target a specific folder or single test file
testguard path/to/module/test/

# Allow empty scope (returns exit 3: not_applicable if no test declarations are found)
testguard --allow-empty path/to/empty/

# Output versioned machine-readable JSON report
testguard --json > audit.json

# Explain a specific rule and remediation steps
testguard explain GUARD-UNIV-IDENTITY-FIXTURE
```

### CLI Modes & Flags

- **Full Scan Default**: Target directory or file inputs are audited in full (`effective_mode: "full"`).
- **`--all` (`-a`)**: Force full scan across all candidate test files. Mutually exclusive with `--diff`.
- **`--diff` (`-d`)**: Scans only modified and untracked test files from git diff (`git diff --name-only`). Accepts `--base <ref>` (defaults to clean-repo fallback, e.g. `origin/main`, `origin/develop`, or `HEAD~1`). Exits `3` (`not_applicable`, reason: `empty_allowed`) when git diff contains no modified test files under `--allow-empty`.
- **`--allow-empty`**: Permits scanning targets with zero analyzable test declarations. Without this flag, discovering zero test declarations is considered an incomplete scan (exit `2`, `empty_scope`). With `--allow-empty`, an empty scan with no findings produces exit `3` (`not_applicable / empty_allowed`).
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
| `2` | **Incomplete / Error** | `incomplete` or `error` | `no_blocker` or `blocker_found` | Incomplete analysis or usage error: `--all` and `--diff` combined, unreadable/missing paths, traversal/parse errors, unsupported file extension targets, or empty test scope without `--allow-empty`. Retained findings survive. |
| `3` | **Not Applicable** | `not_applicable` | `no_blocker` | Empty test scope explicitly authorized via `--allow-empty` with zero findings. **Exit 3 is not passed**; it signifies the check was inapplicable. |

### Machine-Readable JSON Output (`schema_version: 1`)

When run with `--json` or `-f json`, TestGuard outputs a versioned JSON envelope (`ScanReport`) to `stdout`:

```json
{
  "schema_version": 1,
  "tool": {
    "name": "testguard",
    "version": "0.1.7",
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
- **Documented Recall Gaps**: Constructs the engine deliberately does not report (cross-file helpers, hollow tests hidden behind unreferenced nested-function assertions, BLoC shapes outside the modelled subset, adapter lexical boundaries) are listed in `ARCHITECTURE.md` §8. Absence of a finding is not proof that a test is effective.
- **No Production Assurance Claims**: TestGuard provides local-untrusted linting and anti-pattern detection; it does not certify production reliability or complete fault coverage.
- **Observation-First Adoption**: Teams should deploy TestGuard in CI under observation-only (non-blocking) mode. Activating blocking gates requires human owner review and calibrated baseline results. Rollback ownership is explicit: disable the blocking gate while retaining reporting artifacts; never convert missing evidence into success.
- **Published Contract Boundary**: `schema_version: 1`, the exit-code semantics, `--allow-empty`, and `--diff` are published in TestGuard v0.1.6 and v0.1.7. Verify the installed binary's `--version` before depending on a specific envelope field; this documentation describes v0.1.7.

---

## 🎯 Supported Ecosystems & Rules

| Language / Scope | Test Runners | Key Guardrails |
|---|---|---|
| **Universal** | Polyglot | Zero assertions (`GUARD-UNIV-ZERO`), tautological assertions (`GUARD-UNIV-TAUTOLOGY`), committed skipped tests (`GUARD-UNIV-SKIP`), focused tests (`GUARD-UNIV-FOCUS`, TypeScript and Dart), exact duplicate test bodies (`GUARD-UNIV-01`), structural repetition (`GUARD-UNIV-02`), placeholder test names (`GUARD-UNIV-NAME`), and non-identity math & boundary fixtures (`GUARD-UNIV-IDENTITY-FIXTURE`). |
| **Dart / Flutter** | `flutter_test`, `bloc_test` | Seed echoes (`GUARD-DART-SEED-ECHO`), vacuous base-state assertions (`GUARD-DART-VACUOUS-STATE`), identical whole-scenario BLoC duplicates (`GUARD-DART-BLOC-DUPLICATE`), and pass-through mock echoes (`GUARD-DART-MOCK-ECHO`). |
| **Go** | `testing`, `testify`, `gomega`, `quicktest` | Domain-test mock usage as an advisory convention note (`GUARD-GO-01`), table-test duplicate input rows (`GUARD-GO-03`), swallowed errors via bare `return` (`GUARD-GO-SWALLOWED-ERR`), declared-but-unread table expectations (`GUARD-GO-UNCHECKED-WANT`), and payload verification required (`GUARD-GO-EXISTENCE-ONLY`). |
| **TypeScript / Web** | `jest`, `vitest` | Real-timer sleeps in unit tests (`GUARD-REACT-02`), snapshot-only assertions (`GUARD-TS-01`), floating promise assertions (`GUARD-TS-FLOATING-ASSERT`), catch-only error paths (`GUARD-TS-CATCH-ONLY`), conditionally reached assertions (`GUARD-TS-CONDITIONAL-ASSERT`), existence-only oracles (`GUARD-TS-EXISTENCE-ONLY`), self-mocking assertions (`GUARD-TS-SELF-MOCK`), auto-mocked subjects (`GUARD-TS-MOCKED-SUT`), static GraphQL AST / JSON key pinning ban (`GUARD-TS-STATIC-PINNING`), twin `.test.js` + `.test.ts` file ban (`GUARD-TS-DUAL-EXT`), and NestJS controller echoes / unmocked database modules (`GUARD-NEST-01`, `GUARD-NEST-02`). |
| **E2E / Integration** | `Playwright`, `Flutter IntegrationTest` | Fixed arbitrary hard waits in end-to-end tests (`GUARD-E2E-01`). |
| **Mobile (Android/iOS)** | `MockK`, `XCTest` | MockK pass-through echoes (`GUARD-KT-01`), unadvanced coroutine virtual time under `StandardTestDispatcher` (`GUARD-KT-02`), and long expectation timeouts (`GUARD-SWIFT-01`). |

Run `testguard rules` for the authoritative list with severities; `testguard explain <RULE-ID>` prints rationale and remediation.
---

## CI Integration

TestGuard's process exit code is authoritative: `0` passed, `1` blocker, `2` incomplete/error, and `3` not applicable. Each example writes the raw `schema_version: 1` JSON report and preserves the scanner's exit status; none uses `jq`, `|| true`, or a second status gate. The JSON consumer contract is in [QUICKSTART Step 7](QUICKSTART.md#step-7-consuming-machine-readable-json-schema_version-1).

Start by observing a real baseline before making TestGuard a required check. Observation mode changes whether CI blocks; it must not rewrite the scanner's JSON or report exit code:

- **GitHub Actions:** In a standalone workflow, set `continue-on-error: true` on the `testguard` job for observation. A caller of the reusable workflow below cannot set `continue-on-error` on its `uses` job; use the standalone workflow when a non-blocking observation period is required.
- **GitLab CI:** Set `allow_failure: true` on the `testguard` job for observation.
- **Azure Pipelines:** Set `continueOnError: true` on the scan script step for observation. Keep artifact publication conditioned on `succeededOrFailed()` and the scan-start signal shown below; do not use a `SucceededWithIssues` result as an enforcement decision.

### GitHub Actions (standalone)

```yaml
name: TestGuard

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  testguard:
    runs-on: ubuntu-24.04
    timeout-minutes: 20
    # For observation-only rollout, set continue-on-error: true here.
    steps:
      - uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
        with:
          persist-credentials: false
          fetch-depth: 1
      - uses: actions/setup-go@d35c59abb061a4a6fb18e82ac0862c26744d6ab5 # v5.5.0
        with:
          go-version: "1.22.x"
          cache: false
      - name: Install TestGuard v0.1.7
        shell: bash
        env:
          GOBIN: ${{ runner.temp }}/testguard-bin
          GOMODCACHE: ${{ runner.temp }}/go-mod-cache
          GOCACHE: ${{ runner.temp }}/go-build-cache
          GOTOOLCHAIN: local
        run: |
          set -euo pipefail
          mkdir -p "$GOBIN" "$GOMODCACHE" "$GOCACHE"
          go install github.com/HoangNguyen0403/testguard/cmd/testguard@v0.1.7
      - name: Run TestGuard
        id: scan
        shell: bash
        run: |
          "$RUNNER_TEMP/testguard-bin/testguard" --all --json > "$RUNNER_TEMP/testguard-report.json"
      - name: Upload raw TestGuard report
        if: ${{ !cancelled() && (steps.scan.outcome == 'success' || steps.scan.outcome == 'failure') }}
        uses: actions/upload-artifact@65c4c4a1ddee5b72f698fdd19549f0f0fb45cf08 # v4.6.0
        with:
          name: testguard-report
          path: ${{ runner.temp }}/testguard-report.json
          if-no-files-found: error
```

### GitHub Actions (reusable workflow)

```yaml
name: TestGuard

on:
  pull_request:
  push:
    branches: [main]

permissions:
  contents: read

jobs:
  testguard:
    uses: HoangNguyen0403/testguard/.github/workflows/testguard.yml@<40-character-merged-commit-sha>
```

The reusable workflow is source-only until its commit is merged, and published tags `v0.1.6` and `v0.1.7` do not contain it. After merge, replace the placeholder with the full immutable 40-character SHA of the merged commit that contains the workflow. Do not use `@main` or a mutable tag. It checks out the caller repository read-only, installs the pinned `v0.1.7` CLI through Go's module proxy and checksum database, scans all tests, uploads the raw report, and returns the scanner's own status. No secrets or write permissions are required.

### GitLab CI

```yaml
testguard:
  image: golang:1.22
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
    - if: '$CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH'
    - when: never
  script:
    - |
      set -euo pipefail
      rm -f -- testguard-report.json
      bin_dir="$(mktemp -d)"
      trap 'rm -rf "$bin_dir"' EXIT
      export GOBIN="$bin_dir"
      go install github.com/HoangNguyen0403/testguard/cmd/testguard@v0.1.7
      "$GOBIN/testguard" --all --json > testguard-report.json
  artifacts:
    when: always
    paths:
      - testguard-report.json
    expire_in: 1 week
```

`artifacts: when: always` preserves the report when the scan exits non-zero. For observation-only rollout, add `allow_failure: true` to the job; do not change the scan command. `golang:1.22` requires a GitLab Runner able to run OCI container images.

### Azure Pipelines (Services)

```yaml
trigger:
  branches:
    include: [main]
pr:
  branches:
    include: [main]

pool:
  vmImage: ubuntu-24.04

steps:
  - checkout: self
    persistCredentials: false
  - task: GoTool@0
    inputs:
      version: "1.26.2"
  - script: |
      set -euo pipefail
      export GOBIN="$(Agent.TempDirectory)/testguard-bin"
      mkdir -p "$GOBIN" "$(Build.ArtifactStagingDirectory)"
      go install github.com/HoangNguyen0403/testguard/cmd/testguard@v0.1.7
      echo "##vso[task.setvariable variable=TestGuardScanStarted]true"
      "$GOBIN/testguard" --all --json > "$(Build.ArtifactStagingDirectory)/testguard-report.json"
    displayName: Run TestGuard
    # For observation-only rollout, add continueOnError: true here.
  - task: PublishPipelineArtifact@1
    condition: and(succeededOrFailed(), eq(variables['TestGuardScanStarted'], 'true'))
    inputs:
      targetPath: "$(Build.ArtifactStagingDirectory)/testguard-report.json"
      artifact: testguard-report
```

The scan-start variable prevents publication after installation fails; `succeededOrFailed()` still publishes the report after a non-zero scan. `PublishPipelineArtifact@1` applies to Azure DevOps Services; Azure DevOps Server users should use `PublishBuildArtifacts@1` instead. YAML `pr:` triggers apply to GitHub- and Bitbucket-backed Azure Pipelines repositories. For **Azure Repos**, configure a branch policy with **Build validation** pointing to this pipeline; YAML `pr:` does not trigger Azure Repos PR validation.

### Generic shell runner

```bash
#!/usr/bin/env bash
set -euo pipefail

bin_dir="$(mktemp -d)"
trap 'rm -rf "$bin_dir"' EXIT
GOBIN="$bin_dir" go install github.com/HoangNguyen0403/testguard/cmd/testguard@v0.1.7
"$bin_dir/testguard" --all --json > testguard-report.json
```

The report file is the artifact for a generic shell runner; retain it using that runner's artifact mechanism. The pinned `go install` resolves through the Go module proxy and checksum database. These examples do not install from mutable `@latest` or write to the checked-out repository in the reusable GitHub workflow.

---

## 📄 Documentation

- **[Quickstart Guide](QUICKSTART.md)** — Step-by-step installation, CI integration, and troubleshooting.
- **[Architecture & Design Guide](ARCHITECTURE.md)** — Core design principles, execution lifecycle, package boundaries, and extension recipes for maintainers and AI agents.
- **[Changelog](CHANGELOG.md)** — Release history and version updates.
- **[Popular GitHub Corpus Benchmark](docs/benchmarks/github-corpus.md)** — Pinned public-repository scans, sampled finding adjudication, and remaining accuracy limits.
---

## 📄 License

MIT © 2026 [Hoang Nguyen](https://github.com/HoangNguyen0403)
