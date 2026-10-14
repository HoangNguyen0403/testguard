package engine

import (
	"os"
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

// Options configures the engine audit run.
type Options struct {
	TargetPaths []string
	DiffOnly    bool
	ScanAll     bool
	BaseBranch  string
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

// parseGenericTestUnits extracts simple test units for non-Dart languages.
func parseGenericTestUnits(content []byte, filePath string) []parser.TestUnit {
	var units []parser.TestUnit
	src := string(content)
	lines := strings.Split(src, "\n")

	// Group into a generic unit for the file if individual tests aren't split by dedicated AST
	units = append(units, parser.TestUnit{
		Name:       filePath,
		FilePath:   filePath,
		StartLine:  1,
		Kind:       "unit",
		RawBody:    src,
		Assertions: []string{"assert"},
	})

	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "func Test") || strings.HasPrefix(trimmed, "it(") || strings.HasPrefix(trimmed, "test(") || strings.HasPrefix(trimmed, "@Test") {
			units = append(units, parser.TestUnit{
				Name:       trimmed,
				FilePath:   filePath,
				StartLine:  i + 1,
				Kind:       "unit",
				RawBody:    src,
				Assertions: []string{"assert"},
			})
		}
	}

	return units
}

// Run executes the audit across target paths according to options.
func (e *Engine) Run(opts Options) (*Result, error) {
	result := &Result{}

	paths := opts.TargetPaths
	if len(paths) == 0 {
		paths = []string{"."}
	}

	var allTestFiles []string
	for _, p := range paths {
		files, err := FindTestFiles(p, e.dartAdapter)
		if err == nil {
			allTestFiles = append(allTestFiles, files...)
		}
	}

	result.ScannedFiles = len(allTestFiles)

	for _, file := range allTestFiles {
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
