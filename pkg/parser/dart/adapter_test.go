package dart_test

import (
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/parser/dart"
)

func TestLocateBlocTestUnits(t *testing.T) {
	source := []byte(`
void main() {
  group('CustomerCodeBloc', () {
    blocTest<CustomerCodeBloc, CustomerCodeState>(
      'Fetch_WhenValidCustomerCode_EmitsCustomerList',
      seed: () => baseState,
      build: () => CustomerCodeBloc(repo),
      act: (bloc) => bloc.add(const CustomerCodeEvent.fetch()),
      expect: () => [
        baseState.copyWith(isFetching: true),
        baseState.copyWith(isFetching: false, customerCodeList: [mockData]),
      ],
    );

    test('simple unit test', () {
      expect(1 + 1, equals(2));
    });
  });
}
`)

	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests(source, "customer_code_bloc_test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(units) != 2 {
		t.Fatalf("expected 2 test units, got %d", len(units))
	}

	blocTest := units[0]
	if blocTest.Name != "Fetch_WhenValidCustomerCode_EmitsCustomerList" {
		t.Errorf("expected bloc test name to match, got %s", blocTest.Name)
	}
	if blocTest.Kind != "bloc" {
		t.Errorf("expected kind bloc, got %s", blocTest.Kind)
	}
	if blocTest.SeedBody == "" {
		t.Errorf("expected non-empty seed body")
	}
	if blocTest.ExpectBody == "" {
		t.Errorf("expected non-empty expect body")
	}

	simpleTest := units[1]
	if simpleTest.Name != "simple unit test" {
		t.Errorf("expected simple test name to match, got %s", simpleTest.Name)
	}
	if simpleTest.Kind != "unit" {
		t.Errorf("expected kind unit, got %s", simpleTest.Kind)
	}
	if len(simpleTest.Assertions) == 0 {
		t.Errorf("expected simple test to have assertions detected")
	}
}

func TestIsGenerated(t *testing.T) {
	adapter := dart.NewAdapter()
	if !adapter.IsGenerated("foo.freezed.dart", []byte("")) {
		t.Errorf("expected foo.freezed.dart to be detected as generated")
	}
	if !adapter.IsGenerated("bar.g.dart", []byte("")) {
		t.Errorf("expected bar.g.dart to be detected as generated")
	}
	if adapter.IsGenerated("cart_bloc_test.dart", []byte("// pure code")) {
		t.Errorf("expected regular test file to not be generated")
	}
}

func TestLocateUnitTestUnits(t *testing.T) {
	tests := []struct {
		name               string
		source             string
		expectedTestName   string
		expectedAssertions []string
		expectedBodySubstr string
	}{
		{
			name: "simple test with expect",
			source: `
void main() {
  test('simple test with expect', () {
    final result = 1 + 1;
    expect(result, equals(2));
  });
}
`,
			expectedTestName:   "simple test with expect",
			expectedAssertions: []string{"expect"},
			expectedBodySubstr: "expect(result, equals(2));",
		},
		{
			name: "test with nested block closure followed by expect",
			source: `
void main() {
  test('nested closure test', () async {
    final items = <String>[];
    stream.listen((event) {
      items.add(event);
    });
    expect(items, isNotEmpty);
  });
}
`,
			expectedTestName:   "nested closure test",
			expectedAssertions: []string{"expect"},
			expectedBodySubstr: "expect(items, isNotEmpty);",
		},
		{
			name: "test with multiple nested blocks and expectLater",
			source: `
void main() {
  test('multiple nested blocks', () async {
    controller.stream.listen((event) {
      if (event.isValid) {
        processItems(event.items, (item) {
          cache.put(item.id, item);
        });
      }
    });
    await expectLater(controller.stream, emitsDone);
  });
}
`,
			expectedTestName:   "multiple nested blocks",
			expectedAssertions: []string{"expectLater"},
			expectedBodySubstr: "expectLater(controller.stream, emitsDone);",
		},
		{
			name: "double quoted test name with apostrophe",
			source: `
void main() {
  test("doesn't crash on invalid payload", () {
    expect(true, isTrue);
  });
}
`,
			expectedTestName:   "doesn't crash on invalid payload",
			expectedAssertions: []string{"expect"},
			expectedBodySubstr: "expect(true, isTrue);",
		},
		{
			name: "testWidgets with tester closure",
			source: `
void main() {
  testWidgets('renders submit button', (tester) async {
    await tester.pumpWidget(const MyWidget());
    expect(find.byType(ElevatedButton), findsOneWidget);
  });
}
`,
			expectedTestName:   "renders submit button",
			expectedAssertions: []string{"expect"},
			expectedBodySubstr: "findsOneWidget",
		},
	}

	adapter := dart.NewAdapter()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			units, err := adapter.LocateTests([]byte(tc.source), "test.dart")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(units) != 1 {
				t.Fatalf("expected 1 test unit, got %d", len(units))
			}
			unit := units[0]
			if unit.Name != tc.expectedTestName {
				t.Errorf("expected test name %q, got %q", tc.expectedTestName, unit.Name)
			}
			if unit.Kind != "unit" {
				t.Errorf("expected kind %q, got %q", "unit", unit.Kind)
			}
			if len(unit.Assertions) != len(tc.expectedAssertions) {
				t.Fatalf("expected %d assertions, got %d: %v", len(tc.expectedAssertions), len(unit.Assertions), unit.Assertions)
			}
			for i, exp := range tc.expectedAssertions {
				if unit.Assertions[i] != exp {
					t.Errorf("expected assertion[%d] = %q, got %q", i, exp, unit.Assertions[i])
				}
			}
			if !strings.Contains(unit.RawBody, tc.expectedBodySubstr) {
				t.Errorf("expected raw body to contain %q, but got:\n%s", tc.expectedBodySubstr, unit.RawBody)
			}
		})
	}
}

func TestLocateTestsUnmatchedBraceError(t *testing.T) {
	source := []byte(`
void main() {
  test('broken callback test', () {
    expect(1, 1);
    // missing closing brace
`)
	adapter := dart.NewAdapter()
	_, err := adapter.LocateTests(source, "broken_test.dart")
	if err == nil {
		t.Fatalf("expected error for unmatched callback brace, got nil")
	}
	if !strings.Contains(err.Error(), "unmatched brace") {
		t.Errorf("expected error message mentioning unmatched brace, got %v", err)
	}
}
