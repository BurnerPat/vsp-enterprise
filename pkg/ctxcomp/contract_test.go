package ctxcomp

import (
	"strings"
	"testing"
)

func TestExtractContract_Class(t *testing.T) {
	src := `CLASS zcl_example DEFINITION PUBLIC CREATE PUBLIC.
  PUBLIC SECTION.
    CLASS-METHODS escape_json IMPORTING iv_input TYPE string RETURNING VALUE(rv_result) TYPE string.
    METHODS extract_param IMPORTING iv_name TYPE string iv_json TYPE string RETURNING VALUE(rv_value) TYPE string.
    METHODS build_error IMPORTING iv_message TYPE string RETURNING VALUE(rv_json) TYPE string.
  PROTECTED SECTION.
    DATA mv_hidden TYPE string.
  PRIVATE SECTION.
    METHODS internal_helper IMPORTING iv_data TYPE string.
    DATA mv_buffer TYPE string.
ENDCLASS.

CLASS zcl_example IMPLEMENTATION.
  METHOD escape_json.
    rv_result = iv_input.
  ENDMETHOD.
  METHOD extract_param.
    rv_value = iv_name.
  ENDMETHOD.
  METHOD build_error.
    rv_json = iv_message.
  ENDMETHOD.
  METHOD internal_helper.
    mv_buffer = iv_data.
  ENDMETHOD.
ENDCLASS.`

	contract := ExtractContract(src, KindClass)

	if contract == "" {
		t.Fatal("empty contract")
	}
	if !strings.Contains(strings.ToUpper(contract), "CLASS ZCL_EXAMPLE DEFINITION") {
		t.Error("missing CLASS DEFINITION line")
	}
	if !strings.Contains(strings.ToUpper(contract), "PUBLIC SECTION") {
		t.Error("missing PUBLIC SECTION")
	}
	if !strings.Contains(contract, "escape_json") {
		t.Error("missing escape_json method")
	}
	if !strings.Contains(contract, "extract_param") {
		t.Error("missing extract_param method")
	}
	if strings.Contains(strings.ToUpper(contract), "IMPLEMENTATION") {
		t.Error("contract should not contain IMPLEMENTATION section")
	}
	if strings.Contains(strings.ToUpper(contract), "PRIVATE SECTION") {
		t.Error("contract should not contain PRIVATE SECTION")
	}
	ratio := float64(len(contract)) / float64(len(src))
	if ratio > 0.5 {
		t.Errorf("compression ratio too low: %.2f", ratio)
	}
}

func TestExtractContract_ClassInheriting(t *testing.T) {
	src := `CLASS zcl_child DEFINITION PUBLIC INHERITING FROM cl_base CREATE PUBLIC.
  PUBLIC SECTION.
    METHODS on_start REDEFINITION.
    METHODS on_message REDEFINITION.
  PRIVATE SECTION.
    METHODS parse_message IMPORTING iv_text TYPE string.
    METHODS send_response IMPORTING iv_data TYPE string.
ENDCLASS.

CLASS zcl_child IMPLEMENTATION.
  METHOD on_start.
  ENDMETHOD.
  METHOD on_message.
  ENDMETHOD.
  METHOD parse_message.
  ENDMETHOD.
  METHOD send_response.
  ENDMETHOD.
ENDCLASS.`

	contract := ExtractContract(src, KindClass)

	if contract == "" {
		t.Fatal("empty contract")
	}
	if !strings.Contains(strings.ToUpper(contract), "INHERITING FROM") {
		t.Error("missing INHERITING FROM")
	}
	if !strings.Contains(contract, "on_start") {
		t.Error("missing on_start method")
	}
	if strings.Contains(contract, "parse_message") {
		t.Error("should not contain private method parse_message")
	}
	if strings.Contains(strings.ToUpper(contract), "IMPLEMENTATION") {
		t.Error("should not contain IMPLEMENTATION")
	}
}

func TestExtractContract_Interface(t *testing.T) {
	src := `INTERFACE zif_processor PUBLIC.
  METHODS get_domain RETURNING VALUE(rv_domain) TYPE string.
  METHODS handle_message IMPORTING iv_message TYPE string RETURNING VALUE(rv_response) TYPE string.
  TYPES: BEGIN OF ty_node,
    name TYPE string,
    value TYPE string,
  END OF ty_node.
ENDINTERFACE.`

	contract := ExtractContract(src, KindInterface)

	if contract == "" {
		t.Fatal("empty contract")
	}
	if !strings.Contains(contract, "INTERFACE") {
		t.Error("missing INTERFACE keyword")
	}
	if !strings.Contains(contract, "ENDINTERFACE") {
		t.Error("missing ENDINTERFACE keyword")
	}
	if !strings.Contains(contract, "get_domain") {
		t.Error("missing get_domain method")
	}
	if !strings.Contains(contract, "handle_message") {
		t.Error("missing handle_message method")
	}
}

func TestExtractContract_FunctionModule(t *testing.T) {
	src := `FUNCTION ztest_my_func.
*"----------------------------------------------------------------------
*"*"Local Interface:
*"  IMPORTING
*"     VALUE(IV_NAME) TYPE  STRING
*"     VALUE(IV_COUNT) TYPE  I
*"  EXPORTING
*"     VALUE(EV_RESULT) TYPE  STRING
*"  EXCEPTIONS
*"      NOT_FOUND
*"----------------------------------------------------------------------
  DATA lv_temp TYPE string.
  lv_temp = iv_name.
  ev_result = lv_temp.
ENDFUNCTION.`

	contract := ExtractContract(src, KindFunction)

	if contract == "" {
		t.Fatal("empty contract")
	}
	if !strings.Contains(contract, "IV_NAME") {
		t.Error("missing IV_NAME parameter")
	}
	if !strings.Contains(contract, "EV_RESULT") {
		t.Error("missing EV_RESULT parameter")
	}
	if strings.Contains(contract, "lv_temp") {
		t.Error("should not contain function body")
	}
}
