package reporter

import (
	"errors"
	"io"
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
)

type stubReportFile struct {
	writeErr error
	closeErr error
}

func (s *stubReportFile) Write(p []byte) (int, error) {
	if s.writeErr != nil {
		return 0, s.writeErr
	}
	return len(p), nil
}

func (s *stubReportFile) Close() error { return s.closeErr }

func TestWriteMarkdownReportPropagatesFileIOFailures(t *testing.T) {
	orig := createReportFile
	t.Cleanup(func() { createReportFile = orig })

	report := models.NewScanReport()

	openErr := errors.New("open denied")
	createReportFile = func(string) (io.WriteCloser, error) { return nil, openErr }
	if err := WriteMarkdownReport("report.md", report); !errors.Is(err, openErr) {
		t.Errorf("expected open failure, got %v", err)
	}

	writeErr := errors.New("disk full")
	createReportFile = func(string) (io.WriteCloser, error) {
		return &stubReportFile{writeErr: writeErr}, nil
	}
	if err := WriteMarkdownReport("report.md", report); !errors.Is(err, writeErr) {
		t.Errorf("expected post-open write failure, got %v", err)
	}

	closeErr := errors.New("close flush failed")
	createReportFile = func(string) (io.WriteCloser, error) {
		return &stubReportFile{closeErr: closeErr}, nil
	}
	if err := WriteMarkdownReport("report.md", report); !errors.Is(err, closeErr) {
		t.Errorf("expected close failure, got %v", err)
	}
}
