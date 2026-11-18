package dart_test

import (
	"strings"
	"testing"
	"time"

	"github.com/HoangNguyen0403/testguard/pkg/detectors/zeroassert"
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
func TestDartAdapter_32kNestedCallsPerformance(t *testing.T) {
	n := 32000
	src := "void main() {" + strings.Repeat("f(", n) + "0" + strings.Repeat(")", n) + ";}"
	adapter := dart.NewAdapter()

	start := time.Now()
	units, err := adapter.LocateTests([]byte(src), "stress_test.dart")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 0 {
		t.Fatalf("expected 0 units, got %d", len(units))
	}
	if elapsed > time.Second {
		t.Fatalf("expected 32k nested calls to process in under 1s, took %v", elapsed)
	}
}

func TestDartAdapter_32kNestedMapCallsPerformance(t *testing.T) {
	n := 32000
	src := "void main(){" + strings.Repeat("f(", n) + "{}" + strings.Repeat(")", n) + ";}"
	adapter := dart.NewAdapter()

	start := time.Now()
	units, err := adapter.LocateTests([]byte(src), "nested_map_test.dart")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 0 {
		t.Fatalf("expected 0 units, got %d", len(units))
	}
	if elapsed > time.Second {
		t.Fatalf("expected 32k nested map calls to process in under 1s, took %v", elapsed)
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

func TestLocateUnitTestUnits_HelperAssertions(t *testing.T) {
	source := []byte(`
void verifyCartEmpty(tester) {
  expect(find.byType(Cart), findsNothing);
}

void setupCart(tester) {
  print('setting up cart');
}

void verifyTrackingArrow(String id) => expect(id, isNotEmpty);

Future<void> expectCartEmptyAsync(tester) async {
  await expectLater(tester, findsNothing);
}
void transitivelyCallsHelper(tester) {
  verifyCartEmpty(tester);
}

void main() {
  void nestedInsideMainHelper(tester) {
    verify(mock.call());
  }

  testWidgets('cart is empty', (tester) async {
    verifyCartEmpty(tester);
  });

  testWidgets('cart setup only', (tester) async {
    setupCart(tester);
  });

  test('resolves helper defined later', () {
    verifyTrackingLater('checkout_step');
  });

  test('calls arrow helper', () {
    verifyTrackingArrow('123');
  });

  testWidgets('async helper call', (tester) async {
    await expectCartEmptyAsync(tester);
  });

  test('checkCondition', () {
    if (true) {
      print('hello');
    }
  });

  testWidgets('one-hop only ignores transitive helper', (tester) async {
    transitivelyCallsHelper(tester);
  });

  testWidgets('direct assertion and helper together', (tester) async {
    verifyCartEmpty(tester);
    expect(true, isTrue);
  });

  testWidgets('nested helper inside main', (tester) async {
    nestedInsideMainHelper(tester);
  });
}

void verifyTrackingLater(String step) {
  expect(step, isNotEmpty);
}
`)

	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests(source, "helpers_test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 9 {
		t.Fatalf("expected 9 test units, got %d", len(units))
	}

	expected := map[string][]string{
		"cart is empty":                          {"helper:verifyCartEmpty"},
		"cart setup only":                        {},
		"resolves helper defined later":          {"helper:verifyTrackingLater"},
		"calls arrow helper":                     {"helper:verifyTrackingArrow"},
		"async helper call":                      {"helper:expectCartEmptyAsync"},
		"checkCondition":                         {},
		"one-hop only ignores transitive helper": {},
		"direct assertion and helper together":   {"helper:verifyCartEmpty", "expect"},
		"nested helper inside main":              {"helper:nestedInsideMainHelper"},
	}
	for _, unit := range units {
		exp, ok := expected[unit.Name]
		if !ok {
			t.Errorf("unexpected test unit: %q", unit.Name)
			continue
		}
		if len(unit.Assertions) != len(exp) {
			t.Errorf("test %q: expected assertions %v, got %v", unit.Name, exp, unit.Assertions)
			continue
		}
		for i, a := range exp {
			if unit.Assertions[i] != a {
				t.Errorf("test %q: expected assertion[%d]=%q, got %q", unit.Name, i, a, unit.Assertions[i])
			}
		}
	}
}

func TestR2_DartHelper_InsideGroupWithNamedParamsAndMultiLine(t *testing.T) {
	source := []byte(`
void main() {
  group('tracking group', () {
    void verifyTrackingBehavior({
      required String expectedSource,
      required String buttonFrom,
    }) {
      verify(() => repo.setVisitSource(source: expectedSource)).called(1);
      verify(() => mixpanel.trackEvent(eventName: 'x', properties: any(named: 'properties'))).called(1);
    }

    testWidgets('tracks button click', (tester) async {
      verifyTrackingBehavior(
        expectedSource: 'home',
        buttonFrom: 'submit_button',
      );
    });
  });
}
`)
	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests(source, "tracking_test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(units))
	}
	if len(units[0].Assertions) == 0 {
		t.Errorf("expected helper:verifyTrackingBehavior assertion, got 0 assertions: %+v", units[0])
	}
}

func TestBlocTest_SetUpEndingWithSemicolon_FullBodyExtracted(t *testing.T) {
	source := []byte(`
void main() {
  blocTest<SearchBloc, SearchState>(
    'search with failure A',
    setUp: () {
      when(() => repo.get()).thenAnswer((_) async => const Left(ApiFailure.other('e')));
      when(() => repo.fallbackA()).thenReturn(1);
    },
    build: () => SearchBloc(repo),
    act: (bloc) => bloc.add(const SearchEvent('a')),
    expect: () => [SearchState.failureA()],
  );

  blocTest<SearchBloc, SearchState>(
    'search with failure B',
    setUp: () {
      when(() => repo.get()).thenAnswer((_) async => const Left(ApiFailure.other('e')));
      when(() => repo.fallbackB()).thenReturn(2);
    },
    build: () => SearchBloc(repo),
    act: (bloc) => bloc.add(const SearchEvent('b')),
    expect: () => [SearchState.failureB()],
  );
}
`)

	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests(source, "search_bloc_test.dart")
	if err != nil {
		t.Fatalf("unexpected error locating tests: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}

	if units[0].RawBody == units[1].RawBody {
		t.Fatalf("expected different RawBody for two blocTests with differing later setUp and expect, but got identical: %q", units[0].RawBody)
	}
	if !strings.Contains(units[0].RawBody, "fallbackA") {
		t.Errorf("expected unit 0 RawBody to contain fallbackA, got %q", units[0].RawBody)
	}
	if !strings.Contains(units[1].RawBody, "fallbackB") {
		t.Errorf("expected unit 1 RawBody to contain fallbackB, got %q", units[1].RawBody)
	}
}

func TestLocateTests_DirectAssertions(t *testing.T) {
	tests := []struct {
		name       string
		source     string
		wantAssert string
	}{
		{
			name: "verifyZeroInteractions",
			source: `void main() {
				test('zero interactions', () {
					verifyZeroInteractions(mockRepo);
				});
			}`,
			wantAssert: "verifyZeroInteractions",
		},
		{
			name: "verifyNoMoreInteractions",
			source: `void main() {
				test('no more interactions', () {
					verifyNoMoreInteractions(mockRepo);
				});
			}`,
			wantAssert: "verifyNoMoreInteractions",
		},
		{
			name: "fail in branch",
			source: `void main() {
				test('fail in fold', () {
					result.fold((l) => fail('expected right'), (r) => r);
				});
			}`,
			wantAssert: "fail",
		},
		{
			name: "expectAsync0",
			source: `void main() {
				test('async callback 0', () {
					stream.listen(expectAsync0(() {}));
				});
			}`,
			wantAssert: "expectAsync0",
		},
		{
			name: "expectAsync1",
			source: `void main() {
				test('async callback 1', () {
					stream.listen(expectAsync1((val) {}));
				});
			}`,
			wantAssert: "expectAsync1",
		},
		{
			name: "expectAsync6",
			source: `void main() {
				test('async callback 6', () {
					stream.listen(expectAsync6((a, b, c, d, e, f) {}));
				});
			}`,
			wantAssert: "expectAsync6",
		},
		{
			name: "expectAsyncUntil0",
			source: `void main() {
				test('async until callback 0', () {
					stream.listen(expectAsyncUntil0(() {}, () => true));
				});
			}`,
			wantAssert: "expectAsyncUntil0",
		},
		{
			name: "expectAsyncUntil6",
			source: `void main() {
				test('async until callback 6', () {
					stream.listen(expectAsyncUntil6((a, b, c, d, e, f) {}, () => true));
				});
			}`,
			wantAssert: "expectAsyncUntil6",
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
				t.Fatalf("expected 1 unit, got %d", len(units))
			}
			found := false
			for _, a := range units[0].Assertions {
				if a == tc.wantAssert {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected assertion %q in %+v", tc.wantAssert, units[0].Assertions)
			}
		})
	}
}

func TestLocateTests_HelperWithNewAssertion(t *testing.T) {
	source := `
		void checkClean(MockRepo r) {
			verifyZeroInteractions(r);
		}

		void main() {
			test('uses helper with verifyZeroInteractions', () {
				checkClean(repo);
			});
		}
	`
	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests([]byte(source), "test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(units))
	}
	if len(units[0].Assertions) != 1 || units[0].Assertions[0] != "helper:checkClean" {
		t.Errorf("expected helper:checkClean assertion, got %+v", units[0].Assertions)
	}
}

func TestLocateTests_Scope(t *testing.T) {
	source := `
		void main() {
			test('top level test', () {
				expect(1, 1);
			});

			blocTest<UserBloc, UserState>(
				'top level blocTest',
				build: () => UserBloc(),
				expect: () => [],
			);

			group('cache operations', () {
				test('when cache is warm', () {
					expect(true, isTrue);
				});

				group('eviction', () {
					test('evicts oldest', () {
						expect(true, isTrue);
					});

					testWidgets('widget in eviction', (tester) async {
						expect(find.text('clean'), findsOneWidget);
					});

					blocTest<UserBloc, UserState>(
						'blocTest in eviction',
						build: () => UserBloc(),
						expect: () => [],
					);
				});
			});

			group('second root group', () {
				test('second group test', () {
					expect(2, 2);
				});
			});
		}
	`

	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests([]byte(source), "scope_test.dart")
	if err != nil {
		t.Fatalf("unexpected error locating tests: %v", err)
	}

	unitScopeMap := make(map[string]string)
	for _, u := range units {
		unitScopeMap[u.Name] = u.Scope
	}

	expectedScopes := map[string]string{
		"top level test":       "",
		"top level blocTest":   "",
		"when cache is warm":   "cache operations",
		"evicts oldest":        "cache operations > eviction",
		"widget in eviction":   "cache operations > eviction",
		"blocTest in eviction": "cache operations > eviction",
		"second group test":    "second root group",
	}

	for testName, expectedScope := range expectedScopes {
		gotScope, ok := unitScopeMap[testName]
		if !ok {
			t.Errorf("test %q not found in located units", testName)
			continue
		}
		if gotScope != expectedScope {
			t.Errorf("test %q: expected scope %q, got %q", testName, expectedScope, gotScope)
		}
	}
}

func TestLocateTests_NestedSeedBody(t *testing.T) {
	source := `
		void main() {
			blocTest<UserBloc, UserState>(
				'nested seed test',
				build: () => UserBloc(),
				seed: () => UserState(user: User(name: 'alice'), status: Status.idle),
				act: (bloc) => bloc.add(Refresh()),
				expect: () => [
					isA<UserState>()
						.having((s) => s.status, 'status', Status.error)
						.having((s) => s.user, 'user', User(name: 'alice')),
				],
			);
		}
	`
	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests([]byte(source), "test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(units))
	}
	wantSeed := "UserState(user: User(name: 'alice'), status: Status.idle)"
	if units[0].SeedBody != wantSeed {
		t.Errorf("expected SeedBody %q, got %q", wantSeed, units[0].SeedBody)
	}
}

func TestLocateTests_BlocStateTypeMeta(t *testing.T) {
	source := `
		void main() {
			blocTest<UserBloc, UserState>(
				'typed blocTest',
				build: () => UserBloc(),
				expect: () => [],
			);

			blocTest(
				'untyped blocTest',
				build: () => UserBloc(),
				expect: () => [],
			);
		}
	`
	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests([]byte(source), "test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}

	// Typed blocTest
	if units[0].Meta == nil || units[0].Meta[dart.MetaBlocStateType] != "UserState" {
		t.Errorf("expected Meta[%q] = %q, got %+v", dart.MetaBlocStateType, "UserState", units[0].Meta)
	}

	// Untyped blocTest
	if units[1].Meta != nil && units[1].Meta[dart.MetaBlocStateType] != "" {
		t.Errorf("expected untyped Meta[%q] to be empty, got %+v", dart.MetaBlocStateType, units[1].Meta)
	}
}

func TestArrowBodyTestDiscovery(t *testing.T) {
	source := []byte(`void main() {
  test('calc', () => expect(1, 1));
  testWidgets('render', (tester) async => await tester.pump());
}`)

	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests(source, "calc_test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}

	u1 := units[0]
	if u1.Name != "calc" {
		t.Errorf("expected name 'calc', got %q", u1.Name)
	}
	if u1.StartLine != 2 {
		t.Errorf("expected start line 2, got %d", u1.StartLine)
	}
	if u1.EndLine != 2 {
		t.Errorf("expected end line 2, got %d", u1.EndLine)
	}
	if u1.RawBody != "expect(1, 1)" {
		t.Errorf("expected RawBody 'expect(1, 1)', got %q", u1.RawBody)
	}
	if len(u1.Assertions) != 1 || u1.Assertions[0] != "expect" {
		t.Errorf("expected assertions ['expect'], got %v", u1.Assertions)
	}

	u2 := units[1]
	if u2.Name != "render" {
		t.Errorf("expected name 'render', got %q", u2.Name)
	}
	if u2.StartLine != 3 {
		t.Errorf("expected start line 3, got %d", u2.StartLine)
	}
	if u2.EndLine != 3 {
		t.Errorf("expected end line 3, got %d", u2.EndLine)
	}
	if u2.RawBody != "await tester.pump()" {
		t.Errorf("expected RawBody 'await tester.pump()', got %q", u2.RawBody)
	}
}

func TestArrowBodyHollowTestFlaggedByZeroAssert(t *testing.T) {
	source := []byte(`void main() {
  test('hollow', () => 1 + 1);
}`)

	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests(source, "hollow_test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(units) != 1 {
		t.Fatalf("expected 1 unit, got %d", len(units))
	}

	u := units[0]
	if u.Name != "hollow" {
		t.Errorf("expected name 'hollow', got %q", u.Name)
	}
	if u.RawBody != "1 + 1" {
		t.Errorf("expected RawBody '1 + 1', got %q", u.RawBody)
	}
	if len(u.Assertions) != 0 {
		t.Errorf("expected 0 assertions, got %v", u.Assertions)
	}

	detector := zeroassert.New()
	violations := detector.Check(units)
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation from zeroassert, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-UNIV-ZERO" {
		t.Errorf("expected rule GUARD-UNIV-ZERO, got %s", violations[0].RuleID)
	}
}

func TestArrowBodyGenericTypeArguments(t *testing.T) {
	source := []byte(`
void assertPair<A, B>(A a, B b) {
  expect(a, isNotNull);
}

void main() {
  test('generic pair', () => assertPair<String, int>('a', 1));
  test('nested generics', () => verify<Future<Map<String, dynamic>>>(fetchData()));
}
`)

	adapter := dart.NewAdapter()
	units, err := adapter.LocateTests(source, "generic_arrow_test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(units) != 2 {
		t.Fatalf("expected 2 units, got %d", len(units))
	}

	u1 := units[0]
	if u1.Name != "generic pair" {
		t.Errorf("expected name 'generic pair', got %q", u1.Name)
	}
	expectedBody1 := "assertPair<String, int>('a', 1)"
	if u1.RawBody != expectedBody1 {
		t.Errorf("expected RawBody %q, got %q", expectedBody1, u1.RawBody)
	}
	if len(u1.Assertions) != 1 || u1.Assertions[0] != "helper:assertPair" {
		t.Errorf("expected assertions ['helper:assertPair'], got %v", u1.Assertions)
	}

	u2 := units[1]
	if u2.Name != "nested generics" {
		t.Errorf("expected name 'nested generics', got %q", u2.Name)
	}
	expectedBody2 := "verify<Future<Map<String, dynamic>>>(fetchData())"
	if u2.RawBody != expectedBody2 {
		t.Errorf("expected RawBody %q, got %q", expectedBody2, u2.RawBody)
	}
	if len(u2.Assertions) != 1 || u2.Assertions[0] != "verify" {
		t.Errorf("expected assertions ['verify'], got %v", u2.Assertions)
	}
}

func TestLocateUnitTestUnits_RobotPatternAssertions(t *testing.T) {
	source := []byte(`
class _CartPageRobot {
  final WidgetTester tester;
  _CartPageRobot(this.tester);

  void verifyCartTotal(String value) {
    expect(find.text(value), findsOneWidget);
  }

  void tapAddButton() {
    tester.tap(find.byIcon(Icons.add));
  }
}

class LocalAssertions {
  void checkTotal(String value) {
    verify(() => value);
  }
}

void main() {
  testWidgets('robot assertion helper', (tester) async {
    final robot = _CartPageRobot(tester);
    robot.verifyCartTotal('42');
  });

  testWidgets('robot action method', (tester) async {
    final robot = _CartPageRobot(tester);
    robot.tapAddButton();
  });

  testWidgets('robot class from another file', (tester) async {
    final cartRobot = CartRobot(tester);
    cartRobot.verifyTotal();
  });

  testWidgets('local helper class asserting method', (tester) async {
    final helper = LocalAssertions();
    helper.checkTotal('42');
  });
}
`)

	units, err := dart.NewAdapter().LocateTests(source, "cart_page_test.dart")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(units) != 4 {
		t.Fatalf("expected 4 units, got %d", len(units))
	}

	expected := map[string]string{
		"robot assertion helper":              "receiver:robot.verifyCartTotal",
		"robot action method":                 "receiver:robot.tapAddButton",
		"robot class from another file":       "receiver:cartRobot.verifyTotal",
		"local helper class asserting method": "receiver:helper.checkTotal",
	}
	for _, unit := range units {
		want, ok := expected[unit.Name]
		if !ok {
			t.Errorf("unexpected unit %q", unit.Name)
			continue
		}
		if len(unit.Assertions) != 1 || unit.Assertions[0] != want {
			t.Errorf("test %q: expected assertions [%q], got %v", unit.Name, want, unit.Assertions)
		}
	}

	if violations := zeroassert.New().Check(units); len(violations) != 0 {
		t.Fatalf("expected no zero-assertion violations, got %+v", violations)
	}
}
