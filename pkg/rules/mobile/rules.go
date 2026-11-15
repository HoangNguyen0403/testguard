// Package mobile provides lint rules for Kotlin and Swift mobile test files.
package mobile

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	// swiftTimeoutRegex matches wait(for: ..., timeout: N) or fulfillment(of: ..., timeout: N)
	swiftTimeoutRegex = regexp.MustCompile(`(?:wait\s*\(\s*for:\s*\[[^\]]+\],\s*timeout:\s*(\d+(?:\.\d+)?)|fulfillment\s*\(\s*of:\s*\[[^\]]+\],\s*timeout:\s*(\d+(?:\.\d+)?))`)
)

// KotlinCoroutineAdvanceRule checks that async coroutines under StandardTestDispatcher advance virtual time.
type KotlinCoroutineAdvanceRule struct{}

// NewKotlinCoroutineAdvanceRule creates a new rule instance.
func NewKotlinCoroutineAdvanceRule() *KotlinCoroutineAdvanceRule {
	return &KotlinCoroutineAdvanceRule{}
}

// Evaluate reports a test that injects a StandardTestDispatcher into a subject,
// calls it and then reads that subject's state without advancing virtual time.
// It reports nothing when any link of that chain cannot be shown in the test
// body, because unknown provenance is not evidence of a defect.
func (r *KotlinCoroutineAdvanceRule) Evaluate(unit parser.TestUnit) []models.Violation {
	if len(unit.Assertions) == 0 {
		return nil
	}
	if unit.Meta == nil || unit.Meta["hasStandardDispatcher"] != "true" || unit.Meta["hasUnconfinedDispatcher"] == "true" {
		return nil
	}

	code := codeMask(unit.RawBody)
	body := unit.RawBody
	runTestPos, runTestArgs, ok := findRunTest(code)
	if !ok {
		return nil
	}
	dispatcher := injectedDispatcher(code, body, runTestArgs)
	if dispatcher == "" {
		return nil
	}
	if hasAdvancement(code) {
		return nil
	}

	for _, d := range scanInitDecls(code) {
		if d.pos < runTestPos || !declPassesIdent(code, body, d, dispatcher) {
			continue
		}
		callAt := callOnSubject(code, d.name, d.pos)
		if callAt < 0 {
			continue
		}
		chain, ok := stateReadInAssertions(code, d.name, callAt)
		if !ok {
			continue
		}
		return []models.Violation{{
			RuleID:     "GUARD-KT-02",
			Severity:   models.SeverityWarning,
			FilePath:   unit.FilePath,
			LineNumber: unit.StartLine,
			Column:     1,
			Message:    fmt.Sprintf("Unadvanced coroutine state in '%s': it builds '%s' with the StandardTestDispatcher value '%s', calls '%s', then asserts '%s' with no visible advance or await call.", unit.Name, d.name, dispatcher, d.name, chain),
			Suggestion: "Call advanceUntilIdle(), runCurrent(), or awaitItem() before asserting coroutine state under StandardTestDispatcher.",
		}}
	}

	return nil
}

// findRunTest returns the offset just past the callee where runTest is invoked
// in code, together with the spans of its arguments. Both the argument-list
// form `runTest(dispatcher) { … }` and the canonical trailing-lambda form
// `runTest { … }`, which carries no argument spans, are accepted. A brace that
// starts on a later line is a separate block in Kotlin rather than the trailing
// lambda of this call, so it does not count as an invocation.
func findRunTest(code string) (pos int, args []codeSpan, ok bool) {
	for i := 0; i < len(code); {
		if !isIdentStart(code[i]) {
			i++
			continue
		}
		ident, j := identAt(code, i)
		if ident != "runTest" {
			i = j
			continue
		}
		if k := skipInlineSpace(code, j); k < len(code) {
			switch code[k] {
			case '(':
				spans, closeIdx := callArgs(code, k)
				if closeIdx >= 0 {
					return j, spans, true
				}
			case '{':
				return j, nil, true
			}
		}
		i = j
	}
	return 0, nil, false
}

// injectedDispatcher returns the identifier passed to runTest, which is the
// dispatcher the test's coroutines are scheduled on. A dispatcher constructed
// locally inside the body is used when runTest takes no argument, which is the
// case for the trailing-lambda form.
func injectedDispatcher(code, body string, args []codeSpan) string {
	for _, a := range args {
		if name, ok := singleIdent(code, argValueSpan(code, body, a)); ok {
			return name
		}
	}
	for _, d := range scanInitDecls(code) {
		if d.ctor == "StandardTestDispatcher" {
			return d.name
		}
	}
	return ""
}

// advancementTokens are the calls that let StandardTestDispatcher run pending
// coroutines before an assertion reads their state.
var advancementTokens = []string{
	"advanceUntilIdle",
	"advanceTimeBy",
	"runCurrent",
	".join()",
	"joinAll",
	".await()",
	"awaitAll",
	"yield()",
	".test {",
	"awaitItem",
}

func hasAdvancement(code string) bool {
	for _, token := range advancementTokens {
		if strings.Contains(code, token) {
			return true
		}
	}
	return false
}

// callOnSubject returns the offset of the opening parenthesis of the first
// method call made on subject after from, or -1.
func callOnSubject(code, subject string, from int) int {
	for i := from; i < len(code); {
		if !isIdentStart(code[i]) {
			i++
			continue
		}
		ident, j := identAt(code, i)
		if ident == subject {
			if k := skipCodeSpace(code, j); k < len(code) && code[k] == '.' {
				n := skipCodeSpace(code, k+1)
				if n < len(code) && isIdentStart(code[n]) {
					_, m := identAt(code, n)
					if p := skipCodeSpace(code, m); p < len(code) && code[p] == '(' {
						return p
					}
				}
			}
		}
		i = j
	}
	return -1
}

// isAssertionName reports whether name introduces an assertion call.
func isAssertionName(name string) bool {
	if name == "check" || name == "fail" {
		return true
	}
	for _, prefix := range []string{"assert", "should", "verify", "expect"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

// stateReadInAssertions returns the subject chain, e.g. `vm.state.value`, when
// an assertion at or after from reads a state or flow value of subject.
func stateReadInAssertions(code, subject string, from int) (string, bool) {
	for i := from; i < len(code); {
		if !isIdentStart(code[i]) {
			i++
			continue
		}
		ident, j := identAt(code, i)
		if ident == subject {
			segments, end := pureChain(code, i)
			if len(segments) >= 2 && containsSegment(segments, "value") {
				m := skipCodeSpace(code, end)
				if matcher, _ := identAt(code, m); strings.HasPrefix(matcher, "should") {
					return strings.Join(segments, "."), true
				}
			}
		}
		if isAssertionName(ident) {
			if k := skipCodeSpace(code, j); k < len(code) && code[k] == '(' {
				if closeIdx := parser.FindMatchingParen(code, k); closeIdx >= 0 {
					if chain, ok := stateChainInSpan(code, k+1, closeIdx, subject); ok {
						return chain, true
					}
					if next := skipCodeSpace(code, closeIdx+1); next < len(code) && code[next] == '.' {
						matcherStart := skipCodeSpace(code, next+1)
						if _, mj := identAt(code, matcherStart); mj > matcherStart {
							if mk := skipCodeSpace(code, mj); mk < len(code) && code[mk] == '(' {
								if mclose := parser.FindMatchingParen(code, mk); mclose >= 0 {
									if chain, ok := stateChainInSpan(code, mk+1, mclose, subject); ok {
										return chain, true
									}
									i = mclose + 1
									continue
								}
							}
						}
					}
					i = closeIdx + 1
					continue
				}
			}
		}
		i = j
	}
	return "", false
}

// stateChainInSpan finds a member chain of subject that reads a `value`
// property inside the half-open span.
func stateChainInSpan(code string, start, end int, subject string) (string, bool) {
	for i := start; i < end; {
		if !isIdentStart(code[i]) {
			i++
			continue
		}
		ident, j := identAt(code, i)
		if ident == subject {
			segments, chainEnd := pureChain(code, i)
			if len(segments) >= 2 && chainEnd <= end && containsSegment(segments, "value") {
				return strings.Join(segments, "."), true
			}
		}
		i = j
	}
	return "", false
}

func containsSegment(segments []string, want string) bool {
	for _, s := range segments {
		if s == want {
			return true
		}
	}
	return false
}

// KotlinMockEchoRule detects tests that stub a mock and assert the exact stubbed token with no transformation.
type KotlinMockEchoRule struct{}

// NewKotlinMockEchoRule creates a new rule instance.
func NewKotlinMockEchoRule() *KotlinMockEchoRule {
	return &KotlinMockEchoRule{}
}

// Evaluate reports a test that stubs `every { ... } returns X` and then asserts
// the very same token through a subject that was constructed with that stub's
// receiver, with no transformation in between. A matching expected value alone
// is not reported: the stub, the constructing collaborator and the asserted
// value must all be bound to the same receiver.
func (r *KotlinMockEchoRule) Evaluate(unit parser.TestUnit) []models.Violation {
	code := codeMask(unit.RawBody)
	body := unit.RawBody
	decls := scanInitDecls(code)

	for _, st := range scanStubs(code, body) {
		if !hasMockProvenance(decls, st.receiver) {
			continue
		}
		subjects := map[string]bool{st.receiver: true}
		for _, d := range decls {
			if d.pos <= st.pos || mockFactory(d.ctor) {
				continue
			}
			if declPassesIdent(code, body, d, st.receiver) {
				subjects[d.name] = true
			}
		}
		chain, ok := findEcho(code, body, subjects, st.token, st.pos)
		if !ok {
			continue
		}
		return []models.Violation{{
			RuleID:     "GUARD-KT-01",
			Severity:   models.SeverityWarning,
			FilePath:   unit.FilePath,
			LineNumber: unit.StartLine,
			Column:     1,
			Message:    fmt.Sprintf("Mock echo in '%s': '%s' is stubbed to return %s and the assertion compares that same value with '%s' without transformation.", unit.Name, st.stubText, st.token, chain),
			Suggestion: "Assert a transformed outcome or behavior rather than echoing the mock's stubbed value.",
		}}
	}

	return nil
}

// mockFactory reports whether the callee creates a test double.
func mockFactory(ctor string) bool {
	switch ctor {
	case "mockk", "spyk", "mock", "mockkClass":
		return true
	}
	return false
}

// hasMockProvenance reports whether name is known to be a test double: either
// it is declared by a mock factory in the body, or the body does not declare it
// at all, in which case `every { }` is the only lexical evidence available and
// the receiver is treated as declared elsewhere (for example a class field).
func hasMockProvenance(decls []initDecl, name string) bool {
	for _, d := range decls {
		if d.name == name {
			return mockFactory(d.ctor)
		}
	}
	return true
}

// stub is an `every { recv... } returns token` stub whose receiver is a plain
// identifier.
type stub struct {
	receiver string
	stubText string
	token    string
	pos      int
}

// scanStubs returns the MockK stubs in code whose return value is a single
// literal or identifier. Code inside comments and literals is already masked,
// so a commented or quoted stub is not seen.
func scanStubs(code, body string) []stub {
	var stubs []stub
	for i := 0; i < len(code); {
		if !isIdentStart(code[i]) {
			i++
			continue
		}
		ident, j := identAt(code, i)
		if ident != "every" && ident != "coEvery" {
			i = j
			continue
		}
		brace := skipCodeSpace(code, j)
		if brace >= len(code) || code[brace] != '{' {
			i = j
			continue
		}
		closeIdx := parser.FindMatchingBrace(code, brace)
		if closeIdx < 0 {
			i = j
			continue
		}
		receiver, ok := firstIdentIn(code, brace+1, closeIdx)
		if !ok {
			i = closeIdx + 1
			continue
		}
		stubText := strings.Join(strings.Fields(body[i:closeIdx+1]), " ")
		returns := skipCodeSpace(code, closeIdx+1)
		name, afterReturns := identAt(code, returns)
		if name != "returns" {
			i = closeIdx + 1
			continue
		}
		token, tokEnd, ok := parseReturnToken(body, skipInlineSpace(body, afterReturns))
		if !ok {
			i = closeIdx + 1
			continue
		}
		stubs = append(stubs, stub{receiver: receiver, stubText: stubText, token: token, pos: i})
		i = tokEnd
	}
	return stubs
}

func firstIdentIn(code string, start, end int) (string, bool) {
	for i := start; i < end; {
		if isIdentStart(code[i]) {
			ident, _ := identAt(code, i)
			return ident, true
		}
		i++
	}
	return "", false
}

// parseReturnToken reads the single value after `returns`.
func parseReturnToken(body string, start int) (string, int, bool) {
	end := scanExpressionEnd(body, start)
	if end <= start {
		return "", start, false
	}
	token := strings.TrimSpace(body[start:end])
	if !isSimpleToken(token) {
		return "", end, false
	}
	return token, end, true
}

// findEcho returns the asserted subject chain when an assertion at or after
// from compares one of the subjects with token.
func findEcho(code, body string, subjects map[string]bool, token string, from int) (string, bool) {
	for i := from; i < len(code); {
		if !isIdentStart(code[i]) {
			i++
			continue
		}
		ident, j := identAt(code, i)
		k := skipCodeSpace(code, j)

		if k < len(code) && code[k] == '(' {
			switch ident {
			case "assertEquals":
				if args, closeIdx := callArgs(code, k); closeIdx >= 0 {
					if chain, ok := echoPair(code, body, args, subjects, token); ok {
						return chain, true
					}
					i = closeIdx + 1
					continue
				}
			case "assertThat":
				if args, closeIdx := callArgs(code, k); closeIdx >= 0 {
					if len(args) == 1 {
						if chain, ok := subjectChain(code, args[0], subjects); ok {
							if matcherChain, ok := echoedMatcher(code, body, closeIdx, token); ok && matcherChain {
								return chain, true
							}
						}
					}
					i = closeIdx + 1
					continue
				}
			}
		}

		if subjects[ident] {
			if segments, end := pureChain(code, i); len(segments) >= 2 {
				m := skipCodeSpace(code, end)
				if matcher, mj := identAt(code, m); equalityInfix(matcher) {
					valueStart := skipInlineSpace(body, mj)
					valueEnd := scanExpressionEnd(body, valueStart)
					if valueEnd > valueStart && strings.TrimSpace(body[valueStart:valueEnd]) == token {
						return strings.Join(segments, "."), true
					}
				}
			}
		}

		i = j
	}
	return "", false
}

// echoPair reports the subject chain when one argument of assertEquals is the
// stub token and another is a pure member chain rooted at a subject.
func echoPair(code, body string, args []codeSpan, subjects map[string]bool, token string) (string, bool) {
	for a, arg := range args {
		if valueText(body, argValueSpan(code, body, arg)) != token {
			continue
		}
		for b, other := range args {
			if a == b {
				continue
			}
			if chain, ok := subjectChain(code, argValueSpan(code, body, other), subjects); ok {
				return chain, true
			}
		}
	}
	return "", false
}

// echoedMatcher reports whether the matcher chained after assertThat(...) is an
// equality matcher whose argument is the stub token.
func echoedMatcher(code, body string, callClose int, token string) (bool, bool) {
	dot := skipCodeSpace(code, callClose+1)
	if dot >= len(code) || code[dot] != '.' {
		return false, false
	}
	matcherStart := skipCodeSpace(code, dot+1)
	matcher, mj := identAt(code, matcherStart)
	if matcher != "isEqualTo" && matcher != "isSameInstanceAs" {
		return false, false
	}
	open := skipCodeSpace(code, mj)
	if open >= len(code) || code[open] != '(' {
		return false, false
	}
	args, closeIdx := callArgs(code, open)
	if closeIdx < 0 {
		return false, false
	}
	for _, arg := range args {
		if valueText(body, argValueSpan(code, body, arg)) == token {
			return true, true
		}
	}
	return false, true
}

// valueText returns the trimmed original text of a span, so literal values keep
// their quotes and contents.
func valueText(body string, sp codeSpan) string {
	s, e := trimSpan(body, sp.start, sp.end)
	return body[s:e]
}

func equalityInfix(name string) bool {
	switch name {
	case "shouldBe", "shouldEqual", "shouldBeEqualTo":
		return true
	}
	return false
}

// SwiftTimeoutRule checks that XCTest unit tests do not use excessive expectation timeouts (>2.0s).
type SwiftTimeoutRule struct{}

// NewSwiftTimeoutRule creates a new rule instance.
func NewSwiftTimeoutRule() *SwiftTimeoutRule {
	return &SwiftTimeoutRule{}
}

// Evaluate checks if expectation timeout exceeds 2.0s.
func (r *SwiftTimeoutRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	matches := swiftTimeoutRegex.FindAllStringSubmatch(unit.RawBody, -1)
	for _, m := range matches {
		timeoutStr := m[1]
		if timeoutStr == "" && len(m) > 2 {
			timeoutStr = m[2]
		}
		if timeoutStr != "" {
			timeoutVal, err := strconv.ParseFloat(timeoutStr, 64)
			if err == nil && timeoutVal > 2.0 {
				violations = append(violations, models.Violation{
					RuleID:     "GUARD-SWIFT-01",
					Severity:   models.SeverityAdvisory,
					FilePath:   unit.FilePath,
					LineNumber: unit.StartLine,
					Column:     1,
					Message:    fmt.Sprintf("Long expectation timeout in '%s': timeout %.1fs only delays test suite feedback when the test fails.", unit.Name, timeoutVal),
					Suggestion: "Use await fulfillment(of:) with async code or keep timeout minimal to avoid delaying failure feedback.",
				})
			}
		}
	}

	return violations
}
