package engine_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/engine"
	"github.com/HoangNguyen0403/testguard/pkg/models"
)

func TestEngineAuditDartBlocFile(t *testing.T) {
	tmpDir := t.TempDir()
	pubspecPath := filepath.Join(tmpDir, "pubspec.yaml")
	_ = os.WriteFile(pubspecPath, []byte("name: sample_app\ndev_dependencies:\n  flutter_test:\n    sdk: flutter\n"), 0644)

	testDir := filepath.Join(tmpDir, "test")
	_ = os.MkdirAll(testDir, 0755)

	testFile := filepath.Join(testDir, "customer_code_bloc_test.dart")
	content := []byte(`
void main() {
  blocTest<CustomerCodeBloc, CustomerCodeState>(
    'Fetch_MissingLoadingState',
    seed: () => baseState.copyWith(userInfo: fakeUser),
    build: () => CustomerCodeBloc(),
    act: (bloc) => bloc.add(const CustomerCodeEvent.fetch()),
    expect: () => [
      isA<CustomerCodeState>().having((s) => s.userInfo, 'userInfo', fakeUser),
    ],
  );
}
`)
	_ = os.WriteFile(testFile, content, 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
		ScanAll:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error running engine: %v", err)
	}

	for _, v := range res.Violations {
		fmt.Printf("Detected violation: %+v\n", v)
	}

	hasUncoveredMutation := false
	hasSeedEcho := false
	for _, v := range res.Violations {
		if v.RuleID == "GUARD-DART-UNCOVERED-MUTATION" && v.Severity == models.SeverityBlocker {
			hasUncoveredMutation = true
		}
		if v.RuleID == "GUARD-DART-SEED-ECHO" && v.Severity == models.SeverityWarning {
			hasSeedEcho = true
		}
	}

	if !hasUncoveredMutation {
		t.Errorf("expected to detect GUARD-DART-UNCOVERED-MUTATION")
	}
	if !hasSeedEcho {
		t.Errorf("expected to detect GUARD-DART-SEED-ECHO")
	}
}
