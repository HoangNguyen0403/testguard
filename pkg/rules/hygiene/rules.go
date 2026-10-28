package hygiene

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	goparser "go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	numberRegex = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)
	stringRegex = regexp.MustCompile(`^(?:"[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'|` + "`" + `[^` + "`" + `\\]*(?:\\.[^` + "`" + `\\]*)*` + "`" + `)$`)

	strLit   = `(?:"[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'|` + "`" + `[^` + "`" + `\\]*(?:\\.[^` + "`" + `\\]*)*` + "`" + `)`
	parenArg = `(?:` + strLit + `|[^()]+?)`
	commaArg = `(?:` + strLit + `|[^,()]+?)`

	// TS/JS tautology patterns
	tsExpectSameRegex = regexp.MustCompile(`expect\s*\(\s*(` + parenArg + `)\s*\)\s*\.\s*(?:toBe|toEqual|toStrictEqual)\s*\(\s*(` + parenArg + `)\s*\)`)
	tsTruthyTautology = regexp.MustCompile(`expect\s*\(\s*true\s*\)\s*\.\s*toBeTruthy\s*\(\s*\)`)
	tsFalsyTautology  = regexp.MustCompile(`expect\s*\(\s*false\s*\)\s*\.\s*toBeFalsy\s*\(\s*\)`)

	// Dart tautology patterns
	dartTrueIsTrue   = regexp.MustCompile(`expect\s*\(\s*true\s*,\s*isTrue\s*(?:,[^)]*)?\)`)
	dartFalseIsFalse = regexp.MustCompile(`expect\s*\(\s*false\s*,\s*isFalse\s*(?:,[^)]*)?\)`)
	dartEqualsTaut   = regexp.MustCompile(`expect\s*\(\s*(` + commaArg + `)\s*,\s*equals\s*\(\s*(` + parenArg + `)\s*\)\s*(?:,[^)]*)?\)`)
	dartExpectSame   = regexp.MustCompile(`expect\s*\(\s*(` + commaArg + `)\s*,\s*(` + commaArg + `)\s*(?:,[^)]*)?\)`)

	// Go tautology patterns
	goTrueTautology  = regexp.MustCompile(`(?:assert|require)\.True\s*\(\s*\w+\s*,\s*true\s*(?:,[^)]*)?\)`)
	goFalseTautology = regexp.MustCompile(`(?:assert|require)\.False\s*\(\s*\w+\s*,\s*false\s*(?:,[^)]*)?\)`)
	goEqualSame      = regexp.MustCompile(`(?:assert|require)\.Equal(?:Values|f)?\s*\(\s*\w+\s*,\s*(` + commaArg + `)\s*,\s*(` + commaArg + `)\s*(?:,[^)]*)?\)`)
)

func isIdentPart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '$'
}

func isLiteral(s string) bool {
	s = strings.TrimSpace(s)
	if s == "true" || s == "false" || s == "null" || s == "nil" {
		return true
	}
	if numberRegex.MatchString(s) {
		return true
	}
	if stringRegex.MatchString(s) {
		if strings.HasPrefix(s, "`") && strings.Contains(s, "${") {
			return false
		}
		if (strings.HasPrefix(s, "'") || strings.HasPrefix(s, `"`)) && strings.Contains(s, "$") {
			return false
		}
		return true
	}
	return false
}

func isSameLiteral(l, r string) bool {
	l = strings.TrimSpace(l)
	r = strings.TrimSpace(r)
	if l != r {
		return false
	}
	return isLiteral(l)
}

func isTSFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".ts" || ext == ".tsx" || ext == ".js" || ext == ".jsx" || ext == ".mjs" || ext == ".cjs"
}

func isDartFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".dart")
}

func isGoFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".go")
}

type stringRange struct {
	start int
	end   int
}

func blankCommentsAndFindStrings(src string, isTS bool) (string, []stringRange) {
	b := []byte(src)
	n := len(b)
	var strRanges []stringRange
	i := 0
	for i < n {
		if i+1 < n && b[i] == '/' && b[i+1] == '/' {
			b[i] = ' '
			b[i+1] = ' '
			i += 2
			for i < n && b[i] != '\n' {
				if b[i] != '\r' {
					b[i] = ' '
				}
				i++
			}
			continue
		}
		if i+1 < n && b[i] == '/' && b[i+1] == '*' {
			b[i] = ' '
			b[i+1] = ' '
			i += 2
			for i < n {
				if i+1 < n && b[i] == '*' && b[i+1] == '/' {
					b[i] = ' '
					b[i+1] = ' '
					i += 2
					break
				}
				if b[i] != '\n' && b[i] != '\r' {
					b[i] = ' '
				}
				i++
			}
			continue
		}
		if !isTS && i+2 < n && ((b[i] == '\'' && b[i+1] == '\'' && b[i+2] == '\'') || (b[i] == '"' && b[i+1] == '"' && b[i+2] == '"')) {
			delim := b[i]
			start := i
			i += 3
			for i+2 < n {
				if b[i] == '\\' {
					i += 2
					continue
				}
				if b[i] == delim && b[i+1] == delim && b[i+2] == delim {
					i += 3
					break
				}
				i++
			}
			strRanges = append(strRanges, stringRange{start: start, end: i})
			continue
		}
		if !isTS && (b[i] == 'r' || b[i] == 'R') && i+1 < n && (b[i+1] == '\'' || b[i+1] == '"') {
			delim := b[i+1]
			start := i
			i += 2
			for i < n && b[i] != delim && b[i] != '\n' {
				i++
			}
			if i < n && b[i] == delim {
				i++
			}
			strRanges = append(strRanges, stringRange{start: start, end: i})
			continue
		}
		if b[i] == '\'' || b[i] == '"' || (isTS && b[i] == '`') {
			delim := b[i]
			start := i
			i++
			for i < n {
				if b[i] == '\\' {
					i += 2
					continue
				}
				if b[i] == delim {
					i++
					break
				}
				if (delim == '\'' || delim == '"') && b[i] == '\n' {
					break
				}
				i++
			}
			strRanges = append(strRanges, stringRange{start: start, end: i})
			continue
		}
		i++
	}
	return string(b), strRanges
}

func inStringRange(idx int, ranges []stringRange) bool {
	for _, r := range ranges {
		if idx >= r.start && idx < r.end {
			return true
		}
	}
	return false
}

func isIdentTrue(e ast.Expr) bool {
	if ident, ok := e.(*ast.Ident); ok {
		return ident.Name == "true"
	}
	return false
}

func isIdentFalse(e ast.Expr) bool {
	if ident, ok := e.(*ast.Ident); ok {
		return ident.Name == "false"
	}
	return false
}

func isSameGoLiteral(a, b ast.Expr) bool {
	switch va := a.(type) {
	case *ast.BasicLit:
		if vb, ok := b.(*ast.BasicLit); ok {
			return va.Kind == vb.Kind && va.Value == vb.Value
		}
	case *ast.Ident:
		if vb, ok := b.(*ast.Ident); ok {
			if (va.Name == "true" || va.Name == "false" || va.Name == "nil") && va.Name == vb.Name {
				return true
			}
		}
	case *ast.UnaryExpr:
		if vb, ok := b.(*ast.UnaryExpr); ok {
			return va.Op == vb.Op && isSameGoLiteral(va.X, vb.X)
		}
	}
	return false
}

type tautologyMatch struct {
	index int
}

type TautologyRule struct{}

func NewTautologyRule() *TautologyRule {
	return &TautologyRule{}
}

func (r *TautologyRule) Evaluate(unit parser.TestUnit) []models.Violation {
	body := unit.RawBody
	if strings.TrimSpace(body) == "" {
		return nil
	}

	var matches []tautologyMatch

	if isTSFile(unit.FilePath) {
		cleanBody, strRanges := blankCommentsAndFindStrings(body, true)
		for _, loc := range tsTruthyTautology.FindAllStringIndex(cleanBody, -1) {
			if !inStringRange(loc[0], strRanges) {
				matches = append(matches, tautologyMatch{index: loc[0]})
			}
		}
		for _, loc := range tsFalsyTautology.FindAllStringIndex(cleanBody, -1) {
			if !inStringRange(loc[0], strRanges) {
				matches = append(matches, tautologyMatch{index: loc[0]})
			}
		}
		for _, loc := range tsExpectSameRegex.FindAllStringSubmatchIndex(cleanBody, -1) {
			if len(loc) >= 6 && !inStringRange(loc[0], strRanges) {
				l := cleanBody[loc[2]:loc[3]]
				r := cleanBody[loc[4]:loc[5]]
				if isSameLiteral(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
	} else if isDartFile(unit.FilePath) {
		cleanBody, strRanges := blankCommentsAndFindStrings(body, false)
		for _, loc := range dartTrueIsTrue.FindAllStringIndex(cleanBody, -1) {
			if !inStringRange(loc[0], strRanges) {
				matches = append(matches, tautologyMatch{index: loc[0]})
			}
		}
		for _, loc := range dartFalseIsFalse.FindAllStringIndex(cleanBody, -1) {
			if !inStringRange(loc[0], strRanges) {
				matches = append(matches, tautologyMatch{index: loc[0]})
			}
		}
		for _, loc := range dartEqualsTaut.FindAllStringSubmatchIndex(cleanBody, -1) {
			if len(loc) >= 6 && !inStringRange(loc[0], strRanges) {
				l := cleanBody[loc[2]:loc[3]]
				r := cleanBody[loc[4]:loc[5]]
				if isSameLiteral(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
		for _, loc := range dartExpectSame.FindAllStringSubmatchIndex(cleanBody, -1) {
			if len(loc) >= 6 && !inStringRange(loc[0], strRanges) {
				l := cleanBody[loc[2]:loc[3]]
				r := cleanBody[loc[4]:loc[5]]
				if isSameLiteral(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
	} else if isGoFile(unit.FilePath) {
		var fdecl *ast.FuncDecl
		var fset *token.FileSet
		if d, ok := unit.ASTDecl.(*ast.FuncDecl); ok && d != nil {
			fdecl = d
			if fs, ok := unit.FileSet.(*token.FileSet); ok {
				fset = fs
			}
		}
		if fdecl == nil {
			fs := token.NewFileSet()
			src := "package p\nfunc TestUnit() {\n" + unit.RawBody + "\n}"
			f, err := goparser.ParseFile(fs, unit.FilePath, src, 0)
			if err == nil && len(f.Decls) > 0 {
				if fd, ok := f.Decls[0].(*ast.FuncDecl); ok {
					fdecl = fd
					fset = fs
				}
			}
		}
		if fdecl != nil && fset != nil {
			var goMatches []int
			ast.Inspect(fdecl, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				sel, ok := call.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				pkgIdent, ok := sel.X.(*ast.Ident)
				if !ok {
					return true
				}
				pkgName := pkgIdent.Name
				if pkgName != "assert" && pkgName != "require" {
					return true
				}
				fnName := sel.Sel.Name
				switch fnName {
				case "True":
					if len(call.Args) >= 2 {
						if isIdentTrue(call.Args[1]) {
							goMatches = append(goMatches, int(call.Pos()))
						}
					}
				case "False":
					if len(call.Args) >= 2 {
						if isIdentFalse(call.Args[1]) {
							goMatches = append(goMatches, int(call.Pos()))
						}
					}
				case "Equal", "EqualValues", "Equalf", "EqualValuesf":
					if len(call.Args) >= 3 {
						if isSameGoLiteral(call.Args[1], call.Args[2]) {
							goMatches = append(goMatches, int(call.Pos()))
						}
					}
				}
				return true
			})
			if len(goMatches) > 0 {
				pos := fset.Position(token.Pos(goMatches[0]))
				line := pos.Line
				if unit.ASTDecl == nil {
					line = unit.StartLine + (pos.Line - 3)
				}
				if line <= 0 {
					line = unit.StartLine
				}
				return []models.Violation{
					{
						RuleID:     "GUARD-UNIV-TAUTOLOGY",
						Severity:   models.SeverityBlocker,
						FilePath:   unit.FilePath,
						LineNumber: line,
						Column:     1,
						Message:    fmt.Sprintf("Tautological assertion in '%s': assertion will always pass vacuously.", unit.Name),
						Suggestion: "Assert actual behavior or state produced by the system under test rather than comparing literals to themselves.",
					},
				}
			}
			return nil
		}
		return nil
	} else {
		return nil
	}

	if len(matches) == 0 {
		return nil
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].index < matches[j].index
	})

	firstIndex := matches[0].index
	line := unit.StartLine + strings.Count(body[:firstIndex], "\n")
	if line <= 0 {
		line = 1
	}

	return []models.Violation{
		{
			RuleID:     "GUARD-UNIV-TAUTOLOGY",
			Severity:   models.SeverityBlocker,
			FilePath:   unit.FilePath,
			LineNumber: line,
			Column:     1,
			Message:    fmt.Sprintf("Tautological assertion in '%s': assertion will always pass vacuously.", unit.Name),
			Suggestion: "Assert actual behavior or state produced by the system under test rather than comparing literals to themselves.",
		},
	}
}

type SkipRule struct{}

func NewSkipRule() *SkipRule {
	return &SkipRule{}
}

var (
	tsSkipRegex  = regexp.MustCompile(`^(?:it\.skip|test\.skip|describe\.skip|xit|xtest|xdescribe)\s*\(`)
	dartSkipTrue = regexp.MustCompile(`^skip\s*:\s*true\b`)
	dartSkipStr  = regexp.MustCompile(`^skip\s*:\s*(?:'[^']*'|"[^"]*")`)
	goTestFuncRe = regexp.MustCompile(`func\s+Test\w*\s*\(\s*(\w+)\s+\*testing\.T\s*\)`)
)

func (r *SkipRule) CheckFile(filePath string, src string) []models.Violation {
	if isTSFile(filePath) {
		return r.checkTSSkips(filePath, src)
	} else if isDartFile(filePath) {
		return r.checkDartSkips(filePath, src)
	} else if isGoFile(filePath) {
		return r.checkGoSkips(filePath, src)
	}
	return nil
}

func (r *SkipRule) checkTSSkips(filePath string, src string) []models.Violation {
	var violations []models.Violation
	n := len(src)
	line := 1

	for i := 0; i < n; {
		if src[i] == '\n' {
			line++
			i++
			continue
		}

		// Single-line comment
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}

		// Multi-line block comment
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				if src[i] == '\n' {
					line++
				}
				i++
			}
			i += 2
			continue
		}

		// Standard quoted string literal: "..." or '...' or `...`
		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					if i+1 < n && src[i+1] == '\n' {
						line++
					}
					i += 2
					continue
				}
				if src[i] == '\n' {
					line++
					if quote != '`' {
						break
					}
				}
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
			continue
		}

		// Check for TS/JS skip pattern
		if i == 0 || !isIdentPart(src[i-1]) {
			if match := tsSkipRegex.FindString(src[i:]); match != "" {
				snippet := strings.TrimSpace(match)
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-SKIP",
					Severity:   models.SeverityWarning,
					FilePath:   filePath,
					LineNumber: line,
					Column:     1,
					Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", snippet),
					Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
				})
				i += len(match)
				continue
			}
		}

		i++
	}

	return violations
}

func findEnclosingCallName(blanked string, pos int) string {
	parenDepth := 0
	for i := pos - 1; i >= 0; i-- {
		c := blanked[i]
		if c == ')' {
			parenDepth++
		} else if c == '(' {
			if parenDepth > 0 {
				parenDepth--
			} else {
				j := i - 1
				for j >= 0 && (blanked[j] == ' ' || blanked[j] == '\t' || blanked[j] == '\r' || blanked[j] == '\n') {
					j--
				}
				if j >= 0 && isIdentPart(blanked[j]) {
					endIdent := j + 1
					for j >= 0 && isIdentPart(blanked[j]) {
						j--
					}
					return blanked[j+1 : endIdent]
				}
				return ""
			}
		}
	}
	return ""
}

var dartSkipCallNames = map[string]bool{
	"test":        true,
	"testWidgets": true,
	"group":       true,
	"blocTest":    true,
}

func (r *SkipRule) checkDartSkips(filePath string, src string) []models.Violation {
	var violations []models.Violation
	cleanSrc, _ := blankCommentsAndFindStrings(src, false)
	n := len(cleanSrc)
	line := 1

	for i := 0; i < n; {
		if src[i] == '\n' {
			line++
			i++
			continue
		}

		if cleanSrc[i] == ' ' || cleanSrc[i] == '\t' || cleanSrc[i] == '\r' {
			i++
			continue
		}

		if i == 0 || !isIdentPart(cleanSrc[i-1]) {
			rem := src[i:]
			var snippet string
			if match := dartSkipTrue.FindString(rem); match != "" {
				snippet = strings.TrimSpace(match)
			} else if match := dartSkipStr.FindString(rem); match != "" {
				snippet = strings.TrimSpace(match)
			}

			if snippet != "" {
				enclosingCall := findEnclosingCallName(cleanSrc, i)
				if dartSkipCallNames[enclosingCall] {
					violations = append(violations, models.Violation{
						RuleID:     "GUARD-UNIV-SKIP",
						Severity:   models.SeverityWarning,
						FilePath:   filePath,
						LineNumber: line,
						Column:     1,
						Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", snippet),
						Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
					})
					i += len(snippet)
					continue
				}
			}
		}

		i++
	}

	return violations
}

func (r *SkipRule) checkGoSkips(filePath string, src string) []models.Violation {
	var violations []models.Violation
	fset := token.NewFileSet()
	fileAST, err := goparser.ParseFile(fset, filePath, src, 0)
	if err != nil {
		return nil
	}

	testingPkgName := "testing"
	dotTesting := false
	for _, imp := range fileAST.Imports {
		if imp.Path != nil {
			path := strings.Trim(imp.Path.Value, `"`)
			if path == "testing" {
				if imp.Name != nil {
					if imp.Name.Name == "." {
						dotTesting = true
					} else {
						testingPkgName = imp.Name.Name
					}
				}
			}
		}
	}

	provableSuites := make(map[string]bool)
	for _, decl := range fileAST.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.TYPE {
			for _, spec := range gen.Specs {
				if ts, ok := spec.(*ast.TypeSpec); ok {
					if st, ok := ts.Type.(*ast.StructType); ok && st.Fields != nil {
						for _, field := range st.Fields.List {
							if sel, ok := field.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Suite" {
								provableSuites[ts.Name.Name] = true
							}
						}
					}
				}
			}
		}
	}

	for _, decl := range fileAST.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name == nil || fn.Body == nil || len(fn.Body.List) == 0 {
			continue
		}
		if !strings.HasPrefix(fn.Name.Name, "Test") && !strings.HasPrefix(fn.Name.Name, "Benchmark") {
			continue
		}

		// Reject receiver methods unless they belong to a proven suite
		var suiteRecvObj *ast.Object
		if fn.Recv != nil && len(fn.Recv.List) > 0 {
			recvType := ""
			switch t := fn.Recv.List[0].Type.(type) {
			case *ast.StarExpr:
				if id, ok := t.X.(*ast.Ident); ok {
					recvType = id.Name
				}
			case *ast.Ident:
				recvType = t.Name
			}
			if !provableSuites[recvType] {
				continue
			}
			if len(fn.Recv.List[0].Names) > 0 {
				suiteRecvObj = fn.Recv.List[0].Names[0].Obj
			}
		}

		var testingParamObj *ast.Object
		if fn.Recv == nil {
			if fn.Type.Params == nil || len(fn.Type.Params.List) == 0 {
				continue
			}
			for _, field := range fn.Type.Params.List {
				isTesting := false
				if star, ok := field.Type.(*ast.StarExpr); ok {
					if sel, ok := star.X.(*ast.SelectorExpr); ok {
						if ident, ok := sel.X.(*ast.Ident); ok && ident.Name == testingPkgName {
							if sel.Sel.Name == "T" || sel.Sel.Name == "B" || sel.Sel.Name == "F" || sel.Sel.Name == "TB" {
								isTesting = true
							}
						}
					} else if id, ok := star.X.(*ast.Ident); ok && dotTesting {
						if id.Name == "T" || id.Name == "B" || id.Name == "F" || id.Name == "TB" {
							isTesting = true
						}
					}
				}
				if isTesting {
					for _, name := range field.Names {
						if name.Obj != nil {
							testingParamObj = name.Obj
							break
						}
					}
				}
				if testingParamObj != nil {
					break
				}
			}
			if testingParamObj == nil {
				continue
			}
		}

		firstStmt := fn.Body.List[0]
		exprStmt, ok := firstStmt.(*ast.ExprStmt)
		if !ok {
			continue
		}
		call, ok := exprStmt.X.(*ast.CallExpr)
		if !ok {
			continue
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		methodName := sel.Sel.Name
		if methodName != "Skip" && methodName != "Skipf" && methodName != "SkipNow" {
			continue
		}

		isTargetSkip := false
		if recvIdent, ok := sel.X.(*ast.Ident); ok {
			if testingParamObj != nil && recvIdent.Obj == testingParamObj {
				isTargetSkip = true
			}
		} else if innerCall, ok := sel.X.(*ast.CallExpr); ok {
			if innerSel, ok := innerCall.Fun.(*ast.SelectorExpr); ok && innerSel.Sel.Name == "T" {
				if id, ok := innerSel.X.(*ast.Ident); ok && suiteRecvObj != nil && id.Obj == suiteRecvObj {
					isTargetSkip = true
				}
			}
		}
		if !isTargetSkip {
			continue
		}

		pos := fset.Position(call.Pos())
		snippet := fmt.Sprintf("%s(...)", methodName)
		var buf bytes.Buffer
		if err := format.Node(&buf, fset, call); err == nil {
			snippet = buf.String()
		}
		violations = append(violations, models.Violation{
			RuleID:     "GUARD-UNIV-SKIP",
			Severity:   models.SeverityWarning,
			FilePath:   filePath,
			LineNumber: pos.Line,
			Column:     1,
			Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", snippet),
			Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
		})
	}
	return violations
}
