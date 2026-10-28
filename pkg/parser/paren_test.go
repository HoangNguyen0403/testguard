package parser_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

func TestFindMatchingParen(t *testing.T) {
	src := `foo('hello )', (1 + 2) /* ) */, "str \" )")`
	openIdx := 3
	closeIdx := parser.FindMatchingParen(src, openIdx)
	if closeIdx != len(src)-1 {
		t.Fatalf("expected closeIdx=%d, got %d", len(src)-1, closeIdx)
	}
}
