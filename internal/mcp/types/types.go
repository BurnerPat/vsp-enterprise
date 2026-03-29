package types

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// System defines the interface for interacting with a single SAP system.
type System interface {
	// ADT returns the ADT client for the system.
	ADT() *adt.Client

	// Config returns the configuration for this system.
	Config() any

	// IsRfcMode returns true if the system is using RFC/JCo sidecar.
	IsRfcMode() bool

	// Sidecar returns the sidecar manager, if applicable (RFC mode).
	Sidecar() *adt.SidecarManager

	// RequireActiveAMDPSession checks for an active AMDP debug session and returns an error result if missing.
	RequireActiveAMDPSession() *mcp.CallToolResult

	// EnsureWSConnected ensures the WebSocket client for a tool is connected.
	EnsureWSConnected(ctx context.Context, toolName string) *mcp.CallToolResult
}

// ToolHandlerFunc is the signature for MCP tool handlers.
type ToolHandlerFunc func(ctx context.Context, sys System, request mcp.CallToolRequest) (*mcp.CallToolResult, error)

// ParamMapper transforms universal-mode params into handler-specific params.
type ParamMapper func(objectType, objectName string, params map[string]any) map[string]any

// UniversalRoute describes how a tool is accessible via the single SAP(action, target, params) tool.
type UniversalRoute struct {
	Action     string      // universal-mode action: "read", "edit", "create", "delete", "search", "query", "grep", "test", "analyze", "debug", "system"
	TargetType string      // match objectType from target (e.g., "PROG", "INFO"). Empty = don't match on targetType.
	ParamsType string      // match params["type"] (e.g., "list_transports"). Empty = don't match on params.type.
	MapArgs    ParamMapper // optional: transform params before calling handler. nil = pass through.
}

// ToolDef is a self-contained, declarative tool definition.
type ToolDef struct {
	Tool     mcp.Tool
	Handler  ToolHandlerFunc
	AlwaysOn bool     // if true, registered regardless of mode/groups/config
	ReadOnly bool     // tool only reads data, never modifies the SAP system
	Focused  bool     // included in "focused" mode (default mode)
	Groups   []string // group codes for --disabled-groups (e.g., "D", "H", "X")

	// Universal mode routing (optional).
	Routes []UniversalRoute
}

// ErrorResult is a helper to create an MCP error result.
func ErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.NewTextContent(message),
		},
		IsError: true,
	}
}
