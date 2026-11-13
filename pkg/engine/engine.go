package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/detectors/clone"
	"github.com/HoangNguyen0403/testguard/pkg/detectors/naming"
	"github.com/HoangNguyen0403/testguard/pkg/detectors/zeroassert"
	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/parser/dart"
	goparseradapter "github.com/HoangNguyen0403/testguard/pkg/parser/golang"
	"github.com/HoangNguyen0403/testguard/pkg/parser/kotlin"
	"github.com/HoangNguyen0403/testguard/pkg/parser/swift"
	"github.com/HoangNguyen0403/testguard/pkg/parser/typescript"
	dartrules "github.com/HoangNguyen0403/testguard/pkg/rules/dart"
	"github.com/HoangNguyen0403/testguard/pkg/rules/e2e"
	gorules "github.com/HoangNguyen0403/testguard/pkg/rules/golang"
	"github.com/HoangNguyen0403/testguard/pkg/rules/hygiene"
	mobilerules "github.com/HoangNguyen0403/testguard/pkg/rules/mobile"
	tsrules "github.com/HoangNguyen0403/testguard/pkg/rules/typescript"
)

// ProgressFunc receives phase lifecycle events, item counters, and target file paths.
type ProgressFunc func(phase string, current, total int, currentFile string)

// Options configures the engine audit run.
type Options struct {
	TargetPaths []string
	DiffOnly    bool
	ScanAll     bool
	AllowEmpty  bool
	BaseBranch  string
	ProgressFn  ProgressFunc
	Version     string
	Commit      string
	Date        string

	// Fault injection hooks for testing OS permission / traversal / read errors
	FailTraversal func(path string) error
	FailRead      func(path string) error
}

// Result contains aggregated scan statistics, violations, and the formal ScanReport envelope.
type Result struct {
	ScannedFiles int
	ParseErrors  int
	Skipped      int
	Violations   []models.Violation
	Report       *models.ScanReport
}

// Engine coordinates language adapters, universal detectors, and semantic rules.
type Engine struct {
	adapters []parser.LanguageAdapter

	zeroAssert   *zeroassert.Detector
	cloneDetect  *clone.Detector
	namingDetect *naming.Detector

	// Universal hygiene rules
	tautologyRule *hygiene.TautologyRule
	skipRule      *hygiene.SkipRule
	focusRule     *hygiene.FocusRule

	// E2E rules
	hardWaitRule *e2e.HardWaitRule

	// Dart / Flutter rules
	seedEchoRule      *dartrules.SeedEchoRule
	mockEchoRule      *dartrules.MockEchoRule
	blocDuplicateRule *dartrules.BlocDuplicateRule
	vacuousStateRule  *dartrules.VacuousStateRule

	// Go rules
	domainPurityRule   *gorules.DomainPurityRule
	tableDuplicateRule *gorules.TableDuplicateRule
	swallowedErrorRule *gorules.SwallowedErrorRule
	uncheckedWantRule  *gorules.UncheckedWantRule

	// TypeScript / Fullstack rules
	nestControllerEchoRule *tsrules.NestControllerEchoRule
	nestUnmockedDbRule     *tsrules.NestUnmockedDbModuleRule
	snapshotOnlyRule       *tsrules.SnapshotOnlyRule
	floatingAssertRule     *tsrules.FloatingAssertRule
	catchOnlyRule          *tsrules.CatchOnlyRule
	conditionalAssertRule  *tsrules.ConditionalAssertRule
	existenceOnlyRule      *tsrules.ExistenceOnlyRule
	selfMockRule           *tsrules.SelfMockRule
	mockedSutRule          *tsrules.MockedSutRule
	reactSleepRule         *tsrules.ReactSleepRule

	// Native Mobile rules
	kotlinCoroutineRule *mobilerules.KotlinCoroutineAdvanceRule
	kotlinMockEchoRule  *mobilerules.KotlinMockEchoRule
	swiftTimeoutRule    *mobilerules.SwiftTimeoutRule
}

// New creates an initialized testguard engine with all polyglot rules.
func New() *Engine {
	return &Engine{
		adapters: []parser.LanguageAdapter{
			goparseradapter.NewAdapter(),
			dart.NewAdapter(),
			typescript.NewAdapter(),
			kotlin.NewAdapter(),
			swift.NewAdapter(),
		},

		zeroAssert:   zeroassert.New(),
		cloneDetect:  clone.New(),
		namingDetect: naming.New(),

		tautologyRule: hygiene.NewTautologyRule(),
		skipRule:      hygiene.NewSkipRule(),
		focusRule:     hygiene.NewFocusRule(),

		hardWaitRule: e2e.NewHardWaitRule(),

		seedEchoRule:      dartrules.NewSeedEchoRule(),
		mockEchoRule:      dartrules.NewMockEchoRule(),
		blocDuplicateRule: dartrules.NewBlocDuplicateRule(),
		vacuousStateRule:  dartrules.NewVacuousStateRule(),

		domainPurityRule:   gorules.NewDomainPurityRule(),
		tableDuplicateRule: gorules.NewTableDuplicateRule(),
		swallowedErrorRule: gorules.NewSwallowedErrorRule(),
		uncheckedWantRule:  gorules.NewUncheckedWantRule(),

		nestControllerEchoRule: tsrules.NewNestControllerEchoRule(),
		nestUnmockedDbRule:     tsrules.NewNestUnmockedDbModuleRule(),
		snapshotOnlyRule:       tsrules.NewSnapshotOnlyRule(),
		floatingAssertRule:     tsrules.NewFloatingAssertRule(),
		catchOnlyRule:          tsrules.NewCatchOnlyRule(),
		conditionalAssertRule:  tsrules.NewConditionalAssertRule(),
		existenceOnlyRule:      tsrules.NewExistenceOnlyRule(),
		selfMockRule:           tsrules.NewSelfMockRule(),
		mockedSutRule:          tsrules.NewMockedSutRule(),
		reactSleepRule:         tsrules.NewReactSleepRule(),

		kotlinCoroutineRule: mobilerules.NewKotlinCoroutineAdvanceRule(),
		kotlinMockEchoRule:  mobilerules.NewKotlinMockEchoRule(),
		swiftTimeoutRule:    mobilerules.NewSwiftTimeoutRule(),
	}
}

// adapterFor returns the language adapter registered for a file path, matching
// case-insensitively on the extension it declares.
func (e *Engine) adapterFor(filePath string) (parser.LanguageAdapter, bool) {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, adapter := range e.adapters {
		for _, declared := range adapter.FileExtensions() {
			if ext == declared {
				return adapter, true
			}
		}
	}
	return nil, false
}

// Run executes the audit across target paths according to options.
func (e *Engine) Run(opts Options) (*Result, error) {
	report := models.NewScanReport()
	report.Tool = models.NewToolMeta(opts.Version, opts.Commit, opts.Date)

	paths := opts.TargetPaths
	if len(paths) == 0 {
		paths = []string{"."}
	}
	report.Scope.RequestedPaths = append([]string(nil), paths...)

	if opts.DiffOnly {
		report.Analysis.Status = "error"
		report.Analysis.Reason = "invalid_usage"
		report.ExitCode = 2
		report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("invalid_usage", "", "--diff mode is not supported; use full scan"))
		return &Result{
			Report:     report,
			Violations: report.Findings,
		}, nil
	}

	if opts.ProgressFn != nil {
		opts.ProgressFn("discovering", 0, 0, "")
	}

	discovery := DiscoverScope(paths, e.adapters, opts.FailTraversal)
	report.Scope.Excluded = discovery.Excluded
	report.Scope.Unsupported = discovery.Unsupported
	report.Diagnostics = append(report.Diagnostics, discovery.Diagnostics...)
	discoveredFiles := discovery.DiscoveredFiles
	hasInputFailure := discovery.InputFailure

	allTestFiles := discovery.CandidateFiles

	if opts.ProgressFn != nil {
		opts.ProgressFn("discovered", len(allTestFiles), len(allTestFiles), "")
	}

	analyzedFiles := make([]string, 0)
	failedFilesMap := make(map[string]bool)
	var totalAnalyzedTests int

	for i, file := range allTestFiles {
		if opts.ProgressFn != nil {
			opts.ProgressFn("scanning", i+1, len(allTestFiles), file)
		}

		if opts.FailRead != nil {
			if ferr := opts.FailRead(file); ferr != nil {
				failedFilesMap[canonicalAbsPath(file)] = true
				hasInputFailure = true
				report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("read_failed", normalizePath(file), ferr.Error()))
				continue
			}
		}

		content, err := os.ReadFile(file)
		if err != nil {
			failedFilesMap[canonicalAbsPath(file)] = true
			hasInputFailure = true
			report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("read_failed", normalizePath(file), err.Error()))
			continue
		}

		adapter, found := e.adapterFor(file)
		if !found {
			failedFilesMap[canonicalAbsPath(file)] = true
			hasInputFailure = true
			report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("parse_failed", normalizePath(file), fmt.Sprintf("no language adapter handles extension %q", filepath.Ext(file))))
			continue
		}

		units, err := adapter.LocateTests(content, file)
		if err != nil {
			failedFilesMap[canonicalAbsPath(file)] = true
			hasInputFailure = true
			report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("parse_failed", normalizePath(file), err.Error()))
			continue
		}

		analyzedFiles = append(analyzedFiles, normalizePath(file))

		for _, u := range units {
			if u.Kind != "file" {
				totalAnalyzedTests++
			}
		}

		src := string(content)
		suppressions := models.ParseSuppressions(src)
		var fileViolations []models.Violation

		evalUnits := units
		if len(evalUnits) == 0 {
			evalUnits = []parser.TestUnit{{
				Name:      file,
				FilePath:  file,
				StartLine: 1,
				Kind:      "file",
				RawBody:   src,
			}}
		}

		// Hygiene rules apply to every supported language.
		for _, u := range units {
			if u.Kind == "unit" || u.Kind == "bloc" {
				fileViolations = append(fileViolations, e.tautologyRule.Evaluate(u)...)
			}
		}
		fileViolations = append(fileViolations, e.skipRule.CheckFile(file, src)...)
		fileViolations = append(fileViolations, e.focusRule.CheckFile(file, src)...)

		// Universal detectors.
		fileViolations = append(fileViolations, e.zeroAssert.Check(units)...)
		fileViolations = append(fileViolations, e.cloneDetect.Check(evalUnits)...)
		fileViolations = append(fileViolations, e.namingDetect.Check(evalUnits)...)

		isE2E := parser.IsE2EFile(file, content)

		switch adapter.ID() {
		case "dart":
			fileViolations = append(fileViolations, e.blocDuplicateRule.Check(units)...)
			for _, u := range units {
				fileViolations = append(fileViolations, e.seedEchoRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.mockEchoRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.vacuousStateRule.Evaluate(u)...)
			}
			if isE2E {
				fileViolations = append(fileViolations, e.hardWaitRule.EvaluateFile(units, file, src)...)
			}
		case "golang":
			for _, u := range units {
				fileViolations = append(fileViolations, e.domainPurityRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.tableDuplicateRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.swallowedErrorRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.uncheckedWantRule.Evaluate(u)...)
			}
		case "typescript":
			fileViolations = append(fileViolations, e.nestUnmockedDbRule.EvaluateFile(file, src, units)...)
			fileViolations = append(fileViolations, e.mockedSutRule.EvaluateFile(file, src, units)...)
			for _, u := range units {
				fileViolations = append(fileViolations, e.nestControllerEchoRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.floatingAssertRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.catchOnlyRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.conditionalAssertRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.existenceOnlyRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.selfMockRule.Evaluate(u)...)
				if !isE2E {
					fileViolations = append(fileViolations, e.snapshotOnlyRule.Evaluate(u)...)
				}
			}
			if isE2E {
				fileViolations = append(fileViolations, e.hardWaitRule.EvaluateFile(units, file, src)...)
			} else {
				fileViolations = append(fileViolations, e.reactSleepRule.EvaluateFile(units, src)...)
			}
		case "kotlin":
			for _, u := range units {
				fileViolations = append(fileViolations, e.kotlinCoroutineRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.kotlinMockEchoRule.Evaluate(u)...)
			}
		case "swift":
			for _, u := range units {
				fileViolations = append(fileViolations, e.swiftTimeoutRule.Evaluate(u)...)
			}
		}

		// Filter suppressed violations.
		for _, v := range fileViolations {
			if !models.IsSuppressed(suppressions, v.RuleID, v.LineNumber) {
				v.FilePath = normalizePath(v.FilePath)
				report.Findings = append(report.Findings, v)
			}
		}
	}

	sort.Strings(analyzedFiles)
	report.Scope.Files = analyzedFiles
	report.Analysis.DiscoveredFiles = discoveredFiles
	report.Analysis.AnalyzedFiles = len(analyzedFiles)
	report.Analysis.AnalyzedTests = totalAnalyzedTests
	report.Analysis.FailedFiles = len(failedFilesMap)

	blockerCount := 0
	for _, f := range report.Findings {
		if f.Severity == models.SeverityBlocker {
			blockerCount++
		}
	}
	report.Policy.BlockerCount = blockerCount
	if blockerCount > 0 {
		report.Policy.Status = "blocker_found"
	} else {
		report.Policy.Status = "no_blocker"
	}

	// Decision & Exit Precedence (SRS §3.2)
	if hasInputFailure || report.Analysis.FailedFiles > 0 {
		report.Analysis.Status = "incomplete"
		report.Analysis.Reason = "input_failure"
		report.ExitCode = 2
	} else if len(report.Scope.Unsupported) > 0 {
		report.Analysis.Status = "incomplete"
		report.Analysis.Reason = "unsupported_scope"
		report.ExitCode = 2
	} else if totalAnalyzedTests == 0 {
		if len(report.Findings) > 0 {
			report.Analysis.Status = "incomplete"
			report.Analysis.Reason = "empty_scope"
			report.ExitCode = 2
			report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("empty_scope", "", "no analyzable test declarations found"))
		} else if opts.AllowEmpty {
			report.Analysis.Status = "not_applicable"
			report.Analysis.Reason = "empty_allowed"
			report.ExitCode = 3
		} else {
			report.Analysis.Status = "incomplete"
			report.Analysis.Reason = "empty_scope"
			report.ExitCode = 2
			report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("empty_scope", "", "no analyzable test declarations found"))
		}
	} else {
		report.Analysis.Status = "complete"
		report.Analysis.Reason = "analyzed"
		if blockerCount > 0 {
			report.ExitCode = 1
		} else {
			report.ExitCode = 0
		}
	}

	result := &Result{
		ScannedFiles: report.Analysis.AnalyzedFiles,
		ParseErrors:  report.Analysis.FailedFiles,
		Skipped:      len(report.Scope.Excluded),
		Violations:   report.Findings,
		Report:       report,
	}
	return result, nil
}
