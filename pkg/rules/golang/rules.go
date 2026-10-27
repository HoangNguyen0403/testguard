package golang

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	goparser "go/parser"
	"go/token"
	"path/filepath"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	repoOrInfraDirSegments = []string{
		"/repository/",
		"/infrastructure/",
		"/database/",
		"/db/",
		"/dao/",
		"/store/",
	}
	repoOrInfraFileSuffixes = []string{
		"_repository_test.go",
		"_repo_test.go",
		"_dao_test.go",
		"_store_test.go",
	}
)

func isRepositoryOrInfraLayer(filePath string) bool {
	normalized := "/" + strings.ToLower(strings.TrimPrefix(filepath.ToSlash(filePath), "/"))
	for _, seg := range repoOrInfraDirSegments {
		if strings.Contains(normalized, seg) {
			return true
		}
	}
	for _, suf := range repoOrInfraFileSuffixes {
		if strings.HasSuffix(normalized, suf) {
			return true
		}
	}
	return false
}

// getUnitAST retrieves or parses the AST declaration and token.FileSet for a TestUnit.
func getUnitAST(unit parser.TestUnit) (*token.FileSet, *ast.FuncDecl, error) {
	if unit.ASTDecl != nil {
		if fn, ok := unit.ASTDecl.(*ast.FuncDecl); ok && fn != nil {
			if fset, ok := unit.FileSet.(*token.FileSet); ok && fset != nil {
				return fset, fn, nil
			}
		}
	}

	raw := strings.TrimSpace(unit.RawBody)
	if raw == "" {
		return nil, nil, fmt.Errorf("empty test unit raw body")
	}

	fset := token.NewFileSet()
	var src string

	if strings.HasPrefix(raw, "func ") {
		src = "package p\n" + raw
	} else if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		src = "package p\nfunc _testFunc() " + raw
	} else {
		src = "package p\nfunc _testFunc() {\n" + raw + "\n}"
	}

	fileAST, err := goparser.ParseFile(fset, unit.FilePath, src, 0)
	if err != nil {
		return nil, nil, err
	}

	for _, decl := range fileAST.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			return fset, fn, nil
		}
	}

	return nil, nil, fmt.Errorf("no func declaration found in test body")
}

// getNodeLineAndColumn computes the truthful 1-based source line and column for an AST node.
func getNodeLineAndColumn(fset *token.FileSet, node ast.Node, unit parser.TestUnit) (int, int) {
	if fset == nil || node == nil {
		line := unit.StartLine
		if line <= 0 {
			line = 1
		}
		return line, 1
	}

	pos := fset.Position(node.Pos())
	if unit.ASTDecl != nil {
		// When AST was parsed directly from the original file, use exact file position
		return pos.Line, pos.Column
	}

	// Fallback calculation when parsed from synthesized wrapper
	raw := strings.TrimSpace(unit.RawBody)
	headerLines := 2
	if strings.HasPrefix(raw, "func ") {
		headerLines = 1
	} else if strings.HasPrefix(raw, "{") && strings.HasSuffix(raw, "}") {
		headerLines = 1
	}
	relLine := pos.Line - headerLines - 1
	if relLine < 0 {
		relLine = 0
	}
	line := unit.StartLine + relLine
	if line <= 0 {
		line = 1
	}
	return line, pos.Column
}

// isLocalReceiver reports whether ident is lexically bound to a declaration in
// this file (parameter, local variable, range binding, or same-file top-level
// object) rather than an imported package qualifier. go/parser records those
// bindings in ast.Ident.Obj, so this is declaration identity at the selector
// occurrence, not a spelling search across the whole function: a later inner
// shadow never suppresses an earlier genuine package call.
func isLocalReceiver(ident *ast.Ident) bool {
	return ident.Obj != nil
}

// DomainPurityRule enforces that domain entity/model tests contain zero mocks.
// Note: Helper-mediated mock instantiation outside the test AST is outside static boundary.
type DomainPurityRule struct{}

func NewDomainPurityRule() *DomainPurityRule {
	return &DomainPurityRule{}
}

func (r *DomainPurityRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	slashPath := filepath.ToSlash(unit.FilePath)
	if !strings.Contains(slashPath, "/domain/") && !strings.Contains(slashPath, "domain_test.go") {
		return violations
	}

	fset, fn, err := getUnitAST(unit)
	if err != nil {
		return violations
	}

	seenLines := make(map[int]bool)

	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		identName := ident.Name
		methodName := sel.Sel.Name
		isViolation := false

		// A lexically bound identifier is not an imported package qualifier.
		if isLocalReceiver(ident) {
			return true
		}

		if len(unit.FileImports) > 0 {
			importPath, isImported := unit.FileImports[identName]
			if isImported {
				// gomock package call (canonical or aliased)
				if strings.Contains(importPath, "golang/mock/gomock") && methodName == "NewController" {
					isViolation = true
				}
				// testify/mock package call
				if strings.Contains(importPath, "testify/mock") &&
					(strings.HasPrefix(methodName, "New") || methodName == "MatchedBy" || methodName == "Anything") {
					isViolation = true
				}
				// Generated mocks package call
				if (strings.HasSuffix(importPath, "/mocks") || strings.Contains(importPath, "/mock/")) &&
					strings.HasPrefix(methodName, "NewMock") {
					isViolation = true
				}
			}
		} else {
			// Fallback heuristic for synthetic units without import carrier
			pkgName := strings.ToLower(identName)
			if pkgName == "gomock" && methodName == "NewController" {
				isViolation = true
			}
			if pkgName == "mocks" && strings.HasPrefix(methodName, "NewMock") {
				isViolation = true
			}
			if pkgName == "mock" && (strings.HasPrefix(methodName, "New") || methodName == "MatchedBy" || methodName == "Anything") {
				isViolation = true
			}
		}

		if isViolation {
			line, col := getNodeLineAndColumn(fset, call, unit)
			if !seenLines[line] {
				seenLines[line] = true
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-GO-01",
					Severity:   models.SeverityBlocker,
					FilePath:   unit.FilePath,
					LineNumber: line,
					Column:     col,
					Message:    fmt.Sprintf("Domain purity violation in '%s': domain tests must be pure and contain zero mocks.", unit.Name),
					Suggestion: "Remove mock controllers from domain tests. Test business logic using pure domain struct fixtures.",
				})
			}
		}
		return true
	})

	return violations
}

// PortMockingRule verifies that service/usecase tests mock interface ports, not concrete database drivers.
// Note: Helper-mediated database instantiation outside the test AST is outside static boundary.
type PortMockingRule struct{}

func NewPortMockingRule() *PortMockingRule {
	return &PortMockingRule{}
}

func (r *PortMockingRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	if isRepositoryOrInfraLayer(unit.FilePath) {
		return violations
	}

	fset, fn, err := getUnitAST(unit)
	if err != nil {
		return violations
	}

	seenLines := make(map[int]bool)

	ast.Inspect(fn, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}

		identName := ident.Name
		methodName := sel.Sel.Name
		isViolation := false

		// A lexically bound identifier is not an imported package qualifier.
		if isLocalReceiver(ident) {
			return true
		}

		if len(unit.FileImports) > 0 {
			importPath, isImported := unit.FileImports[identName]
			if isImported {
				if strings.Contains(importPath, "sqlmock") && strings.HasPrefix(methodName, "New") {
					isViolation = true
				}
				if strings.Contains(importPath, "pgxmock") && strings.HasPrefix(methodName, "New") {
					isViolation = true
				}
				if strings.Contains(importPath, "gorm") && methodName == "Open" {
					isViolation = true
				}
			}
		} else {
			// Fallback heuristic for synthetic units without import carrier
			pkgName := strings.ToLower(identName)
			switch pkgName {
			case "sqlmock":
				if strings.HasPrefix(methodName, "New") {
					isViolation = true
				}
			case "pgxmock":
				if strings.HasPrefix(methodName, "New") {
					isViolation = true
				}
			case "gorm":
				if methodName == "Open" {
					isViolation = true
				}
			}
		}

		if isViolation {
			line, col := getNodeLineAndColumn(fset, call, unit)
			if !seenLines[line] {
				seenLines[line] = true
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-GO-02",
					Severity:   models.SeverityWarning,
					FilePath:   unit.FilePath,
					LineNumber: line,
					Column:     col,
					Message:    fmt.Sprintf("Port mocking violation in '%s': mocking concrete database connection instead of exported interface Port.", unit.Name),
					Suggestion: "Define and mock an abstract repository interface (e.g. domain.UserRepository) rather than concrete database connections.",
				})
			}
		}
		return true
	})

	return violations
}

// TableDuplicateRule detects identical duplicate rows in Go table-driven tests.
type TableDuplicateRule struct{}

func NewTableDuplicateRule() *TableDuplicateRule {
	return &TableDuplicateRule{}
}

func isPrimitiveScalarType(name string) bool {
	switch name {
	case "string", "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"float32", "float64", "complex64", "complex128",
		"bool", "byte", "rune", "error", "any":
		return true
	}
	return false
}

func isStructRowLiteral(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return true
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			_, isLit := e.X.(*ast.CompositeLit)
			return isLit
		}
	}
	return false
}

// isStructRowElementType reports whether expr is a defensible struct row type
// for a table-driven test. Scalar, array, map, channel, function, and interface
// element types are not table rows. Named row types are accepted syntactically;
// without a type checker a named element may alias a non-struct, and that
// ambiguity is disclosed rather than resolved.
func isStructRowElementType(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.StructType:
		return true
	case *ast.Ident:
		return !isPrimitiveScalarType(e.Name)
	case *ast.SelectorExpr:
		return true
	case *ast.StarExpr:
		return isStructRowElementType(e.X)
	case *ast.IndexExpr:
		return true
	case *ast.IndexListExpr:
		return true
	}
	return false
}

// isTableComposite ensures compLit represents a defensible table-driven test row slice,
// rejecting primitive/scalar fixture arrays, nested array/map payloads, and
// arbitrary non-table struct containers.
func isTableComposite(compLit *ast.CompositeLit) bool {
	if len(compLit.Elts) < 2 {
		return false
	}

	arrType, isArray := compLit.Type.(*ast.ArrayType)
	if !isArray || arrType.Elt == nil {
		return false
	}
	if !isStructRowElementType(arrType.Elt) {
		return false
	}

	// Every element must be a struct row literal ({...} or &{...})
	for _, elt := range compLit.Elts {
		if !isStructRowLiteral(elt) {
			return false
		}
	}

	return true
}

func (r *TableDuplicateRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	fset, fn, err := getUnitAST(unit)
	if err != nil {
		return violations
	}

	ast.Inspect(fn, func(n ast.Node) bool {
		compLit, ok := n.(*ast.CompositeLit)
		if !ok || !isTableComposite(compLit) {
			return true
		}

		// Table boundaries are isolated per composite literal
		seenRows := make(map[string]bool)
		for _, elt := range compLit.Elts {
			var buf bytes.Buffer
			if err := format.Node(&buf, fset, elt); err != nil {
				continue
			}
			rawContent := strings.TrimSpace(buf.String())
			// Strip outer braces if present for clean reporting
			trimmedContent := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(rawContent, "{"), "}"))

			if seenRows[trimmedContent] {
				line, col := getNodeLineAndColumn(fset, elt, unit)
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-GO-03",
					Severity:   models.SeverityAdvisory,
					FilePath:   unit.FilePath,
					LineNumber: line,
					Column:     col,
					Message:    fmt.Sprintf("Duplicate table test row in '%s': row {%s} is duplicated.", unit.Name, trimmedContent),
					Suggestion: "Ensure each table test case tests a distinct equivalence class, boundary condition, or error scenario.",
				})
				// One duplicate row reported per table literal is sufficient
				break
			}
			seenRows[trimmedContent] = true
		}

		return true
	})

	return violations
}
