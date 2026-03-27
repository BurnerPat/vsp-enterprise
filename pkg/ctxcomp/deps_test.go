package ctxcomp

import (
	"testing"
)

func TestExtractDependencies_TypeRefTo(t *testing.T) {
	src := `CLASS zcl_example DEFINITION PUBLIC.
  PUBLIC SECTION.
    DATA mo_helper TYPE REF TO zcl_my_helper.
    DATA mo_intf TYPE REF TO zif_processor.
ENDCLASS.`

	deps := ExtractDependencies(src)

	found := map[string]bool{}
	for _, d := range deps {
		found[d.Name] = true
	}
	if !found["ZCL_MY_HELPER"] {
		t.Error("missing ZCL_MY_HELPER from TYPE REF TO")
	}
	if !found["ZIF_PROCESSOR"] {
		t.Error("missing ZIF_PROCESSOR from TYPE REF TO")
	}
}

func TestExtractDependencies_Inheriting(t *testing.T) {
	src := `CLASS zcl_child DEFINITION PUBLIC INHERITING FROM zcl_base_handler CREATE PUBLIC.
  PUBLIC SECTION.
ENDCLASS.`

	deps := ExtractDependencies(src)

	found := false
	for _, d := range deps {
		if d.Name == "ZCL_BASE_HANDLER" && d.Kind == KindClass {
			found = true
		}
	}
	if !found {
		t.Error("missing ZCL_BASE_HANDLER from INHERITING FROM")
	}
}

func TestExtractDependencies_Interfaces(t *testing.T) {
	src := `CLASS zcl_impl DEFINITION PUBLIC.
  PUBLIC SECTION.
    INTERFACES zif_runnable.
    INTERFACES zif_loggable.
ENDCLASS.`

	deps := ExtractDependencies(src)

	found := map[string]bool{}
	for _, d := range deps {
		found[d.Name] = true
	}
	if !found["ZIF_RUNNABLE"] {
		t.Error("missing ZIF_RUNNABLE")
	}
	if !found["ZIF_LOGGABLE"] {
		t.Error("missing ZIF_LOGGABLE")
	}
}

func TestExtractDependencies_StaticCallAndNew(t *testing.T) {
	src := `METHOD main.
  DATA(lo_util) = NEW zcl_json_util( ).
  DATA(lv_val) = zcl_config=>get_value( 'KEY' ).
  CALL FUNCTION 'Z_MY_FUNC'
    EXPORTING iv_input = lv_val.
ENDMETHOD.`

	deps := ExtractDependencies(src)

	found := map[string]bool{}
	for _, d := range deps {
		found[d.Name] = true
	}
	if !found["ZCL_JSON_UTIL"] {
		t.Error("missing ZCL_JSON_UTIL from NEW")
	}
	if !found["ZCL_CONFIG"] {
		t.Error("missing ZCL_CONFIG from static call")
	}
	if !found["Z_MY_FUNC"] {
		t.Error("missing Z_MY_FUNC from CALL FUNCTION")
	}
}

func TestExtractDependencies_SkipsBuiltins(t *testing.T) {
	src := `METHOD test.
  DATA lv_str TYPE string.
  DATA lv_int TYPE i.
  DATA lo_ref TYPE REF TO zcl_custom.
ENDMETHOD.`

	deps := ExtractDependencies(src)

	for _, d := range deps {
		if d.Name == "STRING" || d.Name == "I" {
			t.Errorf("should skip built-in type %s", d.Name)
		}
	}
	found := false
	for _, d := range deps {
		if d.Name == "ZCL_CUSTOM" {
			found = true
		}
	}
	if !found {
		t.Error("missing ZCL_CUSTOM")
	}
}

func TestExtractDependencies_SkipsComments(t *testing.T) {
	src := `METHOD test.
* TYPE REF TO zcl_commented_out.
" DATA lo TYPE REF TO zcl_also_commented.
  DATA lo_real TYPE REF TO zcl_real_dep.
ENDMETHOD.`

	deps := ExtractDependencies(src)

	for _, d := range deps {
		if d.Name == "ZCL_COMMENTED_OUT" || d.Name == "ZCL_ALSO_COMMENTED" {
			t.Errorf("should skip commented dependency %s", d.Name)
		}
	}
	found := false
	for _, d := range deps {
		if d.Name == "ZCL_REAL_DEP" {
			found = true
		}
	}
	if !found {
		t.Error("missing ZCL_REAL_DEP")
	}
}

func TestExtractDependencies_Cast(t *testing.T) {
	src := `METHOD do_cast.
  DATA(lo) = CAST zif_service( lo_obj ).
ENDMETHOD.`

	deps := ExtractDependencies(src)
	found := false
	for _, d := range deps {
		if d.Name == "ZIF_SERVICE" {
			found = true
		}
	}
	if !found {
		t.Error("missing ZIF_SERVICE from CAST")
	}
}

func TestExtractDependencies_Raising(t *testing.T) {
	src := `CLASS zcl_svc DEFINITION PUBLIC.
  PUBLIC SECTION.
    METHODS run RAISING zcx_my_error.
ENDCLASS.`

	deps := ExtractDependencies(src)
	found := false
	for _, d := range deps {
		if d.Name == "ZCX_MY_ERROR" {
			found = true
		}
	}
	if !found {
		t.Error("missing ZCX_MY_ERROR from RAISING")
	}
}

func TestExtractDependencies_Deduplicates(t *testing.T) {
	src := `METHOD test.
  DATA lo1 TYPE REF TO zcl_helper.
  DATA lo2 TYPE REF TO zcl_helper.
  DATA(lo3) = NEW zcl_helper( ).
ENDMETHOD.`

	deps := ExtractDependencies(src)
	count := 0
	for _, d := range deps {
		if d.Name == "ZCL_HELPER" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 entry for ZCL_HELPER, got %d", count)
	}
}
