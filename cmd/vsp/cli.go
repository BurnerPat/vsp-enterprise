package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/pkg/adt"
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

// resolveSystemParams resolves system parameters from --system flag or env vars.
func resolveSystemParams(cmd *cobra.Command) (*config.SystemResolvedConfig, error) {
	// Debug: show which system is being used
	verbose, _ := cmd.Flags().GetBool("verbose")
	if verbose || os.Getenv("VSP_DEBUG") == "true" {
		fmt.Fprintf(os.Stderr, "[DEBUG] resolveSystemParams: systemName=%q\n", systemName)
	}

	// If --system is specified, load from systems config
	if systemName != "" {
		sysCfg, path, err := config.LoadSystems()
		if err != nil {
			return nil, fmt.Errorf("failed to load systems config: %w", err)
		}
		if sysCfg == nil {
			return nil, fmt.Errorf("no systems config found. Create .vsp.json or ~/.vsp.json\n\nExample:\n%s", config.ExampleConfig())
		}

		sys, err := sysCfg.GetSystem(systemName)
		if err != nil {
			return nil, err
		}

		// Require either password or cookie auth (RFC mode supports SSO without password)
		hasCookieAuth := sys.CookieFile != "" || sys.CookieString != ""
		isRFC := strings.EqualFold(sys.ConnectionMode, "rfc")
		if sys.Password == "" && !hasCookieAuth && !isRFC {
			return nil, fmt.Errorf("auth not found for system '%s'. Set VSP_%s_PASSWORD env var or use cookie_file/cookie_string", systemName, strings.ToUpper(systemName))
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		if verbose || os.Getenv("VSP_VERBOSE") == "true" || os.Getenv("VSP_DEBUG") == "true" {
			fmt.Fprintf(os.Stderr, "[INFO] Using system '%s' from %s\n", systemName, path)
			fmt.Fprintf(os.Stderr, "[DEBUG] URL: %s, User: %s\n", sys.URL, sys.User)
		}

		return sys.ToSystemResolved(), nil
	}

	// Fall back to environment variables
	url := os.Getenv("SAP_URL")
	connMode := os.Getenv("SAP_CONNECTION_MODE")
	if url == "" && !strings.EqualFold(connMode, "rfc") {
		return nil, fmt.Errorf("SAP_URL not set. Use --system flag or set SAP_* env vars")
	}

	user := os.Getenv("SAP_USER")
	password := os.Getenv("SAP_PASSWORD")
	// RFC mode supports SSO — password is optional
	if !strings.EqualFold(connMode, "rfc") && (user == "" || password == "") {
		return nil, fmt.Errorf("SAP_USER and SAP_PASSWORD required")
	}

	return &config.SystemResolvedConfig{
		ConnectionConfig: config.ConnectionConfig{
			URL:      url,
			User:     user,
			Password: password,
			Client:   getEnvOrDefault("SAP_CLIENT", "001"),
			Language: getEnvOrDefault("SAP_LANGUAGE", "EN"),
			Insecure: os.Getenv("SAP_INSECURE") == "true",
		},
	}, nil
}

// getClient creates an ADT client from resolved config.
func getClient(params *config.SystemResolvedConfig) (*adt.Client, error) {
	opts := []adt.Option{
		adt.WithClient(params.Client),
		adt.WithLanguage(params.Language),
	}
	if params.Insecure {
		opts = append(opts, adt.WithInsecureSkipVerify())
	}

	// Use cookie auth if available
	if params.CookieFile != "" {
		cookies, err := adt.LoadCookiesFromFile(params.CookieFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load cookies from %s: %w", params.CookieFile, err)
		}
		opts = append(opts, adt.WithCookies(cookies))
		return adt.NewClient(params.URL, "", "", opts...), nil
	}
	if params.CookieString != "" {
		cookies := adt.ParseCookieString(params.CookieString)
		opts = append(opts, adt.WithCookies(cookies))
		return adt.NewClient(params.URL, "", "", opts...), nil
	}

	return adt.NewClient(params.URL, params.User, params.Password, opts...), nil
}

// getWSClient creates an AMDP WebSocket client for GitExport.
func getWSClient(ctx context.Context, params *config.SystemResolvedConfig) (*adt.AMDPWebSocketClient, error) {
	wsClient := adt.NewAMDPWebSocketClient(
		params.URL,
		params.Client,
		params.User,
		params.Password,
		params.Insecure,
	)

	if err := wsClient.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	return wsClient, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
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

func runSystems(cmd *cobra.Command, args []string) error {
	cfg, path, err := config.LoadSystems()
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
		if name == cfg.Default {
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
