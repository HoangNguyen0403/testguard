package typescript

import (
	"regexp"
	"strings"
)

// AssertionInfo captures the classified properties of a real test assertion.
type AssertionInfo struct {
	Kind         string   // "expect", "expect.soft", "expect.poll", "assert", "sinon.assert", "should", "expectTypeOf", "expectObservable", "expectSubscriptions", "member:<name>", "helper:<name>"
	Raw          string   // full assertion expression text
	Subject      string   // target or expression being asserted
	MatcherChain []string // property and matcher chain, e.g. ["resolves", "toEqual"]
	Matcher      string   // final matcher name, e.g. "toEqual"
	StartOffset  int      // byte offset of assertion start
	EndOffset    int      // byte offset of assertion end
	Line         int      // 1-based line number
}

// chaiGetterTerminals lists chai, sinon-chai and chai-as-promised chain endings
// that assert without a call: expect(x).to.be.empty, .to.have.been.calledOnce,
// await expect(p).to.be.rejected. Naming the terminals keeps arbitrary property
// access from being credited as an assertion.
var chaiGetterTerminals = map[string]bool{
	"exist": true, "ok": true, "true": true, "false": true, "null": true,
	"undefined": true, "empty": true, "NaN": true, "finite": true,
	"extensible": true, "sealed": true, "frozen": true, "arguments": true,
	"called": true, "calledOnce": true, "calledTwice": true, "calledThrice": true,
	"rejected": true, "fulfilled": true, "pending": true,
}

// memberAssertionNames lists verification methods that hang off a receiver:
// supertest `.expect(200)`, testdouble and Angular `.verify()`, Angular
// `.expectOne()`, fast-check `.assert(...)`.
var memberAssertionNames = map[string]bool{
	"expect": true, "verify": true, "expectOne": true, "assert": true,
}

// assertingHelperName matches imported or local assertion helpers by the
// assert*/expect*/verify* convention the Go and Kotlin adapters already credit.
var assertingHelperName = regexp.MustCompile(`^(assert|expect|verify)[A-Z_]`)

// helperScope is the source range in which one asserting-helper declaration is
// visible: a describe block, or the whole file for a top-level helper.
type helperScope struct {
	start int
	end   int
}

// helperIndex maps an asserting helper name to the ranges where a declaration of
// that name can be called, so a helper declared inside one describe block is not
// credited from a sibling block.
type helperIndex map[string][]helperScope

// fileAnalysis carries the per-file facts that per-test assertion classification
// needs, so the whole token stream is scanned once per file.
type fileAnalysis struct {
	helpers    helperIndex
	verifiers  map[string]bool // local names bound to ts-mockito's verify
	namespaces map[string]bool // local names bound to the ts-mockito module
}

// tsMockitoModule is the only module whose bare verify is a verification call;
// ts-mockito's verify returns a wrapper that does nothing until a matcher such
// as .once() is called on it.
const tsMockitoModule = "ts-mockito"

// tsMockitoMatchers lists the matcher calls that turn a ts-mockito
// verify(...) wrapper into an actual assertion.
var tsMockitoMatchers = map[string]bool{
	"once": true, "twice": true, "times": true, "never": true,
	"atLeast": true, "atMost": true, "between": true,
}

// analyzeFile scans the token stream once for the facts every test body needs:
// visible asserting helpers and the local bindings of ts-mockito.
func analyzeFile(src string, tokens []Token) fileAnalysis {
	verifiers, namespaces := collectTsMockitoBindings(tokens)
	facts := fileAnalysis{verifiers: verifiers, namespaces: namespaces}
	facts.helpers = collectHelperScopes(src, tokens, facts)
	return facts
}

// collectTsMockitoBindings returns the local names bound to ts-mockito's verify
// and to the ts-mockito module itself, so an aliased or namespaced import is
// recognised while an unrelated or shadowed `verify` is not.
func collectTsMockitoBindings(tokens []Token) (verifiers, namespaces map[string]bool) {
	verifiers = map[string]bool{}
	namespaces = map[string]bool{}
	n := len(tokens)

	for i := 0; i < n; i++ {
		if tokens[i].Value != "import" || (i > 0 && (tokens[i-1].Value == "." || tokens[i-1].Value == "?.")) {
			continue
		}

		var clauseVerifiers []string
		var clauseNamespaces []string
		inBraces := false
		k := i + 1
		for ; k < n && tokens[k].Value != "from" && tokens[k].Type != TokenString && tokens[k].Type != TokenTemplate; k++ {
			switch tokens[k].Value {
			case "{":
				inBraces = true
			case "}":
				inBraces = false
			case "*":
				if k+2 < n && tokens[k+1].Value == "as" && tokens[k+2].Type == TokenIdent {
					clauseNamespaces = append(clauseNamespaces, tokens[k+2].Value)
					k += 2
				}
			case "verify":
				if !inBraces {
					break
				}
				if k+2 < n && tokens[k+1].Value == "as" && tokens[k+2].Type == TokenIdent {
					clauseVerifiers = append(clauseVerifiers, tokens[k+2].Value)
					k += 2
					continue
				}
				clauseVerifiers = append(clauseVerifiers, "verify")
			}
		}
		if k < n && tokens[k].Value == "from" {
			k++
		}
		if k >= n {
			continue
		}
		specifier := tokens[k]
		if specifier.Type != TokenString && specifier.Type != TokenTemplate {
			continue
		}
		if strings.Trim(specifier.Value, "'\"`") != tsMockitoModule {
			continue
		}
		for _, name := range clauseVerifiers {
			verifiers[name] = true
		}
		for _, name := range clauseNamespaces {
			namespaces[name] = true
		}
	}

	return verifiers, namespaces
}

// findAssertions discovers the assertions inside [startOffset, endOffset) using
// the per-file facts, so per-test classification stays linear in the test body
// size.
func findAssertions(src string, tokens []Token, startOffset, endOffset int, facts fileAnalysis) []AssertionInfo {
	var assertions []AssertionInfo
	n := len(tokens)

	// Filter tokens in range
	startTokIdx := 0
	for startTokIdx < n && tokens[startTokIdx].Offset < startOffset {
		startTokIdx++
	}
	endTokIdx := startTokIdx
	for endTokIdx < n && tokens[endTokIdx].Offset < endOffset {
		endTokIdx++
	}

	subTokens := tokens[startTokIdx:endTokIdx]
	m := len(subTokens)
	matcher := newTokenMatcher(subTokens)

	// Helper to find shadowed helpers in this test body
	shadowed := make(map[string]bool)
	for k := 0; k < m; k++ {
		tok := subTokens[k]
		if tok.Value == "const" || tok.Value == "let" || tok.Value == "var" || tok.Value == "function" {
			if k+1 < m && subTokens[k+1].Type == TokenIdent {
				shadowed[subTokens[k+1].Value] = true
			}
		}
	}

	for i := 0; i < m; i++ {
		tok := subTokens[i]
		prevTokVal := ""
		if i > 0 {
			prevTokVal = subTokens[i-1].Value
		}

		// 1. ts-mockito verification: verify(call).once(), .twice(), .times(n),
		// .never(), .atLeast(n), .atMost(n), .between(a, b). The identifier must
		// be bound to ts-mockito's verify — imported directly, aliased, or read
		// through a ts-mockito namespace — and the matcher call is what makes it
		// an assertion, so it belongs to the recorded span.
		if endIdx, ok := tsMockitoVerificationEnd(subTokens, m, i, facts, shadowed, prevTokVal); ok {
			startIdx := i
			if (prevTokVal == "." || prevTokVal == "?.") && i >= 2 {
				startIdx = i - 2
			}
			astart := subTokens[startIdx].Offset
			aend := subTokens[endIdx].Offset + len(subTokens[endIdx].Value)
			assertions = append(assertions, AssertionInfo{
				Kind:         "ts-mockito",
				Raw:          src[astart:aend],
				Subject:      "verify",
				MatcherChain: []string{"verify"},
				Matcher:      "verify",
				StartOffset:  astart,
				EndOffset:    aend,
				Line:         tok.Line,
			})
			i = endIdx
			continue
		}

		// 2. Verification methods that hang off a receiver: supertest
		// request(app).get('/health').expect(200), testdouble td.verify(...),
		// Angular httpMock.verify()/expectOne(...), fast-check fc.assert(...).
		if memberAssertionNames[tok.Value] && (prevTokVal == "." || prevTokVal == "?.") {
			if i+1 < m && subTokens[i+1].Value == "(" {
				closeParen := matcher.closing(i + 1)
				if closeParen != -1 {
					astart := tok.Offset
					aend := subTokens[closeParen].Offset + len(subTokens[closeParen].Value)
					raw := src[astart:aend]
					assertions = append(assertions, AssertionInfo{
						Kind:         "member:" + tok.Value,
						Raw:          raw,
						Subject:      "receiver",
						MatcherChain: []string{tok.Value},
						Matcher:      tok.Value,
						StartOffset:  astart,
						EndOffset:    aend,
						Line:         tok.Line,
					})
					i = closeParen
					continue
				}
			}
		}

		// 2. expect(...) / expect.soft(...) / expect.poll(...) / expect<T>(...)
		if tok.Value == "expect" && prevTokVal != "." && prevTokVal != "?." && !shadowed["expect"] {
			kind := "expect"
			callStartIdx := i + 1

			// Check for expect.soft or expect.poll
			if i+2 < m && subTokens[i+1].Value == "." && (subTokens[i+2].Value == "soft" || subTokens[i+2].Value == "poll") {
				kind = "expect." + subTokens[i+2].Value
				callStartIdx = i + 3
			} else if i+2 < m && subTokens[i+1].Value == "." {
				// expect.assertions, expect.hasAssertions, expect.extend, expect.any, etc.
				// Do not count!
				continue
			} else if i+1 < m && subTokens[i+1].Value == "<" {
				// Typed expect: expect<number>(value).toBe(3)
				if closeAngle := matcher.closing(i + 1); closeAngle != -1 {
					callStartIdx = closeAngle + 1
				}
			}

			if callStartIdx < m && subTokens[callStartIdx].Value == "(" {
				closeParen := matcher.closing(callStartIdx)
				if closeParen == -1 {
					continue
				}

				// Extract subject
				subjStart := subTokens[callStartIdx].Offset + 1
				subjEnd := subTokens[closeParen].Offset
				subject := ""
				if subjEnd >= subjStart && subjEnd <= len(src) {
					subject = strings.TrimSpace(src[subjStart:subjEnd])
				}

				// Check if followed by member chain: .toBe(...), .resolves..., .to.equal(...)
				nextIdx := closeParen + 1
				if nextIdx >= m || (subTokens[nextIdx].Value != "." && subTokens[nextIdx].Value != "?.") {
					// Bare expect(x) without member access -> do not count!
					continue
				}

				// Collect member chain
				var chain []string
				chainEndTok := closeParen
				hasCall := false

				curr := nextIdx
				for curr < m {
					if subTokens[curr].Value == "." || subTokens[curr].Value == "?." {
						if curr+1 < m && (subTokens[curr+1].Type == TokenIdent || subTokens[curr+1].Type == TokenKeyword) {
							prop := subTokens[curr+1].Value
							chain = append(chain, prop)
							curr += 2
							chainEndTok = curr - 1

							// Check if followed by call: prop(...)
							if curr < m && subTokens[curr].Value == "(" {
								hasCall = true
								argClose := matcher.closing(curr)
								if argClose != -1 {
									chainEndTok = argClose
									curr = argClose + 1
								} else {
									break
								}
							}
							continue
						}
					}
					break
				}

				// Chai style can end in a property getter without a call:
				// .to.be.empty, .to.have.been.calledOnce, .to.be.rejected.
				isChaiGetter := false
				if len(chain) > 0 {
					isChaiGetter = chaiGetterTerminals[chain[len(chain)-1]]
				}

				if hasCall || isChaiGetter {
					astart := tok.Offset
					aend := subTokens[chainEndTok].Offset + len(subTokens[chainEndTok].Value)
					raw := src[astart:aend]
					finalMatcher := ""
					if len(chain) > 0 {
						finalMatcher = chain[len(chain)-1]
					}
					assertions = append(assertions, AssertionInfo{
						Kind:         kind,
						Raw:          raw,
						Subject:      subject,
						MatcherChain: chain,
						Matcher:      finalMatcher,
						StartOffset:  astart,
						EndOffset:    aend,
						Line:         tok.Line,
					})
					i = chainEndTok
					continue
				}
			}
		}

		// 3. assert(...) or assert.<method>(...) or sinon.assert.<method>(...)
		if tok.Value == "assert" && !shadowed["assert"] {
			isSinon := (prevTokVal == "." && i >= 2 && subTokens[i-2].Value == "sinon")
			if isSinon || (prevTokVal != "." && prevTokVal != "?.") {
				kind := "assert"
				if isSinon {
					kind = "sinon.assert"
				}

				if i+1 < m && subTokens[i+1].Value == "(" {
					// assert(...)
					closeParen := matcher.closing(i + 1)
					if closeParen != -1 {
						astart := tok.Offset
						if isSinon {
							astart = subTokens[i-2].Offset
						}
						aend := subTokens[closeParen].Offset + 1
						subjStart := subTokens[i+1].Offset + 1
						subjEnd := subTokens[closeParen].Offset
						subject := ""
						if subjEnd >= subjStart && subjEnd <= len(src) {
							subject = strings.TrimSpace(src[subjStart:subjEnd])
						}
						assertions = append(assertions, AssertionInfo{
							Kind:         kind,
							Raw:          src[astart:aend],
							Subject:      subject,
							MatcherChain: []string{"assert"},
							Matcher:      "assert",
							StartOffset:  astart,
							EndOffset:    aend,
							Line:         tok.Line,
						})
						i = closeParen
						continue
					}
				} else if i+2 < m && subTokens[i+1].Value == "." && subTokens[i+2].Type == TokenIdent {
					// assert.<method>(...)
					methodName := subTokens[i+2].Value
					if i+3 < m && subTokens[i+3].Value == "(" {
						closeParen := matcher.closing(i + 3)
						if closeParen != -1 {
							astart := tok.Offset
							if isSinon {
								astart = subTokens[i-2].Offset
							}
							aend := subTokens[closeParen].Offset + 1
							subjStart := subTokens[i+3].Offset + 1
							subjEnd := subTokens[closeParen].Offset
							subject := ""
							if subjEnd >= subjStart && subjEnd <= len(src) {
								subject = strings.TrimSpace(src[subjStart:subjEnd])
							}
							assertions = append(assertions, AssertionInfo{
								Kind:         kind,
								Raw:          src[astart:aend],
								Subject:      subject,
								MatcherChain: []string{methodName},
								Matcher:      methodName,
								StartOffset:  astart,
								EndOffset:    aend,
								Line:         tok.Line,
							})
							i = closeParen
							continue
						}
					}
				}
			}
		}

		// 4. chai <expr>.should.<...> or should.<method>(...) or Cypress .should(...)
		if tok.Value == "should" {
			if prevTokVal == "." || prevTokVal == "?." {
				// Expression before .should
				// Can be chai: expr.should.equal(...) or cypress: cy.get(...).should(...)
				var chain []string
				chainEndTok := i

				// Collect chain after should
				curr := i + 1
				hasCall := false

				// Cypress call: .should('be.visible')
				if curr < m && subTokens[curr].Value == "(" {
					hasCall = true
					argClose := matcher.closing(curr)
					if argClose != -1 {
						chain = append(chain, "should")
						chainEndTok = argClose
						curr = argClose + 1
					}
				} else {
					for curr < m {
						if subTokens[curr].Value == "." || subTokens[curr].Value == "?." {
							if curr+1 < m && (subTokens[curr+1].Type == TokenIdent || subTokens[curr+1].Type == TokenKeyword) {
								chain = append(chain, subTokens[curr+1].Value)
								curr += 2
								chainEndTok = curr - 1
								if curr < m && subTokens[curr].Value == "(" {
									hasCall = true
									argClose := matcher.closing(curr)
									if argClose != -1 {
										chainEndTok = argClose
										curr = argClose + 1
									} else {
										break
									}
								}
								continue
							}
						}
						break
					}
				}

				isChaiGetter := false
				if len(chain) > 0 {
					isChaiGetter = chaiGetterTerminals[chain[len(chain)-1]]
				}

				if hasCall || isChaiGetter {
					// Estimate subject as preceding token or paren
					subj := "should"
					if i >= 2 {
						subj = subTokens[i-2].Value
					}
					astart := tok.Offset
					if i >= 2 {
						astart = subTokens[i-2].Offset
					}
					aend := subTokens[chainEndTok].Offset + len(subTokens[chainEndTok].Value)
					finalMatcher := "should"
					if len(chain) > 0 {
						finalMatcher = chain[len(chain)-1]
					}
					assertions = append(assertions, AssertionInfo{
						Kind:         "should",
						Raw:          src[astart:aend],
						Subject:      subj,
						MatcherChain: chain,
						Matcher:      finalMatcher,
						StartOffset:  astart,
						EndOffset:    aend,
						Line:         tok.Line,
					})
					i = chainEndTok
					continue
				}
			} else {
				// should.<method>(...)
				if i+2 < m && subTokens[i+1].Value == "." && subTokens[i+2].Type == TokenIdent {
					methodName := subTokens[i+2].Value
					if i+3 < m && subTokens[i+3].Value == "(" {
						closeParen := matcher.closing(i + 3)
						if closeParen != -1 {
							astart := tok.Offset
							aend := subTokens[closeParen].Offset + 1
							subjStart := subTokens[i+3].Offset + 1
							subjEnd := subTokens[closeParen].Offset
							subject := ""
							if subjEnd >= subjStart && subjEnd <= len(src) {
								subject = strings.TrimSpace(src[subjStart:subjEnd])
							}
							assertions = append(assertions, AssertionInfo{
								Kind:         "should",
								Raw:          src[astart:aend],
								Subject:      subject,
								MatcherChain: []string{methodName},
								Matcher:      methodName,
								StartOffset:  astart,
								EndOffset:    aend,
								Line:         tok.Line,
							})
							i = closeParen
							continue
						}
					}
				}
			}
		}

		// 5. expectTypeOf(...).<...>
		if tok.Value == "expectTypeOf" && (prevTokVal != "." && prevTokVal != "?.") {
			if i+1 < m && subTokens[i+1].Value == "(" {
				closeParen := matcher.closing(i + 1)
				if closeParen != -1 && closeParen+2 < m && subTokens[closeParen+1].Value == "." {
					// Followed by .<method>(...)
					curr := closeParen + 1
					var chain []string
					chainEndTok := closeParen
					for curr < m && (subTokens[curr].Value == "." || subTokens[curr].Value == "?.") {
						if curr+1 < m && subTokens[curr+1].Type == TokenIdent {
							chain = append(chain, subTokens[curr+1].Value)
							curr += 2
							chainEndTok = curr - 1
							if curr < m && subTokens[curr].Value == "(" {
								argClose := matcher.closing(curr)
								if argClose != -1 {
									chainEndTok = argClose
									curr = argClose + 1
								}
							}
						} else {
							break
						}
					}
					if len(chain) > 0 {
						astart := tok.Offset
						aend := subTokens[chainEndTok].Offset + len(subTokens[chainEndTok].Value)
						subjStart := subTokens[i+1].Offset + 1
						subjEnd := subTokens[closeParen].Offset
						subject := ""
						if subjEnd >= subjStart && subjEnd <= len(src) {
							subject = strings.TrimSpace(src[subjStart:subjEnd])
						}
						assertions = append(assertions, AssertionInfo{
							Kind:         "expectTypeOf",
							Raw:          src[astart:aend],
							Subject:      subject,
							MatcherChain: chain,
							Matcher:      chain[len(chain)-1],
							StartOffset:  astart,
							EndOffset:    aend,
							Line:         tok.Line,
						})
						i = chainEndTok
						continue
					}
				}
			}
		}

		// 6. expectObservable(...).<...> or expectSubscriptions(...).<...>
		if (tok.Value == "expectObservable" || tok.Value == "expectSubscriptions") && (prevTokVal != "." && prevTokVal != "?.") {
			kind := tok.Value
			if i+1 < m && subTokens[i+1].Value == "(" {
				closeParen := matcher.closing(i + 1)
				if closeParen != -1 && closeParen+2 < m && subTokens[closeParen+1].Value == "." {
					curr := closeParen + 1
					var chain []string
					chainEndTok := closeParen
					for curr < m && (subTokens[curr].Value == "." || subTokens[curr].Value == "?.") {
						if curr+1 < m && subTokens[curr+1].Type == TokenIdent {
							chain = append(chain, subTokens[curr+1].Value)
							curr += 2
							chainEndTok = curr - 1
							if curr < m && subTokens[curr].Value == "(" {
								argClose := matcher.closing(curr)
								if argClose != -1 {
									chainEndTok = argClose
									curr = argClose + 1
								}
							}
						} else {
							break
						}
					}
					if len(chain) > 0 {
						astart := tok.Offset
						aend := subTokens[chainEndTok].Offset + len(subTokens[chainEndTok].Value)
						subjStart := subTokens[i+1].Offset + 1
						subjEnd := subTokens[closeParen].Offset
						subject := ""
						if subjEnd >= subjStart && subjEnd <= len(src) {
							subject = strings.TrimSpace(src[subjStart:subjEnd])
						}
						assertions = append(assertions, AssertionInfo{
							Kind:         kind,
							Raw:          src[astart:aend],
							Subject:      subject,
							MatcherChain: chain,
							Matcher:      chain[len(chain)-1],
							StartOffset:  astart,
							EndOffset:    aend,
							Line:         tok.Line,
						})
						i = chainEndTok
						continue
					}
				}
			}
		}

		// 7. Same-file asserting helper call: helper(...). The helper index
		// records the block each declaration is visible in, so a helper from a
		// sibling describe block is not credited here.
		if _, ok := helperScopeAt(facts.helpers, tok.Value, tok.Offset); ok && (prevTokVal != "." && prevTokVal != "?.") {
			if !shadowed[tok.Value] && i+1 < m && subTokens[i+1].Value == "(" {
				if closeParen := matcher.closing(i + 1); closeParen != -1 && !isMethodDefinition(subTokens, m, closeParen) {
					astart := tok.Offset
					aend := subTokens[closeParen].Offset + 1
					assertions = append(assertions, AssertionInfo{
						Kind:         "helper:" + tok.Value,
						Raw:          src[astart:aend],
						Subject:      tok.Value,
						MatcherChain: []string{tok.Value},
						Matcher:      tok.Value,
						StartOffset:  astart,
						EndOffset:    aend,
						Line:         tok.Line,
					})
					i = closeParen
					continue
				}
			}
		}

		// 8. Conventional asserting helpers: callee and member names following
		// the assert*/expect*/verify* convention, as the Go and Kotlin adapters
		// already credit (assertValidUser(...), page.expectTitle(...)).
		if tok.Type == TokenIdent && assertingHelperName.MatchString(tok.Value) && !shadowed[tok.Value] {
			declarationName := prevTokVal == "function" || prevTokVal == "const" || prevTokVal == "let" || prevTokVal == "var"
			if !declarationName && i+1 < m && subTokens[i+1].Value == "(" {
				if closeParen := matcher.closing(i + 1); closeParen != -1 && !isMethodDefinition(subTokens, m, closeParen) {
					astart := tok.Offset
					aend := subTokens[closeParen].Offset + 1
					assertions = append(assertions, AssertionInfo{
						Kind:         "helper:" + tok.Value,
						Raw:          src[astart:aend],
						Subject:      tok.Value,
						MatcherChain: []string{tok.Value},
						Matcher:      tok.Value,
						StartOffset:  astart,
						EndOffset:    aend,
						Line:         tok.Line,
					})
					i = closeParen
					continue
				}
			}
		}
	}

	return assertions
}

// tsMockitoVerificationEnd reports the index of the token closing a ts-mockito
// verification call. Bare `verify` counts only when the identifier is bound to
// ts-mockito's verify and not shadowed in the test body; the namespaced form
// counts only for a ts-mockito namespace. Both require a trailing matcher call,
// because the wrapper returned by verify asserts nothing on its own.
func tsMockitoVerificationEnd(subTokens []Token, m, i int, facts fileAnalysis, shadowed map[string]bool, prevTokVal string) (int, bool) {
	tok := subTokens[i]
	if tok.Type != TokenIdent || shadowed[tok.Value] {
		return 0, false
	}

	isMember := prevTokVal == "." || prevTokVal == "?."
	if isMember {
		if tok.Value != "verify" || i < 2 {
			return 0, false
		}
		receiver := subTokens[i-2]
		if !facts.namespaces[receiver.Value] || shadowed[receiver.Value] {
			return 0, false
		}
	} else if !facts.verifiers[tok.Value] {
		return 0, false
	}

	if i+1 >= m || subTokens[i+1].Value != "(" {
		return 0, false
	}
	matcher := newTokenMatcher(subTokens)
	closeParen := matcher.closing(i + 1)
	if closeParen == -1 {
		return 0, false
	}
	dot := closeParen + 1
	if dot+1 >= m || (subTokens[dot].Value != "." && subTokens[dot].Value != "?.") {
		return 0, false
	}
	matcherName := subTokens[dot+1]
	if matcherName.Type != TokenIdent || !tsMockitoMatchers[matcherName.Value] {
		return 0, false
	}
	if dot+2 >= m || subTokens[dot+2].Value != "(" {
		return 0, false
	}
	endIdx := matcher.closing(dot + 2)
	if endIdx == -1 {
		return 0, false
	}
	return endIdx, true
}

// isMethodDefinition reports whether the argument list closing at closeParen is
// followed by a body brace, which makes the call a method definition.
func isMethodDefinition(subTokens []Token, m, closeParen int) bool {
	return closeParen+1 < m && subTokens[closeParen+1].Value == "{"
}

// helperScopeAt reports whether a declaring scope for name contains offset.
func helperScopeAt(helpers helperIndex, name string, offset int) (helperScope, bool) {
	for _, scope := range helpers[name] {
		if offset >= scope.start && offset < scope.end {
			return scope, true
		}
	}
	return helperScope{}, false
}

// FindAssertions discovers and classifies the real assertions inside
// [startOffset, endOffset) of src. assertingHelpers adds helper names that the
// caller has already classified, treated as visible file-wide; the adapter
// passes nil because declaration scopes are derived from the source itself.
func FindAssertions(src string, tokens []Token, startOffset, endOffset int, assertingHelpers map[string]bool) []AssertionInfo {
	facts := analyzeFile(src, tokens)
	for name := range assertingHelpers {
		facts.helpers[name] = append(facts.helpers[name], helperScope{start: 0, end: len(src)})
	}
	return findAssertions(src, tokens, startOffset, endOffset, facts)
}

// collectHelperScopes indexes the asserting helpers declared in the file. Each
// entry records the block enclosing its declaration, so a helper declared in one
// describe block is not credited from a sibling block while staying visible
// throughout its own block regardless of declaration order.
func collectHelperScopes(src string, tokens []Token, facts fileAnalysis) helperIndex {
	matcher := newTokenMatcher(tokens)

	blocks := make(map[int]int, len(tokens))
	var open []int
	for i, tok := range tokens {
		switch tok.Value {
		case "{":
			open = append(open, i)
		case "}":
			if len(open) > 0 {
				blocks[open[len(open)-1]] = i
				open = open[:len(open)-1]
			}
		}
	}

	index := helperIndex{}
	var enclosing []int
	for i, tok := range tokens {
		switch tok.Value {
		case "{":
			enclosing = append(enclosing, i)
			continue
		case "}":
			if len(enclosing) > 0 {
				enclosing = enclosing[:len(enclosing)-1]
			}
			continue
		}
		// `async function helper()` is handled at the `async` token.
		if tok.Value == "function" && i > 0 && tokens[i-1].Value == "async" {
			continue
		}
		name, body := helperBodyAt(src, tokens, matcher, i)
		if name == "" {
			continue
		}
		if len(findAssertions(src, tokens, body.start, body.end, facts)) == 0 {
			continue
		}
		scope := helperScope{start: 0, end: len(src)}
		if len(enclosing) > 0 {
			if closeIdx, ok := blocks[enclosing[len(enclosing)-1]]; ok {
				scope = helperScope{
					start: tokens[enclosing[len(enclosing)-1]].Offset,
					end:   tokens[closeIdx].Offset + 1,
				}
			}
		}
		index[name] = append(index[name], scope)
	}
	return index
}

// skipTypeAnnotation advances past an optional `: Type` annotation and the `=`
// that must follow a const/let/var binding name.
func skipTypeAnnotation(matcher tokenMatcher, tokens []Token, idx int) (int, bool) {
	if idx >= len(tokens) {
		return 0, false
	}
	if tokens[idx].Value == "=" {
		return idx + 1, true
	}
	if tokens[idx].Value != ":" {
		return 0, false
	}
	for k := idx + 1; k < len(tokens); k++ {
		switch tokens[k].Value {
		case "=":
			return k + 1, true
		case "(", "[", "{", "<":
			closeIdx := matcher.closing(k)
			if closeIdx == -1 {
				return 0, false
			}
			k = closeIdx
		case ";", ")", "]", "}":
			return 0, false
		}
	}
	return 0, false
}

// helperBodyAt returns the name and body range of an asserting-helper
// declaration starting at token i: a [async] function declaration or a
// const/let/var binding of a function or arrow, with or without a type
// annotation.
func helperBodyAt(src string, tokens []Token, matcher tokenMatcher, i int) (string, helperScope) {
	n := len(tokens)
	tok := tokens[i]

	if tok.Value == "function" || (tok.Value == "async" && i+2 < n && tokens[i+1].Value == "function") {
		nameIdx := i + 1
		if tok.Value == "async" {
			nameIdx = i + 2
		}
		if nameIdx >= n || tokens[nameIdx].Type != TokenIdent {
			return "", helperScope{}
		}
		openParen := nameIdx + 1
		if openParen >= n || tokens[openParen].Value != "(" {
			return "", helperScope{}
		}
		closeParen := matcher.closing(openParen)
		if closeParen == -1 || closeParen+1 >= n || tokens[closeParen+1].Value != "{" {
			return "", helperScope{}
		}
		closeBrace := matcher.closing(closeParen + 1)
		if closeBrace == -1 {
			return "", helperScope{}
		}
		return tokens[nameIdx].Value, helperScope{
			start: tokens[closeParen+1].Offset,
			end:   tokens[closeBrace].Offset + 1,
		}
	}

	if tok.Value != "const" && tok.Value != "let" && tok.Value != "var" {
		return "", helperScope{}
	}
	if i+1 >= n || tokens[i+1].Type != TokenIdent {
		return "", helperScope{}
	}
	name := tokens[i+1].Value
	rhsIdx, ok := skipTypeAnnotation(matcher, tokens, i+2)
	if !ok {
		return "", helperScope{}
	}
	if rhsIdx < n && tokens[rhsIdx].Value == "async" {
		rhsIdx++
	}
	if rhsIdx >= n {
		return "", helperScope{}
	}

	if tokens[rhsIdx].Value == "function" {
		openParen := rhsIdx + 1
		if openParen >= n || tokens[openParen].Value != "(" {
			return "", helperScope{}
		}
		closeParen := matcher.closing(openParen)
		if closeParen == -1 || closeParen+1 >= n || tokens[closeParen+1].Value != "{" {
			return "", helperScope{}
		}
		closeBrace := matcher.closing(closeParen + 1)
		if closeBrace == -1 {
			return "", helperScope{}
		}
		return name, helperScope{
			start: tokens[closeParen+1].Offset,
			end:   tokens[closeBrace].Offset + 1,
		}
	}

	arrowIdx := -1
	if tokens[rhsIdx].Value == "(" {
		if closeParen := matcher.closing(rhsIdx); closeParen != -1 && closeParen+1 < n && tokens[closeParen+1].Value == "=>" {
			arrowIdx = closeParen + 1
		}
	} else if tokens[rhsIdx].Type == TokenIdent && rhsIdx+1 < n && tokens[rhsIdx+1].Value == "=>" {
		arrowIdx = rhsIdx + 1
	}
	if arrowIdx == -1 || arrowIdx+1 >= n {
		return "", helperScope{}
	}

	bodyStart := arrowIdx + 1
	if tokens[bodyStart].Value == "{" {
		closeBrace := matcher.closing(bodyStart)
		if closeBrace == -1 {
			return "", helperScope{}
		}
		return name, helperScope{
			start: tokens[bodyStart].Offset,
			end:   tokens[closeBrace].Offset + 1,
		}
	}

	// A parenthesized expression body spans to its matching paren so a
	// multi-line body is not truncated at the first line break; any other
	// expression body ends at the first semicolon or line break (ASI).
	startOff := tokens[bodyStart].Offset
	if tokens[bodyStart].Value == "(" {
		if closeParen := matcher.closing(bodyStart); closeParen != -1 {
			return name, helperScope{
				start: startOff,
				end:   tokens[closeParen].Offset + len(tokens[closeParen].Value),
			}
		}
	}
	endOff := startOff
	endTok := bodyStart
	for endTok < n && tokens[endTok].Value != ";" && tokens[endTok].Line == tokens[bodyStart].Line {
		endOff = tokens[endTok].Offset + len(tokens[endTok].Value)
		endTok++
	}
	if endTok < n && tokens[endTok].Value == ";" {
		endOff = tokens[endTok].Offset + 1
	}
	return name, helperScope{start: startOff, end: endOff}
}
