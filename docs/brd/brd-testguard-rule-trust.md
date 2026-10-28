# Brief: TestGuard Rule Trust Calibration

lane: direction; snc_tier: high; approval: approved(user, 2026-10-28)
Profile: technical
SNC: S=2 (detectors across Go/Dart/TS + discovery + reporter) N=1 (calibrate existing rules, add few) C=2 (CI Blocker gate = trust boundary for merges) total=5 tier=high

> Data-handling note: the evidence comes from a private client monorepo. This brief keeps only aggregate counts and code shapes; it holds no client paths or source. Per-finding evidence (paths, line refs, snippets) stays local in `/tmp/tg-audit/verdict-{A..F}.md` and must not be committed to this public repo.

## Contract

**Outcome.** A TestGuard Blocker means "this test really gives false confidence" in ≥80% of sampled cases, per rule and per language. Rules that can't meet that are demoted or disabled until fixed.

**Constraints.**
- Static analysis only. No compiler or type checker (existing architecture, `pkg/parser/*`).
- No new runtime dependencies.
- In-code suppression (`testguard:ignore RULE reason:`) and `testguard.yaml` `exclude` stay backward compatible.
- Precision claims must be measured against a labeled corpus, not asserted.

**Non-Goals.**
- Fixing the client's tests.
- Mutation testing or coverage measurement.
- A cross-file call graph beyond one hop inside the same file.
- Rewriting the Dart BLoC rules with full handler control-flow analysis (only a future option, card C).

**Acceptance Criteria.**
- AC1. Discovery skips linked git worktrees (dir whose `.git` is a file containing `gitdir: …/worktrees/…`) and the default vendored dirs (`third_party/`, `vendor/`). Verified on a fixture tree.
- AC2. Go `GUARD-UNIV-ZERO` counts assertions inside `t.Run(name, func(t *testing.T){…})` closures, does not flag `suite.Run(t, …)` entry points, and follows one hop into same-file helpers that take `t`. The probe `TestDoubleTable` (below) yields 0 findings.
- AC3. TS/Dart `GUARD-UNIV-ZERO` resolves one hop into same-file helpers whose bodies contain an assertion.
- AC4. Delete outright, with their code, catalog docs and tests: `GUARD-DART-UNCOVERED-MUTATION` (keyword + literal `isFetching` heuristic, 0/32), `GUARD-DART-DISGUISED-MIRROR` (contradicts `.having()`-on-mutated-fields practice, 0/2) and `GUARD-GO-02` (sqlmock is the correct boundary at the adapter layer, 1/26). Fix in place instead of deleting: `GUARD-DART-BLOC-DUPLICATE` (compare the whole `blocTest` argument body: build/setUp/seed/act/expect, exact tokens) and `GUARD-GO-03` (HEAD already requires struct-row tables; re-measure).
- AC5. `GUARD-REACT-02` fires only for real-timer `setTimeout` with a delay > 0 in the test body itself. It does not fire inside `mockImplementation`/`mockReturnValue`/mock class bodies, in files using `useFakeTimers`, or for `setTimeout(r, 0)` flushes. Severity becomes Warning.
- AC6. `GUARD-UNIV-01` "Duplicate test body" is emitted only for exact-token matches. Normalized-only matches stay labelled "Similar test structure" at Advisory (already true at HEAD `4dad017`; guard it with a regression test).
- AC7. A labeled precision corpus (synthetic fixtures reproducing each FP/TP class below) runs in `go test ./...` and reports per-rule precision.
- AC8. Remove `testguard init` and its starter `testguard.yaml`, including docs. Nothing reads the file, and loading YAML would need a new dependency. Scope control stays with CLI paths plus built-in default excludes (AC1).
- AC9. E2E rule pack. Files are classified as E2E when they import `@playwright/test`, sit under an `integration_test/`, `e2e/` or `playwright/` directory, or (Dart) reference `IntegrationTestWidgetsFlutterBinding`. Unit-only timing rules (`GUARD-REACT-02`) do not apply to E2E files. New `GUARD-E2E-01` (Warning) flags hard waits in E2E tests: Playwright `waitForTimeout(…)` and real-timer `setTimeout` sleeps with delay > 0; Flutter `Future.delayed(…)`, `sleep(…)`. Zero-assertion and clone rules still apply.
- AC10. New hygiene rules: `GUARD-UNIV-TAUTOLOGY` (Blocker) for literal-vs-same-literal assertions (`expect(true).toBe(true)`, `expect(1).toEqual(1)`, `expect(true, isTrue)`, `expect(true, true)`, `assert.True(t, true)`, `assert.Equal(t, 1, 1)`). It is emitted once per test. `GUARD-UNIV-SKIP` (Warning) for skipped tests (`it.skip`, `test.skip`, `describe.skip`, `xit`, `xdescribe`, Dart `skip: true`/`skip: '…'`) and for a Go `t.Skip` that is the unconditional first statement.

## Why

**Problem.** Our client ran installed `testguard v0.1.1-4-g7222622-dirty` (built 2026-10-15) on the client's monorepo: 830 Blockers / 167 Warnings / 2679 Advisories. The team doubted it. Sampling 223 findings against the real code shows the doubt is justified for most rules. A gate this noisy trains teams to ignore or blanket-suppress it, and that also buries the real hollow tests it does find.

**AS-IS → TO-BE.**
- Today: about 21% of Blockers are real `[INFERENCE: extrapolated from samples]`. One Blocker rule is 0% precise. HEAD fixes some false alerts but adds a Go regression.
- Target: per-rule precision is measured, Blockers are ≥80% precise, and the scan scope is canonical (no duplicate worktrees or vendored code).

**SMART metric.** In the next minor release:
- Blocker precision ≥80% per rule×language on the labeled corpus.
- Total Blocker count on the same client snapshot drops ≥60% while all 8 sampled true-positive classes stay detected (recall guard).

Sponsor / validation owner: repo owner (HoangNguyen0403) `assumed`.

## Findings: Rule Scorecard

Precision is measured on the old report (v0.1.1 binary), with worktree copies excluded before sampling. "HEAD" = this repo at `4dad017`, re-run on the same snapshot.

| Rule | Sev | Findings (old → HEAD) | Sample TP / FP / Debatable | Precision | Mechanism (confirmed in code) | Verdict |
|---|---|---|---|---|---|---|
| GUARD-UNIV-ZERO (TS/TSX/Dart) | Blocker | 274 → 274 | 8 / 4 / 1 (n=13) | ~62% | `genericAssertionRegex` / Dart `assertionRegex` only look inside the test body | **Effective, keep Blocker.** TPs are real hollow tests: empty `// Assert`, a snapshot test with no snapshot, guarded `if (btn) click()` with no expect. All FPs are same-file helpers (`expectCartToBeEmpty()`), so AC3 fixes them |
| GUARD-UNIV-ZERO (Go) | Blocker | 222 → 413 | old 0 / 11; HEAD-only 0 / 29 | ~0% | Old: regex saw `func Test` inside `/* */`. HEAD: `findExecutableAssertions` matches only `receiver.Obj == testingParam`, so a `t` shadowed in a `t.Run` closure is never seen; `suite.Run` runners are flagged; methods with receivers are skipped entirely | **Broken in both versions.** 153/413 HEAD findings are `*Suite` runners; 848 testify suite methods are never analyzed (blind spot). Fix per AC2 |
| GUARD-DART-UNCOVERED-MUTATION | Blocker | 192 → 192 | 0 / 32 / 0 | 0% | `uncovered_mutation.go:28-34`: act text contains `fetch\|search\|load\|submit` (substring, so `download`/`upload` match too) AND expect lacks the literal `isFetching`. It never reads the bloc | **False alert. Disable now.** The rule description claims handler analysis it doesn't do. It flags sync events, early-return guards, other field names (`isSearching`, `isLoadMoreFetching`) and full-state `copyWith` equality |
| GUARD-REACT-02 | Blocker | 142 → 142 | 5 / 8 / 15 | ~18% | `sleepRegex` = any `setTimeout(` in the body | **Over-broad.** 78/129 real `setTimeout` delays in the frontend tests are `0`. FPs: mock latency, `useFakeTimers`. TPs: real 500–1100 ms sleeps. Fix per AC5 |
| GUARD-UNIV-01 | Advisory | 1857 → 2096 | 3 / 21 / 0 | ~13% | Old binary labelled normalized (literal-collapsed) matches as "identical execution tokens". HEAD labels 1813/2096 "Similar" and only 283 "Duplicate" | **Message was false in v0.1.1; fixed at HEAD.** Keep exact-only as "Duplicate" |
| GUARD-UNIV-02 | Advisory | 683 → 774 | 8 / 8 / 8 | ~33% (67% incl. debatable) | ≥4 same normalized hash in a file | **Useful but misrouted.** 5/8 FPs are ≥4 *exact* copy-paste stubs (title says "passes prop X", body checks a logo). These are the most valuable findings, yet they get "parameterize" advice. Route exact groups ≥4 to UNIV-01 / a new hollow-clone rule; scope to the same `describe`/`group`; skip bodies already in loops |
| GUARD-GO-02 | Warning | 161 → 337 | 1 / 24 / 1 | ~4% | Any `sqlmock.New*` / `gorm.Open` call | **False alert.** 16/26 have the SUT *as* the repository/adapter (sqlmock is the correct boundary); 8/26 are in commented-out code (old binary). HEAD reports per call line (2× inflation). Restrict to non-adapter packages or drop |
| GUARD-GO-03 | Advisory | 112 → 27 | 0 / 28 / 0 (old) | 0% old; HEAD unverified | Old: any repeated composite literal (mock DB rows `{"id":1}`). HEAD: requires a `[]T{struct rows}` table | **Old false alert, largely fixed at HEAD.** Precision of the remaining 27 is `unknown` |
| GUARD-DART-BLOC-DUPLICATE | Advisory | 27 → 27 | 0 / 8 / 1 | 0% | Compares act+expect only; ignores `seed`/`setUp`; normalizes act args | **False alert.** Include seed/setUp/build in the key |
| GUARD-DART-DISGUISED-MIRROR | Warning | 6 → 6 | 0 / 2 / 0 | 0% | Counts `.having(` across the whole expect list, not per matcher | **False alert.** It also penalizes the client's own mobile standard (`isA<State>().having()` on mutated fields, rule MOB-TEST-02) |

**Scope inflation (rule-independent).**
- 983 findings (26.7%) come from two linked git worktrees of the mobile app. That's a 3× count: 154 of 830 Blockers.
- +35 from a fork of the PMP frontend (27/39 spec files byte-identical).
- +32 from vendored Microsoft `azure-devops-mcp` tests.
- 76 from Playwright E2E specs scored with unit-test rules.

**Blocker trust estimate `[INFERENCE]`.** Of 830 Blockers: 154 are worktree duplicates, about 220 are Go zero-assert at ~0% precision, 64 are uncovered-mutation at 0%, 142 are REACT-02 at ~18%, and about 248 are TS/Dart zero-assert at ~62%. That's ≈180 real (~22%). The sample size per stratum is 13–32, so treat this as ±10 pp.

## Gaps: Real Smells the Tool Misses

Counts are from scout greps on the client repo. Severities are my proposal.

| Candidate rule | Evidence | Proposed sev |
|---|---|---|
| Tautological assertion (`expect(true).toBe(true)`, `assert.True(t, true)`) | 11 in the frontend, usually as "skip" placeholders. They evade UNIV-ZERO because `expect(` matches | Blocker |
| Skipped test without reason/ticket (`it.skip`, `describe.skip`, `t.Skip`) | 16 in the frontend, incl. `/* removed */` bodies | Warning |
| Conditional assertion / guarded action (`if (el) { click }`, `try { expect } catch { expect }`) | Seen in UNIV-ZERO and UNIV-02 TPs; 111 try/catch blocks in frontend tests (unverified how many are vacuous) | Warning |
| Hollow clone: ≥2 exact bodies with different titles | UNIV-02 FP class above (14 copies in one file) | Warning |
| testify suite methods not analyzed | 848 `func (s *X) TestY()` methods are invisible to every Go rule | Coverage fix, not a rule |

## Options And Recommendation

- **A. Calibrate in place (recommended).** Implement AC1–AC8:
  - scope hygiene;
  - Go zero-assert fix (t.Run closure object tracking, `suite.Run`, receiver suite methods, one-hop helpers);
  - TS/Dart one-hop helpers;
  - REACT-02 narrowing;
  - default-disable the 0%-precision rules;
  - labeled corpus as the regression gate.
  
  It's the smallest change that makes Blockers trustworthy and is reversible per rule.
- **B. CI-side triage (stopgap, ship first; no TestGuard change).** `testguard.yaml` cannot do this: `init` writes the file, but nothing in `pkg/` or `cmd/` reads it (no YAML decoding anywhere), so its `exclude` block is inert. Instead:
  - pass explicit canonical module paths to `testguard` (omit `datasource/worktrees/`, the fork, `third_party/`, `integration_test/`);
  - run with `--json` and fail CI only on TS/Dart `GUARD-UNIV-ZERO` findings, by post-filtering on `rule_id`.
  
  It leaves the Go regression and the false rule descriptions in place.
- **C. Rewrite the Dart BLoC rules with lib/ handler resolution.** Rejected for now: high cost, and 0/32 shows the premise is wrong for full-state-equality tests.
- Rejected: **keep as-is and suppress inline**. ~600 suppressions would erode the signal permanently.

## Evidence

| Claim | Status |
|---|---|
| Report produced by pre-`4dad017` binary (`v0.1.1-4-g7222622-dirty`) | confirmed(`testguard --version`; `git log 7222622..HEAD`) |
| UNCOVERED-MUTATION is keyword + literal `isFetching` only | confirmed(`pkg/rules/dart/uncovered_mutation.go:27-35`) |
| REACT-02 matches any `setTimeout(` | confirmed(`pkg/rules/typescript/rules.go:15,83`) |
| HEAD Go zero-assert misses t.Run closures | confirmed(repro: `/tmp/tg-audit/probe/probe_test.go`; HEAD 1 Blocker, v0.1.1 0) and `pkg/parser/golang/adapter.go:358` |
| HEAD skips methods with receivers (suite methods unanalyzed) | confirmed(`pkg/parser/golang/adapter.go:212`) |
| Old binary scanned `/* */`-commented Go tests | confirmed(old report 84 findings in one commented-out file; HEAD 0) |
| UNIV-01 old label "identical" vs HEAD exact/similar split 283/1813 | confirmed(`pkg/detectors/clone/hasher.go:396-401`; HEAD re-run) |
| GO-03 HEAD rejects map-element tables | confirmed(`pkg/rules/golang/rules.go:367-383`) |
| Per-rule TP/FP counts | confirmed(local `/tmp/tg-audit/verdict-A..F.md`; spot-checked 1 Dart claim by hand) |
| Worktree copies are linked git worktrees | confirmed(scouts D and E: `.git` gitdir pointers) |
| Blocker trust ≈22% | assumed (stratified extrapolation) |
| GO-03 HEAD 27 precision | unknown |
| `testguard.yaml` (incl. `exclude`) is read by the scanner | confirmed false (`cmd/testguard/main.go:342-393` writes it; no reader or YAML decoding exists in `pkg/` or `cmd/`) |

### Repro: Go t.Run regression (HEAD)
```go
func TestDoubleTable(t *testing.T) {
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := double(tc.in); got != tc.want { t.Errorf("got %d", got) }
		})
	}
}
// HEAD 0.1.4: [BLOCKER] GUARD-UNIV-ZERO — v0.1.1: PASSED
```

## Standard And Deep Sections

**Stakeholders.**
- TestGuard maintainer (owner).
- Client test owners (consumers of the gate).
- CI owners (block/allow policy).

**Cost-benefit.**
- Option A is about 6 slices (discovery, Go adapter, TS/Dart helper resolution, REACT-02, rule defaults, precision corpus).
- Benefit: the Blocker signal becomes defensible, which saves the review time now spent dismissing ~650 false Blockers per full scan.

**Delivery context.**
- Go 1.x repo; detectors in `pkg/detectors`, `pkg/rules/*`; parsers in `pkg/parser/*`; discovery in `pkg/engine/discovery.go`.

**Glossary.**
- *Precision*: TP / (TP + FP) on sampled findings; debatable counted as not-TP.
- *One-hop helper*: a function declared in the same file and called from the test body.
- *Linked worktree*: a directory whose `.git` is a file pointing at `<repo>/.git/worktrees/<name>`.

**PM handoff checklist.**
- [ ] Approve option A (+B as stopgap).
- [ ] Confirm the labeled corpus must be synthetic (no client code).
- [ ] Decide default-off vs Advisory for the 0%-precision rules.
- [ ] Confirm the release version for the severity changes (changelog: behavior change).

## Approval

approval: approved(user, 2026-10-28). Decisions:
- Delete the always-wrong rules (AC4), keep the fixable ones.
- Include the CI stopgap (option B).
- Give E2E specs (Playwright + Flutter `integration_test`) their own rule pack (AC9).
- After implementation, re-scan the same client snapshot and re-sample precision.

## Outcome Report
{schema_version: 1, run_id: "brainstorm-testguard-rule-trust-2026-10-28", slug: "testguard-rule-trust", workflow: brainstorm-feature, feature_status: implemented_verified, started_at: "2026-10-28T04:00:00Z", completed_at: "2026-10-28T04:40:00Z", requirement_trace: {brd_objectives: ["OBJ-1 Blocker precision >=80% per rule x language", "OBJ-2 canonical scan scope"], requirements: ["REQ-SCOPE-01", "REQ-GO-ZERO-01", "REQ-HELPER-01", "REQ-DART-DEFAULTS-01", "REQ-REACT02-01", "REQ-CLONE-LABEL-01", "REQ-CORPUS-01", "REQ-CONFIG-01"], acceptance_criteria: ["AC1", "AC2", "AC3", "AC4", "AC5", "AC6", "AC7", "AC8"], srs: []}, completed_evidence: ["rule source review", "HEAD re-scan of same snapshot", "223 findings hand-classified across 6 scouts", "t.Run regression repro", "testguard.yaml has no reader"], missing_evidence: ["GO-03 HEAD precision", "precision of new candidate rules"], decision_needed: ["approve option A + B stopgap", "default-off vs Advisory for 0%-precision rules", "load testguard.yaml vs remove init"], recommended_next_workflow: plan-feature, cost: {source: unavailable}, agent: {identity: "omp orchestrator", model: "anthropic/claude-opus-5-5"}}

## Verification Results (same client snapshot, commit 7ed0d4f)

Re-scan totals:
- Blockers: 830 (v0.1.1 report) → 193.
- Warnings: 167 → 63.
- Advisories: 2679 → 1862.
- Scope: 6433 files / 24 modules → 3879 files / 18 modules. Linked worktrees and `third_party/` are now skipped.

Hand-labelled real findings, re-checked automatically against the new report:

| Rule | Labelled true positives still reported | Labelled false positives still reported |
|---|---|---|
| GUARD-UNIV-ZERO | 22/22 | 0/93 |
| GUARD-GO-03 | 8/8 | 0/58 |
| GUARD-DART-BLOC-DUPLICATE | 3/3 | 0/9 |
| GUARD-UNIV-TAUTOLOGY (new) | 11/11 | 0/0 |
| GUARD-UNIV-SKIP (new) | 8/8 | 0/0 |
| GUARD-E2E-01 (new) | 4/4 | 0/0 |
| GUARD-REACT-02 (now Warning) | 19/22 | 4/11 |
| GUARD-UNIV-02 | 18/19 | 2/9 |
| GUARD-UNIV-01 | 7/7 | 19/22, of which 18 now carry the honest "Similar test structure" wording |
| Deleted rules (UNCOVERED-MUTATION, DISGUISED-MIRROR, GO-02) | n/a | 0/58 |

Spot precision on fresh samples of the post-change report:
- UNIV-ZERO TS/TSX: 15/15 TP.
- UNIV-TAUTOLOGY: 11/11.
- REACT-02: 17 TP / 3 FP / 4 debatable.
- E2E-01: 4 TP / 2 debatable.
- UNIV-SKIP: 8 TP / 13 debatable (reason or ticket comment present).
- UNIV-02: 11 TP / 1 FP / 7 debatable.

Known limitations, accepted:
- Split-file testify suites are not analyzed.
- Cross-file helpers are not resolved, except gomock pairing and `t.Run` delegation.
- REACT-02 is silenced for a whole file when the file enables fake timers anywhere (it lost one real 1.1 s sleep).
- Hard waits inside Flutter integration "robot" helper files are not seen, because only test bodies are checked.

## Open Questions
None blocking.

## Next Workflow
plan-feature
