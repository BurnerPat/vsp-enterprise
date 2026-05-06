package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// TransportToolDefs returns tool definitions for transport request tools.
func TransportToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			Tool: mcp.NewTool("GetTransport",
				mcp.WithDescription("Get detailed information about a single transport request by its number. Returns owner, description, status, target system, tasks with their objects."),
				mcp.WithString("number", mcp.Required(),
					mcp.Description("Transport request or task number (e.g. DEVK900001, D61K907178)")),
			),
			Handler:   HandleGetTransport,
			ReadOnly:  true,
			Focused:   true,
			Endpoints: []string{"/sap/bc/adt/cts/transportrequests"},
		},
		{
			Tool: mcp.NewTool("ListTransports",
				mcp.WithDescription("List transport requests for a user. Returns modifiable workbench and customizing requests with their status, target system, and description."),
				mcp.WithString("user",
					mcp.Description("SAP user name to list transports for. Defaults to the connected user if omitted. Use `*` to list transports for all users (requires appropriate permissions).")),
			),
			Handler:   HandleListTransports,
			ReadOnly:  true,
			Focused:   true,
			Endpoints: []string{"/sap/bc/adt/cts/transportrequests"},
		},
	}
}

// --- Transport Handlers ---

func HandleGetTransport(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	number, _ := request.GetArguments()["number"].(string)
	if number == "" {
		return types.ErrorResult("number is required"), nil
	}

	details, err := sys.ADT().GetTransport(ctx, number)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetTransport failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(details, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleListTransports(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	user, _ := request.GetArguments()["user"].(string)

	transports, err := sys.ADT().ListTransports(ctx, user)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("ListTransports failed: %v", err)), nil
	}

	if len(transports) == 0 {
		return mcp.NewToolResultText("No transport requests found."), nil
	}

	output, _ := json.MarshalIndent(transports, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
