package kotlin_test

import (
	"fmt"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser/kotlin"
)

func TestKotlinAdapter_ParameterizedAndRepeated(t *testing.T) {
	src := `
package test

import org.junit.jupiter.params.ParameterizedTest
import org.junit.jupiter.api.RepeatedTest
import org.junit.jupiter.api.Disabled
import org.junit.jupiter.api.Nested

class Outer {
    @ParameterizedTest
    fun paramTest(x: Int) {
        assert(x > 0)
    }

    @Nested
    class Inner {
        @RepeatedTest(5)
        fun repeated() {
            assertEquals(1, 1)
        }

        @Disabled("reason")
        @RepeatedTest(2)
        fun disabledTest() {
            fail("disabled")
        }
    }
}
`
	adapter := kotlin.NewAdapter()
	units, err := adapter.LocateTests([]byte(src), "OuterTest.kt")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}

	if units[0].Name != "paramTest" || units[0].Scope != "Outer" {
		t.Errorf("unexpected unit 0: %+v", units[0])
	}
	if units[1].Name != "repeated" || units[1].Scope != "Outer > Inner" {
		t.Errorf("unexpected unit 1: %+v", units[1])
	}
}

func TestKotlinAdapter_KotestStyles(t *testing.T) {
	src := `
package test

import io.kotest.core.spec.style.FunSpec
import io.kotest.matchers.shouldBe

class MySpec : FunSpec({
    test("addition") {
        1 + 1 shouldBe 2
    }

    it("subtraction") {
        2 - 1 shouldBe 1
    }

    should("multiplication") {
        2 * 2 shouldBe 4
    }
})
`
	adapter := kotlin.NewAdapter()
	units, err := adapter.LocateTests([]byte(src), "MySpec.kt")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	if len(units) != 3 {
		t.Fatalf("expected 3 units, got %d", len(units))
	}
	if units[0].Name != "addition" || len(units[0].Assertions) == 0 {
		t.Errorf("unit 0 issue: %+v", units[0])
	}
	if units[1].Name != "subtraction" || len(units[1].Assertions) == 0 {
		t.Errorf("unit 1 issue: %+v", units[1])
	}
	if units[2].Name != "multiplication" || len(units[2].Assertions) == 0 {
		t.Errorf("unit 2 issue: %+v", units[2])
	}
}

func TestKotlinAdapter_HelpersAndTurbine(t *testing.T) {
	src := `
package test

import org.junit.Test
import app.cash.turbine.test

class FlowTest {
    private fun assertCustom(x: Int) {
        assert(x > 0)
    }

    @Test
    fun testWithHelper() {
        assertCustom(10)
    }

    @Test
    fun testTurbine() = runTest {
        flow.test {
            awaitItem()
            awaitComplete()
        }
    }
}
`
	adapter := kotlin.NewAdapter()
	units, err := adapter.LocateTests([]byte(src), "FlowTest.kt")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}

	if len(units[0].Assertions) == 0 {
		t.Errorf("expected testWithHelper to have assertion from helper, got 0")
	}
	if len(units[1].Assertions) == 0 {
		t.Errorf("expected testTurbine to have assertion awaitComplete, got 0")
	}
}

// Test Intent Record:
// ID: kotlin-generic-fail-reject-adjacent-comparison
// Requirements: review-corrective, recall.
// Owner: kotlin adapter; Contract: comparison expressions never count as fail assertions.
// Fault: scanning through adjacent expression statements can credit a later comparison as a generic call.
// Layer: unit; Cases: repeated newline-separated `fail < limit` expressions.
// Command: go test -count=1 ./pkg/parser/kotlin.
// Oracle: Kotlin type syntax requires a separator between completed adjacent type tokens.
// Boundaries: real Kotlin adapter; no mocks.
// TestKotlinAdapter_GenericFailAssertions pins which fail(...) spellings count
// as an explicit failure assertion. A generic or qualified fail call must not
// leave a test looking hollow, while a mention in a string/comment, a bare
// reference, or a non-invocation must not manufacture an assertion.
func TestKotlinAdapter_GenericFailAssertions(t *testing.T) {
	const wrapper = `package test

import org.junit.Test

class GenericFailTest {
    @Test
    fun failRecognition() {
%s
    }
}
`

	cases := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "plain fail call",
			body: `        fail("boom")`,
			want: true,
		},
		{
			name: "generic fail call",
			body: `        fail<Any>("boom")`,
			want: true,
		},
		{
			name: "generic fail call without arguments",
			body: `        fail<Any>()`,
			want: true,
		},
		{
			name: "nested type arguments",
			body: `        fail<Map<String, List<Int>>>("boom")`,
			want: true,
		},
		{
			name: "function type argument",
			body: `        fail<(String) -> Unit>("expected rejection")`,
			want: true,
		},
		{
			name: "nested function type argument",
			body: `        fail<Map<String, (Int) -> Unit>>("boom")`,
			want: true,
		},
		{
			name: "star projection type argument",
			body: `        fail<List<*>>("boom")`,
			want: true,
		},
		{
			name: "nested star projection type argument",
			body: `        fail<Map<String, List<*>>>("boom")`,
			want: true,
		},
		{
			name: "out projection type argument",
			body: `        fail<MutableList<out Number>>("boom")`,
			want: true,
		},
		{
			name: "in projection type argument",
			body: `        fail<MutableList<in String>>("boom")`,
			want: true,
		},
		{
			name: "backticked out is a type name",
			body: "        fail<`out`>(\"boom\")",
			want: true,
		},
		{
			name: "qualified generic call",
			body: `        someReceiver.fail<Any>("boom")`,
			want: true,
		},
		{
			name: "whitespace variation",
			body: `        fail < Any > ( "boom" )`,
			want: true,
		},
		{
			name: "comment inside type argument",
			body: `        fail<Any /* > */>("boom")`,
			want: true,
		},
		{
			name: "comment between name and call",
			body: `        fail /* generic */ <Any> /* invocation */ ( "boom" )`,
			want: true,
		},
		{
			name: "unterminated type argument",
			body: `        fail<Any("boom")`,
			want: false,
		},
		{
			name: "generic reference without invocation",
			body: `        val f = fail<Any>`,
			want: false,
		},
		{
			name: "generic reference followed by unrelated call",
			body: "        val f = fail<Any>\n        listOf(1)",
			want: false,
		},
		{
			name: "newline-separated comparisons",
			body: "        val fail = 0\n        val limit = 1\n        fail < limit\n        fail < limit\n        fail < limit",
			want: false,
		},
		{
			name: "comparison over statements",
			body: "        val fail = 1\n        val lower = fail < 2\n        val upper = 3 > (1)",
			want: false,
		},
		{
			name: "comparison ending at statement separator",
			body: `        val fail = 1; val lower = fail < limit;`,
			want: false,
		},
		{
			name: "generic reference with lambda but no invocation",
			body: `        fail<Any> { }`,
			want: false,
		},
		{
			name: "bare fail reference",
			body: `        val f = fail`,
			want: false,
		},
		{
			name: "string literal mention",
			body: `        val msg = "fail<Any>(\"boom\")"`,
			want: false,
		},
		{
			name: "line comment mention",
			body: `        // fail<Any>("boom")`,
			want: false,
		},
		{
			name: "block comment mention",
			body: `        /* fail<Any>("boom") */`,
			want: false,
		},
		{
			name: "identifier with fail prefix",
			body: `        failure<Any>("boom")`,
			want: false,
		},
		{
			name: "star after completed type token",
			body: `        fail<List<String *>>("boom")`,
			want: false,
		},
		{
			name: "variance modifier without type",
			body: `        fail<out>("boom")`,
			want: false,
		},
		{
			name: "comparison with multiplication",
			body: "        val fail = 1\n        val scaled = fail < limit * 2",
			want: false,
		},
		{
			name: "hollow test",
			body: `        val x = 1`,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := fmt.Sprintf(wrapper, tc.body)
			units, err := kotlin.NewAdapter().LocateTests([]byte(src), "GenericFailTest.kt")
			if err != nil {
				t.Fatalf("LocateTests error: %v", err)
			}
			if len(units) != 1 {
				t.Fatalf("expected 1 unit, got %d", len(units))
			}

			got := false
			for _, a := range units[0].Assertions {
				if a == "fail" || a == ".fail" {
					got = true
					break
				}
			}
			if got != tc.want {
				t.Errorf("fail assertion credited = %v, want %v (assertions: %v)", got, tc.want, units[0].Assertions)
			}
		})
	}
}

// Test Intent Record:
// ID: kotlin-expression-bodied-asserting-helpers
// Requirements: review-corrective, recall.
// Owner: kotlin adapter; Contract: a same-file helper that asserts in an expression body earns credit for its callers.
// Fault: findAssertingHelpers only read brace bodies, so `fun helper(...) = runBlocking(...) { asserts }` was invisible and its callers looked hollow.
// Layer: unit; Cases: runBlocking expression body, direct expression assertion, helper without assertions, assertions only in literals/comments, bare reference, cross-file helper.
// Command: go test -count=1 ./pkg/parser/kotlin.
// Oracle: only an in-file helper whose own body contributes an assertion may credit a caller; the expression-body scan stops at the line break so it cannot borrow a later declaration's body.
// Boundaries: real Kotlin adapter; no mocks.
func TestKotlinAdapter_ExpressionBodiedAssertingHelpers(t *testing.T) {
	src := `package test

import kotlinx.coroutines.runBlocking
import org.junit.Test

class AppTest {
    private fun checkZip(url: String) = runBlocking(Dispatchers.IO) {
        val outFile = java.io.File(url)
        assertTrue("exists", outFile.exists())
        assertEquals(1, outFile.length())
    }

    private fun directAssert(actual: Int) = assertEquals(1, actual)

    private fun noAssertions(url: String) = runBlocking(Dispatchers.IO) {
        println(url)
    }

    private fun literalOnly(url: String) = runBlocking(Dispatchers.IO) {
        println("assertTrue(true)")
        // assertEquals(1, 1)
    }

    private fun bareReference() = checkZip

    @Test
    fun testZip() =
        checkZip("url") { file, _, _ ->
            file.close()
        }

    @Test
    fun testDirect() = directAssert(1)

    @Test
    fun testHollowHelper() = noAssertions("url")

    @Test
    fun testLiteralOnly() = literalOnly("url")

    @Test
    fun testBareReference() = bareReference()

    @Test
    fun testCrossFileHelper() = external.fileHelper("url")
}
`
	units, err := kotlin.NewAdapter().LocateTests([]byte(src), "AppTest.kt")
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	want := []struct {
		name       string
		assertions []string
		hollow     bool
	}{
		{name: "testZip", assertions: []string{"checkZip"}},
		{name: "testDirect", assertions: []string{"directAssert"}},
		{name: "testHollowHelper", hollow: true},
		{name: "testLiteralOnly", hollow: true},
		{name: "testBareReference", hollow: true},
		{name: "testCrossFileHelper", hollow: true},
	}
	if len(units) != len(want) {
		t.Fatalf("expected %d units, got %d: %+v", len(want), len(units), names(units))
	}
	for i, w := range want {
		unit := units[i]
		if unit.Name != w.name || unit.Scope != "AppTest" {
			t.Errorf("unit %d: expected name=%q scope=AppTest, got name=%q scope=%q", i, w.name, unit.Name, unit.Scope)
		}
		if w.hollow {
			if len(unit.Assertions) != 0 {
				t.Errorf("unit %q: expected no assertions, got %v", w.name, unit.Assertions)
			}
			continue
		}
		found := false
		for _, a := range unit.Assertions {
			if a == w.assertions[0] {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("unit %q: expected helper assertion %q, got %v", w.name, w.assertions[0], unit.Assertions)
		}
	}
}
