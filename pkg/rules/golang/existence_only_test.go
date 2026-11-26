package golang_test

import (
	"testing"

	"github.com/HoangNguyen0403/testguard/pkg/models"
	"github.com/HoangNguyen0403/testguard/pkg/parser"
	"github.com/HoangNguyen0403/testguard/pkg/rules/golang"
)

func TestExistenceOnlyRule(t *testing.T) {
	rule := golang.NewExistenceOnlyRule()

	tests := []struct {
		name     string
		unit     parser.TestUnit
		wantWarn bool
	}{
		{
			name: "Multi-return call asserting only NoError and NotNil triggers warning",
			unit: parser.TestUnit{
				Name:      "TestFetchUser_Success",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestFetchUser_Success(t *testing.T) {
    user, err := svc.GetUser(ctx, 123)
    assert.NoError(t, err)
    assert.NotNil(t, user)
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Multi-return call asserting only err == nil and res != nil triggers warning",
			unit: parser.TestUnit{
				Name:      "TestFindOrder_Success",
				FilePath:  "internal/repo/order_test.go",
				StartLine: 20,
				RawBody: `
func TestFindOrder_Success(t *testing.T) {
    order, err := repo.Find(ctx, "ORD-1")
    if err != nil {
        t.Fatal(err)
    }
    if order == nil {
        t.Fatal("expected order")
    }
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Inspecting res.Field does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestFetchUser_InspectsField",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestFetchUser_InspectsField(t *testing.T) {
    user, err := svc.GetUser(ctx, 123)
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "Alice", user.Name)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Error expectation test does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestFetchUser_NotFound",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestFetchUser_NotFound(t *testing.T) {
    user, err := svc.GetUser(ctx, 999)
    assert.Error(t, err)
    assert.Nil(t, user)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Passing res into assertion helper does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestFetchUser_HelperValidation",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestFetchUser_HelperValidation(t *testing.T) {
    user, err := svc.GetUser(ctx, 123)
    assert.NoError(t, err)
    assertValidUser(t, user)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Single-return error call does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestDeleteUser_Success",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestDeleteUser_Success(t *testing.T) {
    err := svc.DeleteUser(ctx, 123)
    assert.NoError(t, err)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Standard library fixture call does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestMarshalPayload",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestMarshalPayload(t *testing.T) {
    bytes, err := json.Marshal(map[string]string{"foo": "bar"})
    assert.NoError(t, err)
    assert.NotNil(t, bytes)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Type-asserted wrapper payload field does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestUploadGinWrapperField",
				FilePath:  "pkg/handler/fs_test.go",
				StartLine: 33,
				RawBody: `
func TestUploadGinWrapperField(t *testing.T) {
    file, err := c.FormFile("upload")
    assert.NoError(t, err)
    assert.NotNil(t, file)
    assert.Equal(t, "report.pdf", file.(Wrapper).File.Filename)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Parenthesized payload field does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestParenthesizedPayloadField",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestParenthesizedPayloadField(t *testing.T) {
    user, err := svc.GetUser(ctx, 1)
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "Alice", (user).Name)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Parenthesized payload index does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestParenthesizedPayloadIndex",
				FilePath:  "internal/service/items_test.go",
				StartLine: 10,
				RawBody: `
func TestParenthesizedPayloadIndex(t *testing.T) {
    items, err := svc.ListItems(ctx)
    assert.NoError(t, err)
    assert.NotNil(t, items)
    assert.Equal(t, "first", (items)[0])
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Nested parentheses and type assertion payload field does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestNestedWrapperPayloadField",
				FilePath:  "pkg/handler/fs_test.go",
				StartLine: 33,
				RawBody: `
func TestNestedWrapperPayloadField(t *testing.T) {
    file, err := c.FormFile("upload")
    assert.NoError(t, err)
    assert.NotNil(t, file)
    assert.Equal(t, "report.pdf", ((file).(Wrapper)).File.Filename)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Observation of shadowed same-named result retains warning",
			unit: parser.TestUnit{
				Name:      "TestShadowedResultObservation",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestShadowedResultObservation(t *testing.T) {
    user, err := svc.GetUser(ctx, 1)
    assert.NoError(t, err)
    assert.NotNil(t, user)
    {
        user := buildUser("Bob")
        assert.Equal(t, "Bob", user.Name)
    }
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Outer observation after nested shadow block does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestObservationAfterShadowBlock",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestObservationAfterShadowBlock(t *testing.T) {
    user, err := svc.GetUser(ctx, 1)
    assert.NoError(t, err)
    assert.NotNil(t, user)
    {
        user := buildUser("Bob")
        assert.Equal(t, "Bob", user.Name)
    }
    assert.Equal(t, "Alice", user.Name)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Type-asserted observation of shadowed result retains warning",
			unit: parser.TestUnit{
				Name:      "TestShadowedWrapperObservation",
				FilePath:  "pkg/handler/fs_test.go",
				StartLine: 33,
				RawBody: `
func TestShadowedWrapperObservation(t *testing.T) {
    file, err := c.FormFile("upload")
    assert.NoError(t, err)
    assert.NotNil(t, file)
    {
        file := otherFile()
        assert.Equal(t, "x", file.(Wrapper).File.Filename)
    }
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Observation of unrelated variable field retains warning",
			unit: parser.TestUnit{
				Name:      "TestUnrelatedFieldObservation",
				FilePath:  "internal/service/user_test.go",
				StartLine: 10,
				RawBody: `
func TestUnrelatedFieldObservation(t *testing.T) {
    user, err := svc.GetUser(ctx, 1)
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.Equal(t, "prod", cfg.Name)
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Direct boolean result branch condition does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestDarwinAppProcessRunningReportsExitedProcess",
				FilePath:  "app/cmd/app/app_darwin_test.go",
				StartLine: 3000,
				RawBody: `
func TestDarwinAppProcessRunningReportsExitedProcess(t *testing.T) {
    running, err := darwinAppProcessRunning(appProcessIdentity{pid: 1 << 30, startedAt: 1})
    if err != nil {
        t.Fatal(err)
    }
    if running {
        t.Fatal("nonexistent process reported as running")
    }
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Negated boolean result branch condition does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestCloudDisabledAfterMigration",
				FilePath:  "app/store/migration_test.go",
				StartLine: 160,
				RawBody: `
func TestCloudDisabledAfterMigration(t *testing.T) {
    disabled, err := s.CloudDisabled()
    if err != nil {
        t.Fatalf("CloudDisabled() error: %v", err)
    }
    if !disabled {
        t.Fatal("expected cloud to be disabled")
    }
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Boolean result switch condition does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestBooleanSwitchCondition",
				FilePath:  "internal/service/flag_test.go",
				StartLine: 10,
				RawBody: `
func TestBooleanSwitchCondition(t *testing.T) {
    enabled, err := svc.Enabled(ctx)
    if err != nil {
        t.Fatal(err)
    }
    switch enabled {
    case true:
        t.Fatal("unexpected enabled")
    }
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Boolean result for-loop condition does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestBooleanLoopCondition",
				FilePath:  "internal/service/queue_test.go",
				StartLine: 10,
				RawBody: `
func TestBooleanLoopCondition(t *testing.T) {
    pending, err := svc.Pending(ctx)
    if err != nil {
        t.Fatal(err)
    }
    for pending {
        t.Fatal("queue should be drained")
    }
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Boolean result assertion argument does not trigger warning",
			unit: parser.TestUnit{
				Name:      "TestBooleanAssertArgument",
				FilePath:  "internal/service/flag_test.go",
				StartLine: 10,
				RawBody: `
func TestBooleanAssertArgument(t *testing.T) {
    enabled, err := svc.Enabled(ctx)
    assert.NoError(t, err)
    assert.True(t, enabled)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Observation of shadowed same-named boolean predicate retains warning",
			unit: parser.TestUnit{
				Name:      "TestShadowedBooleanPredicate",
				FilePath:  "internal/service/flag_test.go",
				StartLine: 10,
				RawBody: `
func TestShadowedBooleanPredicate(t *testing.T) {
    enabled, err := svc.Enabled(ctx)
    assert.NoError(t, err)
    assert.NotNil(t, enabled)
    {
        enabled := otherFlag()
        if enabled {
            t.Fatal("shadow observation")
        }
    }
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Err-only check with unused boolean result retains warning",
			unit: parser.TestUnit{
				Name:      "TestUnusedBooleanResult",
				FilePath:  "internal/service/flag_test.go",
				StartLine: 10,
				RawBody: `
func TestUnusedBooleanResult(t *testing.T) {
    enabled, err := svc.Enabled(ctx)
    if err != nil {
        t.Fatal(err)
    }
    _ = enabled
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Range operand observes result payload",
			unit: parser.TestUnit{
				Name:      "TestReadDirEntries",
				FilePath:  "src/os/read_test.go",
				StartLine: 123,
				RawBody: `
func TestReadDirEntries(t *testing.T) {
    dirname := "."
    list, err := ReadDir(dirname)
    if err != nil {
        t.Fatalf("ReadDir %s: %v", dirname, err)
    }
    foundFile := false
    for _, dir := range list {
        if dir.Name() == "read_test.go" {
            foundFile = true
        }
    }
    _ = foundFile
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Return operand observes result payload",
			unit: parser.TestUnit{
				Name:      "TestSeekOffset",
				FilePath:  "internal/coverage/slicewriter/slw_test.go",
				StartLine: 52,
				RawBody: `
func TestSeekOffset(t *testing.T) {
    sk := func(t *testing.T, ws *WriteSeeker, offset int64, whence int) int64 {
        t.Helper()
        off, err := ws.Seek(offset, whence)
        if err != nil {
            t.Fatalf("unexpected seek error: %v", err)
        }
        return off
    }
    _ = sk
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Compound-assignment RHS operand observes result payload",
			unit: parser.TestUnit{
				Name:      "TestSocketReadTotal",
				FilePath:  "src/net/tcpsock_test.go",
				StartLine: 559,
				RawBody: `
func TestSocketReadTotal(t *testing.T) {
    read := 0
    n, err := c.Read(buf)
    read += n
    if err != nil {
        t.Log(err)
    }
    _ = read
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Star-deref operand observes result payload",
			unit: parser.TestUnit{
				Name:      "TestParseNamedManifestDeref",
				FilePath:  "server/images_test.go",
				StartLine: 460,
				RawBody: `
func TestParseNamedManifestDeref(t *testing.T) {
    mf, err := manifest.ParseNamedManifestForRunner(name, runner)
    if err != nil {
        t.Fatal(err)
    }
    got := *mf
    _ = got
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Composite literal element observes result payload",
			unit: parser.TestUnit{
				Name:      "TestCompositeElementObservation",
				FilePath:  "internal/service/batch_test.go",
				StartLine: 10,
				RawBody: `
func TestCompositeElementObservation(t *testing.T) {
    items, err := svc.List(ctx)
    if err != nil {
        t.Fatal(err)
    }
    payload := struct{ Items []Item }{Items: items}
    _ = payload
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Channel send operand observes result payload",
			unit: parser.TestUnit{
				Name:      "TestChannelSendObservation",
				FilePath:  "internal/service/queue_test.go",
				StartLine: 10,
				RawBody: `
func TestChannelSendObservation(t *testing.T) {
    n, err := svc.Count(ctx)
    if err != nil {
        t.Fatal(err)
    }
    ch := make(chan int, 1)
    ch <- n
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Append argument observes result payload",
			unit: parser.TestUnit{
				Name:      "TestAppendObservation",
				FilePath:  "internal/service/items_test.go",
				StartLine: 10,
				RawBody: `
func TestAppendObservation(t *testing.T) {
    items, err := svc.List(ctx)
    if err != nil {
        t.Fatal(err)
    }
    all := append(nil, items)
    _ = all
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Only nil comparison on result retains warning",
			unit: parser.TestUnit{
				Name:      "TestOnlyNilComparison",
				FilePath:  "src/os/read_test.go",
				StartLine: 130,
				RawBody: `
func TestOnlyNilComparison(t *testing.T) {
    list, err := ReadDir(dirname)
    if err != nil {
        t.Fatal(err)
    }
    if list != nil {
        t.Fatal("expected nil listing")
    }
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Blank LHS with call RHS still observes selector result",
			unit: parser.TestUnit{
				Name:      "TestMultipartWrite",
				FilePath:  "gin/context_test.go",
				StartLine: 153,
				RawBody: `
func TestMultipartWrite(t *testing.T) {
    w, err := mw.CreateFormFile("file", "test")
    require.NoError(t, err)
    _, err = w.Write([]byte("test"))
    require.NoError(t, err)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "If-init blank LHS with call RHS observes selector result",
			unit: parser.TestUnit{
				Name:      "TestZipWrite",
				FilePath:  "src/archive/zip/fuzz_test.go",
				StartLine: 66,
				RawBody: `
func TestZipWrite(t *testing.T) {
    ww, err := newWriter()
    if err != nil {
        t.Fatal(err)
    }
    if _, err := ww.Write(f.content); err != nil {
        t.Fatal(err)
    }
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Blank LHS with method-call RHS observes selector result",
			unit: parser.TestUnit{
				Name:      "TestWriterName",
				FilePath:  "internal/service/writer_test.go",
				StartLine: 10,
				RawBody: `
func TestWriterName(t *testing.T) {
    w, err := svc.NewWriter(ctx)
    if err != nil {
        t.Fatal(err)
    }
    _ = w.Name()
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Two blank LHS with single call RHS observes call argument",
			unit: parser.TestUnit{
				Name:      "TestFprintWrite",
				FilePath:  "internal/service/format_test.go",
				StartLine: 10,
				RawBody: `
func TestFprintWrite(t *testing.T) {
    w, err := svc.NewWriter(ctx)
    if err != nil {
        t.Fatal(err)
    }
    _, _ = fmt.Fprint(w, 1)
}
`,
			},
			wantWarn: false,
		},
		{
			name: "Blank-discarded bare result retains warning",
			unit: parser.TestUnit{
				Name:      "TestBlankDiscardResult",
				FilePath:  "internal/service/items_test.go",
				StartLine: 10,
				RawBody: `
func TestBlankDiscardResult(t *testing.T) {
    items, err := svc.List(ctx)
    if err != nil {
        t.Fatal(err)
    }
    _ = items
}
`,
			},
			wantWarn: true,
		},
		{
			name: "Positional blank-discarded bare result retains warning",
			unit: parser.TestUnit{
				Name:      "TestPositionalBlankDiscard",
				FilePath:  "internal/service/items_test.go",
				StartLine: 10,
				RawBody: `
func TestPositionalBlankDiscard(t *testing.T) {
    items, err := svc.List(ctx)
    if err != nil {
        t.Fatal(err)
    }
    _, _ = items, other
}
`,
			},
			wantWarn: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			violations := rule.Evaluate(tc.unit)
			if tc.wantWarn {
				if len(violations) == 0 {
					t.Fatalf("expected violation, got 0")
				}
				v := violations[0]
				if v.RuleID != "GUARD-GO-EXISTENCE-ONLY" {
					t.Errorf("expected RuleID GUARD-GO-EXISTENCE-ONLY, got %s", v.RuleID)
				}
				if v.Severity != models.SeverityWarning {
					t.Errorf("expected Severity Warning, got %s", v.Severity)
				}
			} else {
				if len(violations) != 0 {
					t.Fatalf("expected 0 violations, got %d: %+v", len(violations), violations)
				}
			}
		})
	}
}
