package typescript

import (
	"fmt"
	"strings"
)

// TokenType identifies a lexical token in JavaScript/TypeScript source code.
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenKeyword
	TokenString
	TokenTemplate
	TokenNumber
	TokenRegex
	TokenPunct
	TokenComment
)

// Token represents a lexical token with its position and raw content.
type Token struct {
	Type   TokenType
	Value  string
	Offset int
	Line   int
}

type jsxElement struct {
	name      string
	exprDepth int
}

// Lexer scans JavaScript and TypeScript source text into tokens.
type Lexer struct {
	src          string
	n            int
	pos          int
	line         int
	enableJSX    bool
	jsxStack     []jsxElement
	inJSXTag     bool
	jsxExprDepth int
	scanErr      error
}

// delimiterPairs maps an opening bracket to the bracket that closes it.
var delimiterPairs = map[string]string{"(": ")", "[": "]", "{": "}"}

// scanError reports the first structural failure found while tokenising. It
// exists so an adapter can fail closed: a truncated token stream would
// otherwise silently swallow every test that follows the fault.
func (l *Lexer) scanError() error {
	return l.scanErr
}

// fail records the first lexical failure; later failures are consequences of it
// and would only obscure the root cause.
func (l *Lexer) fail(line int, reason string) {
	if l.scanErr == nil {
		l.scanErr = fmt.Errorf("line %d: %s", line, reason)
	}
}

// trackDelimiter keeps the stream structurally honest: an unclosed or
// mismatched bracket means the tokens no longer describe the source, so the
// scan is reported as failed instead of returning a partial test list.
func (l *Lexer) trackDelimiter(stack *[]string, value string, line int) {
	switch value {
	case "(", "[", "{":
		*stack = append(*stack, value)
	case ")", "]", "}":
		if len(*stack) == 0 {
			l.fail(line, fmt.Sprintf("unexpected %q", value))
			return
		}
		top := (*stack)[len(*stack)-1]
		if delimiterPairs[top] != value {
			l.fail(line, fmt.Sprintf("%q closes %q", value, top))
		}
		*stack = (*stack)[:len(*stack)-1]
	}
}

// NewLexer creates a new Lexer for the given source text without JSX support.
func NewLexer(src string) *Lexer {
	return NewLexerWithJSX(src, false)
}

// NewLexerWithJSX creates a new Lexer with optional JSX syntax support.
func NewLexerWithJSX(src string, enableJSX bool) *Lexer {
	return &Lexer{
		src:       src,
		n:         len(src),
		pos:       0,
		line:      1,
		enableJSX: enableJSX,
	}
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || (c >= '0' && c <= '9')
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// Tokenize processes the entire source and returns all non-comment tokens.
func (l *Lexer) Tokenize() []Token {
	return l.TokenizeAll(false)
}

// TokenizeAll processes the entire source, optionally including comments.
func (l *Lexer) TokenizeAll(includeComments bool) []Token {
	var tokens []Token
	var delims []string
	l.scanErr = nil
	prevToken := ""

	for l.pos < l.n {
		// 1. In JSX Children text mode (between opening tag and closing tag or child element/expression)
		if l.enableJSX && !l.inJSXTag && len(l.jsxStack) > 0 && l.jsxExprDepth == l.jsxStack[len(l.jsxStack)-1].exprDepth {
			c := l.src[l.pos]
			if c != '<' && c != '{' {
				for l.pos < l.n && l.src[l.pos] != '<' && l.src[l.pos] != '{' {
					if l.src[l.pos] == '\n' {
						l.line++
					}
					l.pos++
				}
				continue
			}
		}

		c := l.src[l.pos]

		// Whitespace
		if c == ' ' || c == '\t' || c == '\r' {
			l.pos++
			continue
		}
		if c == '\n' {
			l.line++
			l.pos++
			continue
		}

		// Single-line comment: // ...
		if c == '/' && l.pos+1 < l.n && l.src[l.pos+1] == '/' {
			start := l.pos
			startLine := l.line
			l.pos += 2
			for l.pos < l.n && l.src[l.pos] != '\n' {
				l.pos++
			}
			if includeComments {
				tokens = append(tokens, Token{
					Type:   TokenComment,
					Value:  l.src[start:l.pos],
					Offset: start,
					Line:   startLine,
				})
			}
			continue
		}

		// Block comment: /* ... */
		if c == '/' && l.pos+1 < l.n && l.src[l.pos+1] == '*' {
			start := l.pos
			startLine := l.line
			l.pos += 2
			closed := false
			for l.pos+1 < l.n {
				if l.src[l.pos] == '*' && l.src[l.pos+1] == '/' {
					closed = true
					break
				}
				if l.src[l.pos] == '\n' {
					l.line++
				}
				l.pos++
			}
			if closed {
				l.pos += 2 // skip */
			} else {
				l.pos = l.n
				l.fail(startLine, "unterminated block comment")
			}
			if includeComments {
				tokens = append(tokens, Token{
					Type:   TokenComment,
					Value:  l.src[start:l.pos],
					Offset: start,
					Line:   startLine,
				})
			}
			continue
		}

		// Regular Expression literal: /pattern/flags
		if c == '/' && isRegexStarterToken(prevToken) {
			start := l.pos
			startLine := l.line
			l.pos++ // skip leading /
			inClass := false
			closed := false

			for l.pos < l.n {
				rc := l.src[l.pos]
				if rc == '\n' {
					break // regex cannot cross unescaped newline
				}
				if rc == '\\' {
					l.pos += 2
					continue
				}
				if rc == '[' {
					inClass = true
				} else if rc == ']' {
					inClass = false
				} else if rc == '/' && !inClass {
					l.pos++ // consume closing /
					closed = true
					break
				}
				l.pos++
			}

			if closed {
				// consume trailing flags (g, i, m, s, u, y, d)
				for l.pos < l.n && isIdentPart(l.src[l.pos]) {
					l.pos++
				}
				tok := Token{
					Type:   TokenRegex,
					Value:  l.src[start:l.pos],
					Offset: start,
					Line:   startLine,
				}
				tokens = append(tokens, tok)
				prevToken = tok.Value
				continue
			}
			// Fallback: reset to start + 1 if not a valid regex
			l.pos = start
		}

		// Strings: '...' or "..."
		if c == '\'' || c == '"' {
			start := l.pos
			startLine := l.line
			end, closed := l.scanQuoted(l.pos)
			if !closed {
				l.fail(startLine, "unterminated string literal")
			}
			l.pos = end
			tok := Token{
				Type:   TokenString,
				Value:  l.src[start:l.pos],
				Offset: start,
				Line:   startLine,
			}
			tokens = append(tokens, tok)
			prevToken = tok.Value
			continue
		}

		// Template literal: `...`
		if c == '`' {
			start := l.pos
			startLine := l.line
			end, closed := l.scanTemplate(l.pos)
			if !closed {
				l.fail(startLine, "unterminated template literal")
			}
			l.pos = end
			tok := Token{
				Type:   TokenTemplate,
				Value:  l.src[start:l.pos],
				Offset: start,
				Line:   startLine,
			}
			tokens = append(tokens, tok)
			prevToken = tok.Value
			continue
		}

		// Numbers: digits or .digit
		if isDigit(c) || (c == '.' && l.pos+1 < l.n && isDigit(l.src[l.pos+1])) {
			start := l.pos
			startLine := l.line
			for l.pos < l.n && (isIdentPart(l.src[l.pos]) || l.src[l.pos] == '.') {
				l.pos++
			}
			tok := Token{
				Type:   TokenNumber,
				Value:  l.src[start:l.pos],
				Offset: start,
				Line:   startLine,
			}
			tokens = append(tokens, tok)
			prevToken = tok.Value
			continue
		}

		// Identifiers and keywords
		if isIdentStart(c) {
			start := l.pos
			startLine := l.line
			for l.pos < l.n && isIdentPart(l.src[l.pos]) {
				l.pos++
			}
			val := l.src[start:l.pos]
			tok := Token{
				Type:   TokenIdent,
				Value:  val,
				Offset: start,
				Line:   startLine,
			}
			tokens = append(tokens, tok)
			prevToken = val
			continue
		}

		// JSX constructs when enabled
		if c == '<' && l.enableJSX {
			// Closing tag: </Tag> or </>
			if len(l.jsxStack) > 0 && l.pos+1 < l.n && l.src[l.pos+1] == '/' {
				start := l.pos
				startLine := l.line
				l.pos += 2
				for l.pos < l.n && l.src[l.pos] != '>' {
					if l.src[l.pos] == '\n' {
						l.line++
					}
					l.pos++
				}
				if l.pos < l.n && l.src[l.pos] == '>' {
					l.pos++
				}
				if len(l.jsxStack) > 0 {
					l.jsxStack = l.jsxStack[:len(l.jsxStack)-1]
				}
				l.inJSXTag = false
				tok := Token{
					Type:   TokenPunct,
					Value:  l.src[start:l.pos],
					Offset: start,
					Line:   startLine,
				}
				tokens = append(tokens, tok)
				prevToken = ">"
				continue
			}

			// Fragment opener: <>
			if (len(l.jsxStack) > 0 || isExpressionStarterToken(prevToken)) && l.pos+1 < l.n && l.src[l.pos+1] == '>' {
				start := l.pos
				startLine := l.line
				l.pos += 2
				l.jsxStack = append(l.jsxStack, jsxElement{name: "", exprDepth: l.jsxExprDepth})
				l.inJSXTag = false
				tok := Token{
					Type:   TokenPunct,
					Value:  "<>",
					Offset: start,
					Line:   startLine,
				}
				tokens = append(tokens, tok)
				prevToken = ">"
				continue
			}

			// Opening tag: <Tag ...> or <Tag ... />
			if (len(l.jsxStack) > 0 || isExpressionStarterToken(prevToken)) && l.pos+1 < l.n && isIdentStart(l.src[l.pos+1]) && !isArrowGeneric(l.src, l.pos) {
				start := l.pos
				startLine := l.line
				l.pos++ // skip '<'
				tagStart := l.pos
				for l.pos < l.n && (isIdentPart(l.src[l.pos]) || l.src[l.pos] == '.' || l.src[l.pos] == '-') {
					l.pos++
				}
				tagName := l.src[tagStart:l.pos]
				l.skipJSXTypeArguments()

				// Scan inside the tag attributes until '>' or '/>'
				isSelfClosing := false
				for l.pos < l.n {
					tc := l.src[l.pos]
					if tc == ' ' || tc == '\t' || tc == '\r' {
						l.pos++
						continue
					}
					if tc == '\n' {
						l.line++
						l.pos++
						continue
					}
					if tc == '/' && l.pos+1 < l.n && l.src[l.pos+1] == '>' {
						isSelfClosing = true
						l.pos += 2
						break
					}
					if tc == '>' {
						l.pos++
						break
					}
					// String attributes
					if tc == '\'' || tc == '"' {
						q := tc
						l.pos++
						for l.pos < l.n {
							if l.src[l.pos] == '\\' {
								l.pos += 2
								continue
							}
							if l.src[l.pos] == q {
								l.pos++
								break
							}
							if l.src[l.pos] == '\n' {
								l.line++
							}
							l.pos++
						}
						continue
					}
					// Expression attributes: { ... }
					if tc == '{' {
						bDepth := 1
						l.pos++
						for l.pos < l.n && bDepth > 0 {
							nc := l.src[l.pos]
							if nc == '\n' {
								l.line++
							}
							if nc == '{' {
								bDepth++
							} else if nc == '}' {
								bDepth--
							} else if nc == '\'' || nc == '"' || nc == '`' {
								q := nc
								l.pos++
								for l.pos < l.n {
									if l.src[l.pos] == '\\' {
										l.pos += 2
										continue
									}
									if l.src[l.pos] == q {
										l.pos++
										break
									}
									if l.src[l.pos] == '\n' {
										if q == '`' {
											l.line++
										} else {
											break
										}
									}
									l.pos++
								}
								continue
							}
							l.pos++
						}
						continue
					}
					l.pos++
				}

				if !isSelfClosing {
					l.jsxStack = append(l.jsxStack, jsxElement{name: tagName, exprDepth: l.jsxExprDepth})
				}
				tok := Token{
					Type:   TokenPunct,
					Value:  l.src[start:l.pos],
					Offset: start,
					Line:   startLine,
				}
				tokens = append(tokens, tok)
				prevToken = ">"
				continue
			}
		}

		// Braces tracking for JSX expression containers
		if c == '{' {
			start := l.pos
			startLine := l.line
			l.pos++
			if l.enableJSX && len(l.jsxStack) > 0 {
				l.jsxExprDepth++
			}
			l.trackDelimiter(&delims, "{", startLine)
			tok := Token{
				Type:   TokenPunct,
				Value:  "{",
				Offset: start,
				Line:   startLine,
			}
			tokens = append(tokens, tok)
			prevToken = "{"
			continue
		}
		if c == '}' {
			start := l.pos
			startLine := l.line
			l.pos++
			if l.enableJSX && len(l.jsxStack) > 0 && l.jsxExprDepth > 0 {
				l.jsxExprDepth--
			}
			l.trackDelimiter(&delims, "}", startLine)
			tok := Token{
				Type:   TokenPunct,
				Value:  "}",
				Offset: start,
				Line:   startLine,
			}
			tokens = append(tokens, tok)
			prevToken = "}"
			continue
		}

		// Multi-character punctuation: =>, ===, !==, ==, !=, <=, >=, &&, ||, ??, ?., ...
		start := l.pos
		startLine := l.line
		l.pos++

		if start+1 < l.n {
			two := l.src[start : start+2]
			switch two {
			case "=>", "==", "!=", "<=", ">=", "&&", "||", "??", "?.", "++", "--", "+=", "-=", "*=", "/=":
				l.pos = start + 2
				if start+2 < l.n {
					three := l.src[start : start+3]
					if three == "===" || three == "!==" || three == "..." {
						l.pos = start + 3
					}
				}
			}
		}

		tok := Token{
			Type:   TokenPunct,
			Value:  l.src[start:l.pos],
			Offset: start,
			Line:   startLine,
		}
		l.trackDelimiter(&delims, tok.Value, startLine)
		tokens = append(tokens, tok)
		prevToken = tok.Value
	}

	if l.scanErr == nil && len(delims) > 0 {
		l.fail(l.line, fmt.Sprintf("unclosed %q", delims[len(delims)-1]))
	}

	return tokens
}

func isRegexStarterToken(prevToken string) bool {
	switch prevToken {
	case "", "(", ",", "=", ":", "[", "!", "&", "|", "?", "{", "}", ";",
		"return", "typeof", "case", "default", "void", "delete", "throw", "await", "yield",
		"=>", "&&", "||", "??", "==", "===", "!=", "!==", "+", "-", "*", "<", ">", "<=", ">=", "...",
		"in", "of", "else", "instanceof":
		return true
	default:
		return false
	}
}

// scanQuoted consumes a single- or double-quoted string starting at pos and
// returns the offset just past its closing quote, plus whether it closed. A
// bare newline ends the scan because ordinary strings cannot cross one.
func (l *Lexer) scanQuoted(pos int) (int, bool) {
	quote := l.src[pos]
	for i := pos + 1; i < l.n; {
		switch c := l.src[i]; c {
		case '\\':
			if i+1 < l.n && l.src[i+1] == '\n' {
				l.line++
			}
			i += 2
		case quote:
			return i + 1, true
		case '\n':
			return i, false
		default:
			i++
		}
	}
	return l.n, false
}

// scanTemplate consumes a template literal starting at pos, which must hold a
// backtick. Interpolations are scanned recursively, so a template nested inside
// `${...}` and braces or backticks inside nested strings cannot end the literal
// early.
func (l *Lexer) scanTemplate(pos int) (int, bool) {
	for i := pos + 1; i < l.n; {
		switch c := l.src[i]; c {
		case '\\':
			if i+1 < l.n && l.src[i+1] == '\n' {
				l.line++
			}
			i += 2
		case '`':
			return i + 1, true
		case '$':
			if i+1 < l.n && l.src[i+1] == '{' {
				end, closed := l.scanInterpolation(i + 2)
				if !closed {
					return end, false
				}
				i = end
				continue
			}
			i++
		case '\n':
			l.line++
			i++
		default:
			i++
		}
	}
	return l.n, false
}

// scanInterpolation consumes the expression of a `${` interpolation, starting
// just past its opening brace, and returns the offset just past the matching
// closing brace. Regex literals inside an interpolation are treated as plain
// text; a brace inside one therefore surfaces as a lexical error instead of a
// silently truncated scan.
func (l *Lexer) scanInterpolation(pos int) (int, bool) {
	depth := 1
	for i := pos; i < l.n; {
		switch c := l.src[i]; c {
		case '{':
			depth++
			i++
		case '}':
			depth--
			i++
			if depth == 0 {
				return i, true
			}
		case '`':
			end, closed := l.scanTemplate(i)
			if !closed {
				return end, false
			}
			i = end
		case '\'', '"':
			end, closed := l.scanQuoted(i)
			if !closed {
				return end, false
			}
			i = end
		case '/':
			if i+1 < l.n && l.src[i+1] == '/' {
				for i < l.n && l.src[i] != '\n' {
					i++
				}
				continue
			}
			if i+1 < l.n && l.src[i+1] == '*' {
				i += 2
				for i+1 < l.n && !(l.src[i] == '*' && l.src[i+1] == '/') {
					if l.src[i] == '\n' {
						l.line++
					}
					i++
				}
				if i+1 < l.n {
					i += 2
				} else {
					i = l.n
				}
				continue
			}
			i++
		case '\n':
			l.line++
			i++
		default:
			i++
		}
	}
	return l.n, false
}

func isExpressionStarterToken(prevToken string) bool {
	switch prevToken {
	case "", "(", ",", "=", ":", "?", "[", "{", "}", ";", "=>", "&&", "||", "??", "!",
		"return", "yield", "default", "await", "case":
		return true
	default:
		return false
	}
}

func isArrowGeneric(src string, pos int) bool {
	i := pos + 1
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\r' || src[i] == '\n') {
		i++
	}
	if i >= len(src) || !isIdentStart(src[i]) {
		return false
	}
	for i < len(src) && isIdentPart(src[i]) {
		i++
	}
	for i < len(src) && (src[i] == ' ' || src[i] == '\t' || src[i] == '\r' || src[i] == '\n') {
		i++
	}
	if i >= len(src) {
		return false
	}
	if src[i] == ',' || src[i] == '=' {
		return true
	}
	if strings.HasPrefix(src[i:], "extends") {
		after := i + len("extends")
		if after < len(src) && (src[after] == ' ' || src[after] == '\t' || src[after] == '\r' || src[after] == '\n' || src[after] == '{') {
			return true
		}
	}
	return false
}

// skipJSXTypeArguments consumes a balanced type-argument list that follows a JSX
// element name (`<Table<{ n: number }> rows={...} />`). It tracks angle depth
// only outside (), [] and {} so object, function and tuple types may contain
// `>` freely, and it skips strings, template literals and comments. The `>` of
// `=>` never closes the list. An unterminated list leaves the lexer untouched,
// so malformed input still reports the usual delimiter error.
func (l *Lexer) skipJSXTypeArguments() {
	if l.pos >= l.n || l.src[l.pos] != '<' {
		return
	}
	pos, line := l.pos, l.line
	angle, nest := 0, 0
	for pos < l.n {
		c := l.src[pos]
		switch {
		case c == '\n':
			line++
			pos++
		case c == '\'' || c == '"' || c == '`':
			quote := c
			pos++
			for pos < l.n && l.src[pos] != quote {
				if l.src[pos] == '\\' {
					pos++
				}
				if pos < l.n && l.src[pos] == '\n' {
					line++
				}
				pos++
			}
			if pos >= l.n {
				return
			}
			pos++
		case c == '/' && pos+1 < l.n && l.src[pos+1] == '/':
			for pos < l.n && l.src[pos] != '\n' {
				pos++
			}
		case c == '/' && pos+1 < l.n && l.src[pos+1] == '*':
			end := strings.Index(l.src[pos+2:], "*/")
			if end < 0 {
				return
			}
			line += strings.Count(l.src[pos:pos+2+end], "\n")
			pos += end + 4
		case c == '=' && pos+1 < l.n && l.src[pos+1] == '>':
			pos += 2
		case c == '(' || c == '[' || c == '{':
			nest++
			pos++
		case c == ')' || c == ']' || c == '}':
			nest--
			if nest < 0 {
				return
			}
			pos++
		case c == '<' && nest == 0:
			angle++
			pos++
		case c == '>' && nest == 0:
			angle--
			pos++
			if angle == 0 {
				l.pos, l.line = pos, line
				return
			}
		default:
			pos++
		}
	}
}
