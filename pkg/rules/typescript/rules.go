package typescript

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	ts "github.com/HoangNguyen0403/testguard/pkg/parser/typescript"
)

var (
	nestControllerMockRegex     = regexp.MustCompile(`(?:\b(?:mockResolvedValue|mockReturnValue)\s*\(\s*([^,\)]+)\))`)
	nestControllerInlineRegex   = regexp.MustCompile(`expect\s*\(\s*await\s+controller\.\w+\([^)]*\)\s*\)\.(?:resolves\.)?(?:toBe|toEqual|toStrictEqual)\s*\(\s*([^,\)]+)\)`)
	nestControllerAssignedRegex = regexp.MustCompile(`(?:\b(?:const|let|var)\s+([A-Za-z0-9_$]+)\s*=\s*await\s+controller\.\w+\([^)]*\))`)
	fakeTimersRegex             = regexp.MustCompile(`useFakeTimers\s*\(`)
	setTimeoutRegex             = regexp.MustCompile(`\bsetTimeout\s*\(`)
	mockCallRegex               = regexp.MustCompile(`\b(?:mockImplementation|mockImplementationOnce|mockReturnValue|mockReturnValueOnce|mockResolvedValue|mockRejectedValue)\s*\(`)
	classDeclRegex              = regexp.MustCompile(`\bclass\s+\w+`)
	jsIdentRegex                = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*(?:\.[a-zA-Z_$][a-zA-Z0-9_$]*)*$`)

	realDbModuleRegex        = regexp.MustCompile(`\b(?:TypeOrmModule\.forRoot|MongooseModule\.forRoot|SequelizeModule\.forRoot|PrismaModule)\b`)
	createTestingModuleRegex = regexp.MustCompile(`\bTest\.createTestingModule\s*\(`)

	mockCallMatcherRegex = regexp.MustCompile(`^(?:toHaveBeenCalled|toHaveBeenCalledTimes|toHaveBeenCalledWith|toHaveBeenLastCalledWith|toHaveBeenNthCalledWith|toBeCalled|toBeCalledTimes|toBeCalledWith|toBeCalledWithMatch|toBeCalledWithProperty)$`)

	jestOrViMockCallRegex = regexp.MustCompile(`\b(?:jest|vi)\.mock\s*\(\s*['"` + "`" + `]([^'"` + "`" + `]+)['"` + "`" + `]`)
	failCallRegex         = regexp.MustCompile(`\b(?:fail|expect\.fail|assert\.fail|expect\.unreachable)\s*\(|\bthrow\b`)
)

func lexTestBody(unit parser.TestUnit) []ts.Token {
	ext := strings.ToLower(filepath.Ext(unit.FilePath))
	enableJSX := ext == ".tsx" || ext == ".jsx" || ext == ".js"
	return ts.NewLexerWithJSX(unit.RawBody, enableJSX).Tokenize()
}

// bodyStartLine returns the original 1-based file line holding the first byte
// of RawBody. RawBody starts after the declaration title, so a multi-line
// declaration would otherwise shift every statement-level diagnostic by the
// number of title lines. Adapters that cannot supply the fact fall back to the
// declaration line, which keeps whole-test findings anchored as before.
func bodyStartLine(unit parser.TestUnit) int {
	if raw, ok := unit.Meta["body_start_line"]; ok {
		if line, err := strconv.Atoi(raw); err == nil && line > 0 {
			return line
		}
	}
	return unit.StartLine
}

// NestControllerEchoRule flags NestJS controller unit tests that do 1:1 pass-through mock echoing.
type NestControllerEchoRule struct{}

// NewNestControllerEchoRule creates an evaluator for GUARD-NEST-01.
func NewNestControllerEchoRule() *NestControllerEchoRule {
	return &NestControllerEchoRule{}
}

// Evaluate checks whether a controller unit test simply asserts that the controller returns its mocked service stub.
func (r *NestControllerEchoRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	mockMatches := nestControllerMockRegex.FindAllStringSubmatch(unit.RawBody, -1)
	if len(mockMatches) == 0 {
		return nil
	}

	for _, mm := range mockMatches {
		if len(mm) < 2 {
			continue
		}
		mockVal := strings.TrimSpace(mm[1])

		// Form 1: Inline expect(await controller.m()).toBe/toEqual/toStrictEqual(mockVal)
		inlineMatches := nestControllerInlineRegex.FindAllStringSubmatch(unit.RawBody, -1)
		for _, im := range inlineMatches {
			if len(im) > 1 && strings.TrimSpace(im[1]) == mockVal {
				return []models.Violation{{
					RuleID:     "GUARD-NEST-01",
					Severity:   models.SeverityWarning,
					FilePath:   unit.FilePath,
					LineNumber: unit.StartLine,
					Column:     1,
					Message:    fmt.Sprintf("Pass-through controller echo in '%s': controller test simply returns mock '%s' without validation or transformation.", unit.Name, mockVal),
					Suggestion: "Test validation pipes, guards, interceptors, or integration behavior via supertest rather than 1:1 controller pass-through mocks.",
				}}
			}
		}

		// Form 2: Assigned result: const r = await controller.m(); expect(r).toEqual(mockVal)
		assignedMatches := nestControllerAssignedRegex.FindAllStringSubmatch(unit.RawBody, -1)
		for _, am := range assignedMatches {
			if len(am) < 2 {
				continue
			}
			varName := am[1]
			expectVarRegex := regexp.MustCompile(`expect\s*\(\s*` + regexp.QuoteMeta(varName) + `\s*\)\.(?:resolves\.)?(?:toBe|toEqual|toStrictEqual)\s*\(\s*([^,\)]+)\)`)
			expectVarMatches := expectVarRegex.FindAllStringSubmatch(unit.RawBody, -1)
			for _, evm := range expectVarMatches {
				if len(evm) > 1 && strings.TrimSpace(evm[1]) == mockVal {
					return []models.Violation{{
						RuleID:     "GUARD-NEST-01",
						Severity:   models.SeverityWarning,
						FilePath:   unit.FilePath,
						LineNumber: unit.StartLine,
						Column:     1,
						Message:    fmt.Sprintf("Pass-through controller echo in '%s': controller test simply returns mock '%s' through assigned variable '%s' without validation or transformation.", unit.Name, mockVal, varName),
						Suggestion: "Test validation pipes, guards, interceptors, or integration behavior via supertest rather than 1:1 controller pass-through mocks.",
					}}
				}
			}
		}
	}

	return violations
}

// NestUnmockedDbModuleRule traps unmocked database modules inside NestJS unit tests.
type NestUnmockedDbModuleRule struct{}

// NewNestUnmockedDbModuleRule creates an evaluator for GUARD-NEST-02.
func NewNestUnmockedDbModuleRule() *NestUnmockedDbModuleRule {
	return &NestUnmockedDbModuleRule{}
}

// Evaluate evaluates a single unit for GUARD-NEST-02.
func (r *NestUnmockedDbModuleRule) Evaluate(unit parser.TestUnit) []models.Violation {
	return r.EvaluateFile(unit.FilePath, unit.RawBody, []parser.TestUnit{unit})
}

// EvaluateFile evaluates a full source file for unmocked database modules inside Test.createTestingModule.
func (r *NestUnmockedDbModuleRule) EvaluateFile(filePath string, src string, units []parser.TestUnit) []models.Violation {
	var violations []models.Violation

	if parser.IsE2EFile(filePath, []byte(src)) {
		return nil
	}

	fileName := filepath.Base(filePath)
	if strings.Contains(fileName, ".integration.") ||
		strings.Contains(fileName, ".int-spec.") ||
		strings.Contains(fileName, ".e2e-spec.") ||
		strings.Contains(fileName, ".e2e.") {
		return nil
	}

	if strings.Contains(src, "MongoMemoryServer") || strings.Contains(src, "mongodb-memory-server") {
		return nil
	}

	matches := createTestingModuleRegex.FindAllStringIndex(src, -1)
	for _, m := range matches {
		callStart := m[0]
		if !isCodeIndex(src, callStart) {
			continue
		}

		openParen := m[1] - 1
		closeParen := findMatchingParen(src, openParen)
		if closeParen == -1 {
			continue
		}

		argSrc := src[openParen+1 : closeParen]

		// Check in-memory configuration
		if strings.Contains(argSrc, ":memory:") {
			continue
		}

		// Check method chain for .overrideProvider or .overrideModule
		chainEnd := closeParen
		for chainEnd < len(src) && src[chainEnd] != ';' && src[chainEnd] != '\n' {
			chainEnd++
		}
		// If compile() is on next lines, scan up to compile() or statement end
		scanLimit := closeParen + 200
		if scanLimit > len(src) {
			scanLimit = len(src)
		}
		chainSrc := src[closeParen:scanLimit]
		if strings.Contains(chainSrc, ".overrideProvider(") || strings.Contains(chainSrc, ".overrideModule(") {
			continue
		}

		// Find real database modules inside argSrc
		dbMatches := realDbModuleRegex.FindAllStringIndex(argSrc, -1)
		for _, dbm := range dbMatches {
			dbIdx := openParen + 1 + dbm[0]
			if !isCodeIndex(src, dbIdx) {
				continue
			}

			modName := argSrc[dbm[0]:dbm[1]]
			line := strings.Count(src[:dbIdx], "\n") + 1

			violations = append(violations, models.Violation{
				RuleID:     "GUARD-NEST-02",
				Severity:   models.SeverityWarning,
				FilePath:   filePath,
				LineNumber: line,
				Column:     1,
				Message:    fmt.Sprintf("Unmocked database module leak in '%s': NestJS testing module imports '%s' directly.", fileName, modName),
				Suggestion: "Use getRepositoryToken() or mock providers in Test.createTestingModule instead of importing real database connection modules.",
			})
			break
		}
	}

	return violations
}

// SnapshotOnlyRule reports unit tests where every assertion is a snapshot matcher.
type SnapshotOnlyRule struct{}

// NewSnapshotOnlyRule creates an evaluator for GUARD-TS-01.
func NewSnapshotOnlyRule() *SnapshotOnlyRule {
	return &SnapshotOnlyRule{}
}

// Evaluate reports tests where all assertions are snapshot matchers.
func (r *SnapshotOnlyRule) Evaluate(unit parser.TestUnit) []models.Violation {
	if parser.IsE2EFile(unit.FilePath, []byte(unit.RawBody)) {
		return nil
	}
	if len(unit.Assertions) == 0 {
		return nil
	}

	tokens := lexTestBody(unit)
	assertions := ts.FindAssertions(unit.RawBody, tokens, 0, len(unit.RawBody), nil)
	if len(assertions) == 0 {
		return nil
	}

	for _, a := range assertions {
		if a.Matcher != "toMatchSnapshot" && a.Matcher != "toMatchInlineSnapshot" && a.Matcher != "toMatchFileSnapshot" {
			return nil
		}
	}

	return []models.Violation{{
		RuleID:     "GUARD-TS-01",
		Severity:   models.SeverityAdvisory,
		FilePath:   unit.FilePath,
		LineNumber: unit.StartLine,
		Column:     1,
		Message:    fmt.Sprintf("Snapshot-only test in '%s': every assertion is a snapshot matcher.", unit.Name),
		Suggestion: "Add behavioral assertions verifying domain state or return values alongside or in place of snapshot matchers.",
	}}
}

// FloatingAssertRule reports un-awaited promises in expect(...).resolves / rejects.
type FloatingAssertRule struct{}

// NewFloatingAssertRule creates an evaluator for GUARD-TS-FLOATING-ASSERT.
func NewFloatingAssertRule() *FloatingAssertRule {
	return &FloatingAssertRule{}
}

// Evaluate detects floating promise assertion statements.
func (r *FloatingAssertRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	tokens := lexTestBody(unit)
	n := len(tokens)

	for i := 0; i < n; i++ {
		tok := tokens[i]
		if tok.Value != "expect" {
			continue
		}
		if i > 0 && (tokens[i-1].Value == "." || tokens[i-1].Value == "?.") {
			continue
		}

		// Find call expect(...)
		if i+1 >= n || tokens[i+1].Value != "(" {
			continue
		}

		// Check if chain contains resolves or rejects
		curr := i + 1
		hasAsyncMatcher := false
		for curr < n && tokens[curr].Value != ";" && tokens[curr].Line == tok.Line {
			if tokens[curr].Value == "resolves" || tokens[curr].Value == "rejects" {
				hasAsyncMatcher = true
				break
			}
			curr++
		}
		if !hasAsyncMatcher {
			continue
		}

		// Inspect tokens before expect on this statement
		isFloating := true
		for b := i - 1; b >= 0; b-- {
			pt := tokens[b]
			if pt.Value == ";" || pt.Value == "{" || pt.Value == "}" {
				break
			}
			if pt.Value == "await" || pt.Value == "return" || pt.Value == "void" ||
				pt.Value == "=" || pt.Value == "[" || pt.Value == "(" || pt.Value == "," || pt.Value == "=>" {
				isFloating = false
				break
			}
		}

		if isFloating {
			line := bodyStartLine(unit) + (tok.Line - 1)
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-TS-FLOATING-ASSERT",
				Severity:   models.SeverityWarning,
				FilePath:   unit.FilePath,
				LineNumber: line,
				Column:     1,
				Message:    fmt.Sprintf("Floating async assertion in '%s': promise matcher chain is not awaited or returned.", unit.Name),
				Suggestion: "Await the assertion with 'await expect(...)' or return it to prevent false passes.",
			})
		}
	}

	return violations
}

// CatchOnlyRule reports tests where all assertions are swallowed in catch blocks without a fail() guard.
type CatchOnlyRule struct{}

// NewCatchOnlyRule creates an evaluator for GUARD-TS-CATCH-ONLY.
func NewCatchOnlyRule() *CatchOnlyRule {
	return &CatchOnlyRule{}
}

// Evaluate checks whether assertions only exist in catch blocks.
func (r *CatchOnlyRule) Evaluate(unit parser.TestUnit) []models.Violation {
	if strings.Contains(unit.RawBody, "expect.assertions(") || strings.Contains(unit.RawBody, "expect.hasAssertions()") {
		return nil
	}

	tokens := lexTestBody(unit)
	assertions := ts.FindAssertions(unit.RawBody, tokens, 0, len(unit.RawBody), nil)
	if len(assertions) == 0 {
		return nil
	}

	n := len(tokens)
	findMatchingBrace := func(openIdx int) int {
		if openIdx >= n || tokens[openIdx].Value != "{" {
			return -1
		}
		depth := 0
		for k := openIdx; k < n; k++ {
			if tokens[k].Value == "{" {
				depth++
			} else if tokens[k].Value == "}" {
				depth--
				if depth == 0 {
					return k
				}
			}
		}
		return -1
	}

	type catchSpan struct {
		startOff int
		endOff   int
		tryLine  int
	}

	var catches []catchSpan

	for i := 0; i < n; i++ {
		if tokens[i].Value == "try" && (i == 0 || tokens[i-1].Value != ".") {
			tryTok := tokens[i]
			// Try block
			if i+1 < n && tokens[i+1].Value == "{" {
				tryBrace := findMatchingBrace(i + 1)
				if tryBrace != -1 {
					tryBody := unit.RawBody[tokens[i+1].Offset : tokens[tryBrace].Offset+1]
					if failCallRegex.MatchString(tryBody) {
						// Guarded try block
						return nil
					}

					// Look for catch
					curr := tryBrace + 1
					if curr < n && tokens[curr].Value == "catch" {
						catchOpenBrace := -1
						for c := curr + 1; c < n; c++ {
							if tokens[c].Value == "{" {
								catchOpenBrace = c
								break
							}
						}
						if catchOpenBrace != -1 {
							catchCloseBrace := findMatchingBrace(catchOpenBrace)
							if catchCloseBrace != -1 {
								catches = append(catches, catchSpan{
									startOff: tokens[catchOpenBrace].Offset,
									endOff:   tokens[catchCloseBrace].Offset + 1,
									tryLine:  bodyStartLine(unit) + (tryTok.Line - 1),
								})
							}
						}
					}
				}
			}
		}
	}

	if len(catches) == 0 {
		return nil
	}

	// Verify that EVERY assertion is inside one of the catch blocks
	for _, a := range assertions {
		inCatch := false
		for _, c := range catches {
			if a.StartOffset >= c.startOff && a.EndOffset <= c.endOff {
				inCatch = true
				break
			}
		}
		if !inCatch {
			return nil
		}
	}

	return []models.Violation{{
		RuleID:     "GUARD-TS-CATCH-ONLY",
		Severity:   models.SeverityWarning,
		FilePath:   unit.FilePath,
		LineNumber: catches[0].tryLine,
		Column:     1,
		Message:    fmt.Sprintf("Swallowed error path in '%s': all assertions are inside catch block but try block has no fail() guard.", unit.Name),
		Suggestion: "Add 'expect.fail(\"should have thrown\")' or 'throw new Error(\"...\")' at the end of the try block, or use expect(...).rejects.toThrow().",
	}}
}

// ConditionalAssertRule reports tests where all assertions are inside conditional blocks.
type ConditionalAssertRule struct{}

// NewConditionalAssertRule creates an evaluator for GUARD-TS-CONDITIONAL-ASSERT.
func NewConditionalAssertRule() *ConditionalAssertRule {
	return &ConditionalAssertRule{}
}

// Evaluate checks if all assertions are conditionally executed.
func (r *ConditionalAssertRule) Evaluate(unit parser.TestUnit) []models.Violation {
	if strings.Contains(unit.RawBody, "expect.assertions(") || strings.Contains(unit.RawBody, "expect.hasAssertions()") {
		return nil
	}

	tokens := lexTestBody(unit)
	assertions := ts.FindAssertions(unit.RawBody, tokens, 0, len(unit.RawBody), nil)
	if len(assertions) == 0 {
		return nil
	}

	spans := classifyConditionalSpans(tokens, assertions, bodyStartLine(unit))
	if len(spans) == 0 {
		return nil
	}

	var offending *conditionalSpan
	for _, a := range assertions {
		enclosed := false
		for i := range spans {
			if a.StartOffset < spans[i].startOff || a.EndOffset > spans[i].endOff {
				continue
			}
			enclosed = true
			if !spans[i].total && offending == nil {
				offending = &spans[i]
			}
		}
		if !enclosed {
			return nil
		}
	}
	if offending == nil {
		return nil
	}

	return []models.Violation{{
		RuleID:     "GUARD-TS-CONDITIONAL-ASSERT",
		Severity:   models.SeverityWarning,
		FilePath:   unit.FilePath,
		LineNumber: offending.line,
		Column:     1,
		Message:    fmt.Sprintf("Conditional assertion in '%s': assertions are only reached when conditionals on line %d hold, so a failing path asserts nothing.", unit.Name, offending.line),
		Suggestion: "Assert conditions deterministically or declare 'expect.hasAssertions()' to ensure assertions actually execute.",
	}}
}

// conditionalSpan is a token range that executes only when its condition holds.
// total records that every path through the owning construct asserts, so an
// assertion inside it is guaranteed to run and is not a conditional assertion.
type conditionalSpan struct {
	startOff int
	endOff   int
	line     int
	total    bool
}

// classifyConditionalSpans tokenizes branch structure, without a control-flow
// graph, into spans carrying a guaranteed-execution flag. Values are clamped
// conservatively: anything not provably exhaustive stays non-total.
func classifyConditionalSpans(tokens []ts.Token, assertions []ts.AssertionInfo, originLine int) []conditionalSpan {
	n := len(tokens)
	if n == 0 {
		return nil
	}

	conditionRanges := make([][2]int, 0, 4)
	for k := range n {
		switch tokens[k].Value {
		case "if", "while", "switch", "for":
		default:
			continue
		}
		if k+1 >= n || tokens[k+1].Value != "(" {
			continue
		}
		if closeParen := matchingTokenIndex(tokens, k+1); closeParen != -1 {
			conditionRanges = append(conditionRanges, [2]int{tokens[k+1].Offset, tokens[closeParen].Offset + 1})
		}
	}
	inCondition := func(off int) bool {
		for _, r := range conditionRanges {
			if off >= r[0] && off < r[1] {
				return true
			}
		}
		return false
	}

	var spans []conditionalSpan
	for i := range n {
		line := originLine + tokens[i].Line - 1
		switch tokens[i].Value {
		case "if":
			// An `else if` is already part of the chain classified at its opening `if`.
			if i > 0 && tokens[i-1].Value == "else" {
				continue
			}
			branches, exhaustive := ifChainBranches(tokens, i)
			if len(branches) == 0 {
				continue
			}
			total := exhaustive
			for _, b := range branches {
				if !branchTotal(tokens, assertions, b[0], b[1]) {
					total = false
				}
			}
			for _, b := range branches {
				spans = append(spans, conditionalSpan{
					startOff: tokens[b[0]].Offset,
					endOff:   tokens[b[1]].Offset + 1,
					line:     line,
					total:    total,
				})
			}
		case "switch":
			if i+1 >= n || tokens[i+1].Value != "(" {
				continue
			}
			closeParen := matchingTokenIndex(tokens, i+1)
			if closeParen == -1 || closeParen+1 >= n || tokens[closeParen+1].Value != "{" {
				continue
			}
			bodyOpen := closeParen + 1
			bodyClose := matchingTokenIndex(tokens, bodyOpen)
			if bodyClose == -1 {
				continue
			}
			clauses, hasDefault := switchClauses(tokens, bodyOpen, bodyClose)
			if len(clauses) == 0 {
				continue
			}
			total := hasDefault
			for idx, c := range clauses {
				if rangeTotal(tokens, assertions, c.startOff, c.endOff) {
					continue
				}
				followedByClause := idx+1 < len(clauses)
				if c.emptyBody() && followedByClause {
					continue
				}
				total = false
			}
			for _, c := range clauses {
				spans = append(spans, conditionalSpan{
					startOff: c.startOff,
					endOff:   c.endOff,
					line:     line,
					total:    total,
				})
			}
		case "?":
			if inCondition(tokens[i].Offset) {
				continue
			}
			span := ternarySpan(tokens, i, line)
			span.total = rangeTotal(tokens, assertions, span.startOff+1, span.endOff)
			if colon := ternaryColon(tokens, i); colon != -1 {
				span.total = rangeTotal(tokens, assertions, span.startOff+1, colon) &&
					rangeTotal(tokens, assertions, colon+1, span.endOff)
			}
			spans = append(spans, span)
		case "&&", "||":
			if inCondition(tokens[i].Offset) {
				continue
			}
			endOff := tokens[i].Offset
			for k := i + 1; k < n && tokens[k].Value != ";" && tokens[k].Line == tokens[i].Line; k++ {
				endOff = tokens[k].Offset + len(tokens[k].Value)
			}
			spans = append(spans, conditionalSpan{
				startOff: tokens[i].Offset,
				endOff:   endOff,
				line:     line,
				total:    false,
			})
		}
	}
	return spans
}

// assertionWithin reports whether an assertion starts and ends inside the
// range, so the range executes at least one real assertion.
func assertionWithin(assertions []ts.AssertionInfo, startOff, endOff int) bool {
	for _, a := range assertions {
		if a.StartOffset >= startOff && a.EndOffset <= endOff {
			return true
		}
	}
	return false
}

// rangeTotal reports that a token range either asserts or fails the test on
// every path through it, which makes an assertion inside it non-conditional.
func rangeTotal(tokens []ts.Token, assertions []ts.AssertionInfo, startOff, endOff int) bool {
	if assertionWithin(assertions, startOff, endOff) {
		return true
	}
	return failsOutright(tokens, startOff, endOff)
}

// branchTotal applies rangeTotal to a braced if/else branch. The braces are
// excluded from the failure scan so a statement-level failure still counts.
func branchTotal(tokens []ts.Token, assertions []ts.AssertionInfo, openIdx, closeIdx int) bool {
	if closeIdx <= openIdx {
		return false
	}
	if assertionWithin(assertions, tokens[openIdx].Offset, tokens[closeIdx].Offset+1) {
		return true
	}
	if closeIdx <= openIdx+1 {
		return false
	}
	return failsOutright(tokens, tokens[openIdx].Offset+1, tokens[closeIdx].Offset)
}

// failsOutright reports whether the range is guaranteed to fail the test with a
// statement-level throw or explicit failure call, or with a nested if/switch
// whose every bounded path fails that way. Unbraced control bodies, function
// bodies and call arguments are excluded because no path guarantees they run,
// so a lone nested throw never makes the enclosing branch total.
func failsOutright(tokens []ts.Token, startOff, endOff int) bool {
	braceDepth, parenDepth := 0, 0
	guarded := false
	for i, tok := range tokens {
		if tok.Offset < startOff {
			continue
		}
		if tok.Offset >= endOff {
			break
		}
		switch tok.Value {
		case "{":
			braceDepth++
		case "}":
			braceDepth--
			if braceDepth == 0 && parenDepth == 0 {
				guarded = false
			}
		case "(", "[":
			parenDepth++
		case ")", "]":
			parenDepth--
		case ";":
			if braceDepth == 0 && parenDepth == 0 {
				guarded = false
			}
		case "if", "for", "while", "switch", "else":
			if braceDepth != 0 || parenDepth != 0 {
				continue
			}
			if tok.Value == "if" && (i == 0 || tokens[i-1].Value != "else") && nestedFailsOutright(tokens, i) {
				return true
			}
			if tok.Value == "switch" && nestedSwitchFailsOutright(tokens, i) {
				return true
			}
			guarded = true
		case "throw":
			if braceDepth != 0 || parenDepth != 0 || guarded {
				continue
			}
			if i > 0 && tokens[i-1].Value == "." {
				continue
			}
			return true
		case "fail", "unreachable":
			if braceDepth != 0 || parenDepth != 0 || guarded {
				continue
			}
			if i+1 >= len(tokens) || tokens[i+1].Value != "(" {
				continue
			}
			if isFailureCallee(tokens, i) {
				return true
			}
		}
	}
	return false
}

// nestedFailsOutright reports whether the if/else chain starting at ifIdx fails
// the test on every path: the chain must end in a final else and every branch
// must fail outright. A chain with a non-failing fallthrough path returns false,
// so the enclosing branch keeps its conditional warning.
func nestedFailsOutright(tokens []ts.Token, ifIdx int) bool {
	branches, exhaustive := ifChainBranches(tokens, ifIdx)
	if !exhaustive || len(branches) == 0 {
		return false
	}
	for _, b := range branches {
		if b[1] <= b[0]+1 {
			return false
		}
		if !failsOutright(tokens, tokens[b[0]].Offset+1, tokens[b[1]].Offset) {
			return false
		}
	}
	return true
}

// nestedSwitchFailsOutright reports whether the switch starting at switchIdx
// fails the test on every path: it must have a default clause and every clause
// must fail outright. Empty fall-through labels share the following body.
func nestedSwitchFailsOutright(tokens []ts.Token, switchIdx int) bool {
	n := len(tokens)
	if switchIdx+1 >= n || tokens[switchIdx+1].Value != "(" {
		return false
	}
	closeParen := matchingTokenIndex(tokens, switchIdx+1)
	if closeParen == -1 || closeParen+1 >= n || tokens[closeParen+1].Value != "{" {
		return false
	}
	bodyOpen := closeParen + 1
	bodyClose := matchingTokenIndex(tokens, bodyOpen)
	if bodyClose == -1 {
		return false
	}
	clauses, hasDefault := switchClauses(tokens, bodyOpen, bodyClose)
	if !hasDefault || len(clauses) == 0 {
		return false
	}
	for idx, c := range clauses {
		if c.emptyBody() && idx+1 < len(clauses) {
			continue
		}
		if !failsOutright(tokens, c.startOff, c.endOff) {
			return false
		}
	}
	return true
}

// isFailureCallee reports whether tokens[i] is a bare fail() call or the
// failing member accessor of expect/assert, the explicit ways a test marks an
// unreachable path as a failure.
func isFailureCallee(tokens []ts.Token, i int) bool {
	if i < 1 || tokens[i-1].Value != "." {
		return tokens[i].Value == "fail"
	}
	if i < 2 {
		return false
	}
	switch tokens[i-2].Value {
	case "expect", "assert":
		return tokens[i].Value == "fail" || tokens[i].Value == "unreachable"
	default:
		return false
	}
}

// ifChainBranches returns the token ranges of an if/else-if/else chain and
// whether the chain ends in a final else, so no input escapes unasserted.
func ifChainBranches(tokens []ts.Token, ifIdx int) ([][2]int, bool) {
	n := len(tokens)
	if ifIdx+1 >= n || tokens[ifIdx+1].Value != "(" {
		return nil, false
	}
	closeParen := matchingTokenIndex(tokens, ifIdx+1)
	if closeParen == -1 || closeParen+1 >= n || tokens[closeParen+1].Value != "{" {
		return nil, false
	}
	bodyOpen := closeParen + 1
	bodyClose := matchingTokenIndex(tokens, bodyOpen)
	if bodyClose == -1 {
		return nil, false
	}

	branches := [][2]int{{bodyOpen, bodyClose}}
	exhaustive := false
	for cur := bodyClose + 1; cur < n && tokens[cur].Value == "else"; {
		if cur+1 < n && tokens[cur+1].Value == "if" {
			if cur+2 >= n || tokens[cur+2].Value != "(" {
				break
			}
			condClose := matchingTokenIndex(tokens, cur+2)
			if condClose == -1 || condClose+1 >= n || tokens[condClose+1].Value != "{" {
				break
			}
			open := condClose + 1
			closeIdx := matchingTokenIndex(tokens, open)
			if closeIdx == -1 {
				break
			}
			branches = append(branches, [2]int{open, closeIdx})
			cur = closeIdx + 1
			continue
		}
		if cur+1 < n && tokens[cur+1].Value == "{" {
			open := cur + 1
			closeIdx := matchingTokenIndex(tokens, open)
			if closeIdx == -1 {
				break
			}
			branches = append(branches, [2]int{open, closeIdx})
			exhaustive = true
		}
		break
	}
	return branches, exhaustive
}

// switchClause is one case/default clause body of a switch statement.
type switchClause struct {
	startOff int
	endOff   int
}

// emptyBody reports a fall-through label that shares the following clause body.
func (c switchClause) emptyBody() bool {
	return c.endOff <= c.startOff
}

// switchClauses splits a switch body into clause bodies and reports whether a
// default label exists.
func switchClauses(tokens []ts.Token, bodyOpen, bodyClose int) ([]switchClause, bool) {
	var labels []int
	hasDefault := false
	depth := 0
	for k := bodyOpen + 1; k < bodyClose; k++ {
		switch tokens[k].Value {
		case "{":
			depth++
		case "}":
			depth--
		case "case", "default":
			if depth != 0 {
				continue
			}
			labels = append(labels, k)
			if tokens[k].Value == "default" {
				hasDefault = true
			}
		}
	}

	type clauseLabel struct {
		labelIdx int
		endOff   int
	}
	offsets := make([]clauseLabel, 0, len(labels))
	for idx, labelIdx := range labels {
		endOff := tokens[bodyClose].Offset
		if idx+1 < len(labels) {
			endOff = tokens[labels[idx+1]].Offset
		}
		offsets = append(offsets, clauseLabel{labelIdx: labelIdx, endOff: endOff})
	}

	clauses := make([]switchClause, 0, len(offsets))
	for _, o := range offsets {
		bodyStart := o.endOff
		for k := o.labelIdx + 1; k < bodyClose; k++ {
			if tokens[k].Offset >= o.endOff {
				break
			}
			if tokens[k].Value == ":" {
				bodyStart = tokens[k].Offset + 1
				break
			}
		}
		clauses = append(clauses, switchClause{startOff: bodyStart, endOff: o.endOff})
	}
	return clauses, hasDefault
}

// ternarySpan returns the token range of the ternary whose '?' is at qIdx.
func ternarySpan(tokens []ts.Token, qIdx, line int) conditionalSpan {
	span := conditionalSpan{startOff: tokens[qIdx].Offset, endOff: tokens[qIdx].Offset + 1, line: line}
	depth := 0
	for k := qIdx + 1; k < len(tokens); k++ {
		switch tokens[k].Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			if depth == 0 {
				return span
			}
			depth--
		case ";":
			if depth == 0 {
				return span
			}
		}
		span.endOff = tokens[k].Offset + len(tokens[k].Value)
	}
	return span
}

// ternaryColon returns the byte offset of the ':' pairing the '?' at qIdx, or
// -1 when the ternary shape is ambiguous.
func ternaryColon(tokens []ts.Token, qIdx int) int {
	depth := 0
	for k := qIdx + 1; k < len(tokens); k++ {
		switch tokens[k].Value {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			if depth == 0 {
				return -1
			}
			depth--
		case ";":
			if depth == 0 {
				return -1
			}
		case ":":
			if depth == 0 {
				return tokens[k].Offset
			}
		}
	}
	return -1
}

// matchingTokenIndex returns the index of the bracket closing the one at
// openIdx, or -1 when unbalanced.
func matchingTokenIndex(tokens []ts.Token, openIdx int) int {
	if openIdx < 0 || openIdx >= len(tokens) {
		return -1
	}
	var closer string
	switch tokens[openIdx].Value {
	case "(":
		closer = ")"
	case "[":
		closer = "]"
	case "{":
		closer = "}"
	default:
		return -1
	}
	opener := tokens[openIdx].Value
	depth := 0
	for k := openIdx; k < len(tokens); k++ {
		switch tokens[k].Value {
		case opener:
			depth++
		case closer:
			depth--
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

// matchingOpenParenIndex returns the index of the '(' matching the ')' at
// closeIdx, or -1 when unbalanced.
func matchingOpenParenIndex(tokens []ts.Token, closeIdx int) int {
	if closeIdx < 0 || closeIdx >= len(tokens) || tokens[closeIdx].Value != ")" {
		return -1
	}
	depth := 0
	for k := closeIdx; k >= 0; k-- {
		switch tokens[k].Value {
		case ")":
			depth++
		case "(":
			depth--
			if depth == 0 {
				return k
			}
		}
	}
	return -1
}

// ExistenceOnlyRule reports unit tests where assertions only check presence or truthiness without asserting state.
type ExistenceOnlyRule struct{}

// NewExistenceOnlyRule creates an evaluator for GUARD-TS-EXISTENCE-ONLY.
func NewExistenceOnlyRule() *ExistenceOnlyRule {
	return &ExistenceOnlyRule{}
}

// Evaluate checks whether all assertions are existence or truthiness matchers.
func (r *ExistenceOnlyRule) Evaluate(unit parser.TestUnit) []models.Violation {
	tokens := lexTestBody(unit)
	assertions := ts.FindAssertions(unit.RawBody, tokens, 0, len(unit.RawBody), nil)
	if len(assertions) == 0 {
		return nil
	}

	for _, a := range assertions {
		isExist := false
		switch a.Matcher {
		case "toBeDefined", "toBeTruthy", "exist", "ok":
			isExist = true
		case "toBeNull", "toBeUndefined", "toBeFalsy":
			// Check if negated: not.toBeNull, not.toBeUndefined, not.toBeFalsy
			for _, m := range a.MatcherChain {
				if m == "not" {
					isExist = true
					break
				}
			}
		}
		if !isExist {
			return nil
		}
	}

	return []models.Violation{{
		RuleID:     "GUARD-TS-EXISTENCE-ONLY",
		Severity:   models.SeverityAdvisory,
		FilePath:   unit.FilePath,
		LineNumber: unit.StartLine,
		Column:     1,
		Message:    fmt.Sprintf("Existence-only assertions in '%s': test only asserts presence or truthiness without checking values.", unit.Name),
		Suggestion: "Assert exact expected values, types, or domain state rather than bare existence or truthiness.",
	}}
}

// SelfMockRule reports tests where assertions only verify a mock that was created and directly invoked by the test itself.
type SelfMockRule struct{}

// NewSelfMockRule creates an evaluator for GUARD-TS-SELF-MOCK.
func NewSelfMockRule() *SelfMockRule {
	return &SelfMockRule{}
}

// Evaluate checks whether the test asserts directly on mocks created within its own body.
func (r *SelfMockRule) Evaluate(unit parser.TestUnit) []models.Violation {
	tokens := lexTestBody(unit)
	mockIdents := mockDeclarations(tokens)
	if len(mockIdents) == 0 {
		return nil
	}

	assertions := ts.FindAssertions(unit.RawBody, tokens, 0, len(unit.RawBody), nil)
	if len(assertions) == 0 {
		return nil
	}

	directCalls := directMockCalls(tokens, mockIdents)
	resultVars := mockResultVariables(tokens, mockIdents)

	for _, a := range assertions {
		subject := strings.TrimSpace(a.Subject)
		if resultVars[subject] {
			continue
		}

		isSelfMock := false
		for mock := range mockIdents {
			// Direct call: expect(mock()) or expect(repo.find())
			if strings.HasPrefix(subject, mock+"(") {
				isSelfMock = true
				break
			}
			// Call verification matcher on a mock this body invoked itself.
			if subject == mock && directCalls[mock] && mockCallMatcherRegex.MatchString(a.Matcher) {
				isSelfMock = true
				break
			}
		}

		if !isSelfMock {
			return nil
		}
	}

	return []models.Violation{{
		RuleID:     "GUARD-TS-SELF-MOCK",
		Severity:   models.SeverityWarning,
		FilePath:   unit.FilePath,
		LineNumber: unit.StartLine,
		Column:     1,
		Message:    fmt.Sprintf("Self-mock assertion in '%s': test asserts on a mock it created and directly invoked itself.", unit.Name),
		Suggestion: "Invoke system-under-test methods that use the mock collaborator rather than directly asserting on the mock.",
	}}
}

// mockDeclarations returns the mock handles declared in the token stream: a
// variable bound directly to a mock factory ("save") or an object-literal
// property bound to one ("repo.find"). Comments and strings never yield
// declarations because they are single tokens.
func mockDeclarations(tokens []ts.Token) map[string]bool {
	decls := make(map[string]bool)
	n := len(tokens)
	for i := 0; i+3 < n; i++ {
		switch tokens[i].Value {
		case "const", "let", "var":
		default:
			continue
		}
		if tokens[i+2].Value != "=" {
			continue
		}
		name := tokens[i+1].Value
		if isMockFactoryCall(tokens, i+3) {
			decls[name] = true
			continue
		}
		if tokens[i+3].Value != "{" {
			continue
		}
		closeIdx := matchingTokenIndex(tokens, i+3)
		if closeIdx == -1 {
			continue
		}
		depth := 0
		for k := i + 4; k < closeIdx; k++ {
			switch tokens[k].Value {
			case "{":
				depth++
				continue
			case "}":
				depth--
				continue
			}
			if depth != 0 || tokens[k].Type != ts.TokenIdent || k+3 >= n {
				continue
			}
			if tokens[k+1].Value == ":" && isMockFactoryCall(tokens, k+2) {
				decls[name+"."+tokens[k].Value] = true
			}
		}
	}
	return decls
}

// isMockFactoryCall reports whether tokens[i] starts a jest.fn, vi.fn,
// sinon.stub or sinon.spy call.
func isMockFactoryCall(tokens []ts.Token, i int) bool {
	if i+3 >= len(tokens) || tokens[i+1].Value != "." || tokens[i+3].Value != "(" {
		return false
	}
	switch tokens[i].Value {
	case "jest", "vi":
		return tokens[i+2].Value == "fn"
	case "sinon":
		return tokens[i+2].Value == "stub" || tokens[i+2].Value == "spy"
	}
	return false
}

// directMockCalls reports which mock handles the test body invokes itself,
// ignoring the declaration, member access on another object, and bodies of
// nested functions where the identifier may be a shadowed parameter.
func directMockCalls(tokens []ts.Token, mocks map[string]bool) map[string]bool {
	n := len(tokens)
	bodies := functionBodySpans(tokens)
	inFunctionBody := func(idx int) bool {
		for _, b := range bodies {
			if idx > b[0] && idx < b[1] {
				return true
			}
		}
		return false
	}

	calls := make(map[string]bool)
	for i := 0; i < n; i++ {
		if tokens[i].Type != ts.TokenIdent || inFunctionBody(i) {
			continue
		}
		if i > 0 && tokens[i-1].Value == "." {
			continue
		}
		name := tokens[i].Value
		callIdx := i + 1
		if callIdx+2 < n && tokens[callIdx].Value == "." && tokens[callIdx+2].Value == "(" {
			name += "." + tokens[callIdx+1].Value
			callIdx += 2
		}
		if !mocks[name] || callIdx >= n || tokens[callIdx].Value != "(" {
			continue
		}
		if i > 0 && tokens[i-1].Value == "function" {
			continue
		}
		calls[name] = true
	}
	return calls
}

// mockResultVariables returns variables assigned the result of a direct mock
// call, such as `const total = compute()` where compute is a mock.
func mockResultVariables(tokens []ts.Token, mocks map[string]bool) map[string]bool {
	results := make(map[string]bool)
	n := len(tokens)
	for i := 0; i+4 < n; i++ {
		switch tokens[i].Value {
		case "const", "let", "var":
		default:
			continue
		}
		if tokens[i+2].Value != "=" || tokens[i+3].Type != ts.TokenIdent {
			continue
		}
		name := tokens[i+3].Value
		callIdx := i + 4
		if callIdx+2 < n && tokens[callIdx].Value == "." && tokens[callIdx+2].Value == "(" {
			name += "." + tokens[callIdx+1].Value
			callIdx += 2
		}
		if mocks[name] && callIdx < n && tokens[callIdx].Value == "(" {
			results[tokens[i+1].Value] = true
		}
	}
	return results
}

// functionBodySpans returns token ranges of nested function and method bodies,
// so identifier references inside them are not attributed to the test scope.
// The callback wrapper that makes up the test itself is not nested and is
// therefore excluded.
func functionBodySpans(tokens []ts.Token) [][2]int {
	n := len(tokens)
	firstBrace := -1
	for i := range n {
		if tokens[i].Value == "{" {
			firstBrace = i
			break
		}
	}

	var spans [][2]int
	for i := 1; i < n; i++ {
		if tokens[i].Value != "{" {
			continue
		}
		switch tokens[i-1].Value {
		case "=>":
		case ")":
			open := matchingOpenParenIndex(tokens, i-1)
			if open < 1 {
				continue
			}
			switch tokens[open-1].Value {
			case "if", "for", "while", "switch", "catch", "with", "do":
				continue
			}
		default:
			continue
		}
		closeIdx := matchingTokenIndex(tokens, i)
		if closeIdx == -1 {
			continue
		}
		if i == firstBrace && closeIdx == n-1 {
			continue
		}
		spans = append(spans, [2]int{i, closeIdx})
	}
	return spans
}

// MockedSutRule reports test files that mock the file under test without a factory implementation.
type MockedSutRule struct{}

// NewMockedSutRule creates an evaluator for GUARD-TS-MOCKED-SUT.
func NewMockedSutRule() *MockedSutRule {
	return &MockedSutRule{}
}

// EvaluateFile checks whether a test file auto-mocks its own SUT.
func (r *MockedSutRule) EvaluateFile(filePath string, src string, units []parser.TestUnit) []models.Violation {
	var violations []models.Violation

	baseName := filepath.Base(filePath)
	ext := filepath.Ext(baseName)
	withoutExt := strings.TrimSuffix(baseName, ext)

	var sutBase string
	if strings.HasSuffix(withoutExt, ".test") {
		sutBase = strings.TrimSuffix(withoutExt, ".test")
	} else if strings.HasSuffix(withoutExt, ".spec") {
		sutBase = strings.TrimSuffix(withoutExt, ".spec")
	} else {
		return nil
	}

	matches := jestOrViMockCallRegex.FindAllStringSubmatchIndex(src, -1)
	for _, m := range matches {
		callStart := m[0]
		if !isCodeIndex(src, callStart) {
			continue
		}

		mockPath := src[m[2]:m[3]]
		mockPathBase := filepath.Base(mockPath)
		mockExt := filepath.Ext(mockPathBase)
		mockBase := mockPathBase
		if mockExt == ".ts" || mockExt == ".tsx" || mockExt == ".js" || mockExt == ".jsx" {
			mockBase = strings.TrimSuffix(mockPathBase, mockExt)
		}

		if mockBase != sutBase {
			continue
		}

		// Check if there is a second argument
		openParen := strings.Index(src[callStart:], "(") + callStart
		closeParen := findMatchingParen(src, openParen)
		if closeParen == -1 {
			continue
		}

		args := splitCallArgs(src[openParen+1 : closeParen])
		if len(args) == 1 {
			line := strings.Count(src[:callStart], "\n") + 1
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-TS-MOCKED-SUT",
				Severity:   models.SeverityWarning,
				FilePath:   filePath,
				LineNumber: line,
				Column:     1,
				Message:    fmt.Sprintf("Mocked system under test in '%s': file auto-mocks its own subject '%s'.", baseName, sutBase),
				Suggestion: "Do not mock the system under test in its own test suite; mock only its external dependencies.",
			})
		}
	}

	return violations
}

// ReactSleepRule traps hardcoded setTimeout / sleep calls in React and frontend test bodies.
type ReactSleepRule struct{}

// NewReactSleepRule creates an evaluator for GUARD-REACT-02.
func NewReactSleepRule() *ReactSleepRule {
	return &ReactSleepRule{}
}

// EvaluateFile reports real-timer sleeps in unit tests of one file.
func (r *ReactSleepRule) EvaluateFile(units []parser.TestUnit, fileSrc string) []models.Violation {
	var violations []models.Violation

	cleanFileSrc := blankCommentsAndStrings(fileSrc)
	if fakeTimersRegex.MatchString(cleanFileSrc) {
		return nil
	}

	for _, unit := range units {
		if unit.Kind != "" && unit.Kind != "unit" {
			continue
		}

		matches := setTimeoutRegex.FindAllStringIndex(unit.RawBody, -1)
		for _, match := range matches {
			matchIdx := match[0]
			if !isCodeIndex(unit.RawBody, matchIdx) {
				continue
			}

			// Ignore if lexically inside mock argument list
			if isInsideMockCall(unit.RawBody, matchIdx) {
				continue
			}

			// Ignore if inside a class body
			if isInsideClassBody(unit.RawBody, matchIdx) {
				continue
			}

			openParen := strings.Index(unit.RawBody[matchIdx:], "(") + matchIdx
			closeParen := findMatchingParen(unit.RawBody, openParen)
			if closeParen == -1 {
				continue
			}

			args := splitCallArgs(unit.RawBody[openParen+1 : closeParen])
			if len(args) < 2 {
				// Delay argument is absent -> ignore
				continue
			}

			delay := args[1]
			if !isPositiveDelay(delay) {
				continue
			}

			lineOffset := strings.Count(unit.RawBody[:matchIdx], "\n")
			line := bodyStartLine(unit) + lineOffset

			violations = append(violations, models.Violation{
				RuleID:     "GUARD-REACT-02",
				Severity:   models.SeverityWarning,
				FilePath:   unit.FilePath,
				LineNumber: line,
				Column:     1,
				Message:    fmt.Sprintf("Arbitrary sleep/setTimeout fragility in '%s': setTimeout called with delay '%s' in test.", unit.Name, delay),
				Suggestion: "Use React Testing Library's waitFor(...) or async findBy* queries instead of fixed setTimeout sleeps.",
			})

			// At most one violation per unit
			break
		}
	}

	return violations
}

func isCodeIndex(src string, targetIdx int) bool {
	n := len(src)
	if targetIdx < 0 || targetIdx >= n {
		return false
	}

	for i := 0; i < n; {
		if i == targetIdx {
			return true
		}
		if i > targetIdx {
			return false
		}

		// Single-line comment
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			start := i
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			if targetIdx >= start && targetIdx < i {
				return false
			}
			continue
		}

		// Multi-line block comment
		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			start := i
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i += 2
			if targetIdx >= start && targetIdx < i {
				return false
			}
			continue
		}

		// Strings
		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			start := i
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i++
				} else if src[i] == '\n' && quote != '`' {
					break
				}
				i++
			}
			if i < n && src[i] == quote {
				i++
			}
			if targetIdx >= start && targetIdx < i {
				return false
			}
			continue
		}

		i++
	}

	return true
}

func findMatchingParen(src string, openIndex int) int {
	if openIndex < 0 || openIndex >= len(src) || src[openIndex] != '(' {
		return -1
	}

	parenCount := 0
	n := len(src)

	for i := openIndex; i < n; i++ {
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}

		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i++
			continue
		}

		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i++
				} else if src[i] == '\n' && quote != '`' {
					break
				}
				i++
			}
			continue
		}

		if src[i] == '(' {
			parenCount++
		} else if src[i] == ')' {
			parenCount--
			if parenCount == 0 {
				return i
			}
		}
	}

	return -1
}

func splitCallArgs(src string) []string {
	var args []string
	n := len(src)
	start := 0
	parenDepth := 0
	braceDepth := 0
	bracketDepth := 0

	for i := 0; i < n; i++ {
		c := src[i]

		if i+1 < n && c == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}

		if i+1 < n && c == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i++
			continue
		}

		if c == '\'' || c == '"' || c == '`' {
			quote := c
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i++
				} else if src[i] == '\n' && quote != '`' {
					break
				}
				i++
			}
			continue
		}

		if c == '(' {
			parenDepth++
		} else if c == ')' {
			parenDepth--
		} else if c == '{' {
			braceDepth++
		} else if c == '}' {
			braceDepth--
		} else if c == '[' {
			bracketDepth++
		} else if c == ']' {
			bracketDepth--
		} else if c == ',' && parenDepth == 0 && braceDepth == 0 && bracketDepth == 0 {
			args = append(args, strings.TrimSpace(src[start:i]))
			start = i + 1
		}
	}

	if start < n {
		args = append(args, strings.TrimSpace(src[start:]))
	}

	return args
}

func isPositiveDelay(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	if val, err := strconv.ParseFloat(s, 64); err == nil {
		return val > 0
	}

	if s == "0" || s == "0.0" {
		return false
	}

	if jsIdentRegex.MatchString(s) {
		return true
	}

	return true
}

func blankCommentsAndStrings(src string) string {
	b := []byte(src)
	n := len(b)
	for i := 0; i < n; {
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
		if b[i] == '\'' || b[i] == '"' || b[i] == '`' {
			delim := b[i]
			b[i] = ' '
			i++
			for i < n {
				if b[i] == '\\' {
					b[i] = ' '
					if i+1 < n && b[i+1] != '\n' && b[i+1] != '\r' {
						b[i+1] = ' '
					}
					i += 2
					continue
				}
				if b[i] == delim {
					b[i] = ' '
					i++
					break
				}
				if (delim == '\'' || delim == '"') && b[i] == '\n' {
					break
				}
				if b[i] != '\n' && b[i] != '\r' {
					b[i] = ' '
				}
				i++
			}
			continue
		}
		i++
	}
	return string(b)
}

func isInsideMockCall(src string, targetIdx int) bool {
	matches := mockCallRegex.FindAllStringIndex(src, -1)
	for _, m := range matches {
		openParen := m[1] - 1
		closeParen := findMatchingParen(src, openParen)
		if closeParen != -1 && targetIdx > openParen && targetIdx < closeParen {
			return true
		}
	}
	return false
}

func isInsideClassBody(src string, targetIdx int) bool {
	matches := classDeclRegex.FindAllStringIndex(src, -1)
	for _, m := range matches {
		openBraceRel := strings.Index(src[m[0]:], "{")
		if openBraceRel == -1 {
			continue
		}
		openBrace := m[0] + openBraceRel
		closeBrace := findMatchingBrace(src, openBrace)
		if closeBrace != -1 && targetIdx > openBrace && targetIdx < closeBrace {
			return true
		}
	}
	return false
}

func findMatchingBrace(src string, openIndex int) int {
	if openIndex < 0 || openIndex >= len(src) || src[openIndex] != '{' {
		return -1
	}

	braceCount := 0
	n := len(src)

	for i := openIndex; i < n; i++ {
		if i+1 < n && src[i] == '/' && src[i+1] == '/' {
			i += 2
			for i < n && src[i] != '\n' {
				i++
			}
			continue
		}

		if i+1 < n && src[i] == '/' && src[i+1] == '*' {
			i += 2
			for i+1 < n && !(src[i] == '*' && src[i+1] == '/') {
				i++
			}
			i++
			continue
		}

		if src[i] == '\'' || src[i] == '"' || src[i] == '`' {
			quote := src[i]
			i++
			for i < n && src[i] != quote {
				if src[i] == '\\' {
					i++
				} else if src[i] == '\n' && quote != '`' {
					break
				}
				i++
			}
			continue
		}

		if src[i] == '{' {
			braceCount++
		} else if src[i] == '}' {
			braceCount--
			if braceCount == 0 {
				return i
			}
		}
	}

	return -1
}
