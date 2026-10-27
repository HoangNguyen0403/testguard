package parser

// AssertionKind represents whether a call in a test is an assertion.
type AssertionKind int

const (
	AssertionKindNone AssertionKind = iota
	AssertionKindOutcome
	AssertionKindVerification
)

// ModuleRoot represents a discovered project module in the workspace.
type ModuleRoot struct {
	Path     string `json:"path"`
	Language string `json:"language"`
	HasTests bool   `json:"has_tests"`
}

// TestUnit represents an individual test declaration found in a source file.
type TestUnit struct {
	Name       string   `json:"name"`
	FilePath   string   `json:"file_path"`
	StartLine  int      `json:"start_line"`
	EndLine    int      `json:"end_line"`
	Kind       string   `json:"kind"` // "unit", "bloc", "widget", "table"
	RawBody    string   `json:"raw_body"`
	Assertions []string `json:"assertions"`
	SeedBody   string   `json:"seed_body,omitempty"`
	ActBody    string   `json:"act_body,omitempty"`
	ExpectBody string   `json:"expect_body,omitempty"`

	// ASTDecl holds an optional language-specific AST declaration (e.g. *ast.FuncDecl for Go).
	// Excluded from JSON serialization to maintain cross-language model contracts.
	ASTDecl any `json:"-"`
	// FileSet holds an optional token.FileSet for AST source position resolution.
	FileSet any `json:"-"`
	// FileImports holds an optional map of local package names/aliases to canonical import paths.
	FileImports map[string]string `json:"-"`
}

// LanguageAdapter decouples the core engine from language-specific AST structures.
type LanguageAdapter interface {
	ID() string
	FileExtensions() []string
	LocateTests(source []byte, filePath string) ([]TestUnit, error)
	IsGenerated(filePath string, header []byte) bool
}
