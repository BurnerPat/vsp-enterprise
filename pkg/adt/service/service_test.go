package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/oisee/vibing-steampunk/pkg/adt/transport"
)

// --------------------------------------------------------------------------
// SafetyChecker tests
// --------------------------------------------------------------------------

func TestNoopSafety(t *testing.T) {
	s := NoopSafety{}
	if err := s.CheckOp('R', "test"); err != nil {
		t.Errorf("NoopSafety.CheckOp should return nil: %v", err)
	}
	if err := s.CheckPkg("ZTEST"); err != nil {
		t.Errorf("NoopSafety.CheckPkg should return nil: %v", err)
	}
	if !s.IsPkgAllowed("ANYTHING") {
		t.Error("NoopSafety.IsPkgAllowed should always return true")
	}
}

// --------------------------------------------------------------------------
// blockingSafety blocks a specific operation
// --------------------------------------------------------------------------

type blockingSafety struct {
	blockedOp rune
}

func (b *blockingSafety) CheckOp(op rune, opName string) error {
	if op == b.blockedOp {
		return fmt.Errorf("operation %c (%s) is blocked", op, opName)
	}
	return nil
}
func (b *blockingSafety) CheckPkg(string) error                     { return nil }
func (b *blockingSafety) CheckTransportEdit(string, string) error   { return nil }
func (b *blockingSafety) CheckTransport(string, string, bool) error { return nil }
func (b *blockingSafety) IsPkgAllowed(string) bool                  { return true }

// --------------------------------------------------------------------------
// mockSender records requests and returns canned responses
// --------------------------------------------------------------------------

type mockSender struct {
	requests []*transport.AdtRequest
	respBody []byte
	respCode int
	err      error
}

func (m *mockSender) SendRequest(_ context.Context, req *transport.AdtRequest) (*transport.AdtResponse, error) {
	m.requests = append(m.requests, req)
	if m.err != nil {
		return nil, m.err
	}
	code := m.respCode
	if code == 0 {
		code = 200
	}
	return transport.NewAdtResponseFromMap(code, nil, m.respBody), nil
}

// --------------------------------------------------------------------------
// SourceService tests
// --------------------------------------------------------------------------

func TestSourceService_GetProgram(t *testing.T) {
	sender := &mockSender{respBody: []byte("REPORT ZTEST.")}
	svc := NewSourceService(sender, NoopSafety{}, ServiceConfig{})

	source, err := svc.GetProgram(context.Background(), "ztest")
	if err != nil {
		t.Fatalf("GetProgram: %v", err)
	}
	if source != "REPORT ZTEST." {
		t.Errorf("source = %q, want REPORT ZTEST.", source)
	}
	if len(sender.requests) != 1 {
		t.Fatalf("expected 1 request, got %d", len(sender.requests))
	}
	req := sender.requests[0]
	if req.Method != http.MethodGet {
		t.Errorf("Method = %s, want GET", req.Method)
	}
	if !strings.Contains(req.Path, "/programs/programs/ZTEST/source/main") {
		t.Errorf("Path = %s, missing expected path component", req.Path)
	}
}

func TestSourceService_GetClass(t *testing.T) {
	sender := &mockSender{respBody: []byte("CLASS zcl_test DEFINITION.")}
	svc := NewSourceService(sender, NoopSafety{}, ServiceConfig{})

	sources, err := svc.GetClass(context.Background(), "zcl_test")
	if err != nil {
		t.Fatalf("GetClass: %v", err)
	}
	if sources["main"] != "CLASS zcl_test DEFINITION." {
		t.Errorf("main source = %q", sources["main"])
	}
}

func TestSourceService_GetInterface(t *testing.T) {
	sender := &mockSender{respBody: []byte("INTERFACE zif_test PUBLIC.")}
	svc := NewSourceService(sender, NoopSafety{}, ServiceConfig{})

	source, err := svc.GetInterface(context.Background(), "zif_test")
	if err != nil {
		t.Fatalf("GetInterface: %v", err)
	}
	if source != "INTERFACE zif_test PUBLIC." {
		t.Errorf("source = %q", source)
	}
}

func TestSourceService_GetFunction(t *testing.T) {
	sender := &mockSender{respBody: []byte("FUNCTION z_test_func.")}
	svc := NewSourceService(sender, NoopSafety{}, ServiceConfig{})

	source, err := svc.GetFunction(context.Background(), "z_test_func", "z_test_group")
	if err != nil {
		t.Fatalf("GetFunction: %v", err)
	}
	if source != "FUNCTION z_test_func." {
		t.Errorf("source = %q", source)
	}
	req := sender.requests[0]
	if !strings.Contains(req.Path, "/functions/groups/Z_TEST_GROUP/fmodules/Z_TEST_FUNC/source/main") {
		t.Errorf("Path = %s", req.Path)
	}
}

func TestSourceService_RunQuery_SafetyBlocked(t *testing.T) {
	sender := &mockSender{}
	safety := &blockingSafety{blockedOp: OpFreeSQL}
	svc := NewSourceService(sender, safety, ServiceConfig{})

	_, err := svc.RunQuery(context.Background(), "SELECT * FROM T000", 10)
	if err == nil {
		t.Fatal("RunQuery should fail when free SQL is blocked")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("error = %q, should mention 'blocked'", err.Error())
	}
	if len(sender.requests) != 0 {
		t.Error("no request should have been sent when blocked by safety")
	}
}

func TestSourceService_RunQuery_Success(t *testing.T) {
	respXML := `<tableData><columns><metadata name="MANDT" type="C" length="3" keyAttribute="true"/><dataSet><data>001</data></dataSet></columns></tableData>`
	sender := &mockSender{respBody: []byte(respXML)}
	svc := NewSourceService(sender, NoopSafety{}, ServiceConfig{})

	result, err := svc.RunQuery(context.Background(), "SELECT MANDT FROM T000", 10)
	if err != nil {
		t.Fatalf("RunQuery: %v", err)
	}
	if len(result.Columns) != 1 {
		t.Fatalf("expected 1 column, got %d", len(result.Columns))
	}
	if result.Columns[0].Name != "MANDT" {
		t.Errorf("column name = %q, want MANDT", result.Columns[0].Name)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(result.Rows))
	}
}

func TestSourceService_SearchObject(t *testing.T) {
	respXML := `<objectReferences><objectReference uri="/sap/bc/adt/programs/programs/ztest" type="PROG/P" name="ZTEST" packageName="$TMP" description="Test"/></objectReferences>`
	sender := &mockSender{respBody: []byte(respXML)}
	svc := NewSourceService(sender, NoopSafety{}, ServiceConfig{})

	results, err := svc.SearchObject(context.Background(), "ZTEST*", 10)
	if err != nil {
		t.Fatalf("SearchObject: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Name != "ZTEST" {
		t.Errorf("name = %q, want ZTEST", results[0].Name)
	}
}

func TestSourceService_GetDDLS(t *testing.T) {
	sender := &mockSender{respBody: []byte("define view entity ZTEST as select from t000 { mandt }")}
	svc := NewSourceService(sender, NoopSafety{}, ServiceConfig{})

	source, err := svc.GetDDLS(context.Background(), "ztest")
	if err != nil {
		t.Fatalf("GetDDLS: %v", err)
	}
	if !strings.Contains(source, "define view") {
		t.Errorf("source = %q", source)
	}
}

func TestSourceService_GetTable(t *testing.T) {
	sender := &mockSender{respBody: []byte("@EndUserText.label: 'Test'\ndefine table ztest {")}
	svc := NewSourceService(sender, NoopSafety{}, ServiceConfig{})

	source, err := svc.GetTable(context.Background(), "ztest")
	if err != nil {
		t.Fatalf("GetTable: %v", err)
	}
	if !strings.Contains(source, "define table") {
		t.Errorf("source = %q", source)
	}
}

// --------------------------------------------------------------------------
// baseService safety integration
// --------------------------------------------------------------------------

func TestBaseService_CheckSafety_NilSafety(t *testing.T) {
	b := baseService{sender: &mockSender{}, safety: nil}
	if err := b.checkSafety('R', "test"); err != nil {
		t.Errorf("nil safety should allow everything: %v", err)
	}
}
