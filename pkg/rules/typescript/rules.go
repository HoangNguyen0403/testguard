package typescript

import (
	"fmt"
	"regexp"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
)

var (
	nestControllerMockRegex = regexp.MustCompile(`mockResolvedValue\s*\(\s*([^,\)]+)\)`)
	nestControllerExpectRegex = regexp.MustCompile(`expect\s*\(\s*await\s+controller\.\w+\([^)]*\)\s*\)\.(?:resolves\.)?(?:toBe|toEqual)\s*\(\s*([^,\)]+)\)`)
	nestRealDbModuleRegex   = regexp.MustCompile(`(?i)(?:TypeOrmModule\.forRoot|MongooseModule\.forRoot|SequelizeModule\.forRoot|PrismaModule)`)
	sleepRegex              = regexp.MustCompile(`(?i)(?:setTimeout\s*\(|new\s+Promise\s*\(\s*(?:\([^)]*\)|r)\s*=>\s*setTimeout)`)
)

// NestControllerEchoRule flags NestJS controller unit tests that do 1:1 pass-through mock echoing.
type NestControllerEchoRule struct{}

func NewNestControllerEchoRule() *NestControllerEchoRule {
	return &NestControllerEchoRule{}
}

func (r *NestControllerEchoRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	mockMatch := nestControllerMockRegex.FindStringSubmatch(unit.RawBody)
	expectMatch := nestControllerExpectRegex.FindStringSubmatch(unit.RawBody)

	if len(mockMatch) > 1 && len(expectMatch) > 1 {
		if mockMatch[1] == expectMatch[1] {
			violations = append(violations, models.Violation{
				RuleID:     "GUARD-NEST-01",
				Severity:   models.SeverityWarning,
				FilePath:   unit.FilePath,
				LineNumber: unit.StartLine,
				Column:     1,
				Message:    fmt.Sprintf("Pass-through controller echo in '%s': controller test simply returns mock '%s' without validation or transformation.", unit.Name, mockMatch[1]),
				Suggestion: "Test validation pipes, guards, interceptors, or integration behavior via supertest rather than 1:1 controller pass-through mocks.",
			})
		}
	}

	return violations
}

// NestUnmockedDbModuleRule traps unmocked database modules inside NestJS unit tests.
type NestUnmockedDbModuleRule struct{}

func NewNestUnmockedDbModuleRule() *NestUnmockedDbModuleRule {
	return &NestUnmockedDbModuleRule{}
}

func (r *NestUnmockedDbModuleRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	if nestRealDbModuleRegex.MatchString(unit.RawBody) {
		violations = append(violations, models.Violation{
			RuleID:     "GUARD-NEST-02",
			Severity:   models.SeverityBlocker,
			FilePath:   unit.FilePath,
			LineNumber: unit.StartLine,
			Column:     1,
			Message:    fmt.Sprintf("Unmocked database module leak in '%s': unit test imports real database module.", unit.Name),
			Suggestion: "Use getRepositoryToken() or mock providers in Test.createTestingModule instead of importing real database connection modules.",
		})
	}

	return violations
}

// ReactSleepRule traps hardcoded setTimeout / sleep calls in React and frontend test bodies.
type ReactSleepRule struct{}

func NewReactSleepRule() *ReactSleepRule {
	return &ReactSleepRule{}
}

func (r *ReactSleepRule) Evaluate(unit parser.TestUnit) []models.Violation {
	var violations []models.Violation

	if sleepRegex.MatchString(unit.RawBody) {
		violations = append(violations, models.Violation{
			RuleID:     "GUARD-REACT-02",
			Severity:   models.SeverityBlocker,
			FilePath:   unit.FilePath,
			LineNumber: unit.StartLine,
			Column:     1,
			Message:    fmt.Sprintf("Arbitrary sleep/setTimeout fragility in '%s': using hardcoded timer delays in test.", unit.Name),
			Suggestion: "Use React Testing Library's waitFor(...) or async findBy* queries instead of fixed setTimeout sleeps.",
		})
	}

	return violations
}
