package hygiene

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	goparser "go/parser"
	"go/token"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/parser/golang"
)

var (
	numberRegex = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)
	stringRegex = regexp.MustCompile(`^(?:"[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'|` + "`" + `[^` + "`" + `\\]*(?:\\.[^` + "`" + `\\]*)*` + "`" + `)$`)

	strLit   = `(?:"[^"\\]*(?:\\.[^"\\]*)*"|'[^'\\]*(?:\\.[^'\\]*)*'|` + "`" + `[^` + "`" + `\\]*(?:\\.[^` + "`" + `\\]*)*` + "`" + `)`
	parenArg = `(?:` + strLit + `|[^()]+?)`
	commaArg = `(?:` + strLit + `|[^,()]+?)`

	// TS/JS tautology patterns
	tsExpectSameRegex    = regexp.MustCompile(`expect\s*\(\s*(` + parenArg + `)\s*\)\s*\.\s*(?:toBe|toEqual|toStrictEqual)\s*\(\s*(` + parenArg + `)\s*\)`)
	tsExpectNotSameRegex = regexp.MustCompile(`expect\s*\(\s*(` + parenArg + `)\s*\)\s*\.\s*not\s*\.\s*(?:toBe|toEqual|toStrictEqual)\s*\(\s*(` + parenArg + `)\s*\)`)
	tsTruthyTautology    = regexp.MustCompile(`expect\s*\(\s*true\s*\)\s*\.\s*toBeTruthy\s*\(\s*\)`)
	tsFalsyTautology     = regexp.MustCompile(`expect\s*\(\s*false\s*\)\s*\.\s*toBeFalsy\s*\(\s*\)`)

	// Dart tautology patterns
	dartTrueIsTrue   = regexp.MustCompile(`expect\s*\(\s*true\s*,\s*isTrue\s*(?:,[^)]*)?\)`)
	dartFalseIsFalse = regexp.MustCompile(`expect\s*\(\s*false\s*,\s*isFalse\s*(?:,[^)]*)?\)`)
	dartEqualsTaut   = regexp.MustCompile(`expect\s*\(\s*(` + commaArg + `)\s*,\s*equals\s*\(\s*(` + parenArg + `)\s*\)\s*(?:,[^)]*)?\)`)
	dartExpectSame   = regexp.MustCompile(`expect\s*\(\s*(` + commaArg + `)\s*,\s*(` + commaArg + `)\s*(?:,[^)]*)?\)`)

	// Kotlin tautology patterns
	ktAssertTrue  = regexp.MustCompile(`\bassertTrue\s*\(\s*true\s*(?:,[^)]*)?\)`)
	ktAssertFalse = regexp.MustCompile(`\bassertFalse\s*\(\s*false\s*(?:,[^)]*)?\)`)
	ktAssert      = regexp.MustCompile(`\bassert\s*\(\s*true\s*(?:,[^)]*)?\)`)
	ktAssertEqual = regexp.MustCompile(`\bassertEquals\s*\(\s*(` + commaArg + `)\s*,\s*(` + commaArg + `)\s*(?:,[^)]*)?\)`)

	// Swift tautology patterns
	swiftAssertTrue  = regexp.MustCompile(`\bXCTAssertTrue\s*\(\s*true\s*(?:,[^)]*)?\)`)
	swiftAssertFalse = regexp.MustCompile(`\bXCTAssertFalse\s*\(\s*false\s*(?:,[^)]*)?\)`)
	swiftAssertEqual = regexp.MustCompile(`\bXCTAssertEqual\s*\(\s*(` + commaArg + `)\s*,\s*(` + commaArg + `)\s*(?:,[^)]*)?\)`)
	swiftExpectTrue  = regexp.MustCompile(`#expect\s*\(\s*true\s*(?:,[^)]*)?\)`)
	swiftExpectEqual = regexp.MustCompile(`#expect\s*\(\s*(` + commaArg + `)\s*==\s*(` + commaArg + `)\s*(?:,[^)]*)?\)`)
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

// isMemberAccess reports whether the token starting at pos is preceded by a member access
// operator ('.' or '?.'), ignoring intervening whitespace and comments.
func isMemberAccess(blanked string, pos int) bool {
	j := pos - 1
	for j >= 0 && (blanked[j] == ' ' || blanked[j] == '\t' || blanked[j] == '\r' || blanked[j] == '\n') {
		j--
	}
	if j >= 0 && blanked[j] == '.' {
		return true
	}
	return false
}
func isIdent(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i := range s {
		c := s[i]
		if i == 0 {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$') {
				return false
			}
		} else {
			if !isIdentPart(c) {
				return false
			}
		}
	}
	return true
}

func isIdentOrDottedPath(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if s == "true" || s == "false" || s == "null" || s == "nil" {
		return false
	}
	for _, ch := range s {
		if ch == '(' || ch == ')' || ch == '[' || ch == ']' || ch == '+' || ch == '-' || ch == '*' || ch == '/' || ch == '!' || ch == '=' || ch == ',' || ch == ';' || ch == '{' || ch == '}' {
			return false
		}
	}
	parts := strings.Split(s, ".")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if !isIdent(part) {
			return false
		}
	}
	return true
}

func normalizeDottedPath(s string) string {
	parts := strings.Split(s, ".")
	for i, part := range parts {
		parts[i] = strings.TrimSpace(part)
	}
	return strings.Join(parts, ".")
}

func isSameOperand(l, r string) bool {
	if isSameLiteral(l, r) {
		return true
	}
	if isIdentOrDottedPath(l) && isIdentOrDottedPath(r) {
		return normalizeDottedPath(l) == normalizeDottedPath(r)
	}
	return false
}

func evalConstantArithmetic(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	expr, err := goparser.ParseExpr(s)
	if err != nil {
		return 0, false
	}
	return evalASTArithmetic(expr)
}

func evalASTArithmetic(expr ast.Expr) (float64, bool) {
	switch e := expr.(type) {
	case *ast.BasicLit:
		if e.Kind == token.INT || e.Kind == token.FLOAT {
			v, err := strconv.ParseFloat(e.Value, 64)
			if err != nil {
				return 0, false
			}
			return v, true
		}
		return 0, false
	case *ast.UnaryExpr:
		val, ok := evalASTArithmetic(e.X)
		if !ok {
			return 0, false
		}
		if e.Op == token.SUB {
			return -val, true
		} else if e.Op == token.ADD {
			return val, true
		}
		return 0, false
	case *ast.BinaryExpr:
		left, okL := evalASTArithmetic(e.X)
		right, okR := evalASTArithmetic(e.Y)
		if !okL || !okR {
			return 0, false
		}
		switch e.Op {
		case token.ADD:
			return left + right, true
		case token.SUB:
			return left - right, true
		case token.MUL:
			return left * right, true
		case token.QUO:
			if right == 0 {
				return 0, false
			}
			return left / right, true
		case token.REM:
			if right == 0 {
				return 0, false
			}
			return float64(int64(left) % int64(right)), true
		default:
			return 0, false
		}
	case *ast.ParenExpr:
		return evalASTArithmetic(e.X)
	default:
		return 0, false
	}
}

func isConstantArithmeticTautology(l, r string) bool {
	valL, okL := evalConstantArithmetic(l)
	if !okL {
		return false
	}
	valR, okR := evalConstantArithmetic(r)
	if !okR {
		return false
	}
	return math.Abs(valL-valR) < 1e-9
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
func isKotlinFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".kt" || ext == ".kts"
}

func isSwiftFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".swift")
}

type stringRange struct {
	start int
	end   int
}

// sourceLang selects the lexical rules used while blanking comments and
// recording string ranges. The syntaxes handled here differ in whether block
// comments nest and in how a literal embeds an interpolated expression.
type sourceLang int

const (
	langTS sourceLang = iota
	langDart
	langKotlin
	langSwift
)

// nestsBlockComments reports whether the language allows /* /* */ */ comments.
func (l sourceLang) nestsBlockComments() bool {
	return l == langKotlin || l == langSwift
}

// blankCommentsAndFindStrings returns src with comments replaced by spaces and
// the byte ranges of every string literal. Positions and string contents are
// preserved so callers can report original offsets and values while ignoring
// text that only looks like code.
func blankCommentsAndFindStrings(src string, lang sourceLang) (string, []stringRange) {
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
			end := findBlockCommentEnd(src, i, lang.nestsBlockComments())
			for k := i; k < end; k++ {
				if b[k] != '\n' && b[k] != '\r' {
					b[k] = ' '
				}
			}
			i = end
			continue
		}
		if end := scanStringLiteral(src, i, lang); end != i {
			strRanges = append(strRanges, stringRange{start: i, end: end})
			i = end
			continue
		}
		i++
	}
	return string(b), strRanges
}

// findBlockCommentEnd returns the offset just past the block comment starting at
// src[i], nesting when the language allows it. It returns len(src) when the
// comment is unterminated.
func findBlockCommentEnd(src string, i int, nested bool) int {
	n := len(src)
	j := i + 2
	depth := 1
	for j < n {
		if nested && j+1 < n && src[j] == '/' && src[j+1] == '*' {
			depth++
			j += 2
			continue
		}
		if j+1 < n && src[j] == '*' && src[j+1] == '/' {
			depth--
			j += 2
			if depth == 0 {
				return j
			}
			continue
		}
		j++
	}
	return n
}

// scanStringLiteral returns the offset just past the string literal starting at
// src[i], or i when src[i] does not start one. Interpolated expressions are
// scanned as code so a string or brace nested inside them cannot end the outer
// literal early.
func scanStringLiteral(src string, i int, lang sourceLang) int {
	n := len(src)
	if i >= n {
		return i
	}
	switch lang {
	case langTS:
		delim := src[i]
		if delim != '\'' && delim != '"' && delim != '`' {
			return i
		}
		j := i + 1
		for j < n {
			if src[j] == '\\' {
				j += 2
				continue
			}
			if src[j] == delim {
				return j + 1
			}
			if delim != '`' && src[j] == '\n' {
				return j
			}
			j++
		}
		return n
	case langDart:
		return scanDartStringLiteral(src, i)
	case langKotlin:
		return scanKotlinStringLiteral(src, i)
	case langSwift:
		return scanSwiftStringLiteral(src, i)
	}
	return i
}

// scanDartStringLiteral handles single-quoted, double-quoted, triple-quoted,
// r-prefixed raw literals, and ${…} interpolation.
func scanDartStringLiteral(src string, i int) int {
	n := len(src)
	start := i
	raw := false
	if (src[i] == 'r' || src[i] == 'R') && i+1 < n && (src[i+1] == '\'' || src[i+1] == '"') {
		raw = true
		i++
	}
	if i >= n || (src[i] != '\'' && src[i] != '"') {
		return start
	}
	delim := src[i]
	triple := i+2 < n && src[i+1] == delim && src[i+2] == delim
	j := i + 1
	if triple {
		j = i + 3
	}
	for j < n {
		if !triple && src[j] == '\\' {
			j += 2
			continue
		}
		if !raw && src[j] == '$' && j+1 < n {
			if src[j+1] == '{' {
				j = skipInterpolatedExpression(src, j+1, langDart, '{', '}')
				continue
			}
			if isIdentPart(src[j+1]) {
				j += 2
				for j < n && isIdentPart(src[j]) {
					j++
				}
				continue
			}
		}
		if triple {
			if src[j] == delim && j+2 < n && src[j+1] == delim && src[j+2] == delim {
				return j + 3
			}
		} else if src[j] == delim {
			return j + 1
		} else if src[j] == '\n' {
			return j
		}
		j++
	}
	return n
}

// scanKotlinStringLiteral handles '…', "…", """…""", and ${…} interpolation.
func scanKotlinStringLiteral(src string, i int) int {
	n := len(src)
	if src[i] == '\'' {
		j := i + 1
		for j < n {
			if src[j] == '\\' {
				j += 2
				continue
			}
			if src[j] == '\'' {
				return j + 1
			}
			if src[j] == '\n' {
				return j
			}
			j++
		}
		return n
	}
	if src[i] != '"' {
		return i
	}
	triple := i+2 < n && src[i+1] == '"' && src[i+2] == '"'
	j := i + 1
	if triple {
		j = i + 3
	}
	for j < n {
		if !triple && src[j] == '\\' {
			j += 2
			continue
		}
		if src[j] == '$' && j+1 < n {
			if src[j+1] == '{' {
				j = skipInterpolatedExpression(src, j+1, langKotlin, '{', '}')
				continue
			}
			if isIdentPart(src[j+1]) {
				j += 2
				for j < n && isIdentPart(src[j]) {
					j++
				}
				continue
			}
		}
		if triple {
			if src[j] == '"' && j+2 < n && src[j+1] == '"' && src[j+2] == '"' {
				return j + 3
			}
		} else if src[j] == '"' {
			return j + 1
		} else if src[j] == '\n' {
			return j
		}
		j++
	}
	return n
}

// scanSwiftStringLiteral handles "…", """…""", #"…"# raw literals, and \(…)
// interpolation.
func scanSwiftStringLiteral(src string, i int) int {
	n := len(src)
	if src[i] == '#' {
		hashes := 0
		k := i
		for k < n && src[k] == '#' {
			hashes++
			k++
		}
		if k < n && src[k] == '"' {
			suffix := strings.Repeat("#", hashes)
			if k+2 < n && src[k+1] == '"' && src[k+2] == '"' {
				idx := strings.Index(src[k+3:], `"""`+suffix)
				if idx == -1 {
					return n
				}
				return k + 3 + idx + 3 + hashes
			}
			idx := strings.Index(src[k+1:], `"`+suffix)
			if idx == -1 {
				return n
			}
			return k + 1 + idx + 1 + hashes
		}
		return i
	}
	if src[i] != '"' {
		return i
	}
	multiline := i+2 < n && src[i+1] == '"' && src[i+2] == '"'
	j := i + 1
	if multiline {
		j = i + 3
	}
	for j < n {
		if src[j] == '\\' {
			if j+1 < n && src[j+1] == '(' {
				j = skipInterpolatedExpression(src, j+1, langSwift, '(', ')')
				continue
			}
			j += 2
			continue
		}
		if multiline {
			if src[j] == '"' && j+2 < n && src[j+1] == '"' && src[j+2] == '"' {
				return j + 3
			}
		} else if src[j] == '"' {
			return j + 1
		} else if src[j] == '\n' {
			return j
		}
		j++
	}
	return n
}

// skipInterpolatedExpression returns the offset just past the interpolated
// expression whose opening delimiter is src[i]. Inside the expression, strings
// and comments are skipped so a close delimiter that belongs to them does not
// end the interpolation.
func skipInterpolatedExpression(src string, i int, lang sourceLang, open, close byte) int {
	n := len(src)
	depth := 0
	for i < n {
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			i = findBlockCommentEnd(src, i, lang.nestsBlockComments())
			continue
		}
		if end := scanStringLiteral(src, i, lang); end != i {
			i = end
			continue
		}
		switch src[i] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return n
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
func isSameGoIdentOrSelector(a, b ast.Expr) bool {
	switch va := a.(type) {
	case *ast.Ident:
		if vb, ok := b.(*ast.Ident); ok {
			if va.Name == "true" || va.Name == "false" || va.Name == "nil" {
				return false
			}
			return va.Name == vb.Name
		}
	case *ast.SelectorExpr:
		if vb, ok := b.(*ast.SelectorExpr); ok {
			return va.Sel.Name == vb.Sel.Name && isSameGoIdentOrSelector(va.X, vb.X)
		}
	}
	return false
}

type tautologyMatch struct {
	index int
}

// bodyStartLine returns the 1-based file line that holds the first byte of
// unit.RawBody. Adapters that publish Meta["body_start_line"] anchor statement
// diagnostics on the original line even when the declaration spans several
// lines; adapters without that fact fall back to the declaration line so
// whole-test findings stay put.
func bodyStartLine(unit parser.TestUnit) int {
	if raw, ok := unit.Meta["body_start_line"]; ok {
		if line, err := strconv.Atoi(raw); err == nil && line > 0 {
			return line
		}
	}
	return unit.StartLine
}

// TautologyRule reports assertions that compare a value with itself, which can
// never fail and therefore cannot detect a regression.
type TautologyRule struct{}

// NewTautologyRule creates an evaluator for GUARD-UNIV-TAUTOLOGY.
func NewTautologyRule() *TautologyRule {
	return &TautologyRule{}
}

// Evaluate reports at most one GUARD-UNIV-TAUTOLOGY violation per test unit,
// anchored on the line of the first tautological assertion found.
func (r *TautologyRule) Evaluate(unit parser.TestUnit) []models.Violation {
	body := unit.RawBody
	if strings.TrimSpace(body) == "" {
		return nil
	}

	var matches []tautologyMatch

	if isTSFile(unit.FilePath) {
		cleanBody, strRanges := blankCommentsAndFindStrings(body, langTS)
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
				if isSameOperand(l, r) || isConstantArithmeticTautology(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
		for _, loc := range tsExpectNotSameRegex.FindAllStringSubmatchIndex(cleanBody, -1) {
			if len(loc) >= 6 && !inStringRange(loc[0], strRanges) {
				l := cleanBody[loc[2]:loc[3]]
				r := cleanBody[loc[4]:loc[5]]
				if isLiteral(l) && isLiteral(r) && !isSameLiteral(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
	} else if isDartFile(unit.FilePath) {
		cleanBody, strRanges := blankCommentsAndFindStrings(body, langDart)
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
				if isSameOperand(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
		for _, loc := range dartExpectSame.FindAllStringSubmatchIndex(cleanBody, -1) {
			if len(loc) >= 6 && !inStringRange(loc[0], strRanges) {
				l := cleanBody[loc[2]:loc[3]]
				r := cleanBody[loc[4]:loc[5]]
				if isSameOperand(l, r) {
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
				default:
					if strings.HasPrefix(fnName, "Equal") {
						if len(call.Args) >= 3 {
							if isSameGoLiteral(call.Args[1], call.Args[2]) || isSameGoIdentOrSelector(call.Args[1], call.Args[2]) {
								goMatches = append(goMatches, int(call.Pos()))
							}
						}
					}
				}
				return true
			})
			if len(goMatches) > 0 {
				pos := fset.Position(token.Pos(goMatches[0]))
				line := pos.Line
				if unit.ASTDecl == nil {
					// The synthesized wrapper places the first RawBody line at
					// synthetic line 3.
					line = bodyStartLine(unit) + (pos.Line - 3)
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
	} else if isKotlinFile(unit.FilePath) {
		cleanBody, strRanges := blankCommentsAndFindStrings(body, langKotlin)
		for _, loc := range ktAssertTrue.FindAllStringIndex(cleanBody, -1) {
			if !inStringRange(loc[0], strRanges) {
				matches = append(matches, tautologyMatch{index: loc[0]})
			}
		}
		for _, loc := range ktAssertFalse.FindAllStringIndex(cleanBody, -1) {
			if !inStringRange(loc[0], strRanges) {
				matches = append(matches, tautologyMatch{index: loc[0]})
			}
		}
		for _, loc := range ktAssert.FindAllStringIndex(cleanBody, -1) {
			if !inStringRange(loc[0], strRanges) {
				matches = append(matches, tautologyMatch{index: loc[0]})
			}
		}
		for _, loc := range ktAssertEqual.FindAllStringSubmatchIndex(cleanBody, -1) {
			if len(loc) >= 6 && !inStringRange(loc[0], strRanges) {
				l := cleanBody[loc[2]:loc[3]]
				r := cleanBody[loc[4]:loc[5]]
				if isSameOperand(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
	} else if isSwiftFile(unit.FilePath) {
		cleanBody, strRanges := blankCommentsAndFindStrings(body, langSwift)
		for _, loc := range swiftAssertTrue.FindAllStringIndex(cleanBody, -1) {
			if !inStringRange(loc[0], strRanges) {
				matches = append(matches, tautologyMatch{index: loc[0]})
			}
		}
		for _, loc := range swiftAssertFalse.FindAllStringIndex(cleanBody, -1) {
			if !inStringRange(loc[0], strRanges) {
				matches = append(matches, tautologyMatch{index: loc[0]})
			}
		}
		for _, loc := range swiftExpectTrue.FindAllStringIndex(cleanBody, -1) {
			if !inStringRange(loc[0], strRanges) {
				matches = append(matches, tautologyMatch{index: loc[0]})
			}
		}
		for _, loc := range swiftAssertEqual.FindAllStringSubmatchIndex(cleanBody, -1) {
			if len(loc) >= 6 && !inStringRange(loc[0], strRanges) {
				l := cleanBody[loc[2]:loc[3]]
				r := cleanBody[loc[4]:loc[5]]
				if isSameOperand(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
		for _, loc := range swiftExpectEqual.FindAllStringSubmatchIndex(cleanBody, -1) {
			if len(loc) >= 6 && !inStringRange(loc[0], strRanges) {
				l := cleanBody[loc[2]:loc[3]]
				r := cleanBody[loc[4]:loc[5]]
				if isSameOperand(l, r) {
					matches = append(matches, tautologyMatch{index: loc[0]})
				}
			}
		}
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
	line := bodyStartLine(unit) + strings.Count(body[:firstIndex], "\n")
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

// SkipRule reports tests that are disabled outright, which hides the behaviour
// they were meant to protect.
type SkipRule struct{}

// NewSkipRule creates an evaluator for GUARD-UNIV-SKIP.
func NewSkipRule() *SkipRule {
	return &SkipRule{}
}

var (
	tsSkipDeclRegex    = regexp.MustCompile(`^(?:it\.skip|test\.skip|describe\.skip|test\.describe\.skip|xit|xtest|xdescribe|test\.fixme|it\.fixme)\s*\(`)
	tsSkipEachRegex    = regexp.MustCompile(`^(?:it|test|describe|test\.describe)\.skip\.each\s*` + "[(`]")
	dartSkipTrue       = regexp.MustCompile(`^skip\s*:\s*true\b`)
	dartSkipStr        = regexp.MustCompile(`^skip\s*:\s*(?:'[^']*'|"[^"]*")`)
	dartSkipAnnotation = regexp.MustCompile(`^@Skip\s*\(`)
	ktSkipAnnotation   = regexp.MustCompile(`^@(?:[a-zA-Z0-9_.]+\.)?(?:Ignore|Disabled)\b`)
	swiftFuncTestRe    = regexp.MustCompile(`\bfunc\s+(test\w*)\s*\(`)
	swiftThrowXCTSkip  = regexp.MustCompile(`^throw\s+XCTSkip\s*\(`)
	swiftDisabledAttr  = regexp.MustCompile(`^@Test\s*\([^)]*\.disabled\b`)
)

func (r *SkipRule) CheckFile(filePath string, src string) []models.Violation {
	if isTSFile(filePath) {
		return r.checkTSSkips(filePath, src)
	} else if isDartFile(filePath) {
		return r.checkDartSkips(filePath, src)
	} else if isGoFile(filePath) {
		return r.checkGoSkips(filePath, src)
	} else if isKotlinFile(filePath) {
		return r.checkKotlinSkips(filePath, src)
	} else if isSwiftFile(filePath) {
		return r.checkSwiftSkips(filePath, src)
	}
	return nil
}

func (r *SkipRule) checkTSSkips(filePath string, src string) []models.Violation {
	var violations []models.Violation
	cleanSrc, _ := blankCommentsAndFindStrings(src, langTS)
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
		if (i == 0 || !isIdentPart(src[i-1])) && !isMemberAccess(cleanSrc, i) {
			if match := tsSkipEachRegex.FindString(src[i:]); match != "" {
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
			if match := tsSkipDeclRegex.FindString(src[i:]); match != "" {
				rem := src[i+len(match):]
				trimmed := strings.TrimLeft(rem, " \t\r\n")
				if len(trimmed) > 0 && (trimmed[0] == '\'' || trimmed[0] == '"' || trimmed[0] == '`') {
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
	cleanSrc, strRanges := blankCommentsAndFindStrings(src, langDart)
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

		if (i == 0 || !isIdentPart(cleanSrc[i-1])) && !inStringRange(i, strRanges) {
			rem := cleanSrc[i:]
			if match := dartSkipAnnotation.FindString(rem); match != "" {
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-SKIP",
					Severity:   models.SeverityWarning,
					FilePath:   filePath,
					LineNumber: line,
					Column:     1,
					Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", "@Skip(...)"),
					Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
				})
				i += len(match)
				continue
			}
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
func (r *SkipRule) checkKotlinSkips(filePath string, src string) []models.Violation {
	var violations []models.Violation
	cleanSrc, strRanges := blankCommentsAndFindStrings(src, langKotlin)
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

		if (i == 0 || !isIdentPart(cleanSrc[i-1])) && !inStringRange(i, strRanges) {
			if match := ktSkipAnnotation.FindString(cleanSrc[i:]); match != "" {
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-SKIP",
					Severity:   models.SeverityWarning,
					FilePath:   filePath,
					LineNumber: line,
					Column:     1,
					Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", match),
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

func (r *SkipRule) checkSwiftSkips(filePath string, src string) []models.Violation {
	var violations []models.Violation
	cleanSrc, strRanges := blankCommentsAndFindStrings(src, langSwift)
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

		if !inStringRange(i, strRanges) {
			if match := swiftDisabledAttr.FindString(cleanSrc[i:]); match != "" {
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-UNIV-SKIP",
					Severity:   models.SeverityWarning,
					FilePath:   filePath,
					LineNumber: line,
					Column:     1,
					Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", "@Test(.disabled)"),
					Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
				})
				i += len(match)
				continue
			}

			if loc := swiftFuncTestRe.FindStringIndex(cleanSrc[i:]); loc != nil && loc[0] == 0 {
				bracePos := strings.IndexByte(cleanSrc[i+loc[1]:], '{')
				if bracePos != -1 {
					bodyStart := i + loc[1] + bracePos + 1
					j := bodyStart
					for j < n && (cleanSrc[j] == ' ' || cleanSrc[j] == '\t' || cleanSrc[j] == '\r' || cleanSrc[j] == '\n') {
						j++
					}
					if j < n && !inStringRange(j, strRanges) {
						if match := swiftThrowXCTSkip.FindString(cleanSrc[j:]); match != "" {
							stmtLine := 1 + strings.Count(src[:j], "\n")
							violations = append(violations, models.Violation{
								RuleID:     "GUARD-UNIV-SKIP",
								Severity:   models.SeverityWarning,
								FilePath:   filePath,
								LineNumber: stmtLine,
								Column:     1,
								Message:    fmt.Sprintf("Skipped test: '%s' disables a test; delete it or track the reason in a ticket.", "throw XCTSkip(...)"),
								Suggestion: "Remove the skip directive and fix the test, or track the reason in a ticket.",
							})
						}
					}
				}
				i += loc[1]
				continue
			}
		}

		i++
	}

	sort.Slice(violations, func(i, j int) bool {
		return violations[i].LineNumber < violations[j].LineNumber
	})

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

	provableSuites := golang.FindEmbeddedSuiteStructs(fileAST)

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
