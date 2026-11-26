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

// fatalGuardReturns maps each bare-return error check to the structurally
// fitting fatal guard that immediately precedes it in the same statement list.
// Only candidate adjacent if/if pairs are stored and the map stays nil when a
// body has none. Adjacency comes from list order, never from line numbers.
func fatalGuardReturns(body *ast.BlockStmt) map[*ast.IfStmt]*ast.IfStmt {
	var guards map[*ast.IfStmt]*ast.IfStmt
	if body == nil {
		return nil
	}
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		var list []ast.Stmt
		switch stmt := n.(type) {
		case *ast.BlockStmt:
			list = stmt.List
		case *ast.CaseClause:
			list = stmt.Body
		case *ast.CommClause:
			list = stmt.Body
		}
		for i := 1; i < len(list); i++ {
			guard, ok := list[i-1].(*ast.IfStmt)
			if !ok || !isFatalGuardShape(guard) {
				continue
			}
			if !isBareReturnErrorCheck(list[i]) {
				continue
			}
			if guards == nil {
				guards = make(map[*ast.IfStmt]*ast.IfStmt)
			}
			guards[list[i].(*ast.IfStmt)] = guard
		}
		return true
	})
	return guards
}

// isFatalGuardShape reports whether stmt can only abort the test when its
// condition holds: no else branch and a single fatal testing call.
func isFatalGuardShape(stmt *ast.IfStmt) bool {
	if stmt.Else != nil || stmt.Body == nil || len(stmt.Body.List) != 1 {
		return false
	}
	exprStmt, ok := stmt.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "Fatal", "Fatalf", "FailNow":
		return true
	}
	return false
}

// isBareReturnErrorCheck reports whether stmt is `if cond { return }`. The
// if-init must be absent: `if err = operation2(); err != nil { return }` reuses
// the guarded binding for a new value, so it cannot inherit the exemption.
func isBareReturnErrorCheck(stmt ast.Stmt) bool {
	ifStmt, ok := stmt.(*ast.IfStmt)
	if !ok || ifStmt.Init != nil || ifStmt.Else != nil || ifStmt.Body == nil || len(ifStmt.Body.List) != 1 {
		return false
	}
	retStmt, ok := ifStmt.Body.List[0].(*ast.ReturnStmt)
	return ok && len(retStmt.Results) == 0
}

// isVerifiedExpectedErrorReturn reports whether guard is the adjacent dominating
// expected-error mismatch check that already fails the test for errIdent. In
// `if (err != nil) != expectErr { t.Fatalf(...) }` the fatal call aborts on any
// mismatch, so the following bare `if err != nil { return }` is a legitimate
// "expected error, nothing more to assert" exit rather than a swallowed error.
// The returning if and mismatch guard must not initialize or reassign bindings.
func isVerifiedExpectedErrorReturn(
	guard *ast.IfStmt,
	errIdent *ast.Ident,
	inputFacts expectedErrorInputFacts,
	testingParamObjs map[*ast.Object]bool,
	suiteRecvObj *ast.Object,
) bool {
	if guard == nil || guard.Init != nil || errIdent == nil || errIdent.Obj == nil {
		return false
	}
	if !isExpectedErrorMismatch(guard.Cond, errIdent, guard, inputFacts) {
		return false
	}
	exprStmt, ok := guard.Body.List[0].(*ast.ExprStmt)
	if !ok {
		return false
	}
	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}
	return isFatalTestingCall(call, testingParamObjs, suiteRecvObj)
}

// isExpectedErrorMismatch compares the nil-ness of the same error binding
// against a distinct, stable test-input flag.
func isExpectedErrorMismatch(cond ast.Expr, errIdent *ast.Ident, guard *ast.IfStmt, inputFacts expectedErrorInputFacts) bool {
	bin, ok := cond.(*ast.BinaryExpr)
	if !ok || bin.Op != token.NEQ {
		return false
	}
	leftIsNilness := isNilComparisonOf(bin.X, errIdent)
	rightIsNilness := isNilComparisonOf(bin.Y, errIdent)
	if leftIsNilness == rightIsNilness {
		return false
	}
	other := bin.Y
	if rightIsNilness {
		other = bin.X
	}
	return isExpectedErrorFlag(other, errIdent, guard, inputFacts)
}

type expectedErrorInputFacts struct {
	firstWrite   map[*ast.Object]token.Pos
	dependencies map[*ast.Object]map[*ast.Object]token.Pos
}

// collectExpectedErrorInputFacts scans once for writes and direct RHS
// dependencies. The enclosing test AST is shared with t.Run closures so writes
// to captured test inputs are not lost at the closure boundary.
func collectExpectedErrorInputFacts(body *ast.BlockStmt) expectedErrorInputFacts {
	facts := expectedErrorInputFacts{
		firstWrite:   make(map[*ast.Object]token.Pos),
		dependencies: make(map[*ast.Object]map[*ast.Object]token.Pos),
	}
	ast.Inspect(body, func(n ast.Node) bool {
		switch stmt := n.(type) {
		case *ast.AssignStmt:
			isRangeBinding := stmt.Tok == token.DEFINE && len(stmt.Rhs) == 1 &&
				isRangeExpr(stmt.Rhs[0])
			for _, lhs := range stmt.Lhs {
				object := assignedRootObject(lhs)
				if object == nil {
					continue
				}
				if !isRangeBinding && (facts.firstWrite[object] == token.NoPos || stmt.Pos() < facts.firstWrite[object]) {
					facts.firstWrite[object] = stmt.Pos()
				}
				for _, rhs := range stmt.Rhs {
					ast.Inspect(rhs, func(rhsNode ast.Node) bool {
						ident, ok := rhsNode.(*ast.Ident)
						if !ok || ident.Obj == nil {
							return true
						}
						byDependency := facts.dependencies[object]
						if byDependency == nil {
							byDependency = make(map[*ast.Object]token.Pos)
							facts.dependencies[object] = byDependency
						}
						if byDependency[ident.Obj] == token.NoPos || stmt.Pos() < byDependency[ident.Obj] {
							byDependency[ident.Obj] = stmt.Pos()
						}
						return true
					})
				}
			}
		case *ast.IncDecStmt:
			object := assignedRootObject(stmt.X)
			if object != nil && (facts.firstWrite[object] == token.NoPos || stmt.Pos() < facts.firstWrite[object]) {
				facts.firstWrite[object] = stmt.Pos()
			}
		}
		return true
	})
	return facts
}

func isRangeExpr(expr ast.Expr) bool {
	unary, ok := expr.(*ast.UnaryExpr)
	return ok && unary.Op == token.RANGE
}

func assignedRootObject(expr ast.Expr) *ast.Object {
	switch target := expr.(type) {
	case *ast.Ident:
		return target.Obj
	case *ast.SelectorExpr:
		return assignedRootObject(target.X)
	case *ast.IndexExpr:
		return assignedRootObject(target.X)
	case *ast.IndexListExpr:
		return assignedRootObject(target.X)
	case *ast.StarExpr:
		return assignedRootObject(target.X)
	default:
		return nil
	}
}

// isExpectedErrorFlag accepts only resolved parameters, receivers, and range
// inputs, and rejects any input mutated before the mismatch check.
func isExpectedErrorFlag(expr ast.Expr, errIdent *ast.Ident, guard *ast.IfStmt, facts expectedErrorInputFacts) bool {
	for {
		paren, ok := expr.(*ast.ParenExpr)
		if !ok {
			break
		}
		expr = paren.X
	}
	if unary, ok := expr.(*ast.UnaryExpr); ok {
		if unary.Op != token.NOT {
			return false
		}
		expr = unary.X
		for {
			paren, ok := expr.(*ast.ParenExpr)
			if !ok {
				break
			}
			expr = paren.X
		}
	}
	if containsSameErrorBinding(expr, errIdent) {
		return false
	}
	var root *ast.Ident
	switch typed := expr.(type) {
	case *ast.Ident:
		root = typed
	case *ast.SelectorExpr:
		root, _ = typed.X.(*ast.Ident)
	default:
		return false
	}
	if root == nil || root.Obj == nil {
		return false
	}
	switch decl := root.Obj.Decl.(type) {
	case *ast.Field:
	case *ast.AssignStmt:
		if decl.Tok != token.DEFINE || len(decl.Rhs) != 1 {
			return false
		}
		rangeExpr, ok := decl.Rhs[0].(*ast.UnaryExpr)
		if !ok || rangeExpr.Op != token.RANGE || decl.Pos() > errIdent.Obj.Pos() {
			return false
		}
		if containsSameErrorBinding(rangeExpr.X, errIdent) ||
			rangeSourceDependsOnError(rangeExpr.X, decl.Pos(), errIdent.Obj, facts) {
			return false
		}
	default:
		return false
	}
	writePos := facts.firstWrite[root.Obj]
	return writePos == token.NoPos || writePos >= guard.Pos()
}

func rangeSourceDependsOnError(source ast.Expr, rangePos token.Pos, errObject *ast.Object, facts expectedErrorInputFacts) bool {
	sourceIdent, ok := source.(*ast.Ident)
	if !ok || sourceIdent.Obj == nil {
		return false
	}
	errWritePos := facts.dependencies[sourceIdent.Obj][errObject]
	return errWritePos != token.NoPos && errWritePos < rangePos
}

// isNilComparisonOf reports whether expr is `err != nil` or `err == nil` for the
// same error binding, allowing redundant parentheses.
func isNilComparisonOf(expr ast.Expr, errIdent *ast.Ident) bool {
	inner := expr
	for {
		paren, ok := inner.(*ast.ParenExpr)
		if !ok {
			break
		}
		inner = paren.X
	}
	bin, ok := inner.(*ast.BinaryExpr)
	if !ok || (bin.Op != token.NEQ && bin.Op != token.EQL) {
		return false
	}
	if sameErrorBinding(bin.X, errIdent) && isNilIdent(bin.Y) {
		return true
	}
	return sameErrorBinding(bin.Y, errIdent) && isNilIdent(bin.X)
}

// sameErrorBinding reports whether expr is the same declared error object.
func sameErrorBinding(expr ast.Expr, errIdent *ast.Ident) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok || ident.Obj == nil || errIdent == nil {
		return false
	}
	return ident.Obj == errIdent.Obj
}

func containsSameErrorBinding(expr ast.Expr, errIdent *ast.Ident) bool {
	found := false
	ast.Inspect(expr, func(n ast.Node) bool {
		if found {
			return false
		}
		if ident, ok := n.(*ast.Ident); ok && ident.Obj != nil && errIdent != nil && ident.Obj == errIdent.Obj {
			found = true
			return false
		}
		return true
	})
	return found
}

// isFatalTestingCall reports whether call aborts the test through the same
// testing receiver (`t.Fatal`, `t.Fatalf`, `t.FailNow`, or `s.T().Fatal*`).
func isFatalTestingCall(call *ast.CallExpr, testingParamObjs map[*ast.Object]bool, suiteRecvObj *ast.Object) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch sel.Sel.Name {
	case "Fatal", "Fatalf", "FailNow":
	default:
		return false
	}
	return isTestingReceiverExpr(sel.X, testingParamObjs, suiteRecvObj)
}

func isTestingReceiverExpr(expr ast.Expr, testingParamObjs map[*ast.Object]bool, suiteRecvObj *ast.Object) bool {
	if ident, ok := expr.(*ast.Ident); ok {
		if ident.Obj == nil {
			return false
		}
		return testingParamObjs[ident.Obj] || (suiteRecvObj != nil && ident.Obj == suiteRecvObj)
	}
	if call, ok := expr.(*ast.CallExpr); ok && len(call.Args) == 0 {
		if innerSel, ok := call.Fun.(*ast.SelectorExpr); ok && innerSel.Sel.Name == "T" {
			if ident, ok := innerSel.X.(*ast.Ident); ok && suiteRecvObj != nil && ident.Obj == suiteRecvObj {
				return true
			}
		}
	}
	return false
}

func inspectBlockForSwallowedErrors(
	fset *token.FileSet,
	body *ast.BlockStmt,
	unit parser.TestUnit,
	inputFacts expectedErrorInputFacts,
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

	guards := fatalGuardReturns(body)
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
						if errIdent != nil && !isVerifiedExpectedErrorReturn(guards[stmt], errIdent, inputFacts, testingParamObjs, suiteRecvObj) {
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
		closureViolations := inspectBlockForSwallowedErrors(fset, fnLit.Body, unit, inputFacts, closureTestingParams, suiteRecvObj)
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

	inputFacts := collectExpectedErrorInputFacts(fn.Body)
	return inspectBlockForSwallowedErrors(fset, fn.Body, unit, inputFacts, testingParamObjs, suiteRecvObj)
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

func isNameLikeField(name string) bool {
	switch strings.ToLower(name) {
	case "name", "desc", "description", "title":
		return true
	}
	return false
}

// isTestReportingCall reports whether call is a method on the recognized
// testing parameter or suite receiver (t.Run, t.Errorf, b.Logf, s.T().Fatal,
// etc.). Labels passed only to these helpers remain cosmetic.
func isTestReportingCall(call *ast.CallExpr, testingParamObjs map[*ast.Object]bool, suiteRecvObj *ast.Object) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	if id, ok := sel.X.(*ast.Ident); ok {
		if id.Obj == nil {
			return false
		}
		return testingParamObjs[id.Obj] || (suiteRecvObj != nil && id.Obj == suiteRecvObj)
	}
	// Suite pattern s.T().Run / s.T().Errorf
	if innerCall, ok := sel.X.(*ast.CallExpr); ok {
		if innerSel, ok := innerCall.Fun.(*ast.SelectorExpr); ok && innerSel.Sel.Name == "T" {
			if id, ok := innerSel.X.(*ast.Ident); ok && id.Obj != nil && suiteRecvObj != nil && id.Obj == suiteRecvObj {
				return true
			}
		}
	}
	return false
}

// rowNameInputFields returns the lower-cased name-like row fields that the
// range body consumes as call arguments outside the test runner/reporting
// helpers. Binding identity comes from the parser-resolved ast.Object, so a
// same-spelled shadowed row is never credited.
func rowNameInputFields(rng *ast.RangeStmt, rowIdentObj *ast.Object, testingParamObjs map[*ast.Object]bool, suiteRecvObj *ast.Object) map[string]bool {
	if rowIdentObj == nil {
		return nil
	}
	fields := make(map[string]bool)
	ast.Inspect(rng.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if isTestReportingCall(call, testingParamObjs, suiteRecvObj) {
			return true
		}
		for _, arg := range call.Args {
			ast.Inspect(arg, func(an ast.Node) bool {
				sel, ok := an.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				id, ok := sel.X.(*ast.Ident)
				if ok && id.Obj == rowIdentObj && isNameLikeField(sel.Sel.Name) {
					fields[strings.ToLower(sel.Sel.Name)] = true
				}
				return true
			})
		}
		return true
	})
	return fields
}

// normalizeRowContent renders a row literal into a comparison key. Nominal
// name/desc/title fields are stripped as cosmetic labels unless the field is
// listed in retainedNameFields because the loop body consumes that row field as
// a call argument (a genuine test input).
func normalizeRowContent(fset *token.FileSet, elt ast.Expr, compLit *ast.CompositeLit, retainedNameFields map[string]bool) (string, string) {
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

	nameFieldIndexes := make(map[int]string)
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
							if isNameLikeField(name.Name) {
								nameFieldIndexes[fieldIdx] = strings.ToLower(name.Name)
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
				if isNameLikeField(kIdent.Name) && !retainedNameFields[low] {
					continue
				}
			}
			var buf bytes.Buffer
			_ = format.Node(&buf, fset, kv)
			parts = append(parts, buf.String())
		} else {
			if fieldName, isNameField := nameFieldIndexes[idx]; isNameField && !retainedNameFields[fieldName] {
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
	retainedNameFields := make(map[*ast.CompositeLit]map[string]bool)
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
							if isNameLikeField(sel.Sel.Name) {
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

		if inputs := rowNameInputFields(rng, rowIdentObj, testingParamObjs, suiteRecvObj); len(inputs) > 0 {
			retained := retainedNameFields[compLit]
			if retained == nil {
				retained = make(map[string]bool)
				retainedNameFields[compLit] = retained
			}
			for field := range inputs {
				retained[field] = true
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
			normalizedKey, fullContent := normalizeRowContent(fset, elt, compLit, retainedNameFields[compLit])
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
