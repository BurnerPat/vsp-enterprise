package mcp

import "testing"

func TestSimpleGlob(t *testing.T) {
	if !simpleGlobMatch("Get*", "GetSource") ||
		simpleGlobMatch("Get*", "WriteSource") ||
		simpleGlobMatch("Get*", "DebuggerGetVariable") {

		t.Error("prefix match failed")
	}

	if !simpleGlobMatch("*Source", "GetSource") ||
		simpleGlobMatch("*Source", "GetSources") {

		t.Error("suffix match failed")
	}

	if !simpleGlobMatch("GetSource", "GetSource") {
		t.Error("exact match failed")
	}

	if simpleGlobMatch("GetSource", "GetSources") {
		t.Error("exact match failed")
	}
}
