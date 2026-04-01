package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oisee/vibing-steampunk/internal/mcp/tools"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List all available tools with descriptions",
	Long: `List all SAP/ADT tools registered by vsp.

	Displays tool name, description, and flags (read-only, focused mode, groups).
	This does not include meta-tools like ListAvailableTools.`,
	RunE: runTools,
}

var toolsFullDetails bool

func init() {
	rootCmd.AddCommand(toolsCmd)
	toolsCmd.Flags().BoolVarP(&toolsFullDetails, "full", "f", false, "Show full tool metadata")
}

func runTools(_ *cobra.Command, _ []string) error {
	allTools := tools.AllToolDefs()

	sort.Slice(allTools, func(i, j int) bool {
		return allTools[i].Tool.Name < allTools[j].Tool.Name
	})

	for _, t := range allTools {
		fmt.Printf("%s\n", t.Tool.Name)

		if toolsFullDetails {
			fmt.Printf("  - Description: %s\n", t.Tool.Description)
			fmt.Printf("  - Flags: Readonly=%t Focused=%t AlwaysOn=%t\n", t.ReadOnly, t.Focused, t.AlwaysOn)
			fmt.Printf("  - Groups: %s\n", strings.Join(t.Groups, ","))
			fmt.Println()
		}
	}

	fmt.Printf("\nTotal: %d tools\n", len(allTools))
	return nil
}
