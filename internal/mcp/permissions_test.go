package mcp

import (
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

func setupTestConfig() {
	cfg := &config.GlobalConfig{}
	config.SetInstance(cfg)
}

func testToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("GetClass", mcp.WithDescription("test"))},
		{Tool: mcp.NewTool("GetProgram", mcp.WithDescription("test"))},
		{Tool: mcp.NewTool("SearchObject", mcp.WithDescription("test"))},
		{Tool: mcp.NewTool("RunQuery", mcp.WithDescription("test"))},
		{Tool: mcp.NewTool("DataPreview", mcp.WithDescription("test"))},
		{Tool: mcp.NewTool("GetTableDefinition", mcp.WithDescription("test"))},
		{Tool: mcp.NewTool("DeleteObject", mcp.WithDescription("test"))},
	}
}

func boolP(v bool) *bool { return &v }

// ---------------------------------------------------------------------------
// NewPermissionManager
// ---------------------------------------------------------------------------

func TestNewPermissionManager_NoRoles(t *testing.T) {
	setupTestConfig()
	cfg := &config.GlobalConfig{}
	cfg.Systems = map[string]config.SystemConfig{
		"dev": {ConnectionConfig: config.ConnectionConfig{URL: "http://dev:50000"}},
	}

	pm, err := NewPermissionManager(cfg, testToolDefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No roles defined → system gets built-in "default" role (all tools enabled)
	tools := pm.GetEnabledToolsForSystem("dev")
	if len(tools) != len(testToolDefs()) {
		t.Errorf("dev should have all %d tools with default role, got %d", len(testToolDefs()), len(tools))
	}
}

func TestNewPermissionManager_WithRoles(t *testing.T) {
	setupTestConfig()
	cfg := &config.GlobalConfig{}
	cfg.GlobalConfigJSON = config.GlobalConfigJSON{
		Roles: map[string]config.RoleDefinition{
			"reader": {
				Tools: map[string]config.ToolPermission{
					"Get*":    {},
					"Search*": {},
				},
			},
		},
		Systems: map[string]config.SystemConfig{
			"dev": {
				ConnectionConfig: config.ConnectionConfig{URL: "http://dev:50000"},
				Roles:            []string{"reader"},
			},
		},
	}

	pm, err := NewPermissionManager(cfg, testToolDefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// reader role should only enable Get* and Search* tools
	tools := pm.GetEnabledToolsForSystem("dev")
	enabledNames := make(map[string]bool)
	for _, td := range tools {
		enabledNames[td.Tool.Name] = true
	}
	if !enabledNames["GetClass"] {
		t.Error("GetClass should be enabled for reader role")
	}
	if enabledNames["RunQuery"] {
		t.Error("RunQuery should NOT be enabled for reader role")
	}
}

// ---------------------------------------------------------------------------
// GetEnabledToolsForSystem
// ---------------------------------------------------------------------------

func TestPermissionManager_GetEnabledToolsForSystem(t *testing.T) {
	setupTestConfig()
	cfg := &config.GlobalConfig{}
	cfg.GlobalConfigJSON = config.GlobalConfigJSON{
		Roles: map[string]config.RoleDefinition{
			"reader": {
				Tools: map[string]config.ToolPermission{
					"Get*":    {},
					"Search*": {},
				},
			},
		},
		Systems: map[string]config.SystemConfig{
			"dev": {
				ConnectionConfig: config.ConnectionConfig{URL: "http://dev:50000"},
				Roles:            []string{"reader"},
			},
		},
	}

	pm, err := NewPermissionManager(cfg, testToolDefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tools := pm.GetEnabledToolsForSystem("dev")
	if len(tools) == 0 {
		t.Fatal("expected some tools to be enabled")
	}

	// Check that Get* and Search* tools are enabled
	enabledNames := make(map[string]bool)
	for _, td := range tools {
		enabledNames[td.Tool.Name] = true
	}

	if !enabledNames["GetClass"] {
		t.Error("GetClass should be enabled")
	}
	if !enabledNames["GetProgram"] {
		t.Error("GetProgram should be enabled")
	}
	if !enabledNames["SearchObject"] {
		t.Error("SearchObject should be enabled")
	}
	if enabledNames["RunQuery"] {
		t.Error("RunQuery should NOT be enabled")
	}
	if enabledNames["DeleteObject"] {
		t.Error("DeleteObject should NOT be enabled")
	}
}

// ---------------------------------------------------------------------------
// IsToolEnabledForSystem
// ---------------------------------------------------------------------------

func TestPermissionManager_IsToolEnabledForSystem(t *testing.T) {
	setupTestConfig()
	cfg := &config.GlobalConfig{}
	cfg.GlobalConfigJSON = config.GlobalConfigJSON{
		Roles: map[string]config.RoleDefinition{
			"reader": {
				Tools: map[string]config.ToolPermission{
					"Get*": {},
				},
			},
		},
		Systems: map[string]config.SystemConfig{
			"dev": {
				ConnectionConfig: config.ConnectionConfig{URL: "http://dev:50000"},
				Roles:            []string{"reader"},
			},
		},
	}

	pm, err := NewPermissionManager(cfg, testToolDefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !pm.IsToolEnabledForSystem("dev", "GetClass") {
		t.Error("GetClass should be enabled for dev")
	}
	if pm.IsToolEnabledForSystem("dev", "RunQuery") {
		t.Error("RunQuery should NOT be enabled for dev")
	}
}

// ---------------------------------------------------------------------------
// IsObjectAllowedForTool
// ---------------------------------------------------------------------------

func TestPermissionManager_IsObjectAllowedForTool_Allowed(t *testing.T) {
	setupTestConfig()
	cfg := &config.GlobalConfig{}
	cfg.GlobalConfigJSON = config.GlobalConfigJSON{
		Roles: map[string]config.RoleDefinition{
			"restricted": {
				Tools: map[string]config.ToolPermission{
					"DataPreview": {
						AllowedObjects: []string{"Z*", "Y*"},
					},
				},
			},
		},
		Systems: map[string]config.SystemConfig{
			"prod": {
				ConnectionConfig: config.ConnectionConfig{URL: "http://prod:44300"},
				Roles:            []string{"restricted"},
			},
		},
	}

	pm, err := NewPermissionManager(cfg, testToolDefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := pm.IsObjectAllowedForTool("prod", "DataPreview", "ZAPP", ""); err != nil {
		t.Errorf("ZAPP should be allowed: %v", err)
	}
	if err := pm.IsObjectAllowedForTool("prod", "DataPreview", "YCONFIG", ""); err != nil {
		t.Errorf("YCONFIG should be allowed: %v", err)
	}
}

func TestPermissionManager_IsObjectAllowedForTool_Blocked(t *testing.T) {
	setupTestConfig()
	cfg := &config.GlobalConfig{}
	cfg.GlobalConfigJSON = config.GlobalConfigJSON{
		Roles: map[string]config.RoleDefinition{
			"restricted": {
				Tools: map[string]config.ToolPermission{
					"DataPreview": {
						AllowedObjects: []string{"Z*", "Y*"},
						BlockedObjects: []string{"Z001"},
					},
				},
			},
		},
		Systems: map[string]config.SystemConfig{
			"prod": {
				ConnectionConfig: config.ConnectionConfig{URL: "http://prod:44300"},
				Roles:            []string{"restricted"},
			},
		},
	}

	pm, err := NewPermissionManager(cfg, testToolDefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Z001 is blocked even though it matches Z*
	if err := pm.IsObjectAllowedForTool("prod", "DataPreview", "Z001", ""); err == nil {
		t.Error("Z001 should be blocked")
	}

	// Z002 should be allowed (matches Z* and not blocked)
	if err := pm.IsObjectAllowedForTool("prod", "DataPreview", "Z002", ""); err != nil {
		t.Errorf("Z002 should be allowed: %v", err)
	}

	// T001 should NOT be allowed (doesn't match Z* or Y*)
	if err := pm.IsObjectAllowedForTool("prod", "DataPreview", "T001", ""); err == nil {
		t.Error("T001 should not be in allowed list")
	}
}

func TestPermissionManager_IsObjectAllowedForTool_Wildcards(t *testing.T) {
	setupTestConfig()
	cfg := &config.GlobalConfig{}
	cfg.GlobalConfigJSON = config.GlobalConfigJSON{
		Roles: map[string]config.RoleDefinition{
			"cloud_safe": {
				Tools: map[string]config.ToolPermission{
					"DataPreview": {
						BlockedObjects: []string{"T001", "T000", "USR*"},
					},
				},
			},
		},
		Systems: map[string]config.SystemConfig{
			"prod": {
				ConnectionConfig: config.ConnectionConfig{URL: "http://prod:44300"},
				Roles:            []string{"cloud_safe"},
			},
		},
	}

	pm, err := NewPermissionManager(cfg, testToolDefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// T001 is explicitly blocked
	if err := pm.IsObjectAllowedForTool("prod", "DataPreview", "T001", ""); err == nil {
		t.Error("T001 should be blocked")
	}

	// USR* should block all USR tables
	if err := pm.IsObjectAllowedForTool("prod", "DataPreview", "USRT001", ""); err == nil {
		t.Error("USRT001 should be blocked by USR* pattern")
	}

	// ZAPP is not blocked
	if err := pm.IsObjectAllowedForTool("prod", "DataPreview", "ZAPP", ""); err != nil {
		t.Errorf("ZAPP should be allowed: %v", err)
	}
}

func TestPermissionManager_IsObjectAllowedForTool_NoRestrictions(t *testing.T) {
	setupTestConfig()
	cfg := &config.GlobalConfig{}
	cfg.GlobalConfigJSON = config.GlobalConfigJSON{
		Roles: map[string]config.RoleDefinition{
			"open": {
				Tools: map[string]config.ToolPermission{
					"*": {},
				},
			},
		},
		Systems: map[string]config.SystemConfig{
			"dev": {
				ConnectionConfig: config.ConnectionConfig{URL: "http://dev:50000"},
				Roles:            []string{"open"},
			},
		},
	}

	pm, err := NewPermissionManager(cfg, testToolDefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No restrictions → everything allowed
	if err := pm.IsObjectAllowedForTool("dev", "DataPreview", "T001", ""); err != nil {
		t.Errorf("should be allowed with no restrictions: %v", err)
	}
}

// ---------------------------------------------------------------------------
// ObjectNotAllowedMessage
// ---------------------------------------------------------------------------

func TestPermissionManager_ObjectNotAllowedMessage(t *testing.T) {
	setupTestConfig()
	pm := &PermissionManager{}
	msg := pm.ObjectNotAllowedMessage("prod", "DataPreview", "T001")

	if msg == "" {
		t.Error("message should not be empty")
	}
	if !containsStr(msg, "T001") || !containsStr(msg, "DataPreview") || !containsStr(msg, "prod") {
		t.Errorf("message should contain object, tool, and system: %q", msg)
	}
}

// ---------------------------------------------------------------------------
// Default role (system with no roles and roles defined)
// ---------------------------------------------------------------------------

func TestPermissionManager_DefaultRoleFallback(t *testing.T) {
	setupTestConfig()
	cfg := &config.GlobalConfig{}
	cfg.GlobalConfigJSON = config.GlobalConfigJSON{
		Roles: map[string]config.RoleDefinition{
			"reader": {
				Tools: map[string]config.ToolPermission{
					"Get*": {},
				},
			},
		},
		Systems: map[string]config.SystemConfig{
			"dev": {
				ConnectionConfig: config.ConnectionConfig{URL: "http://dev:50000"},
				// No roles → uses built-in "default"
			},
			"prod": {
				ConnectionConfig: config.ConnectionConfig{URL: "http://prod:44300"},
				Roles:            []string{"reader"},
			},
		},
	}

	pm, err := NewPermissionManager(cfg, testToolDefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// dev should have all tools (default role)
	devTools := pm.GetEnabledToolsForSystem("dev")
	if len(devTools) != len(testToolDefs()) {
		t.Errorf("dev should have all %d tools, got %d", len(testToolDefs()), len(devTools))
	}

	// prod should only have Get* tools
	prodTools := pm.GetEnabledToolsForSystem("prod")
	for _, td := range prodTools {
		if td.Tool.Name == "RunQuery" {
			t.Error("prod should not have RunQuery")
		}
	}
}

// ---------------------------------------------------------------------------
// Composite roles
// ---------------------------------------------------------------------------

func TestPermissionManager_CompositeRoles(t *testing.T) {
	setupTestConfig()
	cfg := &config.GlobalConfig{}
	cfg.GlobalConfigJSON = config.GlobalConfigJSON{
		Roles: map[string]config.RoleDefinition{
			"reader": {
				Tools: map[string]config.ToolPermission{
					"Get*":    {},
					"Search*": {},
				},
			},
			"table_viewer": {
				Tools: map[string]config.ToolPermission{
					"GetTableDefinition": {AllowedObjects: []string{"*"}},
					"DataPreview":        {BlockedObjects: []string{"T001", "USR*"}},
				},
			},
			"prod_role": {
				NestedRoles: []string{"reader", "table_viewer"},
				Tools: map[string]config.ToolPermission{
					"RunQuery": {Enabled: boolP(false)},
				},
			},
		},
		Systems: map[string]config.SystemConfig{
			"prod": {
				ConnectionConfig: config.ConnectionConfig{URL: "http://prod:44300"},
				Roles:            []string{"prod_role"},
			},
		},
	}

	pm, err := NewPermissionManager(cfg, testToolDefs())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// RunQuery should be disabled
	if pm.IsToolEnabledForSystem("prod", "RunQuery") {
		t.Error("RunQuery should be disabled")
	}

	// GetClass should be enabled (from reader role)
	if !pm.IsToolEnabledForSystem("prod", "GetClass") {
		t.Error("GetClass should be enabled")
	}

	// DataPreview should block T001
	if err := pm.IsObjectAllowedForTool("prod", "DataPreview", "T001", ""); err == nil {
		t.Error("T001 should be blocked for DataPreview")
	}

	// DataPreview should allow ZAPP
	if err := pm.IsObjectAllowedForTool("prod", "DataPreview", "ZAPP", ""); err != nil {
		t.Errorf("ZAPP should be allowed for DataPreview: %v", err)
	}
}

func TestFilterToolsByEndpoints_EmptyDiscovery(t *testing.T) {
	tools := []*types.ToolDef{
		{Tool: mcp.NewTool("GetProgram"), Endpoints: []string{"/sap/bc/adt/programs/programs"}},
		{Tool: mcp.NewTool("GetClass"), Endpoints: []string{"/sap/bc/adt/oo/classes"}},
	}

	// Nil endpoints = no filtering
	result := FilterToolsByEndpoints(tools, nil)
	if len(result) != 2 {
		t.Errorf("expected 2 tools with nil endpoints, got %d", len(result))
	}

	// Empty endpoints = no filtering
	result = FilterToolsByEndpoints(tools, adt.DiscoveredEndpoints{})
	if len(result) != 2 {
		t.Errorf("expected 2 tools with empty endpoints, got %d", len(result))
	}
}

func TestFilterToolsByEndpoints_NoEndpointsDeclared(t *testing.T) {
	tools := []*types.ToolDef{
		{Tool: mcp.NewTool("GetConnectionInfo")},                                                // no Endpoints
		{Tool: mcp.NewTool("GetProgram"), Endpoints: []string{"/sap/bc/adt/programs/programs"}}, // has Endpoints
	}

	// Only programs/programs is discovered — GetConnectionInfo (no endpoints) always passes
	eps := adt.DiscoveredEndpoints{
		"/sap/bc/adt/programs/programs": adt.ADTEndpoint{Path: "/sap/bc/adt/programs/programs"},
	}

	result := FilterToolsByEndpoints(tools, eps)
	if len(result) != 2 {
		t.Errorf("expected 2 tools (empty Endpoints always passes), got %d", len(result))
	}
}

func TestFilterToolsByEndpoints_FiltersUnavailable(t *testing.T) {
	tools := []*types.ToolDef{
		{Tool: mcp.NewTool("GetProgram"), Endpoints: []string{"/sap/bc/adt/programs/programs"}},
		{Tool: mcp.NewTool("GetClass"), Endpoints: []string{"/sap/bc/adt/oo/classes"}},
		{Tool: mcp.NewTool("ListDumps"), Endpoints: []string{"/sap/bc/adt/runtime/dumps"}},
		{Tool: mcp.NewTool("GetSystemInfo")}, // no Endpoints
	}

	// Only programs and classes are available
	eps := adt.DiscoveredEndpoints{
		"/sap/bc/adt/programs/programs": adt.ADTEndpoint{Path: "/sap/bc/adt/programs/programs"},
		"/sap/bc/adt/oo/classes":        adt.ADTEndpoint{Path: "/sap/bc/adt/oo/classes"},
	}

	result := FilterToolsByEndpoints(tools, eps)
	if len(result) != 3 {
		t.Errorf("expected 3 tools (GetProgram, GetClass, GetSystemInfo), got %d", len(result))
	}

	// Verify ListDumps was filtered
	for _, td := range result {
		if td.Tool.Name == "ListDumps" {
			t.Error("ListDumps should have been filtered out")
		}
	}
}

func TestFilterToolsByEndpoints_MultipleEndpoints_AllRequired(t *testing.T) {
	tools := []*types.ToolDef{
		{Tool: mcp.NewTool("CompareCallGraphs"), Endpoints: []string{"/sap/bc/adt/cai/callgraph", "/sap/bc/adt/runtime/traces"}},
	}

	// Only callgraph available, traces not
	eps := adt.DiscoveredEndpoints{
		"/sap/bc/adt/cai/callgraph": adt.ADTEndpoint{Path: "/sap/bc/adt/cai/callgraph"},
	}

	result := FilterToolsByEndpoints(tools, eps)
	if len(result) != 0 {
		t.Errorf("expected 0 tools (both endpoints needed), got %d", len(result))
	}

	// Both available
	eps["/sap/bc/adt/runtime/traces"] = adt.ADTEndpoint{Path: "/sap/bc/adt/runtime/traces"}
	result = FilterToolsByEndpoints(tools, eps)
	if len(result) != 1 {
		t.Errorf("expected 1 tool (both endpoints available), got %d", len(result))
	}
}

func TestFilterToolsByEndpoints_PrefixMatching(t *testing.T) {
	tools := []*types.ToolDef{
		{Tool: mcp.NewTool("GetSQLTraceState"), Endpoints: []string{"/sap/bc/adt/st05/trace"}},
	}

	// Discovery returns /sap/bc/adt/st05/trace/state which is a sub-path of the tool's endpoint
	// But the tool declares /sap/bc/adt/st05/trace and discovery has /sap/bc/adt/st05/trace/state
	// The tool endpoint should match because it's a prefix of the discovered endpoint... actually no.
	// HasEndpoint checks if a discovered endpoint is a prefix of the requested path.
	// So if discovered has /sap/bc/adt/st05/trace and tool needs /sap/bc/adt/st05/trace, it matches.

	eps := adt.DiscoveredEndpoints{
		"/sap/bc/adt/st05/trace": adt.ADTEndpoint{Path: "/sap/bc/adt/st05/trace"},
	}

	result := FilterToolsByEndpoints(tools, eps)
	if len(result) != 1 {
		t.Errorf("expected 1 tool (exact match), got %d", len(result))
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
