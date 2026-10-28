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

func normalizeRowContent(fset *token.FileSet, elt ast.Expr, compLit *ast.CompositeLit) (string, string) {
	var rowLit *ast.CompositeLit
	switch e := elt.(type) {
	case *ast.CompositeLit:
		rowLit = e
	case *ast.UnaryExpr:
		if e.Op == token.AND {
			if cl, ok := e.X.(*ast.CompositeLit); ok {
				rowLit = cl
			}
		}
	}
	if rowLit == nil {
		var buf bytes.Buffer
		_ = format.Node(&buf, fset, elt)
		raw := strings.TrimSpace(buf.String())
		return raw, raw
	}

	var fullBuf bytes.Buffer
	_ = format.Node(&fullBuf, fset, elt)
	fullContent := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(fullBuf.String()), "{"), "}"))

	nameFieldIndex := -1
	if compLit != nil {
		if arrType, ok := compLit.Type.(*ast.ArrayType); ok {
			var st *ast.StructType
			switch eltType := arrType.Elt.(type) {
			case *ast.StructType:
				st = eltType
			case *ast.Ident:
				if eltType.Obj != nil && eltType.Obj.Kind == ast.Typ {
					if ts, ok := eltType.Obj.Decl.(*ast.TypeSpec); ok {
						if s, ok := ts.Type.(*ast.StructType); ok {
							st = s
						}
					}
				}
			}
			if st != nil && st.Fields != nil {
				fieldIdx := 0
				for _, field := range st.Fields.List {
					if len(field.Names) == 0 {
						fieldIdx++
					} else {
						for _, name := range field.Names {
							low := strings.ToLower(name.Name)
							if low == "name" || low == "desc" || low == "description" || low == "title" {
								nameFieldIndex = fieldIdx
							}
							fieldIdx++
						}
					}
				}
			}
		}
	}

	var parts []string
	for idx, f := range rowLit.Elts {
		if kv, ok := f.(*ast.KeyValueExpr); ok {
			if kIdent, ok := kv.Key.(*ast.Ident); ok {
				low := strings.ToLower(kIdent.Name)
				if low == "name" || low == "desc" || low == "description" || low == "title" {
					continue
				}
			}
			var buf bytes.Buffer
			_ = format.Node(&buf, fset, kv)
			parts = append(parts, buf.String())
		} else {
			if nameFieldIndex != -1 && idx == nameFieldIndex {
				continue
			}
			var buf bytes.Buffer
			_ = format.Node(&buf, fset, f)
			parts = append(parts, buf.String())
		}
	}
	return strings.Join(parts, ", "), fullContent
}

func (r *TableDuplicateRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	fset, fn, err := getUnitAST(unit)
	if err != nil || fn.Body == nil {
		return violations
	}

	var testingParamObj *ast.Object
	if fn.Type != nil && fn.Type.Params != nil && len(fn.Type.Params.List) > 0 {
		for _, f := range fn.Type.Params.List {
			for _, name := range f.Names {
				if name.Obj != nil {
					testingParamObj = name.Obj
					break
				}
			}
			if testingParamObj != nil {
				break
			}
		}
	}
	var suiteRecvObj *ast.Object
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		for _, name := range fn.Recv.List[0].Names {
			if name.Obj != nil {
				suiteRecvObj = name.Obj
				break
			}
		}
	}

	assignedTables := make(map[*ast.Object]*ast.CompositeLit)
	for _, stmt := range fn.Body.List {
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			for i, lhs := range s.Lhs {
				if ident, ok := lhs.(*ast.Ident); ok && ident.Obj != nil && i < len(s.Rhs) {
					if compLit, ok := s.Rhs[i].(*ast.CompositeLit); ok && isTableComposite(compLit) {
						assignedTables[ident.Obj] = compLit
					}
				}
			}
		case *ast.DeclStmt:
			if gen, ok := s.Decl.(*ast.GenDecl); ok && gen.Tok == token.VAR {
				for _, spec := range gen.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for i, name := range vs.Names {
							if name.Obj != nil && i < len(vs.Values) {
								if compLit, ok := vs.Values[i].(*ast.CompositeLit); ok && isTableComposite(compLit) {
									assignedTables[name.Obj] = compLit
								}
							}
						}
					}
				}
			}
		}
	}

	if len(assignedTables) == 0 {
		return violations
	}

	validTables := make(map[*ast.CompositeLit]bool)
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		rng, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}
		ident, ok := rng.X.(*ast.Ident)
		if !ok || ident.Obj == nil {
			return true
		}
		compLit, exists := assignedTables[ident.Obj]
		if !exists {
			return true
		}

		// R3: If the range body references the row's name field in a condition (if tt.name == ... / switch tt.name),
		// the rows are conditionally branched by name and not duplicates by construction — skip that table.
		var rowIdentObj *ast.Object
		if valIdent, ok := rng.Value.(*ast.Ident); ok && valIdent.Obj != nil {
			rowIdentObj = valIdent.Obj
		}
		if rowIdentObj != nil {
			hasConditionOnName := false
			ast.Inspect(rng.Body, func(bn ast.Node) bool {
				switch stmt := bn.(type) {
				case *ast.IfStmt:
					ast.Inspect(stmt.Cond, func(cn ast.Node) bool {
						if sel, ok := cn.(*ast.SelectorExpr); ok {
							if id, ok := sel.X.(*ast.Ident); ok && id.Obj == rowIdentObj {
								low := strings.ToLower(sel.Sel.Name)
								if low == "name" || low == "desc" || low == "description" || low == "title" {
									hasConditionOnName = true
									return false
								}
							}
						}
						return true
					})
				case *ast.SwitchStmt:
					if stmt.Tag != nil {
						ast.Inspect(stmt.Tag, func(cn ast.Node) bool {
							if sel, ok := cn.(*ast.SelectorExpr); ok {
								if id, ok := sel.X.(*ast.Ident); ok && id.Obj == rowIdentObj {
									low := strings.ToLower(sel.Sel.Name)
									if low == "name" || low == "desc" || low == "description" || low == "title" {
										hasConditionOnName = true
										return false
									}
								}
							}
							return true
						})
					}
				}
				return true
			})
			if hasConditionOnName {
				return true
			}
		}

		hasRunCall := false
		ast.Inspect(rng.Body, func(bn ast.Node) bool {
			call, ok := bn.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Run" {
				return true
			}
			// Verify Run is called on recognized testing parameter or suite receiver
			if id, ok := sel.X.(*ast.Ident); ok {
				if (testingParamObj != nil && id.Obj == testingParamObj) || (suiteRecvObj != nil && id.Obj == suiteRecvObj) {
					hasRunCall = true
					return false
				}
			} else if innerCall, ok := sel.X.(*ast.CallExpr); ok {
				if innerSel, ok := innerCall.Fun.(*ast.SelectorExpr); ok && innerSel.Sel.Name == "T" {
					if id, ok := innerSel.X.(*ast.Ident); ok && suiteRecvObj != nil && id.Obj == suiteRecvObj {
						hasRunCall = true
						return false
					}
				}
			}
			return true
		})
		if hasRunCall {
			validTables[compLit] = true
		}
		return true
	})
	if len(validTables) == 0 {
		return violations
	}

	for compLit := range validTables {
		seenRows := make(map[string]bool)
		for _, elt := range compLit.Elts {
			normalizedKey, fullContent := normalizeRowContent(fset, elt, compLit)
			if normalizedKey == "" {
				continue
			}

			if seenRows[normalizedKey] {
				line, col := getNodeLineAndColumn(fset, elt, unit)
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-GO-03",
					Severity:   models.SeverityAdvisory,
					FilePath:   unit.FilePath,
					LineNumber: line,
					Column:     col,
					Message:    fmt.Sprintf("Duplicate table test row in '%s': row {%s} is duplicated.", unit.Name, fullContent),
					Suggestion: "Ensure each table test case tests a distinct equivalence class, boundary condition, or error scenario.",
				})
				break
			}
			seenRows[normalizedKey] = true
		}
	}

	return violations
}
