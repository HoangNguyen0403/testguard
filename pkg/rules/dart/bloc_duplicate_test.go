package dart_test

import (
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/dart"
)

func TestBlocDuplicate_DifferentSeed_NoFinding(t *testing.T) {
	units := []parser.TestUnit{
		{
			Name:       "bloc_test_1",
			FilePath:   "search_bloc_test.dart",
			StartLine:  10,
			Kind:       "bloc",
			RawBody:    "seed: () => baseState,\nact: (bloc) => bloc.add(const SearchEvent('key')),\nexpect: () => [SearchState.loading()],",
			SeedBody:   "baseState",
			ActBody:    "bloc.add(const SearchEvent('key'))",
			ExpectBody: "[SearchState.loading()]",
		},
		{
			Name:       "bloc_test_2",
			FilePath:   "search_bloc_test.dart",
			StartLine:  30,
			Kind:       "bloc",
			RawBody:    "seed: () => otherState,\nact: (bloc) => bloc.add(const SearchEvent('key')),\nexpect: () => [SearchState.loading()],",
			SeedBody:   "otherState",
			ActBody:    "bloc.add(const SearchEvent('key'))",
			ExpectBody: "[SearchState.loading()]",
		},
	}

	checker := dart.NewBlocDuplicateRule()
	violations := checker.Check(units)

	if len(violations) != 0 {
		t.Fatalf("expected 0 violations when seed differs, got %d: %+v", len(violations), violations)
	}
}

func TestBlocDuplicate_DifferingActIdentifiers_NoFinding(t *testing.T) {
	units := []parser.TestUnit{
		{
			Name:       "search with valid key",
			FilePath:   "search_bloc_test.dart",
			StartLine:  10,
			Kind:       "bloc",
			RawBody:    "build: () => SearchBloc(),\nact: (bloc) => bloc.add(SearchEvent(fakeSearchKey)),\nexpect: () => [isA<SearchState>()],",
			ActBody:    "bloc.add(SearchEvent(fakeSearchKey))",
			ExpectBody: "[isA<SearchState>()]",
		},
		{
			Name:       "search with invalid key",
			FilePath:   "search_bloc_test.dart",
			StartLine:  25,
			Kind:       "bloc",
			RawBody:    "build: () => SearchBloc(),\nact: (bloc) => bloc.add(SearchEvent(fakeInvalidSearchKey)),\nexpect: () => [isA<SearchState>()],",
			ActBody:    "bloc.add(SearchEvent(fakeInvalidSearchKey))",
			ExpectBody: "[isA<SearchState>()]",
		},
	}

	checker := dart.NewBlocDuplicateRule()
	violations := checker.Check(units)

	if len(violations) != 0 {
		t.Fatalf("expected 0 violations when act identifiers differ, got %d: %+v", len(violations), violations)
	}
}

func TestBlocDuplicate_IdenticalWholeScenario_ExactlyOneFinding(t *testing.T) {
	units := []parser.TestUnit{
		{
			Name:      "test_one",
			FilePath:  "search_bloc_test.dart",
			StartLine: 10,
			Kind:      "bloc",
			RawBody: `
				build: () => SearchBloc(repo),
				seed: () => baseState,
				act: (bloc) => bloc.add(const SearchEvent.fetch()),
				expect: () => [
					baseState.copyWith(isFetching: true),
				],
			`,
		},
		{
			Name:      "test_two_duplicate",
			FilePath:  "search_bloc_test.dart",
			StartLine: 40,
			Kind:      "bloc",
			RawBody: `
				// Comment in duplicate
				build: () => SearchBloc(repo),
				/* Another comment */
				seed: () => baseState,
				act: (bloc) => bloc.add(const SearchEvent.fetch()),
				expect: () => [ baseState.copyWith(isFetching: true), ],
			`,
		},
	}

	checker := dart.NewBlocDuplicateRule()
	violations := checker.Check(units)

	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation for identical whole scenario, got %d: %+v", len(violations), violations)
	}

	v := violations[0]
	if v.RuleID != "GUARD-DART-BLOC-DUPLICATE" {
		t.Errorf("expected rule GUARD-DART-BLOC-DUPLICATE, got %s", v.RuleID)
	}
	if v.Severity != models.SeverityAdvisory {
		t.Errorf("expected Advisory severity, got %s", v.Severity)
	}
	if v.LineNumber != 40 {
		t.Errorf("expected line number 40 (reporting later unit), got %d", v.LineNumber)
	}
	if !strings.Contains(v.Message, "test_two_duplicate") || !strings.Contains(v.Message, "test_one") || !strings.Contains(v.Message, "10") {
		t.Errorf("expected message referencing unit and first unit line, got %s", v.Message)
	}
}

func TestBlocDuplicate_DifferentBuildOrSetUp_NoFinding(t *testing.T) {
	units := []parser.TestUnit{
		{
			Name:       "bloc_test_setup_a",
			FilePath:   "search_bloc_test.dart",
			StartLine:  10,
			Kind:       "bloc",
			RawBody:    "setUp: () => setUpMockA(),\nbuild: () => SearchBloc(),\nact: (bloc) => bloc.add(SearchEvent()),\nexpect: () => [SearchState.ready()],",
		},
		{
			Name:       "bloc_test_setup_b",
			FilePath:   "search_bloc_test.dart",
			StartLine:  30,
			Kind:       "bloc",
			RawBody:    "setUp: () => setUpMockB(),\nbuild: () => SearchBloc(),\nact: (bloc) => bloc.add(SearchEvent()),\nexpect: () => [SearchState.ready()],",
		},
	}

	checker := dart.NewBlocDuplicateRule()
	violations := checker.Check(units)

	if len(violations) != 0 {
		t.Fatalf("expected 0 violations when setUp differs, got %d: %+v", len(violations), violations)
	}
}

func TestBlocDuplicate_LeadingTestNameStripped(t *testing.T) {
	units := []parser.TestUnit{
		{
			Name:       "First Test Name",
			FilePath:   "search_bloc_test.dart",
			StartLine:  10,
			Kind:       "bloc",
			RawBody:    "'First Test Name', build: () => SearchBloc(), act: (bloc) => bloc.add(const SearchEvent()), expect: () => [SearchState.ready()],",
		},
		{
			Name:       "Second Test Name",
			FilePath:   "search_bloc_test.dart",
			StartLine:  30,
			Kind:       "bloc",
			RawBody:    "'Second Test Name', build: () => SearchBloc(), act: (bloc) => bloc.add(const SearchEvent()), expect: () => [SearchState.ready()],",
		},
	}

	checker := dart.NewBlocDuplicateRule()
	violations := checker.Check(units)

	if len(violations) != 1 {
		t.Fatalf("expected exactly 1 violation when only test names differ in leading RawBody, got %d: %+v", len(violations), violations)
	}
}
