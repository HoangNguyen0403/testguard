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

		// Check how resName is used across the function body
		hasPayloadCheck := false
		ast.Inspect(fn.Body, func(inner ast.Node) bool {
			if hasPayloadCheck {
				return false
			}
			switch node := inner.(type) {
			case *ast.SelectorExpr:
				if id, ok := node.X.(*ast.Ident); ok && id.Name == resName {
					// res.Field or res.Method()
					hasPayloadCheck = true
					return false
				}
			case *ast.IndexExpr:
				if id, ok := node.X.(*ast.Ident); ok && id.Name == resName {
					// res[i]
					hasPayloadCheck = true
					return false
				}
			case *ast.CallExpr:
				for _, arg := range node.Args {
					if isIdentNamed(arg, resName) {
						if isExistenceAssertCall(node) {
							continue
						}
						hasPayloadCheck = true
						return false
					}
				}
			case *ast.BinaryExpr:
				if (isIdentNamed(node.X, resName) && isNilIdent(node.Y)) ||
					(isIdentNamed(node.Y, resName) && isNilIdent(node.X)) {
					return true
				}
				if isIdentNamed(node.X, resName) || isIdentNamed(node.Y, resName) {
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

func isIdentNamed(expr ast.Expr, name string) bool {
	if id, ok := expr.(*ast.Ident); ok {
		return id.Name == name
	}
	return false
}

func isExistenceAssertCall(call *ast.CallExpr) bool {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		return sel.Sel.Name == "NotNil" || sel.Sel.Name == "NotNilf"
	}
	return false
}
