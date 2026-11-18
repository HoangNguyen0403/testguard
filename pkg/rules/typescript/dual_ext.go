package typescript

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var testFileExtRegex = regexp.MustCompile(`^(?i)(?P<stem>.+)\.(?P<infix>test|spec)\.(?P<ext>js|jsx|ts|tsx)$`)

// DualExtRule detects redundant legacy .test.js(x) files coexisting with .test.ts(x) files
// for the same production stem in the same directory.
type DualExtRule struct {
	scannedFiles map[string]bool
}

// NewDualExtRule creates an initialized DualExtRule.
func NewDualExtRule() *DualExtRule {
	return &DualExtRule{
		scannedFiles: make(map[string]bool),
	}
}

// SetScannedFiles records the full set of files included in the current scan scope.
func (r *DualExtRule) SetScannedFiles(files []string) {
	r.scannedFiles = make(map[string]bool, len(files))
	for _, f := range files {
		r.scannedFiles[filepath.Clean(f)] = true
	}
}

// EvaluateFile checks for twin .test.js(x) and .test.ts(x) files.
func (r *DualExtRule) EvaluateFile(filePath string, src string, units []parser.TestUnit) []models.Violation {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)

	match := testFileExtRegex.FindStringSubmatch(base)
	if match == nil {
		return nil
	}

	stem := match[1]
	infix := match[2]
	ext := strings.ToLower(match[3])

	if ext == "js" || ext == "jsx" {
		tsSister := filepath.Join(dir, stem+"."+infix+".ts")
		tsxSister := filepath.Join(dir, stem+"."+infix+".tsx")
		if fileExists(tsSister) || fileExists(tsxSister) {
			return []models.Violation{
				{
					RuleID:     "GUARD-TS-DUAL-EXT",
					Severity:   models.SeverityWarning,
					FilePath:   filePath,
					LineNumber: 1,
					Column:     1,
					Message:    fmt.Sprintf("Twin test file detected: legacy JavaScript test '%s' coexists with TypeScript test file in same directory", base),
					Suggestion: "Consolidate tests into the TypeScript test file and remove the legacy JavaScript test file.",
				},
			}
		}
	} else if ext == "ts" || ext == "tsx" {
		jsSister := filepath.Join(dir, stem+"."+infix+".js")
		jsxSister := filepath.Join(dir, stem+"."+infix+".jsx")
		sisterJs := ""
		if fileExists(jsSister) {
			sisterJs = jsSister
		} else if fileExists(jsxSister) {
			sisterJs = jsxSister
		}

		if sisterJs != "" {
			cleanSister := filepath.Clean(sisterJs)
			jsInScope := false
			if r.scannedFiles != nil {
				if r.scannedFiles[cleanSister] || r.scannedFiles[sisterJs] {
					jsInScope = true
				} else {
					for f := range r.scannedFiles {
						if filepath.Clean(f) == cleanSister || (filepath.Base(f) == filepath.Base(cleanSister) && filepath.Dir(filepath.Clean(f)) == filepath.Dir(cleanSister)) {
							jsInScope = true
							break
						}
					}
				}
			}

			// In diff mode or single-file scan where only the TS file is scanned,
			// warn on the TS file to notify developer of the unremoved JS counterpart.
			if !jsInScope {
				return []models.Violation{
					{
						RuleID:     "GUARD-TS-DUAL-EXT",
						Severity:   models.SeverityWarning,
						FilePath:   filePath,
						LineNumber: 1,
						Column:     1,
						Message:    fmt.Sprintf("Twin test file detected: newly added or modified TypeScript test '%s' has legacy JavaScript counterpart '%s' on disk", base, filepath.Base(sisterJs)),
						Suggestion: "Consolidate tests into the TypeScript test file and remove the legacy JavaScript test file.",
					},
				}
			}
		}
	}

	return nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
