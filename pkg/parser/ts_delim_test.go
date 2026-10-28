package parser_test

import (
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

func TestFindMatchingTSParen_RegexLiteral(t *testing.T) {
	src := `it('asserting regex', () => { const pattern = /\)/; expect(pattern.test(')')).toBe(true); })`
	openIdx := strings.Index(src, "(")
	if openIdx == -1 {
		t.Fatal("open paren not found")
	}

	closeIdx := parser.FindMatchingTSParen(src, openIdx)
	expectedClose := len(src) - 1
	if closeIdx != expectedClose {
		t.Fatalf("expected close paren at %d (%q), got %d (%q)",
			expectedClose, src[expectedClose:], closeIdx, src[closeIdx:])
	}
}

func TestFindMatchingTSParen_CharClassAndEscapes(t *testing.T) {
	src := `test('regex class', () => { const r = /[/)]/; const s = "hello"; })`
	openIdx := strings.Index(src, "(")
	closeIdx := parser.FindMatchingTSParen(src, openIdx)
	expectedClose := len(src) - 1
	if closeIdx != expectedClose {
		t.Fatalf("expected close paren at %d, got %d", expectedClose, closeIdx)
	}
}

func TestFindMatchingTSParen_DivisionNotRegex(t *testing.T) {
	src := `call((a / b) + c)`
	openIdx := strings.Index(src, "(")
	closeIdx := parser.FindMatchingTSParen(src, openIdx)
	expectedClose := len(src) - 1
	if closeIdx != expectedClose {
		t.Fatalf("expected close paren at %d, got %d", expectedClose, closeIdx)
	}
}
