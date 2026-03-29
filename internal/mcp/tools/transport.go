package tools

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// TransportToolDefs returns tool definitions for transport-related tools.
func TransportToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("GetUserTransports",
			mcp.WithDescription("List open transport requests for the current user"),
		), Handler: HandleGetUserTransports, ReadOnly: true, Focused: true,
			Routes: []types.UniversalRoute{{Action: "query", ParamsType: "user_transports"}}},

		{Tool: mcp.NewTool("GetTransportInfo",
			mcp.WithDescription("Get detailed information about a transport request"),
			mcp.WithString("transport", mcp.Required(), mcp.Description("Transport request number")),
		), Handler: HandleGetTransportInfo, ReadOnly: true, Focused: true},

		{Tool: mcp.NewTool("ListTransports",
			mcp.WithDescription("Search for transport requests"),
			mcp.WithString("user", mcp.Description("Filter by user (default: current)")),
			mcp.WithString("status", mcp.Description("Filter by status (D=Modifiable, R=Released)")),
			mcp.WithNumber("limit", mcp.Description("Maximum number of results (default 50)")),
		), Handler: HandleListTransports, ReadOnly: true},

		{Tool: mcp.NewTool("CreateTransport",
			mcp.WithDescription("Create a new transport request"),
			mcp.WithString("description", mcp.Required(), mcp.Description("Transport description")),
			mcp.WithString("target", mcp.Description("Target system (optional)")),
		), Handler: HandleCreateTransport},

		{Tool: mcp.NewTool("ReleaseTransport",
			mcp.WithDescription("Release a transport request"),
			mcp.WithString("transport", mcp.Required(), mcp.Description("Transport request number")),
		), Handler: HandleReleaseTransport},

		{Tool: mcp.NewTool("DeleteTransport",
			mcp.WithDescription("Delete an empty transport request"),
			mcp.WithString("transport", mcp.Required(), mcp.Description("Transport request number")),
		), Handler: HandleDeleteTransport},
	}
}

// --- Transport Handlers ---

func HandleGetUserTransports(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Placeholder - the API seems to have changed or is different in this environment
	return types.ErrorResult("GetUserTransports is not yet fully implemented in the new architecture"), nil
}

func HandleGetTransportInfo(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	trID, _ := request.GetArguments()["transport"].(string)
	if trID == "" {
		return types.ErrorResult("transport is required"), nil
	}

	// Placeholder - mismatch with adt.Client
	return types.ErrorResult("GetTransportInfo is not yet fully implemented in the new architecture"), nil
}

func HandleListTransports(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("ListTransports (placeholder)"), nil
}

func HandleCreateTransport(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("CreateTransport (placeholder)"), nil
}

func HandleReleaseTransport(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("ReleaseTransport (placeholder)"), nil
}

func HandleDeleteTransport(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText("DeleteTransport (placeholder)"), nil
}
