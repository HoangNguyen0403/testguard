package golang

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var excludedFixturePackages = map[string]bool{
	"os": true, "json": true, "sqlmock": true, "http": true, "ioutil": true,
	"template": true, "sql": true, "filepath": true, "io": true, "fmt": true, "time": true,
}

// ExistenceOnlyRule detects Go unit tests calling multi-return functions (T, error)
// where assertions only verify NoError/NotNil without validating payload fields.
type ExistenceOnlyRule struct{}

// NewExistenceOnlyRule creates an initialized ExistenceOnlyRule.
func NewExistenceOnlyRule() *ExistenceOnlyRule {
	return &ExistenceOnlyRule{}
}

// Evaluate checks whether a Go test unit only checks error and non-nil presence on a multi-return call.
func (r *ExistenceOnlyRule) Evaluate(unit parser.TestUnit) []models.Violation {
	fset, fn, err := getUnitAST(unit)
	if err != nil || fn == nil || fn.Body == nil {
		return nil
	}

	// 1. Guardrail: Error-expectation tests (assert.Error, require.Error) are testing failure
	hasErrorExpectation := false
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				name := sel.Sel.Name
				if name == "Error" || name == "Errorf" || name == "ErrorIs" || name == "ErrorContains" || name == "EqualError" {
					hasErrorExpectation = true
					return false
				}
			}
		}
		return true
	})
	if hasErrorExpectation {
		return nil
	}

	var violations []models.Violation

	// Inspect all assignment statements in the test body
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		assign, ok := n.(*ast.AssignStmt)
		if !ok || assign.Tok != token.DEFINE || len(assign.Lhs) < 2 || len(assign.Rhs) < 1 {
			return true
		}

		resIdent, ok1 := assign.Lhs[0].(*ast.Ident)
		errIdent, ok2 := assign.Lhs[1].(*ast.Ident)
		if !ok1 || !ok2 || resIdent.Name == "_" {
			return true
		}
		if !strings.Contains(strings.ToLower(errIdent.Name), "err") {
			return true
		}

		// Check RHS call expression
		call, ok := assign.Rhs[0].(*ast.CallExpr)
		if !ok {
			return true
		}

		// Exclude generic standard library / test fixture utilities
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if xId, ok := sel.X.(*ast.Ident); ok {
				if excludedFixturePackages[xId.Name] {
					return true
				}
			}
		}

		resName := resIdent.Name

		// Check how the result binding is used across the function body. Any
		// binding-identity use of the result identifier is a payload
		// observation except the defining LHS ident, pair-wise bare discards,
		// arguments of existence assertions (NotNil/NotNilf), and operands of
		// nil comparisons. A pair-wise bare discard is `_ = res` or positional
		// `_, x = res, y` where the RHS element at the blank's index is itself
		// the (parenthesis-stripped) result identifier; idents nested in
		// call/selector/index/star expressions are ordinary observations.
		discarded := make(map[*ast.Ident]bool)
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			assign, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for i, lhs := range assign.Lhs {
				blank, ok := lhs.(*ast.Ident)
				if !ok || blank.Name != "_" || i >= len(assign.Rhs) {
					continue
				}
				expr := assign.Rhs[i]
				for {
					paren, ok := expr.(*ast.ParenExpr)
					if !ok {
						break
					}
					expr = paren.X
				}
				id, ok := expr.(*ast.Ident)
				if !ok || resIdent.Obj == nil || id.Obj != resIdent.Obj {
					continue
				}
				discarded[id] = true
			}
			return true
		})

		hasPayloadCheck := false
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			if hasPayloadCheck {
				return false
			}
			switch node := inner.(type) {
			case *ast.CallExpr:
				if isExistenceAssertCall(node) {
					return false
				}
			case *ast.BinaryExpr:
				if node.Op == token.EQL || node.Op == token.NEQ {
					if (isResultIdent(node.X, resIdent) && isNilIdent(node.Y)) ||
						(isResultIdent(node.Y, resIdent) && isNilIdent(node.X)) {
						return false
					}
				}
			case *ast.Ident:
				if node == resIdent || resIdent.Obj == nil || discarded[node] {
					return true
				}
				if node.Obj == resIdent.Obj {
					hasPayloadCheck = true
					return false
				}
			}
			return true
		})

		if !hasPayloadCheck {
			line, col := getNodeLineAndColumn(fset, assign, unit)
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-GO-EXISTENCE-ONLY",
				Severity:   models.SeverityWarning,
				FilePath:   unit.FilePath,
				LineNumber: line,
				Column:     col,
				Message:    fmt.Sprintf("Test '%s' verifies multi-return call '%s' only for non-nil existence without payload assertion", unit.Name, resName),
				Suggestion: "Assert returned payload fields, state, or invariants rather than checking only err == nil and result != nil.",
			})
		}

		return true
	})

	return violations
}

// isResultIdent reports whether expr refers to the multi-return result binding
// declared by resIdent. Parentheses and type assertions are transparent, so
// (res).Field, res[i], and res.(Wrapper).Field all observe the result payload.
// Binding identity comes from the parser-resolved ast.Object, so a same-spelled
// shadowing declaration is never credited to the result.
func isResultIdent(expr ast.Expr, resIdent *ast.Ident) bool {
	for {
		switch e := expr.(type) {
		case *ast.ParenExpr:
			expr = e.X
		case *ast.TypeAssertExpr:
			expr = e.X
		default:
			id, ok := expr.(*ast.Ident)
			if !ok || id.Name != resIdent.Name {
				return false
			}
			if resIdent.Obj == nil {
				// Without resolved bindings, identity cannot be proven; never
				// credit an observation on spelling alone.
				return false
			}
			return id.Obj == resIdent.Obj
		}
	}
}

func isExistenceAssertCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name == "NotNil" || sel.Sel.Name == "NotNilf"
	}
	return false
}
