package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// registerDiscoveryTools registers meta-tools that allow Agents to introspect
// available systems and their permitted tools. These tools bypass the normal
// permission pipeline and are always available.
func (r *Router) registerDiscoveryTools() {
	tool := mcp.NewTool("ListAvailableTools",
		mcp.WithDescription(
			"List available SAP systems and their permitted tools. "+
				"Use this to discover what operations are accessible on each system before calling them. "+
				"Call without parameters to list all systems, or pass system_id to filter to one system.",
		),
		mcp.WithString("system_id",
			mcp.Description(fmt.Sprintf("Optional: filter to a specific system. Available: %s", strings.Join(r.systemIDs, ", "))),
		),
	)

	r.mcpServer.AddTool(tool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return r.handleListAvailableTools(ctx, request)
	})
}

type discoverySystemInfo struct {
	SystemID        string              `json:"system_id"`
	EnabledTools    []string            `json:"enabled_tools"`
	RestrictedTools []DiscoveryToolInfo `json:"restricted_tools,omitempty"`
}

type discoveryResponse struct {
	Systems []discoverySystemInfo `json:"systems"`
}

func (r *Router) handleListAvailableTools(_ context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filterID, _ := request.GetArguments()["system_id"].(string)

	var systems []discoverySystemInfo

	if filterID != "" {
		if _, ok := r.systems[strings.ToLower(filterID)]; !ok {
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.NewTextContent(
					fmt.Sprintf("Unknown system: %s. Available: %s", filterID, strings.Join(r.systemIDs, ", ")),
				)},
				IsError: true,
			}, nil
		}
		systems = append(systems, r.buildSystemInfo(filterID))
	} else {
		for _, id := range r.systemIDs {
			systems = append(systems, r.buildSystemInfo(id))
		}
	}

	resp := discoveryResponse{
		Systems: systems,
	}

	encoded, _ := json.MarshalIndent(resp, "", "  ")
	return mcp.NewToolResultText(string(encoded)), nil
}

func (r *Router) buildSystemInfo(systemID string) discoverySystemInfo {
	toolNames := r.permissionManager.GetEnabledToolNames(systemID)
	sort.Strings(toolNames)

	return discoverySystemInfo{
		SystemID:        systemID,
		EnabledTools:    toolNames,
		RestrictedTools: r.permissionManager.GetRestrictedTools(systemID),
	}
}
