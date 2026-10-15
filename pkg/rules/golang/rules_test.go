package golang_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/golang"
)

func TestDomainPurityRule(t *testing.T) {
	unitWithMockInDomain := parser.TestUnit{
		Name:      "TestUserValidation",
		FilePath:  "internal/domain/user_test.go",
		StartLine: 12,
		RawBody:   "ctrl := gomock.NewController(t); mockRepo := mocks.NewMockUserRepository(ctrl);",
	}

	checker := golang.NewDomainPurityRule()
	violations := checker.Evaluate(unitWithMockInDomain)

	if len(violations) != 1 {
		t.Fatalf("expected 1 domain purity violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-GO-01" {
		t.Errorf("expected GUARD-GO-01, got %s", violations[0].RuleID)
	}
	if violations[0].Severity != models.SeverityBlocker {
		t.Errorf("expected Blocker severity, got %s", violations[0].Severity)
	}
}

func TestPortMockingRule(t *testing.T) {
	checker := golang.NewPortMockingRule()

	t.Run("service test using sqlmock triggers GUARD-GO-02", func(t *testing.T) {
		unitWithConcreteMock := parser.TestUnit{
			Name:      "TestOrderService",
			FilePath:  "internal/service/order_test.go",
			StartLine: 20,
			RawBody:   "db, mock, err := sqlmock.New();",
		}

		violations := checker.Evaluate(unitWithConcreteMock)
		if len(violations) != 1 {
			t.Fatalf("expected 1 port mocking violation, got %d", len(violations))
		}
		if violations[0].RuleID != "GUARD-GO-02" {
			t.Errorf("expected GUARD-GO-02, got %s", violations[0].RuleID)
		}
		if violations[0].Severity != models.SeverityWarning {
			t.Errorf("expected Warning severity, got %s", violations[0].Severity)
		}
	})

	t.Run("repository and infrastructure tests using sqlmock are exempted", func(t *testing.T) {
		testCases := []struct {
			name     string
			filePath string
			rawBody  string
		}{
			{
				name:     "repository file path prefix",
				filePath: "repository/cart_item_repository_test.go",
				rawBody:  "db, mock, err := sqlmock.New();",
			},
			{
				name:     "nested repository directory",
				filePath: "internal/repository/user_test.go",
				rawBody:  "db, mock, err := sqlmock.New();",
			},
			{
				name:     "repository test file suffix",
				filePath: "cart_item_repository_test.go",
				rawBody:  "db, mock, err := sqlmock.New();",
			},
			{
				name:     "infrastructure layer",
				filePath: "internal/infrastructure/database/pgx_test.go",
				rawBody:  "mock, err := pgxmock.NewPool();",
			},
			{
				name:     "database directory",
				filePath: "database/migration_test.go",
				rawBody:  "db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})",
			},
			{
				name:     "db directory",
				filePath: "internal/db/client_test.go",
				rawBody:  "db, mock, err := sqlmock.New();",
			},
			{
				name:     "dao directory",
				filePath: "dao/order_dao_test.go",
				rawBody:  "db, mock, err := sqlmock.New();",
			},
			{
				name:     "store directory",
				filePath: "pkg/store/cache_test.go",
				rawBody:  "db, mock, err := sqlmock.New();",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				unit := parser.TestUnit{
					Name:      "TestDatabaseAdapter",
					FilePath:  tc.filePath,
					StartLine: 10,
					RawBody:   tc.rawBody,
				}

				violations := checker.Evaluate(unit)
				if len(violations) != 0 {
					t.Errorf("expected 0 violations for %s, got %d", tc.filePath, len(violations))
				}
			})
		}
	})
}

func TestTableDuplicateRule(t *testing.T) {
	unitWithDuplicateRows := parser.TestUnit{
		Name:      "TestCalculateTax",
		FilePath:  "service/tax_test.go",
		StartLine: 15,
		RawBody: `
		tests := []struct{ in int; want int }{
			{in: 100, want: 10},
			{in: 100, want: 10}, // duplicate row!
		}
		`,
	}

	checker := golang.NewTableDuplicateRule()
	violations := checker.Evaluate(unitWithDuplicateRows)

	if len(violations) != 1 {
		t.Fatalf("expected 1 table duplicate violation, got %d", len(violations))
	}
	if violations[0].RuleID != "GUARD-GO-03" {
		t.Errorf("expected GUARD-GO-03, got %s", violations[0].RuleID)
	}
}
