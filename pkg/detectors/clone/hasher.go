package clone

import (
	"crypto/sha256"
	"fmt"
	goscanner "go/scanner"
	gotoken "go/token"
	"path/filepath"
	"sort"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

type tokenKind int

const (
	tokOther tokenKind = iota
	tokString
	tokNumber
	tokNewline
)

type token struct {
	kind tokenKind
	text string
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func isIdentStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || c == '$'
}

func isIdentPart(c byte) bool {
	return isIdentStart(c) || isDigit(c)
}

// genericOperators lists multi-character operators recognised by the
// language-neutral lexer so that "a + ++b" and "a++ + b" never collapse to the
// same token stream. Ordered longest first.
var genericOperators = []string{
	">>>=", "<<=", ">>=", "&^=", ">>>", "...", "??=", "===", "!==", "**=",
	"&^", "??", "?.", "=>", "->", "::", "++", "--", "+=", "-=", "*=", "/=",
	"%=", "&=", "|=", "^=", "==", "!=", "<=", ">=", "&&", "||", "<<", ">>",
	"..", "!!",
}

// tokenizeGenericCode lexes a body from a non-Go supported language. It strips
// comments while preserving literal contents verbatim, records line breaks as
// newline tokens (statement boundaries such as JavaScript ASI, including line
// terminators carried by block comments), and recognises multi-character
// operators. It makes no claim to a full language parse.
func tokenizeGenericCode(code string) []token {
	var tokens []token
	n := len(code)
	i := 0

	for i < n {
		c := code[i]

		// Line breaks are execution-relevant in supported languages (ASI).
		if c == '\n' {
			tokens = append(tokens, token{kind: tokNewline, text: "\n"})
			i++
			continue
		}
		if c == ' ' || c == '\t' || c == '\r' {
			i++
			continue
		}

		// Single-line comment: // ...
		if c == '/' && i+1 < n && code[i+1] == '/' {
			i += 2
			for i < n && code[i] != '\n' {
				i++
			}
			continue
		}

		// Multi-line block comment: /* ... */
		if c == '/' && i+1 < n && code[i+1] == '*' {
			i += 2
			for i+1 < n && !(code[i] == '*' && code[i+1] == '/') {
				// A line terminator inside a block comment still participates
				// in automatic semicolon insertion in newline-sensitive
				// languages, so it must remain visible to exact comparison.
				if code[i] == '\n' {
					tokens = append(tokens, token{kind: tokNewline, text: "\n"})
				}
				i++
			}
			if i+1 < n {
				i += 2
			}
			continue
		}

		// Raw string in Dart: r'...' or r"..."
		if (c == 'r' || c == 'R') && i+1 < n && (code[i+1] == '\'' || code[i+1] == '"') {
			quote := code[i+1]
			start := i
			i += 2
			for i < n && code[i] != quote && code[i] != '\n' {
				i++
			}
			if i < n && code[i] == quote {
				i++
			}
			tokens = append(tokens, token{kind: tokString, text: code[start:i]})
			continue
		}

		// Raw string: `...`
		if c == '`' {
			start := i
			i++
			for i < n && code[i] != '`' {
				i++
			}
			if i < n && code[i] == '`' {
				i++
			}
			tokens = append(tokens, token{kind: tokString, text: code[start:i]})
			continue
		}

		// Triple-quoted string: """...""" or '''...'''
		if (c == '\'' || c == '"') && i+2 < n && code[i+1] == c && code[i+2] == c {
			quote := c
			start := i
			i += 3
			for i+2 < n && !(code[i] == quote && code[i+1] == quote && code[i+2] == quote) {
				if code[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				i++
			}
			if i+2 < n {
				i += 3
			}
			tokens = append(tokens, token{kind: tokString, text: code[start:i]})
			continue
		}

		// Standard quoted string: '...' or "..."
		if c == '\'' || c == '"' {
			quote := c
			start := i
			i++
			for i < n && code[i] != quote {
				if code[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				if code[i] == '\n' {
					break
				}
				i++
			}
			if i < n && code[i] == quote {
				i++
			}
			tokens = append(tokens, token{kind: tokString, text: code[start:i]})
			continue
		}

		// Numbers (sign handling is resolved during normalization).
		if isDigit(c) {
			start := i
			i++
			for i < n && (isDigit(code[i]) || code[i] == '.' || code[i] == 'e' || code[i] == 'E' ||
				((code[i] == '+' || code[i] == '-') && (code[i-1] == 'e' || code[i-1] == 'E'))) {
				i++
			}
			tokens = append(tokens, token{kind: tokNumber, text: code[start:i]})
			continue
		}

		// Identifiers and keywords
		if isIdentStart(c) {
			start := i
			i++
			for i < n && isIdentPart(code[i]) {
				i++
			}
			tokens = append(tokens, token{kind: tokOther, text: code[start:i]})
			continue
		}

		// Multi-character operators
		matched := false
		for _, op := range genericOperators {
			if strings.HasPrefix(code[i:], op) {
				tokens = append(tokens, token{kind: tokOther, text: op})
				i += len(op)
				matched = true
				break
			}
		}
		if matched {
			continue
		}

		// Other operators, punctuation, single characters
		tokens = append(tokens, token{kind: tokOther, text: string(c)})
		i++
	}

	return tokens
}

// tokenizeGoCode lexes a Go body with the standard library scanner. Comments
// are skipped, literal values are preserved byte for byte, and go/scanner's
// semicolon insertion provides lexical statement boundaries.
func tokenizeGoCode(code string) []token {
	var tokens []token
	fset := gotoken.NewFileSet()
	file := fset.AddFile("", fset.Base(), len(code))
	var scanner goscanner.Scanner
	scanner.Init(file, []byte(code), nil, 0)

	for {
		_, tok, lit := scanner.Scan()
		if tok == gotoken.EOF {
			break
		}
		switch tok {
		case gotoken.STRING, gotoken.CHAR:
			tokens = append(tokens, token{kind: tokString, text: lit})
		case gotoken.INT, gotoken.FLOAT, gotoken.IMAG:
			tokens = append(tokens, token{kind: tokNumber, text: lit})
		case gotoken.SEMICOLON:
			// go/scanner reports an implicit newline semicolon as lit == "\n";
			// both explicit and implicit boundaries normalize to ";".
			tokens = append(tokens, token{kind: tokOther, text: ";"})
		default:
			text := lit
			if text == "" {
				text = tok.String()
			}
			tokens = append(tokens, token{kind: tokOther, text: text})
		}
	}

	return tokens
}

// isGoSource reports whether filePath uses Go's lexical rules.
func isGoSource(filePath string) bool {
	return strings.EqualFold(filepath.Ext(filePath), ".go")
}

// tokensForCode lexes a test body with the lexical rules of its language.
func tokensForCode(code, filePath string) []token {
	if isGoSource(filePath) {
		return tokenizeGoCode(code)
	}
	return tokenizeGenericCode(code)
}

// isUnaryBoundary reports whether the previous normalized token allows a
// following "-" or "+" to be read as part of a numeric literal.
func isUnaryBoundary(previous string) bool {
	switch previous {
	case "", "(", "[", "{", ",", ";", ":", "=", ":=", "==", "!=", "<", ">",
		"<=", ">=", "+", "-", "*", "/", "%", "&&", "||", "!", "&", "|", "^",
		"<<", ">>", "=>", "?", "??", "return", "case", "when", "throw", "yield",
		"in", "is", "as":
		return true
	}
	return false
}

// normalizedTokens collapses literal values for structural comparison while
// preserving operator identity. Line breaks are excluded here but remain
// visible to exact-token comparison.
func normalizedTokens(tokens []token) []string {
	parts := make([]string, 0, len(tokens))
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok.kind == tokNewline {
			continue
		}
		// Signed numeric literals: "Calc(-1)" and "Calc(- 1)" normalize alike,
		// while spacing never turns binary subtraction into a sign.
		if tok.kind == tokOther && (tok.text == "-" || tok.text == "+") &&
			i+1 < len(tokens) && tokens[i+1].kind == tokNumber {
			previous := ""
			if len(parts) > 0 {
				previous = parts[len(parts)-1]
			}
			if isUnaryBoundary(previous) {
				parts = append(parts, "_NUM_")
				i++
				continue
			}
		}
		switch tok.kind {
		case tokString:
			parts = append(parts, "_LIT_")
		case tokNumber:
			parts = append(parts, "_NUM_")
		default:
			parts = append(parts, tok.text)
		}
	}
	return parts
}

// hashTokens produces a SHA256 hex string over the normalized token stream.
func hashTokens(tokens []token) string {
	norm := strings.Join(normalizedTokens(tokens), " ")
	sum := sha256.Sum256([]byte(norm))
	return fmt.Sprintf("%x", sum)
}

// isExactTokens returns true if two token streams have identical tokens and
// literal values, including statement line breaks.
func isExactTokens(t1, t2 []token) bool {
	if len(t1) != len(t2) {
		return false
	}
	for i := range t1 {
		if t1[i].kind != t2[i].kind || t1[i].text != t2[i].text {
			return false
		}
	}
	return true
}

// meaningfulTokenCount counts tokens that carry executable structure.
func meaningfulTokenCount(tokens []token) int {
	count := 0
	for _, tok := range tokens {
		if tok.kind != tokNewline {
			count++
		}
	}
	return count
}

// Detector identifies duplicate tests and structural repetition.
type Detector struct{}

// New creates a new clone detector.
func New() *Detector {
	return &Detector{}
}

type analyzedUnit struct {
	unit   parser.TestUnit
	tokens []token
}

// Check compares test units within the same file for duplicate execution bodies.
func (d *Detector) Check(units []parser.TestUnit) []models.Violation {
	var violations []models.Violation
	byHash := make(map[string][]analyzedUnit)

	for _, unit := range units {
		if unit.Kind == "file" {
			continue
		}
		body := strings.TrimSpace(unit.RawBody)
		if body == "" {
			continue
		}
		tokens := tokensForCode(body, unit.FilePath)
		if meaningfulTokenCount(tokens) < 3 {
			continue
		}
		h := hashTokens(tokens)
		byHash[h] = append(byHash[h], analyzedUnit{unit: unit, tokens: tokens})
	}

	hashes := make([]string, 0, len(byHash))
	for h := range byHash {
		hashes = append(hashes, h)
	}
	sort.Strings(hashes)

	for _, h := range hashes {
		group := byHash[h]
		if len(group) < 2 {
			continue
		}

		// 1. Partition the group into exact-token subgroups.
		var exactSubgroups [][]analyzedUnit
		for _, u := range group {
			found := false
			for i := range exactSubgroups {
				if isExactTokens(exactSubgroups[i][0].tokens, u.tokens) {
					exactSubgroups[i] = append(exactSubgroups[i], u)
					found = true
					break
				}
			}
			if !found {
				exactSubgroups = append(exactSubgroups, []analyzedUnit{u})
			}
		}

		var groupViolations []models.Violation

		// 2. For each exact subgroup of size >= 2, emit GUARD-UNIV-01 Duplicate test body for every member after the first.
		for _, subgroup := range exactSubgroups {
			if len(subgroup) >= 2 {
				first := subgroup[0]
				for j := 1; j < len(subgroup); j++ {
					member := subgroup[j]
					groupViolations = append(groupViolations, models.Violation{
						RuleID:     "GUARD-UNIV-01",
						Severity:   models.SeverityAdvisory,
						FilePath:   member.unit.FilePath,
						LineNumber: member.unit.StartLine,
						Column:     1,
						Message:    fmt.Sprintf("Duplicate test body: '%s' has identical execution tokens to '%s' (line %d).", member.unit.Name, first.unit.Name, first.unit.StartLine),
						Suggestion: "Consolidate identical duplicate tests or verify whether distinct inputs were intended.",
					})
				}
			}
		}

		// 3. Members of the normalized group that are NOT exact duplicates of anything.
		// Count only distinct exact-variants: len(exactSubgroups).
		numVariants := len(exactSubgroups)
		firstUnit := group[0]
		if numVariants >= 2 && numVariants <= 3 {
			for i := 1; i < numVariants; i++ {
				variantRep := exactSubgroups[i][0]
				groupViolations = append(groupViolations, models.Violation{
					RuleID:     "GUARD-UNIV-01",
					Severity:   models.SeverityAdvisory,
					FilePath:   variantRep.unit.FilePath,
					LineNumber: variantRep.unit.StartLine,
					Column:     1,
					Message:    fmt.Sprintf("Similar test structure: '%s' shares normalized token structure with '%s' (line %d). Note: literal normalization indicates structural similarity, not proven semantic redundancy.", variantRep.unit.Name, firstUnit.unit.Name, firstUnit.unit.StartLine),
					Suggestion: "Check whether this test exercises a distinct edge case or boundary condition before consolidating. Do not treat syntactic clone similarity as proof of semantic redundancy or delete cases.",
				})
			}
		} else if numVariants >= 4 {
			for i := 3; i < numVariants; i++ {
				variantRep := exactSubgroups[i][0]
				groupViolations = append(groupViolations, models.Violation{
					RuleID:     "GUARD-UNIV-02",
					Severity:   models.SeverityAdvisory,
					FilePath:   variantRep.unit.FilePath,
					LineNumber: variantRep.unit.StartLine,
					Column:     1,
					Message:    fmt.Sprintf("Structural repetition: '%s' shares normalized token structure with %d other tests in this file (first seen at '%s', line %d). Note: structural similarity does not prove semantic redundancy.", variantRep.unit.Name, len(group)-1, firstUnit.unit.Name, firstUnit.unit.StartLine),
					Suggestion: "Consolidate repetitive test structures into a parameterized table-driven test loop without removing valid boundary conditions or distinct equivalence classes.",
				})
			}
		}

		sort.Slice(groupViolations, func(i, j int) bool {
			return groupViolations[i].LineNumber < groupViolations[j].LineNumber
		})
		violations = append(violations, groupViolations...)
	}

	return violations
}
