package dart_test

import (
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
