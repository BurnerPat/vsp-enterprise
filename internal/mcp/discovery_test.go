package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

func newTestRouter(systemIDs []string, toolsPerSystem map[string][]string) *Router {
	return newTestRouterWithPerms(systemIDs, toolsPerSystem, nil, nil)
}

// newTestRouterWithPerms creates a test router with optional per-tool restrictions and role names.
// toolPerms: map[systemID]map[toolName]*config.ResolvedToolPermission
// roleNames: map[systemID][]string
func newTestRouterWithPerms(
	systemIDs []string,
	toolsPerSystem map[string][]string,
	toolPerms map[string]map[string]*config.ResolvedToolPermission,
	roleNames map[string][]string,
) *Router {
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
			SystemID:        id,
			DisabledTools:   make(map[string]bool),
			ToolPermissions: make(map[string]*config.ResolvedToolPermission),
			RoleNames:       []string{"default"},
		}
		if roleNames != nil {
			if rn, ok := roleNames[id]; ok {
				sp.RoleNames = rn
			}
		}
		enabledSet := make(map[string]bool)
		for _, name := range toolsPerSystem[id] {
			enabledSet[name] = true
		}
		for i := range allDefs {
			toolName := allDefs[i].Tool.Name
			if enabledSet[toolName] {
				sp.EnabledTools = append(sp.EnabledTools, &allDefs[i])
				// Set tool permission if provided
				if toolPerms != nil {
					if sysPerms, ok := toolPerms[id]; ok {
						if tp, ok := sysPerms[toolName]; ok {
							sp.ToolPermissions[toolName] = tp
						}
					}
				}
			} else {
				sp.DisabledTools[toolName] = true
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

	if len(resp.Systems) != 2 {
		t.Fatalf("len(Systems) = %d, want 2", len(resp.Systems))
	}

	// Systems should be in the order they were added (DEV, PRD)
	if resp.Systems[0].SystemID != "DEV" {
		t.Errorf("Systems[0].SystemID = %q, want DEV", resp.Systems[0].SystemID)
	}
	if len(resp.Systems[0].EnabledTools) != 3 {
		t.Errorf("DEV enabled tools = %d, want 3", len(resp.Systems[0].EnabledTools))
	}
	if resp.Systems[1].SystemID != "PRD" {
		t.Errorf("Systems[1].SystemID = %q, want PRD", resp.Systems[1].SystemID)
	}
	if len(resp.Systems[1].EnabledTools) != 2 {
		t.Errorf("PRD enabled tools = %d, want 2", len(resp.Systems[1].EnabledTools))
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
	if len(resp.Systems[0].EnabledTools) != 1 {
		t.Errorf("enabled tools = %d, want 1", len(resp.Systems[0].EnabledTools))
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

// --- Discovery: restrictions ---

func TestListAvailableTools_NoRestrictions_OmitsBlock(t *testing.T) {
	r := newTestRouter(
		[]string{"DEV"},
		map[string][]string{"DEV": {"ReadObject", "GetSystemInfo"}},
	)

	result, err := r.handleListAvailableTools(context.Background(), makeRequest(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text, _ := mcp.AsTextContent(result.Content[0])

	// "restricted_tools" key should not appear when there are no object-level restrictions
	if strings.Contains(text.Text, "restricted_tools") {
		t.Errorf("response should omit restricted_tools when none exist, got:\n%s", text.Text)
	}
}

func TestListAvailableTools_WithRestrictions(t *testing.T) {
	r := newTestRouterWithPerms(
		[]string{"PRD"},
		map[string][]string{
			"PRD": {"DataPreview", "GetTableDefinition", "ReadObject"},
		},
		map[string]map[string]*config.ResolvedToolPermission{
			"PRD": {
				"DataPreview": {
					ObjectRestricted: true,
					BlockedObjects:   []string{"T001", "T000", "USR*"},
				},
				"GetTableDefinition": {
					ObjectRestricted: true,
					AllowedObjects:   []string{"Z*", "Y*"},
					AllowedPackages:  []string{"ZTEST"},
				},
				// ReadObject has no restrictions — should NOT appear in restricted_tools
			},
		},
		nil,
	)

	result, err := r.handleListAvailableTools(context.Background(), makeRequest(map[string]any{"system_id": "PRD"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp discoveryResponse
	text, _ := mcp.AsTextContent(result.Content[0])
	if err := json.Unmarshal([]byte(text.Text), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Systems) != 1 {
		t.Fatalf("expected 1 system, got %d", len(resp.Systems))
	}
	prd := resp.Systems[0]

	// restricted_tools should be at the system root level
	if len(prd.RestrictedTools) != 2 {
		t.Fatalf("restricted_tools count = %d, want 2", len(prd.RestrictedTools))
	}

	// Find DataPreview and GetTableDefinition restrictions
	var dpRestr, gtdRestr *DiscoveryToolInfo
	for i := range prd.RestrictedTools {
		switch prd.RestrictedTools[i].Name {
		case "DataPreview":
			dpRestr = &prd.RestrictedTools[i]
		case "GetTableDefinition":
			gtdRestr = &prd.RestrictedTools[i]
		}
	}

	if dpRestr == nil {
		t.Fatal("DataPreview should appear in restricted_tools")
	}
	if len(dpRestr.BlockedObjects) != 3 {
		t.Errorf("DataPreview blocked_objects = %v, want 3 entries", dpRestr.BlockedObjects)
	}
	if len(dpRestr.AllowedObjects) != 0 {
		t.Errorf("DataPreview allowed_objects should be empty, got %v", dpRestr.AllowedObjects)
	}

	if gtdRestr == nil {
		t.Fatal("GetTableDefinition should appear in restricted_tools")
	}
	if len(gtdRestr.AllowedObjects) != 2 {
		t.Errorf("GetTableDefinition allowed_objects = %v, want [Z*, Y*]", gtdRestr.AllowedObjects)
	}
	if len(gtdRestr.AllowedPackages) != 1 || gtdRestr.AllowedPackages[0] != "ZTEST" {
		t.Errorf("GetTableDefinition allowed_packages = %v, want [ZTEST]", gtdRestr.AllowedPackages)
	}
}
