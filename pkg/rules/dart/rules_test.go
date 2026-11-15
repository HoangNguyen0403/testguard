package dart_test

import (
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	dartparser "github.com/HoangNguyen0403/testguard/pkg/parser/dart"
	"github.com/HoangNguyen0403/testguard/pkg/rules/dart"
)

func TestDetectSeedEcho(t *testing.T) {
	unitWithEcho := parser.TestUnit{
		Name:       "test_with_echo",
		FilePath:   "customer_code_bloc_test.dart",
		StartLine:  10,
		Kind:       "bloc",
		SeedBody:   "baseState.copyWith(userInfo: fakeUser)",
		ExpectBody: "[isA<CustomerCodeState>().having((s) => s.userInfo, 'userInfo', fakeUser)]",
	}

	checker := dart.NewSeedEchoRule()
	violations := checker.Evaluate(unitWithEcho)

	if len(violations) != 1 {
		t.Fatalf("expected 1 seed echo violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-DART-SEED-ECHO" {
		t.Errorf("expected GUARD-DART-SEED-ECHO, got %s", violations[0].RuleID)
	}
	if violations[0].Severity != models.SeverityWarning {
		t.Errorf("expected Warning severity, got %s", violations[0].Severity)
	}
}

func TestSeedEcho_RefinedBehavior(t *testing.T) {
	tests := []struct {
		name           string
		seedBody       string
		expectBody     string
		wantViolations int
		wantFieldInMsg string
	}{
		{
			name:           "positive: seed echoing name with no mutation",
			seedBody:       "UserState(name: 'bob', age: 3)",
			expectBody:     "[isA<UserState>().having((s) => s.name, 'name', 'bob')]",
			wantViolations: 1,
			wantFieldInMsg: "name",
		},
		{
			name:           "positive: multiple fields echoed",
			seedBody:       "UserState(name: 'bob', age: 3)",
			expectBody:     "[isA<UserState>().having((s) => s.name, 'name', 'bob').having((s) => s.age, 'age', 3)]",
			wantViolations: 1,
			wantFieldInMsg: "name",
		},
		{
			name:           "negative: keeps seeded user when status changes to error",
			seedBody:       "UserState(user: User(name: 'alice'), status: Status.idle)",
			expectBody:     "[isA<UserState>().having((s) => s.status, 'status', Status.error).having((s) => s.user, 'user', User(name: 'alice'))]",
			wantViolations: 0,
		},
		{
			name:           "negative: nested constructor argument is not a state field",
			seedBody:       "UserState(user: User(name: 'alice'), name: 'bob')",
			expectBody:     "[isA<UserState>().having((s) => s.name, 'name', 'alice')]",
			wantViolations: 0,
		},
		{
			name:           "negative: asserts unset field not present in seed",
			seedBody:       "UserState(name: 'bob')",
			expectBody:     "[isA<UserState>().having((s) => s.name, 'name', 'bob').having((s) => s.age, 'age', 3)]",
			wantViolations: 0,
		},
		{
			name:           "negative: expect has no having matchers",
			seedBody:       "UserState(name: 'bob')",
			expectBody:     "[isA<UserState>()]",
			wantViolations: 0,
		},
		{
			name:           "positive: whitespace normalization in nested constructor value",
			seedBody:       "UserState(\n  name: 'bob',\n  age: 3,\n)",
			expectBody:     "[isA<UserState>().having((s) => s.name, 'name', 'bob')]",
			wantViolations: 1,
			wantFieldInMsg: "name",
		},
	}

	rule := dart.NewSeedEchoRule()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:       tc.name,
				FilePath:   "bloc_test.dart",
				StartLine:  15,
				Kind:       "bloc",
				SeedBody:   tc.seedBody,
				ExpectBody: tc.expectBody,
			}
			violations := rule.Evaluate(unit)
			if len(violations) != tc.wantViolations {
				t.Fatalf("expected %d violations, got %d: %+v", tc.wantViolations, len(violations), violations)
			}
			if tc.wantViolations > 0 {
				v := violations[0]
				if v.RuleID != "GUARD-DART-SEED-ECHO" {
					t.Errorf("expected RuleID GUARD-DART-SEED-ECHO, got %s", v.RuleID)
				}
				if v.Severity != models.SeverityWarning {
					t.Errorf("expected Warning severity, got %s", v.Severity)
				}
				if tc.wantFieldInMsg != "" && !strings.Contains(v.Message, tc.wantFieldInMsg) {
					t.Errorf("expected message to contain %q, got %q", tc.wantFieldInMsg, v.Message)
				}
				if v.Suggestion == "" {
					t.Errorf("expected non-empty suggestion")
				}
			}
		})
	}
}

func TestDetectMockEcho(t *testing.T) {
	unitWithMockEcho := parser.TestUnit{
		Name:      "test_mock_echo",
		FilePath:  "repo_test.dart",
		StartLine: 20,
		Kind:      "unit",
		RawBody:   "when(() => api.fetch()).thenAnswer((_) => Right(mockData));\nexpect(repo.fetch(), equals(Right(mockData)));",
	}

	checker := dart.NewMockEchoRule()
	violations := checker.Evaluate(unitWithMockEcho)

	if len(violations) != 1 {
		t.Fatalf("expected 1 mock echo violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-DART-MOCK-ECHO" {
		t.Errorf("expected GUARD-DART-MOCK-ECHO, got %s", violations[0].RuleID)
	}
}

func TestDetectMockEcho_Extended(t *testing.T) {
	tests := []struct {
		name           string
		rawBody        string
		wantViolations int
	}{
		{
			name: "thenAnswer Left",
			rawBody: `
				when(() => api.fetch()).thenAnswer((_) async => Left(appError));
				final res = await repo.fetch();
				expect(res, Left(appError));
			`,
			wantViolations: 1,
		},
		{
			name: "thenReturn Right",
			rawBody: `
				when(() => api.fetch()).thenReturn(Right(fakeUser));
				final res = repo.fetch();
				expect(res, Right(fakeUser));
			`,
			wantViolations: 1,
		},
		{
			name: "thenReturn Left with equals",
			rawBody: `
				when(() => api.fetch()).thenReturn(Left(appError));
				final res = repo.fetch();
				expect(res, equals(Left(appError)));
			`,
			wantViolations: 1,
		},
		{
			name: "multiple stubs: second stub matches expect",
			rawBody: `
				when(() => api.check()).thenReturn(Right(otherToken));
				when(() => api.fetch()).thenAnswer((_) async => Right(targetToken));
				final res = await repo.fetch();
				expect(res, Right(targetToken));
			`,
			wantViolations: 1,
		},
		{
			name: "opposite side: stub Right, expect Left -> no violation",
			rawBody: `
				when(() => api.fetch()).thenReturn(Right(commonToken));
				final res = repo.fetch();
				expect(res, Left(commonToken));
			`,
			wantViolations: 0,
		},
		{
			name: "mapper transforms model to entity -> no violation",
			rawBody: `
				when(() => dataSource.fetch()).thenAnswer((_) async => Right(fakeUserModel));
				final result = await UserRepository(dataSource).fetch();
				expect(result, Right(fakeUser));
			`,
			wantViolations: 0,
		},
	}

	checker := dart.NewMockEchoRule()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:      tc.name,
				FilePath:  "repo_test.dart",
				StartLine: 25,
				Kind:      "unit",
				RawBody:   tc.rawBody,
			}
			violations := checker.Evaluate(unit)
			if len(violations) != tc.wantViolations {
				t.Fatalf("expected %d violations, got %d: %+v", tc.wantViolations, len(violations), violations)
			}
			if tc.wantViolations > 0 {
				v := violations[0]
				if v.RuleID != "GUARD-DART-MOCK-ECHO" {
					t.Errorf("expected RuleID GUARD-DART-MOCK-ECHO, got %s", v.RuleID)
				}
				if v.Severity != models.SeverityWarning {
					t.Errorf("expected Warning severity, got %s", v.Severity)
				}
				if v.Suggestion == "" {
					t.Errorf("expected non-empty suggestion")
				}
			}
		})
	}
}

func TestAuditProbe_DartRulesProof(t *testing.T) {
	source := `
import 'package:bloc_test/bloc_test.dart';
import 'package:dartz/dartz.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:mocktail/mocktail.dart';

void main() {
  late MockRepo repo;
  late MockDataSource dataSource;

  setUp(() {
    repo = MockRepo();
    dataSource = MockDataSource();
  });

  group('zero-assert styles', () {
    test('verifyZeroInteractions only', () {
      Service(repo).noop();
      verifyZeroInteractions(repo);
    });

    test('verifyNoMoreInteractions only', () {
      Service(repo).noop();
      verifyNoMoreInteractions(repo);
    });

    test('fail branch', () {
      final result = Service(repo).parse('1');
      result.fold((l) => fail('expected right'), (r) => r);
    });

    test('expectAsync callback', () {
      Service(repo).onDone(expectAsync1((int v) {}));
    });

    test('truly empty', () {
      Service(repo).noop();
    });
  });

  group('when cache is warm', () {
    setUp(() => when(() => repo.cached()).thenReturn(true));
    test('returns items', () {
      expect(Service(repo).items(), hasLength(2));
    });
  });

  group('when cache is cold', () {
    setUp(() => when(() => repo.cached()).thenReturn(false));
    test('returns items', () {
      expect(Service(repo).items(), hasLength(2));
    });
  });

  group('mock echo', () {
    test('pass-through use case', () async {
      when(() => repo.fetch()).thenAnswer((_) async => Right(fakeUser));
      final result = await GetUser(repo)();
      expect(result, Right(fakeUser));
    });

    test('repository maps model to entity', () async {
      when(() => dataSource.fetch()).thenAnswer((_) async => Right(fakeUserModel));
      final result = await UserRepository(dataSource).fetch();
      expect(result, Right(fakeUser));
    });
  });

  blocTest<UserBloc, UserState>(
    'keeps the seeded user when refresh fails',
    build: () => UserBloc(repo),
    seed: () => UserState(user: User(name: 'alice'), status: Status.idle),
    act: (bloc) => bloc.add(const RefreshFailed()),
    expect: () => [
      isA<UserState>()
          .having((s) => s.status, 'status', Status.error)
          .having((s) => s.user, 'user', User(name: 'alice')),
    ],
  );

  blocTest<UserBloc, UserState>(
    'nested seed field is not a state field',
    build: () => UserBloc(repo),
    seed: () => UserState(user: User(name: 'alice'), name: 'bob'),
    act: (bloc) => bloc.add(const Renamed('alice')),
    expect: () => [
      isA<UserState>().having((s) => s.name, 'name', 'alice'),
    ],
  );

  test('skipped with flag', () {
    expect(1 + 1, 2);
  }, skip: true);

  test('same identifier tautology', () {
    final x = Service(repo).count();
    expect(x, x);
  });
}
`
	adapter := dartparser.NewAdapter()
	units, err := adapter.LocateTests([]byte(source), "rules_test.dart")
	if err != nil {
		t.Fatalf("unexpected error locating tests: %v", err)
	}

	unitMap := make(map[string]parser.TestUnit)
	for _, u := range units {
		unitMap[u.Name] = u
	}

	// 1. Proof 1: only 'truly empty' has zero assertions
	zeroAssertUnits := []string{}
	for _, u := range units {
		if u.Kind == "unit" && len(u.Assertions) == 0 {
			zeroAssertUnits = append(zeroAssertUnits, u.Name)
		}
	}
	if len(zeroAssertUnits) != 1 || zeroAssertUnits[0] != "truly empty" {
		t.Errorf("expected only 'truly empty' to have zero assertions, got %v", zeroAssertUnits)
	}

	// 2. Proof 2: scopes of the two 'returns items' tests
	warmUnit, warmOk := unitMap["returns items"]
	if !warmOk {
		t.Fatalf("expected 'returns items' to exist")
	}
	_ = warmUnit
	// Since both are named 'returns items', let's check both by walking units
	var scopes []string
	for _, u := range units {
		if u.Name == "returns items" {
			scopes = append(scopes, u.Scope)
		}
	}
	if len(scopes) != 2 || scopes[0] != "when cache is warm" || scopes[1] != "when cache is cold" {
		t.Errorf("expected scopes ['when cache is warm', 'when cache is cold'], got %v", scopes)
	}

	// 3. Proof 3: GUARD-DART-SEED-ECHO produces no finding on either blocTest
	seedEchoRule := dart.NewSeedEchoRule()
	for _, u := range units {
		if u.Kind == "bloc" {
			violations := seedEchoRule.Evaluate(u)
			if len(violations) != 0 {
				t.Errorf("expected 0 seed echo violations for %q, got %+v", u.Name, violations)
			}
		}
	}

	// 4. Proof 4: GUARD-DART-MOCK-ECHO fires on 'pass-through use case' only
	mockEchoRule := dart.NewMockEchoRule()
	passThroughUnit, ok := unitMap["pass-through use case"]
	if !ok {
		t.Fatalf("expected 'pass-through use case' unit to exist")
	}
	passThroughViolations := mockEchoRule.Evaluate(passThroughUnit)
	if len(passThroughViolations) != 1 || passThroughViolations[0].RuleID != "GUARD-DART-MOCK-ECHO" {
		t.Errorf("expected 1 GUARD-DART-MOCK-ECHO violation on 'pass-through use case', got %+v", passThroughViolations)
	}

	mapperUnit, ok := unitMap["repository maps model to entity"]
	if !ok {
		t.Fatalf("expected 'repository maps model to entity' unit to exist")
	}
	mapperViolations := mockEchoRule.Evaluate(mapperUnit)
	if len(mapperViolations) != 0 {
		t.Errorf("expected 0 mock echo violations on 'repository maps model to entity', got %+v", mapperViolations)
	}
}
