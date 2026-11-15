package dart_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	dartparser "github.com/HoangNguyen0403/testguard/pkg/parser/dart"
	"github.com/HoangNguyen0403/testguard/pkg/rules/dart"
)

func TestVacuousStateRule(t *testing.T) {
	tests := []struct {
		name           string
		stateType      string
		expectBody     string
		rawBody        string
		wantViolations int
	}{
		{
			name:           "positive: bare isA<S>() without having or verify",
			stateType:      "UserState",
			expectBody:     "[isA<UserState>()]",
			rawBody:        "build: () => UserBloc(), act: (b) => b.add(Ev()), expect: () => [isA<UserState>()]",
			wantViolations: 1,
		},
		{
			name:           "positive: multiple bare isA<S>()",
			stateType:      "UserState",
			expectBody:     "[isA<UserState>(), isA<UserState>()]",
			rawBody:        "build: () => UserBloc(), expect: () => [isA<UserState>(), isA<UserState>()]",
			wantViolations: 1,
		},
		{
			name:           "negative: element has having matcher",
			stateType:      "UserState",
			expectBody:     "[isA<UserState>().having((s) => s.status, 'status', Status.ready)]",
			rawBody:        "build: () => UserBloc(), expect: () => [isA<UserState>().having((s) => s.status, 'status', Status.ready)]",
			wantViolations: 0,
		},
		{
			name:           "negative: more specific subtypes",
			stateType:      "UserState",
			expectBody:     "[isA<UserLoading>(), isA<UserLoaded>()]",
			rawBody:        "build: () => UserBloc(), expect: () => [isA<UserLoading>(), isA<UserLoaded>()]",
			wantViolations: 0,
		},
		{
			name:           "negative: has verify argument block",
			stateType:      "UserState",
			expectBody:     "[isA<UserState>()]",
			rawBody:        "build: () => UserBloc(), expect: () => [isA<UserState>()], verify: (b) { verify(() => repo.call()).called(1); }",
			wantViolations: 0,
		},
		{
			name:           "negative: concrete state values",
			stateType:      "UserState",
			expectBody:     "[UserState.initial()]",
			rawBody:        "build: () => UserBloc(), expect: () => [UserState.initial()]",
			wantViolations: 0,
		},
		{
			name:           "negative: unknown S state type",
			stateType:      "",
			expectBody:     "[isA<UserState>()]",
			rawBody:        "build: () => UserBloc(), expect: () => [isA<UserState>()]",
			wantViolations: 0,
		},
	}

	rule := dart.NewVacuousStateRule()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			unit := parser.TestUnit{
				Name:       tc.name,
				FilePath:   "user_bloc_test.dart",
				StartLine:  20,
				Kind:       "bloc",
				RawBody:    tc.rawBody,
				ExpectBody: tc.expectBody,
			}
			if tc.stateType != "" {
				unit.Meta = map[string]string{
					dartparser.MetaBlocStateType: tc.stateType,
				}
			}

			violations := rule.Evaluate(unit)
			if len(violations) != tc.wantViolations {
				t.Fatalf("expected %d violations, got %d: %+v", tc.wantViolations, len(violations), violations)
			}

			if tc.wantViolations > 0 {
				v := violations[0]
				if v.RuleID != "GUARD-DART-VACUOUS-STATE" {
					t.Errorf("expected RuleID GUARD-DART-VACUOUS-STATE, got %s", v.RuleID)
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
