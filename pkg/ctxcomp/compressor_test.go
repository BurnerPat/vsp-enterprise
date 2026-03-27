package ctxcomp

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// mockProvider implements SourceProvider for testing.
type mockProvider struct {
	sources map[string]string
}

func (m *mockProvider) GetSource(_ context.Context, kind DependencyKind, name string) (string, error) {
	key := string(kind) + ":" + name
	if src, ok := m.sources[key]; ok {
		return src, nil
	}
	return "", fmt.Errorf("not found: %s", name)
}

func TestCompressor_EndToEnd(t *testing.T) {
	mainSrc := `CLASS zcl_main DEFINITION PUBLIC.
  PUBLIC SECTION.
    DATA mo_helper TYPE REF TO zcl_helper.
    DATA mo_intf TYPE REF TO zif_service.
ENDCLASS.

CLASS zcl_main IMPLEMENTATION.
  METHOD constructor.
    mo_helper = NEW zcl_helper( ).
  ENDMETHOD.
ENDCLASS.`

	helperSrc := `CLASS zcl_helper DEFINITION PUBLIC CREATE PUBLIC.
  PUBLIC SECTION.
    METHODS run RETURNING VALUE(rv_ok) TYPE abap_bool.
    METHODS get_name RETURNING VALUE(rv_name) TYPE string.
  PRIVATE SECTION.
    DATA mv_internal TYPE string.
ENDCLASS.

CLASS zcl_helper IMPLEMENTATION.
  METHOD run.
    rv_ok = abap_true.
  ENDMETHOD.
  METHOD get_name.
    rv_name = 'helper'.
  ENDMETHOD.
ENDCLASS.`

	intfSrc := `INTERFACE zif_service PUBLIC.
  METHODS execute IMPORTING iv_input TYPE string RETURNING VALUE(rv_output) TYPE string.
ENDINTERFACE.`

	provider := &mockProvider{
		sources: map[string]string{
			"CLAS:ZCL_HELPER":  helperSrc,
			"INTF:ZIF_SERVICE": intfSrc,
		},
	}

	comp := NewCompressor(provider, 20)
	result, err := comp.Compress(context.Background(), mainSrc, "ZCL_MAIN", "CLAS")
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if result.Prologue == "" {
		t.Error("empty prologue")
	}
	if !strings.Contains(result.Prologue, "ZCL_HELPER") {
		t.Error("prologue missing ZCL_HELPER")
	}
	if !strings.Contains(result.Prologue, "ZIF_SERVICE") {
		t.Error("prologue missing ZIF_SERVICE")
	}
	if result.Stats.DepsResolved != 2 {
		t.Errorf("expected 2 resolved deps, got %d", result.Stats.DepsResolved)
	}
	if result.Stats.DepsFailed != 0 {
		t.Errorf("expected 0 failed deps, got %d", result.Stats.DepsFailed)
	}
}

func TestCompressor_MaxDeps(t *testing.T) {
	src := `METHOD test.
  DATA lo1 TYPE REF TO zcl_one.
  DATA lo2 TYPE REF TO zcl_two.
  DATA lo3 TYPE REF TO zcl_three.
  DATA lo4 TYPE REF TO zcl_four.
  DATA lo5 TYPE REF TO zcl_five.
ENDMETHOD.`

	provider := &mockProvider{sources: map[string]string{
		"CLAS:ZCL_ONE":   "CLASS zcl_one DEFINITION PUBLIC. ENDCLASS.",
		"CLAS:ZCL_TWO":   "CLASS zcl_two DEFINITION PUBLIC. ENDCLASS.",
		"CLAS:ZCL_THREE": "CLASS zcl_three DEFINITION PUBLIC. ENDCLASS.",
		"CLAS:ZCL_FOUR":  "CLASS zcl_four DEFINITION PUBLIC. ENDCLASS.",
		"CLAS:ZCL_FIVE":  "CLASS zcl_five DEFINITION PUBLIC. ENDCLASS.",
	}}

	comp := NewCompressor(provider, 3)
	result, err := comp.Compress(context.Background(), src, "ZCL_TEST", "CLAS")
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if result.Stats.DepsFound > 5 {
		t.Errorf("found too many deps: %d", result.Stats.DepsFound)
	}
	total := result.Stats.DepsResolved + result.Stats.DepsFailed
	if total > 3 {
		t.Errorf("expected at most 3 resolved+failed, got %d", total)
	}
}

func TestCompressor_FailedDeps(t *testing.T) {
	src := `METHOD test.
  DATA lo TYPE REF TO zcl_missing.
ENDMETHOD.`

	provider := &mockProvider{sources: map[string]string{}}

	comp := NewCompressor(provider, 20)
	result, err := comp.Compress(context.Background(), src, "ZCL_TEST", "CLAS")
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if result.Stats.DepsFailed == 0 {
		t.Error("expected at least 1 failed dep")
	}
}

func TestCompressor_EmptySource(t *testing.T) {
	provider := &mockProvider{sources: map[string]string{}}

	comp := NewCompressor(provider, 20)
	result, err := comp.Compress(context.Background(), "", "ZCL_EMPTY", "CLAS")
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	if result.Stats.DepsFound != 0 {
		t.Errorf("expected 0 deps for empty source, got %d", result.Stats.DepsFound)
	}
}

func TestCompressor_FiltersSelfReference(t *testing.T) {
	src := `CLASS zcl_self DEFINITION PUBLIC.
  PUBLIC SECTION.
    DATA mo TYPE REF TO zcl_self.
    DATA mo_other TYPE REF TO zcl_other.
ENDCLASS.`

	provider := &mockProvider{sources: map[string]string{
		"CLAS:ZCL_OTHER": `CLASS zcl_other DEFINITION PUBLIC.
  PUBLIC SECTION.
    METHODS ping.
ENDCLASS.`,
	}}

	comp := NewCompressor(provider, 20)
	result, err := comp.Compress(context.Background(), src, "ZCL_SELF", "CLAS")
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}

	for _, c := range result.Contracts {
		if c.Name == "ZCL_SELF" {
			t.Error("should filter self-reference")
		}
	}
	if result.Stats.DepsResolved < 1 {
		t.Error("expected at least 1 resolved dep (ZCL_OTHER)")
	}
}
