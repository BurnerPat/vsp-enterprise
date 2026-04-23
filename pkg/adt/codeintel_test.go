package adt

import (
	"testing"
)

func TestParseObjectOutline(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<abapsource:objectStructureElement xmlns:abapsource="http://www.sap.com/adt/abapsource"
    xmlns:adtcore="http://www.sap.com/adt/core"
    xmlns:atom="http://www.w3.org/2005/Atom"
    adtcore:name="ZCL_TEST" adtcore:type="CLAS/OC" visibility="public">
  <atom:link href="/sap/bc/adt/oo/classes/ZCL_TEST" rel="self" type="application/xml"/>
  <abapsource:objectStructureElement adtcore:name="CONSTRUCTOR" adtcore:type="CLAS/OM"
      visibility="public" isStatic="false" isFinal="false" isAbstract="false"
      description="Constructor">
    <atom:link href="/sap/bc/adt/oo/classes/ZCL_TEST/methods/CONSTRUCTOR" rel="self"/>
  </abapsource:objectStructureElement>
  <abapsource:objectStructureElement adtcore:name="PROCESS_DATA" adtcore:type="CLAS/OM"
      visibility="public" isStatic="false" isFinal="true" isAbstract="false"
      description="Process data method">
    <atom:link href="/sap/bc/adt/oo/classes/ZCL_TEST/methods/PROCESS_DATA" rel="self"/>
  </abapsource:objectStructureElement>
  <abapsource:objectStructureElement adtcore:name="HELPER" adtcore:type="CLAS/OM"
      visibility="private" isStatic="true" isFinal="false" isAbstract="false">
    <atom:link href="/sap/bc/adt/oo/classes/ZCL_TEST/methods/HELPER" rel="self"/>
  </abapsource:objectStructureElement>
  <abapsource:objectStructureElement adtcore:name="MV_DATA" adtcore:type="CLAS/OD"
      visibility="private" readOnly="false">
    <atom:link href="/sap/bc/adt/oo/classes/ZCL_TEST/attributes/MV_DATA" rel="self"/>
  </abapsource:objectStructureElement>
  <abapsource:objectStructureElement adtcore:name="MC_CONSTANT" adtcore:type="CLAS/OD"
      visibility="public" constant="true" readOnly="true">
    <atom:link href="/sap/bc/adt/oo/classes/ZCL_TEST/attributes/MC_CONSTANT" rel="self"/>
  </abapsource:objectStructureElement>
  <abapsource:objectStructureElement adtcore:name="ON_CHANGE" adtcore:type="CLAS/OE"
      visibility="public">
    <atom:link href="/sap/bc/adt/oo/classes/ZCL_TEST/events/ON_CHANGE" rel="self"/>
  </abapsource:objectStructureElement>
</abapsource:objectStructureElement>`

	result, err := parseObjectOutline([]byte(xmlData), "/sap/bc/adt/oo/classes/ZCL_TEST/objectstructure")
	if err != nil {
		t.Fatalf("parseObjectOutline failed: %v", err)
	}

	// Check root element
	if result.Name != "ZCL_TEST" {
		t.Errorf("expected name 'ZCL_TEST', got '%s'", result.Name)
	}
	if result.Type != "CLAS/OC" {
		t.Errorf("expected type 'CLAS/OC', got '%s'", result.Type)
	}
	if result.Visibility != "public" {
		t.Errorf("expected visibility 'public', got '%s'", result.Visibility)
	}

	// Check children count
	if len(result.Children) != 6 {
		t.Fatalf("expected 6 children, got %d", len(result.Children))
	}

	// Check method with description
	constructor := result.Children[0]
	if constructor.Name != "CONSTRUCTOR" {
		t.Errorf("expected first child name 'CONSTRUCTOR', got '%s'", constructor.Name)
	}
	if constructor.Description != "Constructor" {
		t.Errorf("expected description 'Constructor', got '%s'", constructor.Description)
	}

	// Check final method
	processData := result.Children[1]
	if !processData.IsFinal {
		t.Error("expected PROCESS_DATA to be final")
	}

	// Check static method
	helper := result.Children[2]
	if !helper.IsStatic {
		t.Error("expected HELPER to be static")
	}
	if helper.Visibility != "private" {
		t.Errorf("expected HELPER visibility 'private', got '%s'", helper.Visibility)
	}

	// Check constant attribute
	constant := result.Children[4]
	if !constant.IsConstant {
		t.Error("expected MC_CONSTANT to be constant")
	}
	if !constant.IsReadOnly {
		t.Error("expected MC_CONSTANT to be read-only")
	}

	// Check event
	event := result.Children[5]
	if event.Name != "ON_CHANGE" {
		t.Errorf("expected event name 'ON_CHANGE', got '%s'", event.Name)
	}

	// Check that hrefs are absolute
	if constructor.Href != "/sap/bc/adt/oo/classes/ZCL_TEST/methods/CONSTRUCTOR" {
		t.Errorf("expected absolute href, got '%s'", constructor.Href)
	}
}

func TestParseObjectOutlineRelativeHref(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<abapsource:objectStructureElement xmlns:abapsource="http://www.sap.com/adt/abapsource"
    xmlns:adtcore="http://www.sap.com/adt/core"
    xmlns:atom="http://www.w3.org/2005/Atom"
    adtcore:name="ZCL_61CAR_TLOG_TOOL" adtcore:type="CLAS/OC" visibility="public">
  <abapsource:objectStructureElement adtcore:name="GET_FILES_FOR_QUICK_ACCESS" adtcore:type="CLAS/OM" visibility="private">
    <atom:link href="./../zcl_61car_tlog_tool/source/main#start=61,12;end=61,38" rel="http://www.sap.com/adt/relations/definitionIdentifier"/>
  </abapsource:objectStructureElement>
  <abapsource:objectStructureElement adtcore:name="ALREADY_ABSOLUTE" adtcore:type="CLAS/OM" visibility="public">
    <atom:link href="/sap/bc/adt/oo/classes/ZCL_61CAR_TLOG_TOOL/methods/ALREADY_ABSOLUTE" rel="self"/>
  </abapsource:objectStructureElement>
</abapsource:objectStructureElement>`

	result, err := parseObjectOutline([]byte(xmlData), "/sap/bc/adt/oo/classes/ZCL_61CAR_TLOG_TOOL/objectstructure")
	if err != nil {
		t.Fatalf("parseObjectOutline failed: %v", err)
	}

	// Relative href should be resolved to absolute
	method := result.Children[0]
	expected := "/sap/bc/adt/oo/classes/zcl_61car_tlog_tool/source/main#start=61,12;end=61,38"
	if method.Href != expected {
		t.Errorf("expected resolved href '%s', got '%s'", expected, method.Href)
	}

	// Already absolute href should stay absolute
	abs := result.Children[1]
	if abs.Href != "/sap/bc/adt/oo/classes/ZCL_61CAR_TLOG_TOOL/methods/ALREADY_ABSOLUTE" {
		t.Errorf("expected absolute href unchanged, got '%s'", abs.Href)
	}
}

func TestParseObjectOutlineEmpty(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<abapsource:objectStructureElement xmlns:abapsource="http://www.sap.com/adt/abapsource"
    xmlns:adtcore="http://www.sap.com/adt/core"
    adtcore:name="ZCL_EMPTY" adtcore:type="CLAS/OC" visibility="public">
</abapsource:objectStructureElement>`

	result, err := parseObjectOutline([]byte(xmlData), "/sap/bc/adt/oo/classes/ZCL_EMPTY/objectstructure")
	if err != nil {
		t.Fatalf("parseObjectOutline failed: %v", err)
	}

	if result.Name != "ZCL_EMPTY" {
		t.Errorf("expected name 'ZCL_EMPTY', got '%s'", result.Name)
	}
	if len(result.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(result.Children))
	}
}

func TestParseObjectOutlineNested(t *testing.T) {
	// Test nested children (e.g., local types within methods)
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<abapsource:objectStructureElement xmlns:abapsource="http://www.sap.com/adt/abapsource"
    xmlns:adtcore="http://www.sap.com/adt/core"
    adtcore:name="ZCL_NESTED" adtcore:type="CLAS/OC" visibility="public">
  <abapsource:objectStructureElement adtcore:name="OUTER_METHOD" adtcore:type="CLAS/OM" visibility="public">
    <abapsource:objectStructureElement adtcore:name="LT_LOCAL" adtcore:type="CLAS/OT" visibility="private">
    </abapsource:objectStructureElement>
  </abapsource:objectStructureElement>
</abapsource:objectStructureElement>`

	result, err := parseObjectOutline([]byte(xmlData), "/sap/bc/adt/oo/classes/ZCL_NESTED/objectstructure")
	if err != nil {
		t.Fatalf("parseObjectOutline failed: %v", err)
	}

	if len(result.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(result.Children))
	}

	outerMethod := result.Children[0]
	if outerMethod.Name != "OUTER_METHOD" {
		t.Errorf("expected outer method name 'OUTER_METHOD', got '%s'", outerMethod.Name)
	}

	// Check nested child
	if len(outerMethod.Children) != 1 {
		t.Fatalf("expected 1 nested child, got %d", len(outerMethod.Children))
	}

	localType := outerMethod.Children[0]
	if localType.Name != "LT_LOCAL" {
		t.Errorf("expected local type name 'LT_LOCAL', got '%s'", localType.Name)
	}
}
