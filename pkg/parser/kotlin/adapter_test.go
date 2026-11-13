package kotlin_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser/kotlin"
)

func TestKotlinAdapter_LocateTests_StylesTest(t *testing.T) {
	fixturePath := filepath.Join("testdata", "StylesTest.kt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}

	adapter := kotlin.NewAdapter()
	units, err := adapter.LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests failed: %v", err)
	}

	// 10 units expected (ignored excluded)
	if len(units) != 10 {
		t.Fatalf("expected 10 units, got %d", len(units))
	}

	unitsByName := make(map[string]bool)
	for _, u := range units {
		unitsByName[u.Name] = true
		if u.Name == "ignored" {
			t.Errorf("expected @Ignore test 'ignored' to be excluded")
		}
	}

	if !unitsByName["backtick name with spaces"] {
		t.Errorf("expected test named 'backtick name with spaces' without backticks, got names: %+v", unitsByName)
	}

	// Only trulyEmpty should have zero assertions
	for _, u := range units {
		if u.Name == "trulyEmpty" {
			if len(u.Assertions) != 0 {
				t.Errorf("expected trulyEmpty to have 0 assertions, got %v", u.Assertions)
			}
		} else {
			if len(u.Assertions) == 0 {
				t.Errorf("expected %s to have assertions, got 0", u.Name)
			}
		}
		if u.Scope != "StylesTest" {
			t.Errorf("expected scope 'StylesTest' for %s, got %q", u.Name, u.Scope)
		}
	}
}

func TestKotlinAdapter_LocateTests_CacheTest(t *testing.T) {
	fixturePath := filepath.Join("testdata", "CacheTest.kt")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("failed to read fixture file: %v", err)
	}

	adapter := kotlin.NewAdapter()
	units, err := adapter.LocateTests(content, fixturePath)
	if err != nil {
		t.Fatalf("LocateTests failed: %v", err)
	}

	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}

	for _, u := range units {
		if len(u.Assertions) == 0 {
			t.Errorf("expected %s to have assertions, got 0", u.Name)
		}
	}
}
