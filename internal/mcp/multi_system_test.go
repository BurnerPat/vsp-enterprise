package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

func TestMultiSystemRouterRouting(t *testing.T) {
	router := newMultiSystemRouter([]string{"dev", "prod"})

	// Create mock handlers that return identifiable results
	devHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("dev-response"), nil
	}
	prodHandler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("prod-response"), nil
	}

	router.registerHandler("GetProgram", "dev", devHandler)
	router.registerHandler("GetProgram", "prod", prodHandler)

	routeHandler := router.routeHandler("GetProgram")

	// Test routing to "dev"
	req := newRequest(map[string]any{"system_id": "dev", "program_name": "ZTEST"})
	result, err := routeHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text := extractText(result)
	if text != "dev-response" {
		t.Errorf("expected dev-response, got %s", text)
	}

	// Test routing to "prod"
	req = newRequest(map[string]any{"system_id": "prod", "program_name": "ZTEST"})
	result, err = routeHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	text = extractText(result)
	if text != "prod-response" {
		t.Errorf("expected prod-response, got %s", text)
	}
}

func TestMultiSystemRouterCaseInsensitive(t *testing.T) {
	router := newMultiSystemRouter([]string{"A4H", "PROD"})

	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("a4h-response"), nil
	}
	router.registerHandler("GetProgram", "A4H", handler)

	routeHandler := router.routeHandler("GetProgram")

	// Test case-insensitive routing
	for _, sysID := range []string{"a4h", "A4H", "A4h", "a4H"} {
		req := newRequest(map[string]any{"system_id": sysID, "program_name": "ZTEST"})
		result, err := routeHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error for system_id=%q: %v", sysID, err)
		}
		text := extractText(result)
		if text != "a4h-response" {
			t.Errorf("system_id=%q: expected a4h-response, got %s", sysID, text)
		}
	}
}

func TestMultiSystemRouterMissingSystemID(t *testing.T) {
	router := newMultiSystemRouter([]string{"dev", "prod"})
	router.registerHandler("GetProgram", "dev", nil)

	routeHandler := router.routeHandler("GetProgram")

	// Test missing system_id
	req := newRequest(map[string]any{"program_name": "ZTEST"})
	result, err := routeHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for missing system_id")
	}
	text := extractText(result)
	if !strings.Contains(text, "system_id is required") {
		t.Errorf("expected 'system_id is required' in error, got: %s", text)
	}
}

func TestMultiSystemRouterUnknownSystem(t *testing.T) {
	router := newMultiSystemRouter([]string{"dev", "prod"})
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	}
	router.registerHandler("GetProgram", "dev", handler)

	routeHandler := router.routeHandler("GetProgram")

	// Test unknown system_id
	req := newRequest(map[string]any{"system_id": "staging", "program_name": "ZTEST"})
	result, err := routeHandler(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for unknown system_id")
	}
	text := extractText(result)
	if !strings.Contains(text, "Unknown system_id") {
		t.Errorf("expected 'Unknown system_id' in error, got: %s", text)
	}
}

func TestAddSystemIDToTool(t *testing.T) {
	tool := mcp.NewTool("GetProgram",
		mcp.WithDescription("Get ABAP program"),
		mcp.WithString("program_name",
			mcp.Required(),
			mcp.Description("Name of the program"),
		),
	)

	modified := addSystemIDToTool(tool, []string{"dev", "prod"})

	// Check system_id was added to properties
	if _, ok := modified.InputSchema.Properties["system_id"]; !ok {
		t.Fatal("system_id property not added to tool schema")
	}

	// Check system_id is required
	found := false
	for _, req := range modified.InputSchema.Required {
		if req == "system_id" {
			found = true
			break
		}
	}
	if !found {
		t.Error("system_id not added to required list")
	}

	// Check original properties still exist
	if _, ok := modified.InputSchema.Properties["program_name"]; !ok {
		t.Error("original program_name property missing")
	}

	// Check enum values match system IDs
	sysIDProp := modified.InputSchema.Properties["system_id"].(map[string]interface{})
	enum := sysIDProp["enum"].([]string)
	if len(enum) != 2 || enum[0] != "dev" || enum[1] != "prod" {
		t.Errorf("unexpected enum values: %v", enum)
	}
}

func TestAddToolSingleSystem(t *testing.T) {
	cfg := &Config{
		BaseURL:  "https://sap.example.com:44300",
		Username: "testuser",
		Password: "testpass",
		Client:   "001",
		Language: "EN",
	}

	s := NewServer(cfg)

	// In single-system mode, tools should NOT have system_id
	rawResponse := s.mcpServer.HandleMessage(context.Background(), []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/list",
		"params": {}
	}`))

	response, ok := rawResponse.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("expected JSONRPCResponse, got %T", rawResponse)
	}

	var tools []mcp.Tool
	switch result := response.Result.(type) {
	case mcp.ListToolsResult:
		tools = result.Tools
	case *mcp.ListToolsResult:
		tools = result.Tools
	default:
		t.Fatalf("expected ListToolsResult, got %T", response.Result)
	}

	for _, tool := range tools {
		if _, hasSysID := tool.InputSchema.Properties["system_id"]; hasSysID {
			t.Errorf("tool %s should NOT have system_id in single-system mode", tool.Name)
		}
	}
}

func TestAddToolBuilderMode(t *testing.T) {
	// In builder mode, handlers should be stored in handlerMap
	s := &Server{
		handlerMap: make(map[string]mcpserver.ToolHandlerFunc),
		toolMap:    make(map[string]mcp.Tool),
	}

	tool := mcp.NewTool("TestTool", mcp.WithDescription("test"))
	handler := func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("ok"), nil
	}

	s.addTool(tool, handler)

	if _, ok := s.handlerMap["TestTool"]; !ok {
		t.Error("handler not stored in handlerMap")
	}
	if _, ok := s.toolMap["TestTool"]; !ok {
		t.Error("tool not stored in toolMap")
	}
}

func TestNewMultiSystemServer(t *testing.T) {
	cfg := &Config{
		Mode:        "focused",
		Language:    "EN",
		Client:      "001",
		MultiSystem: true,
		MultiSystems: map[string]*SystemConfigResolved{
			"dev": {
				BaseURL:  "https://dev.example.com:44300",
				Username: "devuser",
				Password: "devpass",
				Client:   "001",
				Language: "EN",
			},
			"prod": {
				BaseURL:  "https://prod.example.com:44300",
				Username: "produser",
				Password: "prodpass",
				Client:   "100",
				Language: "EN",
				ReadOnly: true,
			},
		},
	}

	srv, err := NewMultiSystemServer(cfg)
	if err != nil {
		t.Fatalf("NewMultiSystemServer failed: %v", err)
	}
	if srv == nil {
		t.Fatal("NewMultiSystemServer returned nil")
	}
	if !srv.multiSystem {
		t.Error("multiSystem should be true")
	}
	if srv.router == nil {
		t.Error("router should not be nil")
	}
	if len(srv.router.systems) != 2 {
		t.Errorf("expected 2 systems, got %d", len(srv.router.systems))
	}

	// Verify tools have system_id parameter
	rawResponse := srv.mcpServer.HandleMessage(context.Background(), []byte(`{
		"jsonrpc": "2.0",
		"id": 1,
		"method": "tools/list",
		"params": {}
	}`))

	response, ok := rawResponse.(mcp.JSONRPCResponse)
	if !ok {
		t.Fatalf("expected JSONRPCResponse, got %T", rawResponse)
	}

	var tools []mcp.Tool
	switch result := response.Result.(type) {
	case mcp.ListToolsResult:
		tools = result.Tools
	case *mcp.ListToolsResult:
		tools = result.Tools
	default:
		t.Fatalf("expected ListToolsResult, got %T", response.Result)
	}

	if len(tools) == 0 {
		t.Fatal("no tools registered")
	}

	for _, tool := range tools {
		if _, hasSysID := tool.InputSchema.Properties["system_id"]; !hasSysID {
			t.Errorf("tool %s should have system_id in multi-system mode", tool.Name)
		}
	}

	// Verify shutdown cleans up
	srv.Shutdown()
}

// extractText gets the text content from a CallToolResult.
func extractText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}
