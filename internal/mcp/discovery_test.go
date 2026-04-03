package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

func newTestRouter(systemIDs []string, toolsPerSystem map[string][]string) *Router {
	mcpSrv := server.NewMCPServer("test", "0.0.0")
	r := NewRouter(mcpSrv)

	allToolNames := make(map[string]bool)
	for _, names := range toolsPerSystem {
		for _, n := range names {
			allToolNames[n] = true
		}
	}

	// Build ToolDef stubs
	var allDefs []types.ToolDef
	for name := range allToolNames {
		allDefs = append(allDefs, types.ToolDef{
			Tool: mcp.NewTool(name, mcp.WithDescription("test tool")),
		})
	}
	r.allTools = allDefs

	for _, id := range systemIDs {
		r.systems[strings.ToLower(id)] = nil // no real System needed for these tests
		r.systemIDs = append(r.systemIDs, id)
	}

	// Build a PermissionManager with the given tool assignments
	pm := &PermissionManager{
		systemPermissions: make(map[string]*SystemPermissions),
		allTools:          allDefs,
	}

	for _, id := range systemIDs {
		sp := &SystemPermissions{
			SystemID:      id,
			DisabledTools: make(map[string]bool),
		}
		enabledSet := make(map[string]bool)
		for _, name := range toolsPerSystem[id] {
			enabledSet[name] = true
		}
		for i := range allDefs {
			if enabledSet[allDefs[i].Tool.Name] {
				sp.EnabledTools = append(sp.EnabledTools, &allDefs[i])
			} else {
				sp.DisabledTools[allDefs[i].Tool.Name] = true
			}
		}
		pm.systemPermissions[strings.ToLower(id)] = sp
	}
	r.permissionManager = pm

	return r
}

func makeRequest(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

// --- ListAvailableTools tests ---

func TestListAvailableTools_AllSystems(t *testing.T) {
	r := newTestRouter(
		[]string{"DEV", "PRD"},
		map[string][]string{
			"DEV": {"ReadObject", "SearchObjects", "GetSystemInfo"},
			"PRD": {"ReadObject", "GetSystemInfo"},
		},
	)

	result, err := r.handleListAvailableTools(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result)
	}

	var resp discoveryResponse
	text, _ := mcp.AsTextContent(result.Content[0])
	if err := json.Unmarshal([]byte(text.Text), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.TotalSystems != 2 {
		t.Errorf("TotalSystems = %d, want 2", resp.TotalSystems)
	}
	if len(resp.Systems) != 2 {
		t.Fatalf("len(Systems) = %d, want 2", len(resp.Systems))
	}

	// Systems should be in the order they were added (DEV, PRD)
	if resp.Systems[0].SystemID != "DEV" {
		t.Errorf("Systems[0].SystemID = %q, want DEV", resp.Systems[0].SystemID)
	}
	if resp.Systems[0].TotalEnabled != 3 {
		t.Errorf("DEV TotalEnabled = %d, want 3", resp.Systems[0].TotalEnabled)
	}
	if resp.Systems[1].SystemID != "PRD" {
		t.Errorf("Systems[1].SystemID = %q, want PRD", resp.Systems[1].SystemID)
	}
	if resp.Systems[1].TotalEnabled != 2 {
		t.Errorf("PRD TotalEnabled = %d, want 2", resp.Systems[1].TotalEnabled)
	}
}

func TestListAvailableTools_FilterBySystemID(t *testing.T) {
	r := newTestRouter(
		[]string{"DEV", "PRD"},
		map[string][]string{
			"DEV": {"ReadObject", "SearchObjects"},
			"PRD": {"ReadObject"},
		},
	)

	result, err := r.handleListAvailableTools(context.Background(), makeRequest(map[string]any{"system_id": "PRD"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error result: %+v", result)
	}

	var resp discoveryResponse
	text, _ := mcp.AsTextContent(result.Content[0])
	if err := json.Unmarshal([]byte(text.Text), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Systems) != 1 {
		t.Fatalf("len(Systems) = %d, want 1", len(resp.Systems))
	}
	if resp.Systems[0].SystemID != "PRD" {
		t.Errorf("SystemID = %q, want PRD", resp.Systems[0].SystemID)
	}
	if resp.Systems[0].TotalEnabled != 1 {
		t.Errorf("TotalEnabled = %d, want 1", resp.Systems[0].TotalEnabled)
	}
	// TotalSystems still reports the global count
	if resp.TotalSystems != 2 {
		t.Errorf("TotalSystems = %d, want 2", resp.TotalSystems)
	}
}

func TestListAvailableTools_UnknownSystem(t *testing.T) {
	r := newTestRouter(
		[]string{"DEV"},
		map[string][]string{"DEV": {"ReadObject"}},
	)

	result, err := r.handleListAvailableTools(context.Background(), makeRequest(map[string]any{"system_id": "NOPE"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected error result for unknown system")
	}

	text, _ := mcp.AsTextContent(result.Content[0])
	if !strings.Contains(text.Text, "Unknown system: NOPE") {
		t.Errorf("error message %q should contain 'Unknown system: NOPE'", text.Text)
	}
	if !strings.Contains(text.Text, "DEV") {
		t.Errorf("error message %q should list available system IDs", text.Text)
	}
}

func TestListAvailableTools_SingleSystem(t *testing.T) {
	r := newTestRouter(
		[]string{"DEV"},
		map[string][]string{"DEV": {"GetSystemInfo", "ReadObject"}},
	)

	result, err := r.handleListAvailableTools(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp discoveryResponse
	text, _ := mcp.AsTextContent(result.Content[0])
	if err := json.Unmarshal([]byte(text.Text), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.TotalSystems != 1 {
		t.Errorf("TotalSystems = %d, want 1", resp.TotalSystems)
	}
	if len(resp.Systems) != 1 {
		t.Fatalf("len(Systems) = %d, want 1", len(resp.Systems))
	}
	// Tool names should be sorted
	if resp.Systems[0].EnabledTools[0] != "GetSystemInfo" {
		t.Errorf("first tool = %q, want GetSystemInfo (sorted)", resp.Systems[0].EnabledTools[0])
	}
}

func TestListAvailableTools_ToolsSorted(t *testing.T) {
	r := newTestRouter(
		[]string{"SYS"},
		map[string][]string{"SYS": {"Zebra", "Alpha", "Middle"}},
	)

	result, err := r.handleListAvailableTools(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp discoveryResponse
	text, _ := mcp.AsTextContent(result.Content[0])
	json.Unmarshal([]byte(text.Text), &resp)

	want := []string{"Alpha", "Middle", "Zebra"}
	for i, name := range resp.Systems[0].EnabledTools {
		if name != want[i] {
			t.Errorf("EnabledTools[%d] = %q, want %q", i, name, want[i])
		}
	}
}

// --- Permission denied error message tests ---

func TestPermissionDeniedMessage_ShowsAvailableTools(t *testing.T) {
	r := newTestRouter(
		[]string{"DEV"},
		map[string][]string{"DEV": {"ReadObject", "SearchObjects", "GetSystemInfo"}},
	)

	msg := r.permissionDeniedMessage("DeleteObject", "DEV", r.permissionManager.GetEnabledToolsForSystem("DEV"))

	if !strings.Contains(msg, "Permission denied for tool DeleteObject on system DEV") {
		t.Errorf("message %q missing denial statement", msg)
	}
	if !strings.Contains(msg, "ReadObject") {
		t.Errorf("message %q should list available tools", msg)
	}
	if !strings.Contains(msg, "Available tools:") {
		t.Errorf("message %q should contain 'Available tools:'", msg)
	}
}

func TestPermissionDeniedMessage_NoTools(t *testing.T) {
	r := newTestRouter(
		[]string{"DEV"},
		map[string][]string{"DEV": {}},
	)

	msg := r.permissionDeniedMessage("ReadObject", "DEV", r.permissionManager.GetEnabledToolsForSystem("DEV"))

	if !strings.Contains(msg, "No tools are enabled") {
		t.Errorf("message %q should indicate no tools enabled", msg)
	}
}

func TestPermissionDeniedMessage_CapsAt10(t *testing.T) {
	tools := make([]string, 15)
	for i := range tools {
		tools[i] = "Tool" + string(rune('A'+i))
	}

	r := newTestRouter([]string{"SYS"}, map[string][]string{"SYS": tools})
	msg := r.permissionDeniedMessage("Blocked", "SYS", r.permissionManager.GetEnabledToolsForSystem("SYS"))

	if !strings.Contains(msg, "and 5 more") {
		t.Errorf("message %q should indicate truncated list", msg)
	}
	if !strings.Contains(msg, "ListAvailableTools") {
		t.Errorf("message %q should hint at ListAvailableTools", msg)
	}
}

// --- Unknown system error message test ---

func TestHandleToolCall_UnknownSystemListsAvailable(t *testing.T) {
	r := newTestRouter(
		[]string{"DEV", "PRD"},
		map[string][]string{
			"DEV": {"ReadObject"},
			"PRD": {"ReadObject"},
		},
	)

	td := &types.ToolDef{Tool: mcp.NewTool("ReadObject")}
	result, err := r.HandleToolCall(context.Background(), td, makeRequest(map[string]any{"system_id": "NOPE"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text, _ := mcp.AsTextContent(result.Content[0])
	if !strings.Contains(text.Text, "DEV") || !strings.Contains(text.Text, "PRD") {
		t.Errorf("error %q should list available system IDs", text.Text)
	}
}
