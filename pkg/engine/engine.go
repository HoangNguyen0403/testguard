package engine

import (
	"os"

	"github.com/HoangNguyen0403/testguard/pkg/detectors/clone"
	"github.com/HoangNguyen0403/testguard/pkg/detectors/zeroassert"
	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser/dart"
	dartrules "github.com/HoangNguyen0403/testguard/pkg/rules/dart"
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
	dartAdapter *dart.Adapter
	zeroAssert  *zeroassert.Detector
	cloneDetect *clone.Detector

	seedEchoRule          *dartrules.SeedEchoRule
	uncoveredMutationRule *dartrules.UncoveredMutationRule
}

// New creates an initialized testguard engine.
func New() *Engine {
	return &Engine{
		dartAdapter:           dart.NewAdapter(),
		zeroAssert:            zeroassert.New(),
		cloneDetect:           clone.New(),
		seedEchoRule:          dartrules.NewSeedEchoRule(),
		uncoveredMutationRule: dartrules.NewUncoveredMutationRule(),
	}
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

		units, err := e.dartAdapter.LocateTests(content, file)
		if err != nil {
			result.ParseErrors++
			continue
		}

		// 1. Universal Zero-Assertion check
		result.Violations = append(result.Violations, e.zeroAssert.Check(units)...)

		// 2. Universal Clone check
		result.Violations = append(result.Violations, e.cloneDetect.Check(units)...)

		// 3. Language Semantic rules (Dart)
		for _, u := range units {
			result.Violations = append(result.Violations, e.seedEchoRule.Evaluate(u)...)
			result.Violations = append(result.Violations, e.uncoveredMutationRule.Evaluate(u)...)
		}
	}

	return result, nil
}
