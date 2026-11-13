package swift_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser/swift"
)

func TestSwiftAdapter_LocateTests_StylesTests(t *testing.T) {
	fixturePath := filepath.Join("testdata", "StylesTests.swift")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	adapter := swift.NewAdapter()
	units, err := adapter.LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	// 9 units (testSkipped excluded)
	if len(units) != 9 {
		t.Fatalf("expected 9 units, got %d", len(units))
	}

	for _, u := range units {
		if u.Name == "testSkipped" {
			t.Errorf("testSkipped should have been excluded")
		}
		if u.Name == "testTrulyEmpty" || u.Name == "swiftTestingEmpty" {
			if len(u.Assertions) != 0 {
				t.Errorf("expected %s to have 0 assertions, got %v", u.Name, u.Assertions)
			}
		} else {
			if len(u.Assertions) == 0 {
				t.Errorf("expected %s to have assertions, got 0", u.Name)
			}
		}
	}
}

func TestSwiftAdapter_AdversarialTests(t *testing.T) {
	fixturePath := filepath.Join("testdata", "AdversarialTests.swift")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture: %v", err)
	}

	adapter := swift.NewAdapter()
	units, err := adapter.LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests error: %v", err)
	}

	// Expect exactly 10 units:
	// testAsyncThrows, testMainActorAttribute, testUsesSameFileHelper,
	// testNimble, testShortTimeout, testPerformanceOfSort, testOnlyPrints,
	// adds, positive, emptySwiftTesting.
	// (disabledTest is @Test(.disabled(...)), so excluded.
	// testFakeInString is inside doc string literal, so excluded.)
	if len(units) != 10 {
		t.Fatalf("expected 10 units, got %d", len(units))
	}

	unitsByName := make(map[string]int)
	for _, u := range units {
		unitsByName[u.Name] = len(u.Assertions)
	}

	for name, assertCount := range unitsByName {
		if name == "testOnlyPrints" || name == "emptySwiftTesting" {
			if assertCount != 0 {
				t.Errorf("expected %s to have 0 assertions, got %d", name, assertCount)
			}
		} else {
			if assertCount == 0 {
				t.Errorf("expected %s to have assertions, got 0", name)
			}
		}
	}

	if _, ok := unitsByName["testPerformanceOfSort"]; !ok {
		t.Fatalf("expected testPerformanceOfSort to be discovered")
	}
	if unitsByName["testPerformanceOfSort"] == 0 {
		t.Errorf("expected testPerformanceOfSort to have assertion (measure call), got 0")
	}
}
