package clone_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/detectors/clone"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/parser/golang"
)

func TestDetectExactClones(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test1",
		FilePath:  "auth_test.dart",
		StartLine: 10,
		RawBody:   "expect(auth.login('foo', 'bar'), equals(true));",
	}

	test2 := parser.TestUnit{
		Name:      "test2",
		FilePath:  "auth_test.dart",
		StartLine: 25,
		RawBody:   "expect(auth.login('baz', 'qux'), equals(true));",
	}

	detector := clone.New()
	violations := detector.Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 clone violation, got %d", len(violations))
	}

	if violations[0].RuleID != "GUARD-UNIV-01" {
		t.Errorf("expected GUARD-UNIV-01, got %s", violations[0].RuleID)
	}
}

func TestDetectExactClones_DistinctBoundaryValuesMessage(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_zero_boundary",
		FilePath:  "calc_test.go",
		StartLine: 10,
		RawBody:   "res := Calc(0); assert.Equal(t, 0, res);",
	}

	test2 := parser.TestUnit{
		Name:      "test_negative_boundary",
		FilePath:  "calc_test.go",
		StartLine: 20,
		RawBody:   "res := Calc(-1); assert.Equal(t, 0, res);",
	}

	detector := clone.New()
	violations := detector.Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 advisory violation for structural similarity, got %d", len(violations))
	}
	v := violations[0]
	if !strings.Contains(v.Message, "not proven semantic redundancy") {
		t.Errorf("expected message to note that literal normalization does not prove semantic redundancy, got: %s", v.Message)
	}
	if !strings.Contains(v.Suggestion, "Do not treat syntactic clone similarity as proof of semantic redundancy or delete cases") {
		t.Errorf("expected suggestion to caution against deleting cases, got: %s", v.Suggestion)
	}
}

func TestDetectExactClones_WhitespaceInsideStringsNotExact(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_double_space",
		FilePath:  "format_test.go",
		StartLine: 10,
		RawBody:   `assert.Equal(t, "a  b", actual);`,
	}

	test2 := parser.TestUnit{
		Name:      "test_single_space",
		FilePath:  "format_test.go",
		StartLine: 20,
		RawBody:   `assert.Equal(t, "a b", actual);`,
	}

	detector := clone.New()
	violations := detector.Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 advisory violation for structural similarity, got %d", len(violations))
	}
	v := violations[0]
	if strings.Contains(v.Message, "identical execution tokens") {
		t.Errorf("expected structural similarity message, not identical execution tokens: %s", v.Message)
	}
	if !strings.Contains(v.Message, "not proven semantic redundancy") {
		t.Errorf("expected structural similarity message, got: %s", v.Message)
	}
}

func TestDetectExactClones_CommentsDifferExact(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_comment_1",
		FilePath:  "math_test.dart",
		StartLine: 10,
		RawBody: `
		// Arrange and Act
		expect(calculator.add(10, 20), equals(30));
		`,
	}

	test2 := parser.TestUnit{
		Name:      "test_comment_2",
		FilePath:  "math_test.dart",
		StartLine: 25,
		RawBody: `
		/* Different comment style */
		expect(calculator.add(10, 20), equals(30));
		`,
	}

	detector := clone.New()
	violations := detector.Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 exact clone violation, got %d", len(violations))
	}
	if !strings.Contains(violations[0].Message, "identical execution tokens") {
		t.Errorf("expected identical execution tokens message, got: %s", violations[0].Message)
	}
}

func TestDetectExactClonesFromParsedGoTests(t *testing.T) {
	source := []byte(`package service

import "testing"

func TestAlpha(t *testing.T) {
	x := 42
	if x != 42 {
		t.Fatal("fail")
	}
}

func TestBeta(t *testing.T) {
	x := 42
	if x != 42 {
		t.Fatal("fail")
	}
}
`)
	adapter := golang.NewAdapter()
	units, err := adapter.LocateTests(source, "service_test.go")
	if err != nil {
		t.Fatalf("unexpected adapter error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 located Go tests, got %d", len(units))
	}

	detector := clone.New()
	violations := detector.Check(units)

	if len(violations) != 1 {
		t.Fatalf("expected 1 exact clone violation despite different test names, got %d (%+v)", len(violations), violations)
	}
	if violations[0].RuleID != "GUARD-UNIV-01" {
		t.Errorf("expected GUARD-UNIV-01, got %s", violations[0].RuleID)
	}
	if !strings.Contains(violations[0].Message, "identical execution tokens") {
		t.Errorf("expected identical execution tokens, got: %s", violations[0].Message)
	}
}

func TestDetectExactClonesFromParsedGoTestsDistinctLiterals(t *testing.T) {
	source := []byte(`package service

import "testing"

func TestAlpha(t *testing.T) {
	x := 42
	if x != 42 {
		t.Fatal("fail")
	}
}

func TestBeta(t *testing.T) {
	x := 7
	if x != 7 {
		t.Fatal("fail")
	}
}
`)
	adapter := golang.NewAdapter()
	units, err := adapter.LocateTests(source, "service_test.go")
	if err != nil {
		t.Fatalf("unexpected adapter error: %v", err)
	}

	violations := clone.New().Check(units)

	if len(violations) != 1 {
		t.Fatalf("expected 1 structural similarity advisory, got %d (%+v)", len(violations), violations)
	}
	if strings.Contains(violations[0].Message, "identical execution tokens") {
		t.Errorf("distinct literals must not claim identical execution tokens: %s", violations[0].Message)
	}
	if !strings.Contains(violations[0].Message, "not proven semantic redundancy") {
		t.Errorf("expected structural similarity message, got: %s", violations[0].Message)
	}
}

func TestDetectStructuralRepetition(t *testing.T) {
	var tests []parser.TestUnit
	for i := 1; i <= 5; i++ {
		tests = append(tests, parser.TestUnit{
			Name:      "repetition_case",
			FilePath:  "pricing_test.go",
			StartLine: i * 10,
			RawBody:   fmt.Sprintf("var input = %d; result := Calc(input); assert.Equal(t, %d, result);", i*100, i*100),
		})
	}

	detector := clone.New()
	violations := detector.Check(tests)

	hasRepetition := false
	for _, v := range violations {
		if v.RuleID == "GUARD-UNIV-02" {
			hasRepetition = true
			if strings.Contains(v.Message, "AST structure") {
				t.Errorf("message should not claim AST structure: %s", v.Message)
			}
		}
	}

	if !hasRepetition {
		t.Errorf("expected GUARD-UNIV-02 to be triggered for 5 repetitive tests")
	}
}

func TestDetectExactClones_DartIncrementOperatorIdentity(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_prefix_increment",
		FilePath:  "counter_test.dart",
		StartLine: 10,
		RawBody:   "expect(a + ++b, equals(2));",
	}

	test2 := parser.TestUnit{
		Name:      "test_postfix_increment",
		FilePath:  "counter_test.dart",
		StartLine: 20,
		RawBody:   "expect(a++ + b, equals(2));",
	}

	violations := clone.New().Check([]parser.TestUnit{test1, test2})

	if len(violations) != 0 {
		t.Fatalf("operator-distinct bodies must not be reported as clones, got %d (%+v)", len(violations), violations)
	}
}

func TestDetectExactClones_JavaScriptASINotExact(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_return_newline",
		FilePath:  "flow_test.js",
		StartLine: 10,
		RawBody:   "return\nvalue;",
	}

	test2 := parser.TestUnit{
		Name:      "test_return_inline",
		FilePath:  "flow_test.js",
		StartLine: 20,
		RawBody:   "return value;",
	}

	violations := clone.New().Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 structural advisory for ASI-sensitive bodies, got %d (%+v)", len(violations), violations)
	}
	if strings.Contains(violations[0].Message, "identical execution tokens") {
		t.Errorf("newline-sensitive bodies must not claim identical execution tokens: %s", violations[0].Message)
	}
	if !strings.Contains(violations[0].Message, "not proven semantic redundancy") {
		t.Errorf("expected structural similarity message, got: %s", violations[0].Message)
	}
}

func TestDetectExactClones_SignedBoundarySpacingInsensitive(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_spaced_negative_boundary",
		FilePath:  "calc_test.go",
		StartLine: 10,
		RawBody:   "res := Calc(- 1); assert.Equal(t, 0, res);",
	}

	test2 := parser.TestUnit{
		Name:      "test_zero_boundary",
		FilePath:  "calc_test.go",
		StartLine: 20,
		RawBody:   "res := Calc(0); assert.Equal(t, 0, res);",
	}

	violations := clone.New().Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 structural advisory regardless of sign spacing, got %d (%+v)", len(violations), violations)
	}
	if strings.Contains(violations[0].Message, "identical execution tokens") {
		t.Errorf("distinct boundary values must not claim identical execution tokens: %s", violations[0].Message)
	}
}

func TestDetectExactClones_RawAndEscapedLiteralsNotExact(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_raw_string",
		FilePath:  "literal_test.go",
		StartLine: 10,
		RawBody:   "assert.Equal(t, `line1\\nline2`, actual)",
	}

	test2 := parser.TestUnit{
		Name:      "test_escaped_string",
		FilePath:  "literal_test.go",
		StartLine: 20,
		RawBody:   "assert.Equal(t, \"line1\\nline2\", actual)",
	}

	violations := clone.New().Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 structural advisory for literal variants, got %d (%+v)", len(violations), violations)
	}
	if strings.Contains(violations[0].Message, "identical execution tokens") {
		t.Errorf("distinct literal bytes must not claim identical execution tokens: %s", violations[0].Message)
	}
}

func TestDetectExactClones_GoCommentsDoNotChangeExactness(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_line_comment",
		FilePath:  "calc_test.go",
		StartLine: 10,
		RawBody:   "// arrange\nres := Calc(2)\nassert.Equal(t, 4, res)",
	}

	test2 := parser.TestUnit{
		Name:      "test_block_comment",
		FilePath:  "calc_test.go",
		StartLine: 20,
		RawBody:   "/* arrange */\nres := Calc(2)\nassert.Equal(t, 4, res)",
	}

	violations := clone.New().Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 exact clone violation, got %d (%+v)", len(violations), violations)
	}
	if !strings.Contains(violations[0].Message, "identical execution tokens") {
		t.Errorf("comments must not change executable tokens, got: %s", violations[0].Message)
	}
}

func TestDetectExactClones_JavaScriptBlockCommentNewlineNotExact(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_return_comment_newline",
		FilePath:  "flow_test.js",
		StartLine: 10,
		RawBody:   "return /*\n*/ value;",
	}

	test2 := parser.TestUnit{
		Name:      "test_return_inline",
		FilePath:  "flow_test.js",
		StartLine: 20,
		RawBody:   "return value;",
	}

	violations := clone.New().Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 structural advisory for comment-carried ASI, got %d (%+v)", len(violations), violations)
	}
	if strings.Contains(violations[0].Message, "identical execution tokens") {
		t.Errorf("a line terminator inside a block comment must not allow an exact claim: %s", violations[0].Message)
	}
	if !strings.Contains(violations[0].Message, "not proven semantic redundancy") {
		t.Errorf("expected structural similarity message, got: %s", violations[0].Message)
	}
}

func TestDetectExactClones_JavaScriptInlineBlockCommentStillExact(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_inline_comment",
		FilePath:  "flow_test.js",
		StartLine: 10,
		RawBody:   "return /* inline */ value;",
	}

	test2 := parser.TestUnit{
		Name:      "test_no_comment",
		FilePath:  "flow_test.js",
		StartLine: 20,
		RawBody:   "return value;",
	}

	violations := clone.New().Check([]parser.TestUnit{test1, test2})

	if len(violations) != 1 {
		t.Fatalf("expected 1 exact clone violation, got %d (%+v)", len(violations), violations)
	}
	if !strings.Contains(violations[0].Message, "identical execution tokens") {
		t.Errorf("single-line comment without a line terminator must remain execution-identical, got: %s", violations[0].Message)
	}
}

func TestDetectExactClones_StructurallyDifferentControl(t *testing.T) {
	test1 := parser.TestUnit{
		Name:      "test_positive_path",
		FilePath:  "calc_test.go",
		StartLine: 10,
		RawBody:   "res := Calc(3)\nassert.Equal(t, 6, res)",
	}

	test2 := parser.TestUnit{
		Name:      "test_negative_path",
		FilePath:  "calc_test.go",
		StartLine: 20,
		RawBody:   "res := Calc(4)\nassert.NotEqual(t, 6, res)",
	}

	violations := clone.New().Check([]parser.TestUnit{test1, test2})

	if len(violations) != 0 {
		t.Errorf("structurally different bodies must not be reported as clones, got %d (%+v)", len(violations), violations)
	}
}

func TestCloneRouting_FiveByteIdenticalBodies(t *testing.T) {
	var tests []parser.TestUnit
	for i := 1; i <= 5; i++ {
		tests = append(tests, parser.TestUnit{
			Name:      fmt.Sprintf("test_%d", i),
			FilePath:  "pricing_test.go",
			StartLine: i * 10,
			RawBody:   "renderHeader(); expect(logo);",
		})
	}

	violations := clone.New().Check(tests)
	duplicateCount := 0
	repetitionCount := 0
	for _, v := range violations {
		if v.RuleID == "GUARD-UNIV-01" && strings.Contains(v.Message, "Duplicate test body") {
			duplicateCount++
		}
		if v.RuleID == "GUARD-UNIV-02" {
			repetitionCount++
		}
	}

	if duplicateCount != 4 {
		t.Errorf("expected 4 Duplicate test body violations, got %d", duplicateCount)
	}
	if repetitionCount != 0 {
		t.Errorf("expected 0 UNIV-02 repetition violations, got %d", repetitionCount)
	}
}

func TestCloneRouting_FourBodiesDifferingOnlyInStringLiteral(t *testing.T) {
	var tests []parser.TestUnit
	titles := []string{"apple", "banana", "cherry", "date"}
	for i, title := range titles {
		tests = append(tests, parser.TestUnit{
			Name:      fmt.Sprintf("test_%s", title),
			FilePath:  "fruit_test.go",
			StartLine: (i + 1) * 10,
			RawBody:   fmt.Sprintf(`renderHeader("%s"); expect(logo);`, title),
		})
	}

	violations := clone.New().Check(tests)
	univ02Count := 0
	for _, v := range violations {
		if v.RuleID == "GUARD-UNIV-02" {
			univ02Count++
			if v.LineNumber != 40 {
				t.Errorf("expected UNIV-02 on 4th test (line 40), got line %d", v.LineNumber)
			}
		}
	}

	if univ02Count != 1 {
		t.Errorf("expected 1 UNIV-02 violation on the 4th, got %d", univ02Count)
	}
}

func TestCloneRouting_TwoIdenticalPlusOneLiteralVariant(t *testing.T) {
	tests := []parser.TestUnit{
		{
			Name:      "test_a",
			FilePath:  "auth_test.go",
			StartLine: 10,
			RawBody:   `res := Login("admin", "secret"); assert.True(t, res);`,
		},
		{
			Name:      "test_b",
			FilePath:  "auth_test.go",
			StartLine: 20,
			RawBody:   `res := Login("admin", "secret"); assert.True(t, res);`,
		},
		{
			Name:      "test_c",
			FilePath:  "auth_test.go",
			StartLine: 30,
			RawBody:   `res := Login("guest", "secret"); assert.True(t, res);`,
		},
	}

	violations := clone.New().Check(tests)
	duplicateCount := 0
	similarCount := 0
	for _, v := range violations {
		if v.RuleID == "GUARD-UNIV-01" {
			if strings.Contains(v.Message, "Duplicate test body") {
				duplicateCount++
			} else if strings.Contains(v.Message, "Similar test structure") {
				similarCount++
			}
		}
	}

	if duplicateCount != 1 {
		t.Errorf("expected 1 Duplicate test body violation, got %d", duplicateCount)
	}
	if similarCount != 1 {
		t.Errorf("expected 1 Similar test structure violation, got %d", similarCount)
	}
}
