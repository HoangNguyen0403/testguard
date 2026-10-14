package docs_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/docs"
	"github.com/HoangNguyen0403/testguard/pkg/models"
)

func TestExplainKnownRule(t *testing.T) {
	doc, found := docs.Explain("GUARD-DART-SEED-ECHO")
	if !found {
		t.Fatalf("expected rule GUARD-DART-SEED-ECHO to be found")
	}
	if doc.ID != "GUARD-DART-SEED-ECHO" {
		t.Errorf("expected ID to be GUARD-DART-SEED-ECHO, got %s", doc.ID)
	}
	if doc.Severity != models.SeverityWarning {
		t.Errorf("expected severity Warning, got %s", doc.Severity)
	}
	if doc.Title == "" || doc.Remediation == "" {
		t.Errorf("expected complete documentation for GUARD-DART-SEED-ECHO")
	}
}

func TestListAllRules(t *testing.T) {
	catalog := docs.AllRules()
	if len(catalog) == 0 {
		t.Fatalf("expected non-empty rule catalog")
	}
}
