package reporter_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/reporter"
)

func TestProgressReporterUpdatesAndClears(t *testing.T) {
	var buf bytes.Buffer
	p := reporter.NewProgressReporterWithOptions(&buf, reporter.ProgressConfig{
		IsTerminal: true,
		NoColor:    false,
		Disabled:   false,
	})

	p.Update("discovering", 0, 0, "")
	p.Update("scanning", 5, 20, "path/to/test.go")
	p.Finish()

	out := buf.String()
	if !strings.Contains(out, "Discovering test files...") {
		t.Errorf("expected buffer to contain discovering notice, got %q", out)
	}
	if !strings.Contains(out, "Scanning test files [5/20]") {
		t.Errorf("expected buffer to contain scanning count [5/20], got %q", out)
	}
	if !strings.Contains(out, "path/to/test.go") {
		t.Errorf("expected buffer to contain file path, got %q", out)
	}
	if !strings.Contains(out, "\033[2K") {
		t.Errorf("expected buffer to contain line clear escape, got %q", out)
	}
}

func TestProgressReporterNonTerminal(t *testing.T) {
	var buf bytes.Buffer
	p := reporter.NewProgressReporterWithOptions(&buf, reporter.ProgressConfig{
		IsTerminal: false,
		NoColor:    true,
		Disabled:   false,
	})

	p.Update("discovering", 0, 0, "")
	p.Update("scanning", 5, 20, "path/to/test.go")
	p.Finish()

	out := buf.String()
	if strings.Contains(out, "\033[2K") || strings.Contains(out, "\r") {
		t.Errorf("expected non-terminal output to omit carriage returns and ANSI escapes, got %q", out)
	}
}

func TestProgressReporterDisabled(t *testing.T) {
	var buf bytes.Buffer
	p := reporter.NewProgressReporterWithOptions(&buf, reporter.ProgressConfig{
		Disabled: true,
	})

	p.Update("discovering", 0, 0, "")
	p.Update("scanning", 1, 10, "foo_test.go")
	p.Finish()

	if buf.Len() != 0 {
		t.Errorf("expected zero bytes for disabled reporter, got %d", buf.Len())
	}
}

func TestProgressReporterTruncatePath(t *testing.T) {
	longPath := "datasource/ezrx/mobile/submodules/features/order/presentation/bloc/order_detail_bloc_test.dart"
	short := reporter.TruncatePath(longPath, 40)
	if len(short) > 40 {
		t.Errorf("expected truncated length <= 40, got %d (%s)", len(short), short)
	}
	if !strings.HasPrefix(short, "...") {
		t.Errorf("expected truncated path to start with '...', got %s", short)
	}

	multiByte := "datasource/ezrx/mobile/订单详情_test.dart"
	truncatedRune := reporter.TruncatePath(multiByte, 15)
	if len([]rune(truncatedRune)) > 15 {
		t.Errorf("expected truncated runes <= 15, got %d", len([]rune(truncatedRune)))
	}
}
