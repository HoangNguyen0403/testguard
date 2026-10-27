package engine

import (
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/detectors/clone"
	"github.com/HoangNguyen0403/testguard/pkg/detectors/naming"
	"github.com/HoangNguyen0403/testguard/pkg/detectors/zeroassert"
	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/parser/dart"
	goparseradapter "github.com/HoangNguyen0403/testguard/pkg/parser/golang"
	dartrules "github.com/HoangNguyen0403/testguard/pkg/rules/dart"
	gorules "github.com/HoangNguyen0403/testguard/pkg/rules/golang"
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
	dartAdapter  *dart.Adapter
	goAdapter    *goparseradapter.Adapter
	zeroAssert   *zeroassert.Detector
	cloneDetect  *clone.Detector
	namingDetect *naming.Detector

	// Dart / Flutter rules
	seedEchoRule          *dartrules.SeedEchoRule
	uncoveredMutationRule *dartrules.UncoveredMutationRule
	disguisedMirrorRule   *dartrules.DisguisedMirrorRule
	mockEchoRule          *dartrules.MockEchoRule
	blocDuplicateRule     *dartrules.BlocDuplicateRule

	// Go rules
	domainPurityRule   *gorules.DomainPurityRule
	portMockingRule    *gorules.PortMockingRule
	tableDuplicateRule *gorules.TableDuplicateRule

	// TypeScript / Fullstack rules
	nestControllerEchoRule *tsrules.NestControllerEchoRule
	nestUnmockedDbRule     *tsrules.NestUnmockedDbModuleRule
	reactSleepRule         *tsrules.ReactSleepRule

	// Native Mobile rules
	kotlinCoroutineRule *mobilerules.KotlinCoroutineAdvanceRule
	swiftTimeoutRule    *mobilerules.SwiftTimeoutRule
}

// New creates an initialized testguard engine with all polyglot rules.
func New() *Engine {
	return &Engine{
		dartAdapter:  dart.NewAdapter(),
		goAdapter:    goparseradapter.NewAdapter(),
		zeroAssert:   zeroassert.New(),
		cloneDetect:  clone.New(),
		namingDetect: naming.New(),

		seedEchoRule:          dartrules.NewSeedEchoRule(),
		uncoveredMutationRule: dartrules.NewUncoveredMutationRule(),
		disguisedMirrorRule:   dartrules.NewDisguisedMirrorRule(),
		mockEchoRule:          dartrules.NewMockEchoRule(),
		blocDuplicateRule:     dartrules.NewBlocDuplicateRule(),

		domainPurityRule:   gorules.NewDomainPurityRule(),
		portMockingRule:    gorules.NewPortMockingRule(),
		tableDuplicateRule: gorules.NewTableDuplicateRule(),

		nestControllerEchoRule: tsrules.NewNestControllerEchoRule(),
		nestUnmockedDbRule:     tsrules.NewNestUnmockedDbModuleRule(),
		reactSleepRule:         tsrules.NewReactSleepRule(),

		kotlinCoroutineRule: mobilerules.NewKotlinCoroutineAdvanceRule(),
		swiftTimeoutRule:    mobilerules.NewSwiftTimeoutRule(),
	}
}

var (
	tsTestRegex           = regexp.MustCompile(`(?m)^[ \t]*(?:it|test)\s*\(\s*(?:"((?:\\.|[^"])+)"|'((?:\\.|[^'])+)')`)
	ktTestRegex           = regexp.MustCompile(`(?m)^[ \t]*@Test\s+(?:fun\s+)?(\w+)`)
	swiftTestRegex        = regexp.MustCompile(`(?m)^[ \t]*func\s+(test\w+)\s*\(`)
	genericAssertionRegex = regexp.MustCompile(`(?i)(?:assert\.|require\.|t\.(?:Error|Fatal|Fail)|expect\(|verify\(|verifyNever\(|XCTAssert)`)
)

// parseGenericTestUnits extracts test units and function bodies for non-Dart languages.
func parseGenericTestUnits(content []byte, filePath string) []parser.TestUnit {
	var units []parser.TestUnit
	src := string(content)

	lineOffsets := []int{0}
	for i := range len(src) {
		if src[i] == '\n' {
			lineOffsets = append(lineOffsets, i+1)
		}
	}
	offsetToLine := func(byteOffset int) int {
		low, high := 0, len(lineOffsets)-1
		line := 1
		for low <= high {
			mid := (low + high) / 2
			if lineOffsets[mid] <= byteOffset {
				line = mid + 1
				low = mid + 1
			} else {
				high = mid - 1
			}
		}
		return line
	}
	var targetRegex *regexp.Regexp
	switch {
	case strings.HasSuffix(filePath, ".ts") || strings.HasSuffix(filePath, ".tsx") || strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".jsx"):
		targetRegex = tsTestRegex
	case strings.HasSuffix(filePath, ".kt"):
		targetRegex = ktTestRegex
	case strings.HasSuffix(filePath, ".swift"):
		targetRegex = swiftTestRegex
	default:
		targetRegex = tsTestRegex
	}

	matches := targetRegex.FindAllStringSubmatchIndex(src, -1)
	for _, m := range matches {
		var testName string
		for group := 2; group < len(m); group += 2 {
			if m[group] != -1 {
				testName = src[m[group]:m[group+1]]
				break
			}
		}
		if testName == "" {
			testName = src[m[0]:m[1]]
		}
		startOffset := m[0]
		startLine := offsetToLine(startOffset)

		openBrace := strings.Index(src[startOffset:], "{")
		var body string
		if openBrace != -1 {
			openIdx := startOffset + openBrace
			closeIdx := parser.FindMatchingBrace(src, openIdx)
			if closeIdx != -1 {
				body = src[startOffset : closeIdx+1]
			}
		}
		if body == "" {
			lineEnd := strings.Index(src[startOffset:], "\n")
			if lineEnd != -1 {
				body = src[startOffset : startOffset+lineEnd]
			} else {
				body = src[startOffset:]
			}
		}

		var assertions []string
		if genericAssertionRegex.MatchString(body) {
			assertions = []string{"assert"}
		}

		units = append(units, parser.TestUnit{
			Name:       testName,
			FilePath:   filePath,
			StartLine:  startLine,
			Kind:       "unit",
			RawBody:    body,
			Assertions: assertions,
		})
	}

	if len(units) == 0 {
		var assertions []string
		if genericAssertionRegex.MatchString(src) {
			assertions = []string{"assert"}
		}
		units = append(units, parser.TestUnit{
			Name:       filePath,
			FilePath:   filePath,
			StartLine:  1,
			Kind:       "file",
			RawBody:    src,
			Assertions: assertions,
		})
	}

	return units
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

	discovery := DiscoverScope(paths, e.dartAdapter, opts.FailTraversal)
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

		var units []parser.TestUnit
		if strings.HasSuffix(file, ".go") {
			u, err := e.goAdapter.LocateTests(content, file)
			if err != nil {
				failedFilesMap[canonicalAbsPath(file)] = true
				hasInputFailure = true
				report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("parse_failed", normalizePath(file), err.Error()))
				continue
			}
			units = u
		} else if strings.HasSuffix(file, ".dart") {
			u, err := e.dartAdapter.LocateTests(content, file)
			if err != nil {
				failedFilesMap[canonicalAbsPath(file)] = true
				hasInputFailure = true
				report.Diagnostics = append(report.Diagnostics, models.NewDiagnostic("parse_failed", normalizePath(file), err.Error()))
				continue
			}
			units = u
		} else {
			units = parseGenericTestUnits(content, file)
		}

		analyzedFiles = append(analyzedFiles, normalizePath(file))

		for _, u := range units {
			if u.Kind != "file" {
				totalAnalyzedTests++
			}
		}

		suppressions := models.ParseSuppressions(string(content))
		var fileViolations []models.Violation

		evalUnits := units
		if len(evalUnits) == 0 {
			evalUnits = []parser.TestUnit{{
				Name:      file,
				FilePath:  file,
				StartLine: 1,
				Kind:      "file",
				RawBody:   string(content),
			}}
		}

		// 1. Universal checks
		fileViolations = append(fileViolations, e.zeroAssert.Check(units)...)
		fileViolations = append(fileViolations, e.cloneDetect.Check(evalUnits)...)
		fileViolations = append(fileViolations, e.namingDetect.Check(evalUnits)...)
		// 2. Language & Framework rules
		if strings.HasSuffix(file, ".dart") {
			fileViolations = append(fileViolations, e.blocDuplicateRule.Check(units)...)
			for _, u := range units {
				fileViolations = append(fileViolations, e.seedEchoRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.uncoveredMutationRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.disguisedMirrorRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.mockEchoRule.Evaluate(u)...)
			}
		} else if strings.HasSuffix(file, ".go") {
			for _, u := range units {
				fileViolations = append(fileViolations, e.domainPurityRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.portMockingRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.tableDuplicateRule.Evaluate(u)...)
			}
		} else if strings.HasSuffix(file, ".ts") || strings.HasSuffix(file, ".tsx") || strings.HasSuffix(file, ".js") || strings.HasSuffix(file, ".jsx") {
			for _, u := range units {
				fileViolations = append(fileViolations, e.nestControllerEchoRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.nestUnmockedDbRule.Evaluate(u)...)
				fileViolations = append(fileViolations, e.reactSleepRule.Evaluate(u)...)
			}
		} else if strings.HasSuffix(file, ".kt") {
			for _, u := range units {
				fileViolations = append(fileViolations, e.kotlinCoroutineRule.Evaluate(u)...)
			}
		} else if strings.HasSuffix(file, ".swift") {
			for _, u := range units {
				fileViolations = append(fileViolations, e.swiftTimeoutRule.Evaluate(u)...)
			}
		}

		// 3. Filter suppressed violations
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
