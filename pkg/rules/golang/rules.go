package golang

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	goparser "go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

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

	if strings.HasPrefix(raw, "func ") || strings.HasPrefix(raw, "type ") || strings.HasPrefix(raw, "var ") || strings.HasPrefix(raw, "const ") {
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

// DomainPurityRule flags mock usage in domain entity/model tests as an advisory convention note.
// Note: Helper-mediated mock instantiation outside the test AST is outside static boundary.
type DomainPurityRule struct{}

// NewDomainPurityRule creates an initialized DomainPurityRule.
func NewDomainPurityRule() *DomainPurityRule {
	return &DomainPurityRule{}
}

// Evaluate inspects a domain test unit for mock controller instantiation and reports advisory convention findings.
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
		}

		if isViolation {
			line, col := getNodeLineAndColumn(fset, call, unit)
			if !seenLines[line] {
				seenLines[line] = true
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-GO-01",
					Severity:   models.SeverityAdvisory,
					FilePath:   unit.FilePath,
					LineNumber: line,
					Column:     col,
					Message:    fmt.Sprintf("Domain test '%s' uses mocks; consider pure fixtures if the team keeps domain tests mock-free.", unit.Name),
					Suggestion: "Consider replacing mocks with pure domain fixtures if the team follows a mock-free domain test convention.",
				})
			}
		}
		return true
	})

	return violations
}

// SwallowedErrorRule flags if err != nil { return } statements in test bodies or
// t.Run closures where errors are silently swallowed with bare returns.
type SwallowedErrorRule struct{}

// NewSwallowedErrorRule creates an initialized SwallowedErrorRule.
func NewSwallowedErrorRule() *SwallowedErrorRule {
	return &SwallowedErrorRule{}
}

func isNilIdent(expr ast.Expr) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name == "nil" && ident.Obj == nil
	}
	return false
}

func inspectBlockForSwallowedErrors(
	fset *token.FileSet,
	body *ast.BlockStmt,
	unit parser.TestUnit,
	testingParamObjs map[*ast.Object]bool,
	suiteRecvObj *ast.Object,
) []models.Violation {
	var violations []models.Violation
	if body == nil {
		return violations
	}

	callAssignedVars := make(map[string]bool)
	tRunFuncLits := make(map[*ast.FuncLit]bool)

	// Identify t.Run closures within this body
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Run" || len(call.Args) < 2 {
			return true
		}
		fnLit, ok := call.Args[1].(*ast.FuncLit)
		if !ok {
			return true
		}
		isTestingRunner := false
		if id, ok := sel.X.(*ast.Ident); ok {
			if testingParamObjs[id.Obj] || (suiteRecvObj != nil && id.Obj == suiteRecvObj) {
				isTestingRunner = true
			}
		} else if innerCall, ok := sel.X.(*ast.CallExpr); ok {
			if innerSel, ok := innerCall.Fun.(*ast.SelectorExpr); ok && innerSel.Sel.Name == "T" {
				if id, ok := innerSel.X.(*ast.Ident); ok && suiteRecvObj != nil && id.Obj == suiteRecvObj {
					isTestingRunner = true
				}
			}
		}
		if isTestingRunner {
			tRunFuncLits[fnLit] = true
		}
		return true
	})

	seenLines := make(map[int]bool)

	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			// Do not descend into non-t.Run closures, and let t.Run closures be evaluated independently
			return false
		}

		switch stmt := n.(type) {
		case *ast.AssignStmt:
			hasCall := false
			for _, rhs := range stmt.Rhs {
				if _, ok := rhs.(*ast.CallExpr); ok {
					hasCall = true
					break
				}
			}
			if hasCall {
				for _, lhs := range stmt.Lhs {
					if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" {
						callAssignedVars[id.Name] = true
					}
				}
			}
		case *ast.IfStmt:
			if stmt.Init != nil {
				if assign, ok := stmt.Init.(*ast.AssignStmt); ok {
					hasCall := false
					for _, rhs := range assign.Rhs {
						if _, ok := rhs.(*ast.CallExpr); ok {
							hasCall = true
							break
						}
					}
					if hasCall {
						for _, lhs := range assign.Lhs {
							if id, ok := lhs.(*ast.Ident); ok && id.Name != "_" {
								callAssignedVars[id.Name] = true
							}
						}
					}
				}
			}

			if stmt.Else == nil && stmt.Body != nil && len(stmt.Body.List) == 1 {
				if retStmt, ok := stmt.Body.List[0].(*ast.ReturnStmt); ok && len(retStmt.Results) == 0 {
					if bin, ok := stmt.Cond.(*ast.BinaryExpr); ok && bin.Op == token.NEQ {
						var errIdent *ast.Ident
						if left, ok := bin.X.(*ast.Ident); ok && isNilIdent(bin.Y) && callAssignedVars[left.Name] {
							errIdent = left
						} else if right, ok := bin.Y.(*ast.Ident); ok && isNilIdent(bin.X) && callAssignedVars[right.Name] {
							errIdent = right
						}
						if errIdent != nil {
							line, col := getNodeLineAndColumn(fset, stmt, unit)
							if !seenLines[line] {
								seenLines[line] = true
								violations = append(violations, models.Violation{
									RuleID:     "GUARD-GO-SWALLOWED-ERR",
									Severity:   models.SeverityWarning,
									FilePath:   unit.FilePath,
									LineNumber: line,
									Column:     col,
									Message:    fmt.Sprintf("Swallowed error in '%s': error '%s' is checked with a bare return instead of failing the test.", unit.Name, errIdent.Name),
									Suggestion: "Fail the test with t.Fatalf or require.NoError instead of returning early.",
								})
							}
						}
					}
				}
			}
		}
		return true
	})

	// Evaluate each t.Run closure in its own scope
	for fnLit := range tRunFuncLits {
		closureTestingParams := make(map[*ast.Object]bool)
		if fnLit.Type != nil && fnLit.Type.Params != nil {
			for _, f := range fnLit.Type.Params.List {
				if isRecognizedTestingParam(f.Type, unit.FileImports) {
					for _, name := range f.Names {
						if name.Obj != nil {
							closureTestingParams[name.Obj] = true
						}
					}
				}
			}
		}
		closureViolations := inspectBlockForSwallowedErrors(fset, fnLit.Body, unit, closureTestingParams, suiteRecvObj)
		violations = append(violations, closureViolations...)
	}

	return violations
}

// Evaluate inspects test functions and t.Run closures for swallowed errors that return early instead of failing.
func (r *SwallowedErrorRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	fset, fn, err := getUnitAST(unit)
	if err != nil || fn.Body == nil {
		return violations
	}

	testingParamObjs := make(map[*ast.Object]bool)
	if fn.Type != nil && fn.Type.Params != nil {
		for _, f := range fn.Type.Params.List {
			if isRecognizedTestingParam(f.Type, unit.FileImports) {
				for _, name := range f.Names {
					if name.Obj != nil {
						testingParamObjs[name.Obj] = true
					}
				}
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

	return inspectBlockForSwallowedErrors(fset, fn.Body, unit, testingParamObjs, suiteRecvObj)
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

func isRecognizedTestingParam(expr ast.Expr, fileImports map[string]string) bool {
	testingPkgName := "testing"
	dotTesting := false
	if fileImports != nil {
		for local, path := range fileImports {
			if path == "testing" {
				if local == "." {
					dotTesting = true
				} else {
					testingPkgName = local
				}
				break
			}
		}
	}
	if star, ok := expr.(*ast.StarExpr); ok {
		if sel, ok := star.X.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == testingPkgName {
				if sel.Sel.Name == "T" || sel.Sel.Name == "B" || sel.Sel.Name == "TB" {
					return true
				}
			}
		} else if ident, ok := star.X.(*ast.Ident); ok {
			if dotTesting && (ident.Name == "T" || ident.Name == "B" || ident.Name == "TB") {
				return true
			}
			if ident.Name == "T" || ident.Name == "B" || ident.Name == "TB" {
				return true
			}
		}
	}
	return false
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

// Evaluate inspects table-driven test cases for duplicate row literals with identical non-name field contents.
func (r *TableDuplicateRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	fset, fn, err := getUnitAST(unit)
	if err != nil || fn.Body == nil {
		return violations
	}

	testingParamObjs := make(map[*ast.Object]bool)
	if fn.Type != nil && fn.Type.Params != nil {
		for _, f := range fn.Type.Params.List {
			if isRecognizedTestingParam(f.Type, unit.FileImports) {
				for _, name := range f.Names {
					if name.Obj != nil {
						testingParamObjs[name.Obj] = true
					}
				}
			}
		}
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if fnLit, ok := n.(*ast.FuncLit); ok {
			if fnLit.Type != nil && fnLit.Type.Params != nil {
				for _, f := range fnLit.Type.Params.List {
					if isRecognizedTestingParam(f.Type, unit.FileImports) {
						for _, name := range f.Names {
							if name.Obj != nil {
								testingParamObjs[name.Obj] = true
							}
						}
					}
				}
			}
		}
		return true
	})
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
			checkExprForName := func(expr ast.Expr) bool {
				found := false
				ast.Inspect(expr, func(cn ast.Node) bool {
					if sel, ok := cn.(*ast.SelectorExpr); ok {
						if id, ok := sel.X.(*ast.Ident); ok && id.Obj == rowIdentObj {
							low := strings.ToLower(sel.Sel.Name)
							if low == "name" || low == "desc" || low == "description" || low == "title" {
								found = true
								return false
							}
						}
					}
					return true
				})
				return found
			}

			ast.Inspect(rng.Body, func(bn ast.Node) bool {
				switch stmt := bn.(type) {
				case *ast.IfStmt:
					if checkExprForName(stmt.Cond) {
						hasConditionOnName = true
						return false
					}
				case *ast.SwitchStmt:
					if stmt.Tag != nil && checkExprForName(stmt.Tag) {
						hasConditionOnName = true
						return false
					}
				case *ast.CaseClause:
					for _, expr := range stmt.List {
						if checkExprForName(expr) {
							hasConditionOnName = true
							return false
						}
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
				if testingParamObjs[id.Obj] || (suiteRecvObj != nil && id.Obj == suiteRecvObj) {
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

// isWantFieldName reports whether name is a recognized want/expect field name.
// Lower-cased name starts with want, expect, or exp followed by nothing or an upper-case letter.
func isWantFieldName(name string) bool {
	lower := strings.ToLower(name)
	for _, prefix := range []string{"expected", "expect", "want", "exp"} {
		if strings.HasPrefix(lower, prefix) {
			rem := name[len(prefix):]
			if len(rem) == 0 {
				return true
			}
			r, _ := utf8.DecodeRuneInString(rem)
			if unicode.IsUpper(r) {
				return true
			}
		}
	}
	return false
}

func resolveRowStructType(eltType ast.Expr, fn *ast.FuncDecl, unit parser.TestUnit) *ast.StructType {
	switch t := eltType.(type) {
	case *ast.StructType:
		return t
	case *ast.StarExpr:
		return resolveRowStructType(t.X, fn, unit)
	case *ast.Ident:
		typeName := t.Name
		if t.Obj != nil {
			if ts, ok := t.Obj.Decl.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					return st
				}
			}
		}
		if fn != nil && fn.Body != nil {
			var found *ast.StructType
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == typeName {
					if st, ok := ts.Type.(*ast.StructType); ok {
						found = st
						return false
					}
				}
				return true
			})
			if found != nil {
				return found
			}
		}
		if unit.FilePath != "" {
			if src, err := os.ReadFile(unit.FilePath); err == nil {
				fileFset := token.NewFileSet()
				if parsedFile, err := goparser.ParseFile(fileFset, unit.FilePath, src, 0); err == nil {
					for _, decl := range parsedFile.Decls {
						if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
							for _, spec := range gen.Specs {
								if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == typeName {
									if st, ok := ts.Type.(*ast.StructType); ok {
										return st
									}
								}
							}
						}
					}
				}
			}
		}
		if unit.RawBody != "" {
			fileFset := token.NewFileSet()
			src := "package p\n" + unit.RawBody
			if parsedFile, err := goparser.ParseFile(fileFset, unit.FilePath, src, 0); err == nil {
				for _, decl := range parsedFile.Decls {
					if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
						for _, spec := range gen.Specs {
							if ts, ok := spec.(*ast.TypeSpec); ok && ts.Name.Name == typeName {
								if st, ok := ts.Type.(*ast.StructType); ok {
									return st
								}
							}
						}
					}
				}
			}
		}
	}
	return nil
}

func isTableCompositeForWant(compLit *ast.CompositeLit) bool {
	if len(compLit.Elts) < 1 {
		return false
	}

	arrType, isArray := compLit.Type.(*ast.ArrayType)
	if !isArray || arrType.Elt == nil {
		return false
	}
	if !isStructRowElementType(arrType.Elt) {
		return false
	}

	for _, elt := range compLit.Elts {
		if !isStructRowLiteral(elt) {
			return false
		}
	}

	return true
}

// UncheckedWantRule detects table-driven test row struct fields that define expected values
// but are never read or asserted against in the test loop.
type UncheckedWantRule struct{}

// NewUncheckedWantRule creates an initialized UncheckedWantRule.
func NewUncheckedWantRule() *UncheckedWantRule {
	return &UncheckedWantRule{}
}

// Evaluate inspects table-driven test structures to flag want/expect fields never asserted in the test loop.
func (r *UncheckedWantRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	fset, fn, err := getUnitAST(unit)
	if err != nil || fn.Body == nil {
		return violations
	}

	testingParamObjs := make(map[*ast.Object]bool)
	if fn.Type != nil && fn.Type.Params != nil {
		for _, f := range fn.Type.Params.List {
			if isRecognizedTestingParam(f.Type, unit.FileImports) {
				for _, name := range f.Names {
					if name.Obj != nil {
						testingParamObjs[name.Obj] = true
					}
				}
			}
		}
	}

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if fnLit, ok := n.(*ast.FuncLit); ok {
			if fnLit.Type != nil && fnLit.Type.Params != nil {
				for _, f := range fnLit.Type.Params.List {
					if isRecognizedTestingParam(f.Type, unit.FileImports) {
						for _, name := range f.Names {
							if name.Obj != nil {
								testingParamObjs[name.Obj] = true
							}
						}
					}
				}
			}
		}
		return true
	})

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
					if compLit, ok := s.Rhs[i].(*ast.CompositeLit); ok && isTableCompositeForWant(compLit) {
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
								if compLit, ok := vs.Values[i].(*ast.CompositeLit); ok && isTableCompositeForWant(compLit) {
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

	reportedFields := make(map[*ast.Ident]bool)

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

		valIdent, ok := rng.Value.(*ast.Ident)
		if !ok || valIdent.Name == "_" || valIdent.Obj == nil {
			return true
		}
		rowObj := valIdent.Obj
		rowName := valIdent.Name

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
			if id, ok := sel.X.(*ast.Ident); ok {
				if testingParamObjs[id.Obj] || (suiteRecvObj != nil && id.Obj == suiteRecvObj) {
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
		if !hasRunCall {
			return true
		}

		passedWhole := false
		ast.Inspect(rng.Body, func(bn ast.Node) bool {
			call, ok := bn.(*ast.CallExpr)
			if !ok {
				return true
			}
			for _, arg := range call.Args {
				if id, ok := arg.(*ast.Ident); ok {
					if id.Obj == rowObj || id.Name == rowName {
						passedWhole = true
						return false
					}
				}
			}
			return true
		})
		if passedWhole {
			return true
		}

		arrType, isArray := compLit.Type.(*ast.ArrayType)
		if !isArray || arrType.Elt == nil {
			return true
		}

		structType := resolveRowStructType(arrType.Elt, fn, unit)
		if structType == nil || structType.Fields == nil {
			return true
		}

		type wantFieldItem struct {
			name string
			node *ast.Ident
		}
		var wantFields []wantFieldItem
		for _, field := range structType.Fields.List {
			for _, nameIdent := range field.Names {
				if isWantFieldName(nameIdent.Name) {
					wantFields = append(wantFields, wantFieldItem{
						name: nameIdent.Name,
						node: nameIdent,
					})
				}
			}
		}
		if len(wantFields) == 0 {
			return true
		}

		readFields := make(map[string]bool)
		ast.Inspect(rng.Body, func(bn ast.Node) bool {
			if sel, ok := bn.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok {
					if id.Obj == rowObj || id.Name == rowName {
						readFields[sel.Sel.Name] = true
					}
				}
			}
			return true
		})

		for _, wf := range wantFields {
			if !readFields[wf.name] && !reportedFields[wf.node] {
				reportedFields[wf.node] = true
				line, col := getNodeLineAndColumn(fset, wf.node, unit)
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-GO-UNCHECKED-WANT",
					Severity:   models.SeverityWarning,
					FilePath:   unit.FilePath,
					LineNumber: line,
					Column:     col,
					Message:    fmt.Sprintf("Unchecked want field in '%s': field '%s' is declared in table row but never read in the test loop.", unit.Name, wf.name),
					Suggestion: fmt.Sprintf("Assert against '%s' in the test loop or remove the unused field.", wf.name),
				})
			}
		}

		return true
	})

	return violations
}
