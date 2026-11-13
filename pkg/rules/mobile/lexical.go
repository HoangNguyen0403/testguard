package mobile

import (
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

// codeMask returns a view of src of exactly the same length in which comments
// and the text of string, character and backtick literals are replaced by
// spaces. Newlines are preserved, so every offset in the mask addresses the
// same byte of the original source. Rules use it so that code hidden inside
// comments or literals cannot be mistaken for a real construct, while literal
// values stay readable from the original text at the same offsets.
func codeMask(src string) string {
	if src == "" {
		return ""
	}
	masked := []byte(src)
	n := len(src)
	for i := 0; i < n; {
		switch {
		case src[i] == '/' && i+1 < n && src[i+1] == '/':
			for i < n && src[i] != '\n' {
				masked[i] = ' '
				i++
			}
		case src[i] == '/' && i+1 < n && src[i+1] == '*':
			depth := 1
			masked[i], masked[i+1] = ' ', ' '
			i += 2
			for i < n && depth > 0 {
				switch {
				case src[i] == '/' && i+1 < n && src[i+1] == '*':
					depth++
					masked[i], masked[i+1] = ' ', ' '
					i += 2
				case src[i] == '*' && i+1 < n && src[i+1] == '/':
					depth--
					masked[i], masked[i+1] = ' ', ' '
					i += 2
				default:
					if src[i] != '\n' {
						masked[i] = ' '
					}
					i++
				}
			}
		case src[i] == '"' || src[i] == '\'' || src[i] == '`':
			end := stringEnd(src, i)
			if end < 0 {
				end = n
			}
			for ; i < end; i++ {
				if src[i] != '\n' {
					masked[i] = ' '
				}
			}
		default:
			i++
		}
	}
	return string(masked)
}

// stringEnd returns the offset just past the literal that starts at i, or -1
// when the literal is unterminated on its line.
func stringEnd(src string, i int) int {
	n := len(src)
	quote := src[i]
	if quote == '`' {
		for j := i + 1; j < n; j++ {
			if src[j] == '`' {
				return j + 1
			}
		}
		return -1
	}
	if i+2 < n && src[i+1] == quote && src[i+2] == quote {
		for j := i + 3; j+2 < n; j++ {
			if src[j] == quote && src[j+1] == quote && src[j+2] == quote {
				return j + 3
			}
		}
		return -1
	}
	for j := i + 1; j < n; j++ {
		switch src[j] {
		case '\\':
			j++
		case quote:
			return j + 1
		case '\n':
			return -1
		}
	}
	return -1
}

func isCodeSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\r' || b == '\n'
}

func isIdentStart(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isIdentPart(b byte) bool {
	return isIdentStart(b) || (b >= '0' && b <= '9')
}

func skipCodeSpace(code string, i int) int {
	for i < len(code) && isCodeSpace(code[i]) {
		i++
	}
	return i
}

// skipInlineSpace skips horizontal whitespace only, so a masked literal on the
// same line is not skipped as if it were blank.
func skipInlineSpace(src string, i int) int {
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\r') {
		i++
	}
	return i
}

// identAt reads the identifier that starts at i and returns it with the offset
// just past its last byte.
func identAt(code string, i int) (string, int) {
	j := i
	for j < len(code) && isIdentPart(code[j]) {
		j++
	}
	return code[i:j], j
}

type codeSpan struct{ start, end int }

// trimBodySpan trims ASCII whitespace using the original source, so a literal
// that the code mask turned into spaces is not trimmed away.
func trimBodySpan(body string, s, e int) (int, int) {
	for s < e && isCodeSpace(body[s]) {
		s++
	}
	for e > s && isCodeSpace(body[e-1]) {
		e--
	}
	return s, e
}

func trimSpan(code string, s, e int) (int, int) {
	for s < e && isCodeSpace(code[s]) {
		s++
	}
	for e > s && isCodeSpace(code[e-1]) {
		e--
	}
	return s, e
}

// callArgs splits the argument list of the call whose opening parenthesis is at
// openParen and returns the argument spans plus the closing parenthesis offset.
// Spans are left untrimmed because the mask turns literals into whitespace.
func callArgs(code string, openParen int) ([]codeSpan, int) {
	if openParen < 0 || openParen >= len(code) || code[openParen] != '(' {
		return nil, -1
	}
	closeIdx := parser.FindMatchingParen(code, openParen)
	if closeIdx < 0 {
		return nil, -1
	}
	var spans []codeSpan
	depth := 0
	start := openParen + 1
	for i := openParen + 1; i < closeIdx; i++ {
		switch code[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case ',':
			if depth == 0 {
				spans = append(spans, codeSpan{start, i})
				start = i + 1
			}
		}
	}
	if len(spans) > 0 || start < closeIdx {
		spans = append(spans, codeSpan{start, closeIdx})
	}
	return spans, closeIdx
}

// argValueSpan strips a leading `name =` from a Kotlin named argument.
func argValueSpan(code, body string, sp codeSpan) codeSpan {
	s, e := trimBodySpan(body, sp.start, sp.end)
	if s >= e || !isIdentStart(code[s]) {
		return codeSpan{s, e}
	}
	_, j := identAt(code, s)
	k := skipCodeSpace(code, j)
	if k >= e || code[k] != '=' || (k+1 < e && code[k+1] == '=') {
		return codeSpan{s, e}
	}
	return argValueSpan(code, body, codeSpan{k + 1, e})
}

func singleIdent(code string, sp codeSpan) (string, bool) {
	s, e := trimSpan(code, sp.start, sp.end)
	if s >= e || !isIdentStart(code[s]) {
		return "", false
	}
	name, j := identAt(code, s)
	if j != e {
		return "", false
	}
	return name, true
}

// findAssign returns the offset of the assignment operator that binds the
// declaration starting at from, or -1 when the declaration does not assign on
// the same statement.
func findAssign(code string, from int) int {
	depth := 0
	for i := from; i < len(code); i++ {
		switch code[i] {
		case '(', '[', '{':
			depth++
		case ')', ']', '}':
			depth--
		case '\n':
			if depth <= 0 {
				return -1
			}
		case '=':
			if depth != 0 {
				continue
			}
			if i+1 < len(code) && code[i+1] == '=' {
				return -1
			}
			if i > from && strings.IndexByte("!<>=", code[i-1]) >= 0 {
				continue
			}
			return i
		}
	}
	return -1
}

// skipAngleGroups skips one generic argument list, e.g. `<Repo>`, and returns
// the offset just past its closing angle bracket.
func skipAngleGroups(code string, i int) int {
	depth := 0
	for ; i < len(code); i++ {
		switch code[i] {
		case '<':
			depth++
		case '>':
			if i > 0 && code[i-1] == '-' {
				continue
			}
			depth--
			if depth == 0 {
				return i + 1
			}
		case '\n':
			return i
		}
	}
	return i
}

// initDecl is a local value declaration initialised by a call, e.g.
// `val vm = SearchViewModel(repo)`.
type initDecl struct {
	name string
	ctor string
	args []codeSpan
	pos  int
}

// scanInitDecls returns every `val`/`var` declaration in code whose initializer
// is a constructor or factory call.
func scanInitDecls(code string) []initDecl {
	var decls []initDecl
	for i := 0; i < len(code); {
		if !isIdentStart(code[i]) {
			i++
			continue
		}
		keyword, j := identAt(code, i)
		if keyword == "val" || keyword == "var" {
			if d, next, ok := parseInitDecl(code, j); ok {
				decls = append(decls, d)
				i = next
				continue
			}
		}
		i = j
	}
	return decls
}

func parseInitDecl(code string, afterKeyword int) (initDecl, int, bool) {
	i := skipCodeSpace(code, afterKeyword)
	if i >= len(code) || !isIdentStart(code[i]) {
		return initDecl{}, i, false
	}
	name, j := identAt(code, i)
	eq := findAssign(code, j)
	if eq < 0 {
		return initDecl{}, j, false
	}
	v := skipCodeSpace(code, eq+1)
	if v >= len(code) || !isIdentStart(code[v]) {
		return initDecl{}, v, false
	}
	ctor, w := identAt(code, v)
	w = skipCodeSpace(code, w)
	if w < len(code) && code[w] == '<' {
		w = skipCodeSpace(code, skipAngleGroups(code, w))
	}
	if w >= len(code) || code[w] != '(' {
		return initDecl{}, w, false
	}
	args, closeIdx := callArgs(code, w)
	if closeIdx < 0 {
		return initDecl{}, w, false
	}
	return initDecl{name: name, ctor: ctor, args: args, pos: i}, closeIdx + 1, true
}

// declPassesIdent reports whether one of the declaration's arguments is exactly
// the identifier name, which is how a collaborator is injected into the subject.
func declPassesIdent(code, body string, d initDecl, name string) bool {
	for _, a := range d.args {
		if got, ok := singleIdent(code, argValueSpan(code, body, a)); ok && got == name {
			return true
		}
	}
	return false
}

// pureChain reads an identifier member chain that contains no calls, e.g.
// `vm.state.value`. It returns the dotted segments and the offset just past the
// last segment.
func pureChain(code string, i int) ([]string, int) {
	name, j := identAt(code, i)
	if name == "" {
		return nil, i
	}
	segments := []string{name}
	end := j
	for {
		k := skipCodeSpace(code, end)
		if k >= len(code) || code[k] != '.' {
			return segments, end
		}
		n := skipCodeSpace(code, k+1)
		if n >= len(code) || !isIdentStart(code[n]) {
			return segments, end
		}
		segment, m := identAt(code, n)
		segments = append(segments, segment)
		end = m
	}
}

// subjectChain reports whether the whole span is a pure member chain rooted at
// one of the given subjects, and returns the chain text.
func subjectChain(code string, sp codeSpan, subjects map[string]bool) (string, bool) {
	s, e := trimSpan(code, sp.start, sp.end)
	if s >= e || !isIdentStart(code[s]) {
		return "", false
	}
	segments, end := pureChain(code, s)
	if len(segments) < 2 || end != e || !subjects[segments[0]] {
		return "", false
	}
	return strings.Join(segments, "."), true
}

// scanExpressionEnd returns the offset where the expression that starts at
// start ends: the newline, semicolon or unbalanced closing bracket outside
// literals, or the start of a trailing comment.
func scanExpressionEnd(body string, start int) int {
	depth := 0
	for i := start; i < len(body); i++ {
		c := body[i]
		switch {
		case c == '"' || c == '\'':
			end := stringEnd(body, i)
			if end < 0 {
				return len(body)
			}
			i = end - 1
		case c == '/' && i+1 < len(body) && (body[i+1] == '/' || body[i+1] == '*'):
			return i
		case c == '(' || c == '[' || c == '{':
			depth++
		case c == ')' || c == ']' || c == '}':
			if depth == 0 {
				return i
			}
			depth--
		case (c == '\n' || c == ';') && depth == 0:
			return i
		}
	}
	return len(body)
}

func isNumberToken(tok string) bool {
	i := 0
	if i < len(tok) && tok[i] == '-' {
		i++
	}
	digits := 0
	for ; i < len(tok); i++ {
		if tok[i] >= '0' && tok[i] <= '9' {
			digits++
			continue
		}
		if tok[i] == '.' {
			continue
		}
		if strings.IndexByte("fFlLdD", tok[i]) >= 0 && i == len(tok)-1 {
			return digits > 0
		}
		return false
	}
	return digits > 0
}

func isSimpleToken(tok string) bool {
	if tok == "" {
		return false
	}
	if tok[0] == '"' || tok[0] == '\'' {
		return stringEnd(tok, 0) == len(tok)
	}
	if isIdentStart(tok[0]) {
		for i := 1; i < len(tok); i++ {
			if !isIdentPart(tok[i]) {
				return false
			}
		}
		return true
	}
	return isNumberToken(tok)
}
