// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_servicebinding.go contains handlers for RAP service binding publish/unpublish.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
)

// ServiceBindingToolDefs returns tool definitions for service binding tools.
func ServiceBindingToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("PublishServiceBinding",
			mcp.WithDescription("Publish a RAP service binding"),
			mcp.WithString("service_name", mcp.Required(), mcp.Description("Service binding name")),
			mcp.WithString("service_version", mcp.Description("Service version (default: 0001)")),
		), Handler: HandlePublishServiceBinding},

		{Tool: mcp.NewTool("UnpublishServiceBinding",
			mcp.WithDescription("Unpublish a RAP service binding"),
			mcp.WithString("service_name", mcp.Required(), mcp.Description("Service binding name")),
			mcp.WithString("service_version", mcp.Description("Service version (default: 0001)")),
		), Handler: HandleUnpublishServiceBinding},
	}
}

// --- Service Binding Publish/Unpublish Handlers ---

func HandlePublishServiceBinding(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName, ok := request.GetArguments()["service_name"].(string)
	if !ok || serviceName == "" {
		return types.ErrorResult("service_name is required"), nil
	}

	serviceVersion := "0001"
	if sv, ok := request.GetArguments()["service_version"].(string); ok && sv != "" {
		serviceVersion = sv
	}

	result, err := sys.ADT().PublishServiceBinding(ctx, serviceName, serviceVersion)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to publish service binding: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleUnpublishServiceBinding(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	serviceName, ok := request.GetArguments()["service_name"].(string)
	if !ok || serviceName == "" {
		return types.ErrorResult("service_name is required"), nil
	}

	serviceVersion := "0001"
	if sv, ok := request.GetArguments()["service_version"].(string); ok && sv != "" {
		serviceVersion = sv
	}

	result, err := sys.ADT().UnpublishServiceBinding(ctx, serviceName, serviceVersion)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to unpublish service binding: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
