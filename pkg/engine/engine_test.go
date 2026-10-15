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

func TestEngineProgressCallback(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	_ = os.MkdirAll(testDir, 0755)

	file1 := filepath.Join(testDir, "a_test.go")
	_ = os.WriteFile(file1, []byte("package test\nfunc TestA(t *testing.T) {}\n"), 0644)

	file2 := filepath.Join(testDir, "b_test.go")
	_ = os.WriteFile(file2, []byte("package test\nfunc TestB(t *testing.T) {}\n"), 0644)

	type progressEvent struct {
		phase       string
		current     int
		total       int
		currentFile string
	}

	var events []progressEvent
	progressFn := func(phase string, current, total int, currentFile string) {
		events = append(events, progressEvent{
			phase:       phase,
			current:     current,
			total:       total,
			currentFile: currentFile,
		})
	}

	eng := engine.New()
	_, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir},
		ScanAll:     true,
		ProgressFn:  progressFn,
	})
	if err != nil {
		t.Fatalf("unexpected error running engine: %v", err)
	}

	if len(events) < 3 {
		t.Fatalf("expected at least 3 progress events, got %d", len(events))
	}

	// First event should be discovering
	if events[0].phase != "discovering" {
		t.Errorf("expected first event phase 'discovering', got %q", events[0].phase)
	}

	// Second event should be discovered with total = 2
	if events[1].phase != "discovered" || events[1].total != 2 {
		t.Errorf("expected second event 'discovered' with total 2, got %+v", events[1])
	}

	// Remaining events should be scanning
	scanningCount := 0
	for _, e := range events[2:] {
		if e.phase == "scanning" {
			scanningCount++
		}
	}
	if scanningCount != 2 {
		t.Errorf("expected 2 scanning events, got %d", scanningCount)
	}
}

func TestEngineTargetNotFound(t *testing.T) {
	eng := engine.New()
	_, err := eng.Run(engine.Options{
		TargetPaths: []string{"/nonexistent-directory-testguard-path-12345"},
	})
	if err == nil {
		t.Errorf("expected error for nonexistent target path, got nil")
	}
}

func TestEnginePathDeduplication(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test")
	_ = os.MkdirAll(testDir, 0755)

	file1 := filepath.Join(testDir, "sample_test.go")
	_ = os.WriteFile(file1, []byte("package test\nfunc TestSample(t *testing.T) {}\n"), 0644)

	eng := engine.New()
	res, err := eng.Run(engine.Options{
		TargetPaths: []string{tmpDir, testDir},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res.ScannedFiles != 1 {
		t.Errorf("expected exactly 1 scanned file due to deduplication, got %d", res.ScannedFiles)
	}
}
