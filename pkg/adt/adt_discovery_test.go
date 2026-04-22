package adt

import (
	"testing"
)

func TestParseDiscoveryResponse(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<app:service xmlns:app="http://www.w3.org/2007/app" xmlns:atom="http://www.w3.org/2005/Atom">
  <app:workspace>
    <atom:title>Programs</atom:title>
    <app:collection href="/sap/bc/adt/programs/programs">
      <atom:title>Programs</atom:title>
      <app:accept>application/vnd.sap.adt.programs.programs.v2+xml</app:accept>
      <app:accept>application/vnd.sap.adt.programs.programs.v3+xml</app:accept>
    </app:collection>
  </app:workspace>
  <app:workspace>
    <atom:title>OO</atom:title>
    <app:collection href="/sap/bc/adt/oo/classes">
      <atom:title>Classes</atom:title>
      <app:accept>application/vnd.sap.adt.oo.classes.v4+xml</app:accept>
    </app:collection>
    <app:collection href="/sap/bc/adt/oo/interfaces">
      <atom:title>Interfaces</atom:title>
    </app:collection>
  </app:workspace>
  <app:workspace>
    <atom:title>Data Dictionary</atom:title>
    <app:collection href="/sap/bc/adt/ddic/tables">
      <atom:title>Tables</atom:title>
    </app:collection>
  </app:workspace>
</app:service>`

	endpoints, err := parseDiscoveryResponse([]byte(xml))
	if err != nil {
		t.Fatalf("parseDiscoveryResponse failed: %v", err)
	}

	if len(endpoints) != 4 {
		t.Fatalf("expected 4 endpoints, got %d", len(endpoints))
	}

	// Check specific endpoints exist
	if _, ok := endpoints["/sap/bc/adt/programs/programs"]; !ok {
		t.Error("expected /sap/bc/adt/programs/programs endpoint")
	}
	if _, ok := endpoints["/sap/bc/adt/oo/classes"]; !ok {
		t.Error("expected /sap/bc/adt/oo/classes endpoint")
	}
	if _, ok := endpoints["/sap/bc/adt/oo/interfaces"]; !ok {
		t.Error("expected /sap/bc/adt/oo/interfaces endpoint")
	}
	if _, ok := endpoints["/sap/bc/adt/ddic/tables"]; !ok {
		t.Error("expected /sap/bc/adt/ddic/tables endpoint")
	}

	// Check accept headers
	ep := endpoints["/sap/bc/adt/programs/programs"]
	if len(ep.Accept) != 2 {
		t.Errorf("expected 2 accept headers for programs, got %d", len(ep.Accept))
	}

	ep = endpoints["/sap/bc/adt/oo/interfaces"]
	if len(ep.Accept) != 0 {
		t.Errorf("expected 0 accept headers for interfaces, got %d", len(ep.Accept))
	}
}

func TestDiscoveredEndpoints_HasEndpoint(t *testing.T) {
	eps := DiscoveredEndpoints{
		"/sap/bc/adt/programs/programs": ADTEndpoint{Path: "/sap/bc/adt/programs/programs"},
		"/sap/bc/adt/oo/classes":        ADTEndpoint{Path: "/sap/bc/adt/oo/classes"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"/sap/bc/adt/programs/programs", true},                   // exact match
		{"/sap/bc/adt/programs/programs/ZTEST", true},             // sub-path
		{"/sap/bc/adt/programs/programs/ZTEST/source/main", true}, // deep sub-path
		{"/sap/bc/adt/oo/classes", true},                          // exact match
		{"/sap/bc/adt/oo/classes/ZCL_TEST/source/main", true},     // sub-path
		{"/sap/bc/adt/oo/interfaces", false},                      // not in map
		{"/sap/bc/adt/debugger", false},                           // not in map
		{"/sap/bc/adt/programs/includes", false},                  // different endpoint
		{"/sap/bc/adt/programs/programsXXX", false},               // not a valid path segment
	}

	for _, tt := range tests {
		got := eps.HasEndpoint(tt.path)
		if got != tt.want {
			t.Errorf("HasEndpoint(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestDiscoveredEndpoints_HasEndpoint_Empty(t *testing.T) {
	var eps DiscoveredEndpoints
	if eps.HasEndpoint("/sap/bc/adt/programs/programs") {
		t.Error("empty endpoints should not match anything")
	}
}

func TestParseDiscoveryResponse_Empty(t *testing.T) {
	xml := `<?xml version="1.0" encoding="utf-8"?>
<app:service xmlns:app="http://www.w3.org/2007/app" xmlns:atom="http://www.w3.org/2005/Atom">
</app:service>`

	endpoints, err := parseDiscoveryResponse([]byte(xml))
	if err != nil {
		t.Fatalf("parseDiscoveryResponse failed: %v", err)
	}

	if len(endpoints) != 0 {
		t.Fatalf("expected 0 endpoints, got %d", len(endpoints))
	}
}
