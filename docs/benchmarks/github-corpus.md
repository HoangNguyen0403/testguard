# Public GitHub corpus: diagnostic benchmark

## Scope and method

Each supported language was searched independently with GitHub's repository search (`language:<L> fork:false archived:false stars:>1000`, sorted by stars) on 2026-11-25. The three highest-starred repositories with a verifiable open-source license were attempted first; where a top result had no applicable supported tests (curated lists, empty scope) the next test-bearing repositories were added and the empty result is **kept and reported, never counted as a pass**. Across all languages this yielded **26 unique attempted repositories**, not the top three only and not one ranked set. JavaScript is treated separately from TypeScript. GitHub's "language" label is repository-level, so ranked results include non-test-bearing and tool repositories; selection is purposive and **not a ranking of test quality**. Star counts are discovery-time values.

Only TestGuard ran. No downloaded script, dependency installer, upstream test or mutation tool executed. Source was fetched as pinned commit archives and, for the five supplementary repositories (vuejs/core, gin-gonic/gin, felangel/bloc, lysine-dev/okhttp, Alamofire/Alamofire), as full shallow clones. Archive checkouts were projected to regular supported-language source files and module manifests; the retained shallow clones kept documentation, Xcode projects and package files (the Alamofire clone is one example). Non-source assets and symlinks were not retained (`omitted_source_links` is recorded per repository in the evidence manifest). Therefore results describe the scanned source projection, not whole-repository assurance, and a reduced archive projection is not interchangeable with an original clone. Each scan used `testguard --all --json <checkout>`. `analyzed_tests` counts static declaration sites, not executed or parameter-expanded test cases.

Binary identities: baseline = commit `4faae296ee7dda96d2d894e5e58b790f2b533a23` (SHA-256 `4c5e61a5…c965909`); final = the same commit plus the uncommitted changes in this change set (SHA-256 `f032cb95aa869eb9042d263de1fd72fa4b0b397dcfe535a951e60945dccde28b`). Byte hashes are local evidence identities and change with toolchain and build path.

## Corpus and result per repository

"Before" and "after" are the baseline and final binaries on identical pinned source. All exit codes matched their JSON envelopes. Exit 1 is an observed policy result (blockers retained); exit 2 is an incomplete scan and was never converted to success.

| Language | Repository (commit) | License | Before | After |
|---|---|---|---|---|
| Go | [avelino/awesome-go](https://github.com/avelino/awesome-go) (`2520247d`) | MIT | complete; 5 files; 1 finding | unchanged |
| Go | [ollama/ollama](https://github.com/ollama/ollama) (`b207a5a3`) | MIT | complete; 575 files; 627 findings | complete; 540 findings |
| Go | [golang/go](https://github.com/golang/go) (`f9d574dc`) | BSD-3-Clause | **incomplete**; 2,023 files analyzed, 7 parse failures; 2,138 findings | still **incomplete** (same 7); 2,098 findings |
| Go | [gin-gonic/gin](https://github.com/gin-gonic/gin) (`0f09c3a9`) | MIT | complete; 5 findings | 3 findings |
| Dart | [flutter/flutter](https://github.com/flutter/flutter) (`67fec024`) | BSD-3-Clause | complete; 2,891 files; 30,786 decl; 1,239 findings | unchanged |
| Dart | [localsend/localsend](https://github.com/localsend/localsend) (`44f48f18`) | Apache-2.0 | complete; 2 findings | unchanged |
| Dart | [AppFlowy-IO/AppFlowy](https://github.com/AppFlowy-IO/AppFlowy) (`8fcee27d`) | AGPL-3.0 | complete; 47 findings | unchanged |
| Dart (supplement) | [felangel/bloc](https://github.com/felangel/bloc) (`654f15a5`) | MIT | complete; 23 findings | unchanged |
| TypeScript | [morluto/rea](https://github.com/morluto/rea) (`3dae5671`) | MIT | **incomplete** (2 parse failures); 247 findings | complete; 28,162 decl; 221 findings |
| TypeScript | [freeCodeCamp/freeCodeCamp](https://github.com/freeCodeCamp/freeCodeCamp) (`52030263`) | BSD-3-Clause | complete; 101 findings | unchanged |
| TypeScript | [openclaw/openclaw](https://github.com/openclaw/openclaw) (`54dc5639`) | MIT | **timeout after 180 s, no report** | complete; 24,248 files / 149,370 decl; 1,412 findings; 150 s (one local run) |
| TypeScript (supplement) | [vuejs/core](https://github.com/vuejs/core) (`50aaa3c7`) | MIT | complete; 358 findings | 352 findings |
| JavaScript | [affaan-m/ECC](https://github.com/affaan-m/ECC) (`4adeee67`) | MIT | **incomplete** (9 parse failures); 1,043 findings | complete; 7,638 decl; 1,126 findings |
| JavaScript | [react/react](https://github.com/react/react) (`edae846a`) | MIT | complete but **only 16 files discovered**; 49 findings | complete; **376 files**; 957 findings |
| JavaScript | [DietrichGebert/ponytail](https://github.com/DietrichGebert/ponytail) (`ebd8654c`) | MIT | complete; 27 findings | unchanged |
| Kotlin | [2dust/v2rayNG](https://github.com/2dust/v2rayNG) (`71631de1`) | GPL-3.0 | complete; 1 finding | unchanged |
| Kotlin | [topjohnwu/Magisk](https://github.com/topjohnwu/Magisk) (`a212cd92`) | GPL-3.0 | complete; 2 findings | 0 findings |
| Kotlin | [JetBrains/kotlin](https://github.com/JetBrains/kotlin) (`e02fc449`) | Apache-2.0 (`license/LICENSE.txt`) | **incomplete** (1 parse failure); 2,140 findings | complete; 2,004 findings |
| Kotlin (supplement) | [lysine-dev/okhttp](https://github.com/lysine-dev/okhttp) (`223a0cc7`) | Apache-2.0 | complete; 248 findings | 243 findings |
| Swift | [jaywcjlove/awesome-mac](https://github.com/jaywcjlove/awesome-mac) (`e4cf8eb3`) | CC0-1.0 | incomplete: `empty_scope` | unchanged (not applicable, not clean) |
| Swift | [swiftlang/swift](https://github.com/swiftlang/swift) (`b70bb115`) | Apache-2.0 | complete; 121 files; 113 findings; 105 s | 113 findings; 0.27 s (single runs) |
| Swift | [vsouza/awesome-ios](https://github.com/vsouza/awesome-ios) (`630ca87d`) | MIT | incomplete: `empty_scope` | unchanged |
| Swift (supplement) | [apple/container](https://github.com/apple/container) (`16e8d488`) | Apache-2.0 | complete; 92 files; 113 findings | 59 findings |
| Swift (supplement) | iina/iina, exelban/stats | GPL-3.0, MIT | incomplete: `empty_scope` | unchanged |
| Swift (supplement) | [Alamofire/Alamofire](https://github.com/Alamofire/Alamofire) (`dea4bab9`) | MIT | complete; 48 findings | unchanged |

Totals across the **26 unique attempted repositories** are not summed: finding counts mix advisory, warning and blocker severities and scopes that changed (React discovery grew 16 → 376 files, so its finding count grew by design). The table's repository rows are attempts, not a top-three rank per language; several languages needed supplements because an early attempt had empty scope. `golang/go` remains incomplete, not clean, and OpenClaw has no paired baseline findings report.

## What the corpus exposed, and what changed

Every added finding was traced to its cause. Removed findings were checked against source only on a stratified spot-check (not exhaustively): all **six sampled removals** were justified, and **21/21 first-wave mechanism-positive locations are still emitted** in the final reports. That 21/21 check is retention of known examples, not 21 new source adjudications and not a recall measurement. Machine-readable first-wave labels are in [github-corpus-labels.json](github-corpus-labels.json); the 36-row source adjudication is in [github-corpus-adjudications.json](github-corpus-adjudications.json). All 83 added ECC findings, all 908 added React findings, and rea's single added finding (`AdbProvider.test.ts`) are in files that were previously unparsed or undiscovered; none appeared in previously analyzed files.

| Language | Defect found on public code | Evidence | Fix | Measured effect |
|---|---|---|---|---|
| JavaScript/TypeScript | `*-test.js/ts` under `__tests__` were never discovered | React: 360 of 376 conventional test files missed (95.7%; files, not cases) | path-aware discovery in full, explicit and `--diff` scopes; root `build-and-test.js` and `.vscode-test.mjs` stay excluded | React 16 → 376 files |
| JavaScript/TypeScript | regex literal inside a template interpolation treated as a string | rea 2/2 and ECC 9/9 parse failures contained `.replace(/'/g, …)`-style interpolations | linear, regex-aware interpolation lexing sharing the main lexer's regex-vs-division rule | all 11 files parse; scans complete |
| TypeScript | typed local assertion helper (`): Promise<void> =>`) not recognized | rea `Iso9660ArtifactReader.boundaries.test.ts` | optional return-type skipping | 26 `UNIV-ZERO` blockers removed in rea (report delta); the sampled typed-helper removal was source-confirmed, not all 26 |
| TypeScript | concise-arrow callbacks counted as direct mock calls | Vue computed/effect specs | span-aware attribution with type-annotation, IIFE and ASI controls | 6 Vue, 1 rea finding removed |
| Go | payload read via type assertion, boolean branch, range, return, operand not recognized; table `name` treated as cosmetic when passed to the SUT | Gin `fs_test.go`, Ollama, Go standard library | binding-identity use inversion (any non-existence-assertion use of the result counts), narrow blank-discard rule, name-as-input retention | 87 Ollama, 40 `golang/go`, 2 Gin findings removed; 0 added |
| Kotlin | multi-dollar string literals rejected; same-file expression-bodied helpers and generic `fail<T>()` (including `->`, `*`, `in`/`out`) not credited | JetBrains compiler parse failure; Magisk; OkHttp | threshold-aware literals; bounded type-argument scanner; expression-body helper analysis | compiler scan complete; 136 + 2 + 5 findings removed; 4,000-comparison workload stays at 16 ms |
| Swift | closure-form `#expect { … } throws:` not credited; skip rule rescanned the file suffix at every byte | apple/container; profile of the Swift compiler sample | closure-form macros; anchored linear skip scan | 54 container findings removed; 3-file profile 33.76 s → 0.043 s (96% of baseline CPU was this scan) |
| Dart | none changed | see limits | none | unchanged |

OpenClaw completed within 150 s with the final binary after exceeding 180 s with the baseline. The cause was not isolated; the repository contains Swift tests, which the baseline's quadratic skip scan would slow, but that attribution is an inference, not a measurement.

## Benchmark denominators

Report the three denominators separately; none is a single "accuracy" number.

- **Discovery / parse.** For each repository and language: independently enumerable eligible files, files discovered, successful parses, parse failures, unsupported constructs and empty scopes. React 16 → 376 discovered files and the resolution of the named parse failures support scanner improvements; they do not estimate finding precision or recall.
- **Precision.** Emitted findings labelled for **mechanism presence** and **actionability** separately, stratified by rule, severity, actual file language and repository. The 36-row ledger is a purposive sample, so its counts describe those rows only.
- **Recall.** Independently known defects, whether emitted or silent. Recall was **not measured**: there is no independent defect enumeration over the corpus.

## Adjudication ledger (36 rows)

[github-corpus-adjudications.json](github-corpus-adjudications.json) records six source-adjudicated rows per actual file language across 19 repositories, with pinned full commit SHA, primary and duplicate-partner locations, baseline/final presence, mechanism verdict, actionability verdict, detector decision and cited oracle evidence. It is agent adjudication of retained pinned source, not blind human ground truth; no upstream code was executed.

Of the 36 rows: 26 are present in both reports, four are final-only and six are baseline-only. The reported mechanism is present in 23 rows and absent in 13. The source review identified nine actionable findings, 20 false positives as defect claims, and seven debatable or unsupported cases. An informational disposition does not resolve an uncertain verdict. **Mechanism presence is not the same as an actionable defect**—fixtures, no-throw contracts, compile-time checks and intentional controls can carry the reported syntax while remaining legitimate. Do not compute population precision, recall or a confidence interval from these rows.

## Held-out and paired controls (proposed)

Before any blocking gate is considered:

1. Freeze rule versions and the gate criterion.
2. Draw a reproducible probability sample of **current blocker alerts on previously untouched repositories**, recording repository clustering and rule/language strata.
3. Have two independent reviewers label source without seeing prior verdicts; retain disagreements and human adjudication.
4. Separately seed paired defects and legitimate controls into eligible held-out test contexts, including silent files and helper, receiver and branch variants.
5. Report seeded detection over eligible seeds separately from natural-defect recall, with per-stratum denominators, unresolved labels and uncertainty.

A future opt-in gate should qualify **one rule, one language and one supported runner at a time**, not inherit credibility from an aggregate finding count. Until then, use PR annotations for developer review and a distinct scan-health status for incomplete analysis.

## Post-adjudication repair verification

Rescanned all 26 unique pinned snapshots against the preceding `f032cb95…` binary. Repair binary SHA-256: `97225c5ed67404c8f510fdf8f08c09693c367b2ab604714350e9cb72d432dd19`. The ledger preserves the historical reports it reviewed; this section records the later repair.

| Targeted false positive | Verified repair |
|---|---|
| Ollama `api/types_test.go:1047` — expected error called swallowed | Removed the warning for the adjacent fatal check against an independent table expectation. A self-derived expectation still emits the warning in a separate CLI control. |
| Ponytail `pi-extension/test/extension.test.js:88` — substring check called existence-only | Removed the advisory. Tests retain alerts for bare truthiness, short-circuit bypasses and runtime-empty substrings. Across affected repositories, Ponytail removed 10 advisories and ECC removed 422; these are report deltas, not individually adjudicated false positives. |
| Swift compiler `test/CAS/swift-scan-test.swift:58` — free fixture function treated as a test | Removed the blocker. The Swift compiler projection went from 199 to 169 declarations and 113 to 81 findings (33 removed, one added). The added `UNIV-ZERO` at `test/SILOptimizer/OSLogMandatoryOptTest.swift:568` is a FileCheck fixture method in a protocol extension, not a runtime test. Type-member XCTest recognition still relies on a name heuristic and does not establish runner membership. |

All three target findings disappeared; all nine source-adjudicated actionable findings and all 21 first-wave mechanism-positive locations remained. File discovery, parse counts and exit codes were unchanged. `go test ./...` passed (1,028 tests), as did `go vet ./...`, the formatting check and the CLI build. Independent source rereview accepted the three bounded repairs; this was not a whole-change review or a population precision/recall study.

## Remaining limits and adoption risk

- **Parse failures that are not defects.** All 7 `golang/go` failures are syntax newer than the native parser used by the build host's Go version accepts (bare composite-literal operands, generic methods). The scan remains `incomplete` (exit 2) and must be treated so.
- **Imported helpers and custom wrappers.** Dart/Kotlin/TypeScript helper credit is same-file. AppFlowy imported extension helpers, Flutter's `@isTest` `testUsingContext` wrapper, FreeCodeCamp's imported `alertToBeVisible`, and Kotlin inherited base-class `test`/`doTest` methods are not resolved. Do not hide them with name whitelists; they are documented unsupported context, not clean results.
- **Fixture and intent mismatch.** Flutter's `dev/automated_tests` contains deliberately empty or throwing subprocess fixtures; the Swift compiler's `lit`/`FileCheck` fixtures and free `test*` functions are not XCTest or Swift Testing declarations; Kotlin compiler test bases are inherited. Findings there describe local syntax, not test-quality defects. A static scanner cannot tell the runner contract.
- **No-throw, compile-only and process-exit oracles** are legitimate and look hollow (`fail<…>`-free no-throw smoke tests, existential-conformance checks, subprocess exit status). Blanket-whitelisting calls would create false negatives.
- **Swift scope.** Swift applicability is narrow: **four of seven attempted Swift repositories returned `empty_scope`** (awesome-mac, awesome-ios, iina, stats); three had analyzed scope. Run-time test counts, `.testTarget` membership and inherited XCTest bases are not modelled.
- **Recall is not measured.** Labels exist only for the stratified samples reviewed (73 first-wave rows plus the 36-row source adjudication and the second-wave per-language samples retained in the evidence archive). They are agent-adjudicated, non-random and must not be extrapolated to population precision or recall.
- **Independent review gap.** The final integration round passed orchestrator-run local gates (full `go test ./...`, `go vet`, `gofmt`, label retention, rescans of all pinned snapshots). Changed findings were classified by **report delta**, not by exhaustive source adjudication of every removed finding; the 36-row ledger adjudicates a purposive sample only. The independent review route was exhausted by a provider quota limit before a whole-change review could complete, and no substitute reviewer was approved. This document does **not** claim an independent whole-change code review; that review remains outstanding.
- **Genuine issues also exist**: empty template tests, a commented-out assertion, self-comparison assertions, fixed real-time waits and `test.skip` were confirmed in Bloc, Vue, FreeCodeCamp and OkHttp. These are **confirmed mechanisms**, which is not the same as actionable upstream defects; source adjudication in the 36-row ledger scores mechanism presence and actionability separately.

**Adoption decision:** observation-only reporting remains appropriate. These results do not qualify TestGuard as a universal blocking quality gate, and a lower finding count must not be read as stronger tests.

## Reproduce the scans

Build TestGuard from the intended revision and use fresh, isolated, pinned checkouts or commit archives; do not install dependencies or execute repository scripts. Example for Gin:

```sh
go build -o /tmp/testguard-corpus ./cmd/testguard
work=$(mktemp -d)
git -c core.hooksPath=/dev/null clone --no-checkout https://github.com/gin-gonic/gin.git "$work/gin"
git -C "$work/gin" -c core.hooksPath=/dev/null checkout --detach 0f09c3a9b4626d9fe9979cf4d0521d6ae32ab646
/tmp/testguard-corpus --all --json "$work/gin" > "$work/gin.json"
status=$?
jq --argjson status "$status" '.analysis, .diagnostics, (.exit_code == $status)' "$work/gin.json"
```

Preserve non-zero exits. Compare findings after normalizing checkout-relative paths, and compare discovery counts, not only finding counts: a scan that discovers more files can legitimately report more findings.
