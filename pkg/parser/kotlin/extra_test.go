package kotlin_test

import (
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
