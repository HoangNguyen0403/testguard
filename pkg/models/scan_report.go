package models

// ExcludedPath represents a path excluded from scan analysis and the exclusion reason.
type ExcludedPath struct {
	Path   string `json:"path"`
	Reason string `json:"reason"` // "default_directory" | "generated" | "non_test"
}

// ToolMeta records metadata identifying TestGuard and its build revision.
type ToolMeta struct {
	Name    string  `json:"name"`
	Version string  `json:"version"`
	Commit  *string `json:"commit"`
	BuiltAt *string `json:"built_at"`
}

// ScanScope records caller requested paths and effective scope resolution.
type ScanScope struct {
	RequestedPaths []string       `json:"requested_paths"`
	EffectiveMode  string         `json:"effective_mode"` // "full"
	Files          []string       `json:"files"`
	Excluded       []ExcludedPath `json:"excluded"`
	Unsupported    []string       `json:"unsupported"`
	ConfigApplied  bool           `json:"config_applied"` // false
}

// AnalysisSummary records analysis status, reason, and test file counters.
type AnalysisSummary struct {
	Status          string `json:"status"` // "complete" | "incomplete" | "not_applicable" | "error"
	Reason          string `json:"reason"` // "analyzed" | "input_failure" | "empty_scope" | "empty_allowed" | "unsupported_scope" | "invalid_usage" | "output_failure"
	DiscoveredFiles int    `json:"discovered_files"`
	AnalyzedFiles   int    `json:"analyzed_files"`
	AnalyzedTests   int    `json:"analyzed_tests"`
	FailedFiles     int    `json:"failed_files"`
}

// Diagnostic represents a structured failure or warning diagnostic.
type Diagnostic struct {
	Code    string  `json:"code"`
	Path    *string `json:"path"`
	Message string  `json:"message"`
}

// ScanPolicy records blocker count and policy evaluation status.
type ScanPolicy struct {
	BlockerCount int    `json:"blocker_count"`
	Status       string `json:"status"` // "no_blocker" | "blocker_found"
}

// ScanReport is the exact v1 JSON envelope emitted by TestGuard.
type ScanReport struct {
	SchemaVersion int             `json:"schema_version"`
	Tool          ToolMeta        `json:"tool"`
	Scope         ScanScope       `json:"scope"`
	Analysis      AnalysisSummary `json:"analysis"`
	Findings      []Violation     `json:"findings"`
	Diagnostics   []Diagnostic    `json:"diagnostics"`
	Policy        ScanPolicy      `json:"policy"`
	ExitCode      int             `json:"exit_code"`
}

// NewToolMeta initializes tool metadata, mapping "none" or "unknown" to nil pointers.
func NewToolMeta(version, commit, builtAt string) ToolMeta {
	var commitPtr *string
	if commit != "" && commit != "none" {
		c := commit
		commitPtr = &c
	}
	var builtAtPtr *string
	if builtAt != "" && builtAt != "unknown" {
		b := builtAt
		builtAtPtr = &b
	}
	return ToolMeta{
		Name:    "testguard",
		Version: version,
		Commit:  commitPtr,
		BuiltAt: builtAtPtr,
	}
}

// NewDiagnostic creates a diagnostic with an optional path pointer.
func NewDiagnostic(code, path, message string) Diagnostic {
	var pathPtr *string
	if path != "" {
		p := path
		pathPtr = &p
	}
	return Diagnostic{
		Code:    code,
		Path:    pathPtr,
		Message: message,
	}
}

// NewScanReport creates an initialized ScanReport with non-nil slices.
func NewScanReport() *ScanReport {
	return &ScanReport{
		SchemaVersion: 1,
		Tool: ToolMeta{
			Name: "testguard",
		},
		Scope: ScanScope{
			RequestedPaths: make([]string, 0),
			EffectiveMode:  "full",
			Files:          make([]string, 0),
			Excluded:       make([]ExcludedPath, 0),
			Unsupported:    make([]string, 0),
			ConfigApplied:  false,
		},
		Analysis: AnalysisSummary{
			Status: "complete",
			Reason: "analyzed",
		},
		Findings:    make([]Violation, 0),
		Diagnostics: make([]Diagnostic, 0),
		Policy: ScanPolicy{
			BlockerCount: 0,
			Status:       "no_blocker",
		},
		ExitCode: 0,
	}
}
