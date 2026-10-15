package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"github.com/HoangNguyen0403/testguard/pkg/detectors/clone"
	"github.com/HoangNguyen0403/testguard/pkg/detectors/naming"
	"github.com/HoangNguyen0403/testguard/pkg/detectors/zeroassert"
	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/parser/dart"
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
	BaseBranch  string
	ProgressFn  ProgressFunc
}

// Result contains the aggregated scan statistics and violations.
type Result struct {
	ScannedFiles int
	ParseErrors  int
	Skipped      int
	Violations   []models.Violation
}

// Engine coordinates language adapters, universal detectors, and semantic rules.
type Engine struct {
	dartAdapter  *dart.Adapter
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
	goTestRegex           = regexp.MustCompile(`(?m)^[ \t]*func\s+(Test\w+)\s*\(`)
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
	case strings.HasSuffix(filePath, ".go"):
		targetRegex = goTestRegex
	case strings.HasSuffix(filePath, ".ts") || strings.HasSuffix(filePath, ".tsx") || strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".jsx"):
		targetRegex = tsTestRegex
	case strings.HasSuffix(filePath, ".kt"):
		targetRegex = ktTestRegex
	case strings.HasSuffix(filePath, ".swift"):
		targetRegex = swiftTestRegex
	default:
		targetRegex = goTestRegex
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
	result := &Result{
		Violations: make([]models.Violation, 0),
	}

	paths := opts.TargetPaths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("target path not found: %s", p)
		}
	}

	if opts.ProgressFn != nil {
		opts.ProgressFn("discovering", 0, 0, "")
	}

	seenFiles := make(map[string]bool)
	var allTestFiles []string
	for _, p := range paths {
		files, err := FindTestFiles(p, e.dartAdapter)
		if err == nil {
			for _, f := range files {
				clean := filepath.Clean(f)
				if !seenFiles[clean] {
					seenFiles[clean] = true
					allTestFiles = append(allTestFiles, f)
				}
			}
		}
	}

	result.ScannedFiles = len(allTestFiles)

	if opts.ProgressFn != nil {
		opts.ProgressFn("discovered", len(allTestFiles), len(allTestFiles), "")
	}

	for i, file := range allTestFiles {
		if opts.ProgressFn != nil {
			opts.ProgressFn("scanning", i+1, len(allTestFiles), file)
		}
		content, err := os.ReadFile(file)
		if err != nil {
			result.ParseErrors++
			continue
		}

		suppressions := models.ParseSuppressions(string(content))

		var units []parser.TestUnit
		if strings.HasSuffix(file, ".dart") {
			u, err := e.dartAdapter.LocateTests(content, file)
			if err != nil {
				result.ParseErrors++
				continue
			}
			units = u
		} else {
			units = parseGenericTestUnits(content, file)
		}

		var fileViolations []models.Violation

		// 1. Universal checks
		fileViolations = append(fileViolations, e.zeroAssert.Check(units)...)
		fileViolations = append(fileViolations, e.cloneDetect.Check(units)...)
		fileViolations = append(fileViolations, e.namingDetect.Check(units)...)

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
				result.Violations = append(result.Violations, v)
			}
		}
	}

	return result, nil
}
