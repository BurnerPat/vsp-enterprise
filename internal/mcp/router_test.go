package mcp

import "testing"

func TestSimpleGlob(t *testing.T) {
	if !simpleGlobMatch("Get*", "GetSource") {
		t.Error("prefix match failed")
	}

	if !simpleGlobMatch("*Source", "GetSource") {
		t.Error("suffix match failed")
	}

	if !simpleGlobMatch("GetSource", "GetSource") {
		t.Error("exact match failed")
	}

	if simpleGlobMatch("GetSource", "GetSources") {
		t.Error("exact match failed")
	}
}
