package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/spf13/cobra"
)

var (
	systemName string
)

func init() {
	// Add persistent --system flag to root command
	rootCmd.PersistentFlags().StringVarP(&systemName, "system", "s", "", "System name from config (e.g., 'a4h')")

	// Add CLI subcommands
	rootCmd.AddCommand(systemsCmd)
}

// --- systems command ---

var systemsCmd = &cobra.Command{
	Use:   "systems",
	Short: "List configured systems",
	Long: `List all configured SAP systems from the systems config file.

Config file locations (searched in order):
  .vsp-systems.json
  .vsp/systems.json
  ~/.vsp-systems.json
  ~/.vsp/systems.json`,
	RunE: runSystems,
}

func init() {
	systemsCmd.AddCommand(systemsInitCmd)
}

func runSystems(_ *cobra.Command, _ []string) error {
	cfg, path, err := config.LoadConfiguration()
	if err != nil {
		return err
	}

	if cfg == nil {
		fmt.Println("No systems config found.")
		fmt.Println("\nCreate .vsp-systems.json with:")
		fmt.Println(config.ExampleConfig())
		return nil
	}

	fmt.Printf("Config: %s\n\n", path)
	fmt.Println("Systems:")
	for name, sys := range cfg.Systems {
		defaultMark := ""
		if name == cfg.DefaultSystem {
			defaultMark = " (default)"
		}

		// Determine auth method
		authStatus := ""
		if sys.CookieFile != "" {
			authStatus = fmt.Sprintf("cookie-file:%s", sys.CookieFile)
		} else if sys.CookieString != "" {
			authStatus = "cookie-string:***"
		} else {
			// Password auth
			if sys.Password != "" {
				authStatus = "pwd:inline"
			} else if os.Getenv(fmt.Sprintf("VSP_%s_PASSWORD", strings.ToUpper(name))) != "" {
				authStatus = "pwd:env ✓"
			} else {
				authStatus = "pwd:env ✗"
			}
		}

		userInfo := sys.User
		if userInfo == "" {
			userInfo = "(cookie)"
		}
		fmt.Printf("  %-12s %s [%s@%s] %s%s\n", name, sys.URL, userInfo, sys.Client, authStatus, defaultMark)
	}

	return nil
}

var systemsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create example systems config",
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := ".vsp-systems.json"
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("%s already exists", configPath)
		}

		if err := os.WriteFile(configPath, []byte(config.ExampleConfig()), 0600); err != nil {
			return err
		}

		fmt.Printf("Created %s\n", configPath)
		fmt.Println("\nEdit the file to add your SAP systems.")
		fmt.Println("Set passwords via environment variables: VSP_<SYSTEM>_PASSWORD")
		return nil
	},
}
