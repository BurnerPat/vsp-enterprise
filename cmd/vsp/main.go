// vsp is an MCP server providing ABAP Development Tools (ADT) functionality.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/internal/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Version information (set by build flags)
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

var cfg = &config.ResolvedConfig{}

// Multi-system mode
var (
	multiSystem bool
	configFile  string
)

var rootCmd = &cobra.Command{
	Use:   "vsp",
	Short: "ABAP Development Tools for AI agents and DevOps",
	Long: `vsp — ABAP Development Tools for AI agents and DevOps.

Single binary, 9 platforms, no dependencies. Download from GitHub releases,
point your MCP config at it, done.

Two modes of operation:

  MCP Server (default)  Connects Claude, Gemini CLI, Copilot, Codex, Qwen Code,
                        and other MCP-compatible agents to SAP systems.
                        81 tools (focused), 122 (expert), or 1 universal tool (hyperfocused).

  CLI Mode              Direct terminal access: search, source, export, debug.
                        Multi-system profiles. Useful for scripts and pipelines.

Quick start:
  # 1. MCP server (reads .env or SAP_* env vars)
  vsp --url https://host:44300 --user dev --password secret

  # 2. CLI mode with saved system profile
  vsp -s dev search "ZCL_ORDER*"
  vsp -s dev source CLAS ZCL_ORDER_PROCESSING
  vsp -s dev export '$ZPACKAGE' -o backup.zip

  # 3. Enterprise safety (hand to AI without fear)
  vsp --read-only                                    # no writes at all
  vsp --allowed-packages 'Z*,$TMP' --block-free-sql  # sandbox AI to custom code
  vsp --disallowed-ops CDUA                           # block create/delete/update/activate

Configuration files:
  .env          Default SAP connection (MCP server mode). SAP_URL, SAP_USER, etc.
  .vsp.json     Multi-system profiles for CLI mode (vsp -s dev, vsp -s prod).
  .mcp.json     MCP server entries for Claude Desktop / other MCP clients.

  vsp config init       Generate example files (.env.example, .vsp.json.example, .mcp.json.example)
  vsp config show       Display effective configuration
  vsp config mcp-to-vsp Import systems from .mcp.json into .vsp.json
  vsp config vsp-to-mcp Export .vsp.json systems to .mcp.json format
  vsp config tools      Manage per-tool visibility in .vsp.json

Configuration priority: CLI flags > env vars > .env file > defaults
Ready-to-use configs for 8 AI agents: docs/cli-agents/`,
	Version: fmt.Sprintf("%s (commit: %s, built: %s)", Version, Commit, BuildDate),
	RunE:    runServer,
}

func init() {
	// Load .env file if it exists
	godotenv.Load()

	// Service URL
	rootCmd.Flags().StringVar(&cfg.URL, "url", "", "SAP system URL (e.g., https://host:44300)")
	rootCmd.Flags().StringVar(&cfg.URL, "service", "", "SAP system URL (alias for --url)")

	// Authentication flags
	rootCmd.Flags().StringVarP(&cfg.User, "user", "u", "", "SAP username")
	rootCmd.Flags().StringVarP(&cfg.Password, "password", "p", "", "SAP password")
	rootCmd.Flags().StringVar(&cfg.Password, "pass", "", "SAP password (alias for --password)")

	// SAP connection options
	rootCmd.Flags().StringVar(&cfg.Client, "client", "001", "SAP client number")
	rootCmd.Flags().StringVar(&cfg.Language, "language", "EN", "SAP language")
	rootCmd.Flags().BoolVar(&cfg.Insecure, "insecure", false, "Skip TLS certificate verification")

	// Cookie authentication
	rootCmd.Flags().String("cookie-file", "", "Path to cookie file in Netscape format")
	rootCmd.Flags().String("cookie-string", "", "Cookie string (key1=val1; key2=val2)")

	// Browser-based SSO authentication
	rootCmd.Flags().Bool("browser-auth", false, "Open browser for SSO login (Kerberos, SAML, Keycloak)")
	rootCmd.Flags().Duration("browser-auth-timeout", 120*time.Second, "Timeout for browser-based SSO login")
	rootCmd.Flags().String("browser-exec", "", "Path to Chromium-based browser (default: auto-detect Edge, Chrome, Chromium)")
	rootCmd.Flags().String("cookie-save", "", "Save browser auth cookies to file for reuse with --cookie-file")

	// Session keep-alive
	rootCmd.Flags().Duration("keepalive", 5*time.Minute, "Session keep-alive interval (e.g., 60s, 5m). Prevents session timeout during idle periods. 0 = disabled")

	// Safety options
	rootCmd.Flags().BoolVar(&cfg.ReadOnly, "read-only", false, "Block all write operations (create, update, delete, activate)")
	rootCmd.Flags().BoolVar(&cfg.BlockFreeSQL, "block-free-sql", false, "Block execution of arbitrary SQL queries via RunQuery")
	rootCmd.Flags().StringVar(&cfg.AllowedOps, "allowed-ops", "", "Whitelist of allowed operation types (e.g., \"RSQ\" for Read, Search, Query only)")
	rootCmd.Flags().StringVar(&cfg.DisallowedOps, "disallowed-ops", "", "Blacklist of operation types to block (e.g., \"CDUA\" for Create, Delete, Update, Activate)")
	rootCmd.Flags().StringSliceVar(&cfg.AllowedPackages, "allowed-packages", nil, "Restrict operations to specific packages (comma-separated, supports wildcards like Z*)")
	rootCmd.Flags().BoolVar(&cfg.EnableTransports, "enable-transports", false, "Enable transport management operations (disabled by default for safety)")
	rootCmd.Flags().BoolVar(&cfg.TransportReadOnly, "transport-read-only", false, "Only allow read operations on transports (list, get)")
	rootCmd.Flags().StringSliceVar(&cfg.AllowedTransports, "allowed-transports", nil, "Restrict transport operations to specific transports (comma-separated, supports wildcards like A4HK*)")
	rootCmd.Flags().BoolVar(&cfg.AllowTransportableEdits, "allow-transportable-edits", false, "Allow editing objects in transportable packages (requires transport parameter)")

	// Mode options
	rootCmd.Flags().StringVar(&cfg.Mode, "mode", "focused", "Tool mode: focused (81 tools), expert (122 tools), or hyperfocused (single universal SAP tool)")
	rootCmd.Flags().StringVar(&cfg.DisabledGroups, "disabled-groups", "", "Disable tool groups: 5/U=UI5, T=Tests, H=HANA, D=Debug (e.g., \"TH\" disables Tests and HANA)")

	// Multi-system mode
	rootCmd.Flags().BoolVar(&multiSystem, "multi-system", false, "Enable multi-system mode: route tool requests to systems from .vsp.json config")
	rootCmd.Flags().StringVar(&configFile, "config", "", "Path to .vsp.json configuration file (auto-discovered if not set)")

	// Feature configuration (safety network)
	// Values: "auto" (default), "on", "off"
	rootCmd.Flags().StringVar(&cfg.FeatureHANA, "feature-hana", "auto", "HANA database detection: auto, on, off")
	rootCmd.Flags().StringVar(&cfg.FeatureAbapGit, "feature-abapgit", "auto", "abapGit integration: auto, on, off")
	rootCmd.Flags().StringVar(&cfg.FeatureRAP, "feature-rap", "auto", "RAP/OData development: auto, on, off")
	rootCmd.Flags().StringVar(&cfg.FeatureAMDP, "feature-amdp", "auto", "AMDP/HANA debugger: auto, on, off")
	rootCmd.Flags().StringVar(&cfg.FeatureUI5, "feature-ui5", "auto", "UI5/Fiori BSP management: auto, on, off")
	rootCmd.Flags().StringVar(&cfg.FeatureTransport, "feature-transport", "auto", "CTS transport management: auto, on, off")

	// Debugger configuration
	rootCmd.Flags().StringVar(&cfg.TerminalID, "terminal-id", "", "SAP GUI terminal ID for cross-tool breakpoint sharing")

	// RFC connection settings
	rootCmd.Flags().StringVar(&cfg.ConnectionMode, "connection-mode", "http", "Connection mode: http (default) or rfc")
	rootCmd.Flags().StringVar(&cfg.AsHost, "ashost", "", "SAP application server hostname (RFC mode)")
	rootCmd.Flags().StringVar(&cfg.SysNr, "sysnr", "00", "SAP system number (RFC mode)")
	rootCmd.Flags().StringVar(&cfg.MsHost, "mshost", "", "SAP message server host (RFC load balancing)")
	rootCmd.Flags().StringVar(&cfg.MsServ, "msserv", "", "SAP message server service/port (RFC load balancing)")
	rootCmd.Flags().StringVar(&cfg.R3Name, "r3name", "", "SAP system name (RFC load balancing)")
	rootCmd.Flags().StringVar(&cfg.Group, "group", "", "SAP logon group (RFC load balancing)")
	rootCmd.Flags().StringVar(&cfg.JcoProxyJar, "jco-proxy-jar", "", "Path to jco-proxy JAR file")
	rootCmd.Flags().StringVar(&cfg.JcoLibsDir, "jco-libs-dir", "", "Path to JCo libraries directory")
	rootCmd.Flags().StringVar(&cfg.JavaPath, "java-path", "java", "Path to Java binary")
	rootCmd.Flags().IntVar(&cfg.RfcProxyPort, "rfc-proxy-port", 0, "Fixed sidecar port (0=auto)")
	rootCmd.Flags().IntVar(&cfg.RfcMaxConcurrent, "rfc-max-concurrent", 5, "Max concurrent RFC calls")
	rootCmd.Flags().StringVar(&cfg.SidecarTransport, "jco-sidecar-transport", "http", "Sidecar transport: http (default) or stdio")

	// SNC/SSO configuration (via SAP UI Landscape)
	rootCmd.Flags().BoolVar(&cfg.SNC, "snc", false, "Enable SNC single sign-on via JCo (requires --sysid)")
	rootCmd.Flags().StringVar(&cfg.SysID, "sysid", "", "SAP System ID for SNC logon (3-char SID, reads connection from SAP UI Landscape)")
	rootCmd.Flags().StringVar(&cfg.LandscapeFile, "landscape-file", "", "Path to SAP UI Landscape XML (auto-discovered if not set)")

	// Output options
	rootCmd.Flags().BoolVarP(&cfg.Verbose, "verbose", "v", false, "Enable verbose output to stderr")

	// Bind flags to viper for environment variable support
	viper.BindPFlag("url", rootCmd.Flags().Lookup("url"))
	viper.BindPFlag("user", rootCmd.Flags().Lookup("user"))
	viper.BindPFlag("password", rootCmd.Flags().Lookup("password"))
	viper.BindPFlag("client", rootCmd.Flags().Lookup("client"))
	viper.BindPFlag("language", rootCmd.Flags().Lookup("language"))
	viper.BindPFlag("insecure", rootCmd.Flags().Lookup("insecure"))
	viper.BindPFlag("cookie-file", rootCmd.Flags().Lookup("cookie-file"))
	viper.BindPFlag("cookie-string", rootCmd.Flags().Lookup("cookie-string"))
	viper.BindPFlag("browser-auth", rootCmd.Flags().Lookup("browser-auth"))
	viper.BindPFlag("browser-auth-timeout", rootCmd.Flags().Lookup("browser-auth-timeout"))
	viper.BindPFlag("browser-exec", rootCmd.Flags().Lookup("browser-exec"))
	viper.BindPFlag("cookie-save", rootCmd.Flags().Lookup("cookie-save"))
	viper.BindPFlag("keepalive", rootCmd.Flags().Lookup("keepalive"))
	viper.BindPFlag("read-only", rootCmd.Flags().Lookup("read-only"))
	viper.BindPFlag("block-free-sql", rootCmd.Flags().Lookup("block-free-sql"))
	viper.BindPFlag("allowed-ops", rootCmd.Flags().Lookup("allowed-ops"))
	viper.BindPFlag("disallowed-ops", rootCmd.Flags().Lookup("disallowed-ops"))
	viper.BindPFlag("allowed-packages", rootCmd.Flags().Lookup("allowed-packages"))
	viper.BindPFlag("enable-transports", rootCmd.Flags().Lookup("enable-transports"))
	viper.BindPFlag("transport-read-only", rootCmd.Flags().Lookup("transport-read-only"))
	viper.BindPFlag("allowed-transports", rootCmd.Flags().Lookup("allowed-transports"))
	viper.BindPFlag("allow-transportable-edits", rootCmd.Flags().Lookup("allow-transportable-edits"))
	viper.BindPFlag("mode", rootCmd.Flags().Lookup("mode"))
	viper.BindPFlag("disabled-groups", rootCmd.Flags().Lookup("disabled-groups"))
	viper.BindPFlag("verbose", rootCmd.Flags().Lookup("verbose"))

	// Feature configuration
	viper.BindPFlag("feature-hana", rootCmd.Flags().Lookup("feature-hana"))
	viper.BindPFlag("feature-abapgit", rootCmd.Flags().Lookup("feature-abapgit"))
	viper.BindPFlag("feature-rap", rootCmd.Flags().Lookup("feature-rap"))
	viper.BindPFlag("feature-amdp", rootCmd.Flags().Lookup("feature-amdp"))
	viper.BindPFlag("feature-ui5", rootCmd.Flags().Lookup("feature-ui5"))
	viper.BindPFlag("feature-transport", rootCmd.Flags().Lookup("feature-transport"))

	// Debugger configuration
	viper.BindPFlag("terminal-id", rootCmd.Flags().Lookup("terminal-id"))

	// RFC connection settings
	viper.BindPFlag("connection-mode", rootCmd.Flags().Lookup("connection-mode"))
	viper.BindPFlag("ashost", rootCmd.Flags().Lookup("ashost"))
	viper.BindPFlag("sysnr", rootCmd.Flags().Lookup("sysnr"))
	viper.BindPFlag("mshost", rootCmd.Flags().Lookup("mshost"))
	viper.BindPFlag("msserv", rootCmd.Flags().Lookup("msserv"))
	viper.BindPFlag("r3name", rootCmd.Flags().Lookup("r3name"))
	viper.BindPFlag("group", rootCmd.Flags().Lookup("group"))
	viper.BindPFlag("jco-proxy-jar", rootCmd.Flags().Lookup("jco-proxy-jar"))
	viper.BindPFlag("jco-libs-dir", rootCmd.Flags().Lookup("jco-libs-dir"))
	viper.BindPFlag("java-path", rootCmd.Flags().Lookup("java-path"))
	viper.BindPFlag("rfc-proxy-port", rootCmd.Flags().Lookup("rfc-proxy-port"))
	viper.BindPFlag("rfc-max-concurrent", rootCmd.Flags().Lookup("rfc-max-concurrent"))
	viper.BindPFlag("jco-sidecar-transport", rootCmd.Flags().Lookup("jco-sidecar-transport"))

	// SNC/SSO configuration
	viper.BindPFlag("snc", rootCmd.Flags().Lookup("snc"))
	viper.BindPFlag("sysid", rootCmd.Flags().Lookup("sysid"))
	viper.BindPFlag("landscape-file", rootCmd.Flags().Lookup("landscape-file"))

	// Set up environment variable mapping
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
	viper.SetEnvPrefix("SAP")
}

func runServer(cmd *cobra.Command, args []string) error {
	// Resolve configuration with priority: flags > env vars > defaults
	resolveConfig(cmd)

	// In multi-system mode, skip single-system validation and cookie processing
	// (each system validates independently during server creation)
	if !multiSystem {
		// Resolve SNC/SSO configuration from SAP UI Landscape file
		if cfg.SNC {
			if cfg.SysID == "" {
				return fmt.Errorf("--sysid is required when --snc is specified")
			}
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] SNC mode: resolving system %q from SAP UI Landscape\n", cfg.SysID)
				if cfg.LandscapeFile != "" {
					fmt.Fprintf(os.Stderr, "[VERBOSE] Using landscape file: %s\n", cfg.LandscapeFile)
				}
			}
			jcoProps, err := adt.ResolveSNCJcoProperties(cfg.SysID, cfg.LandscapeFile, cfg.Client, cfg.Language)
			if err != nil {
				return fmt.Errorf("SNC configuration failed: %w", err)
			}
			cfg.JcoProperties = jcoProps
			cfg.ConnectionMode = "rfc" // SNC requires RFC mode via JCo sidecar
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] SNC: resolved %d JCo properties for system %q\n", len(jcoProps), cfg.SysID)
				for k, v := range jcoProps {
					if k == "jco.client.snc_partnername" {
						fmt.Fprintf(os.Stderr, "[VERBOSE]   %s = %s\n", k, v)
					} else {
						fmt.Fprintf(os.Stderr, "[VERBOSE]   %s = %s\n", k, v)
					}
				}
			}
		}

		// Validate configuration
		if err := validateConfig(); err != nil {
			return err
		}

		// Browser-based SSO authentication (must run before processCookieAuth)
		if err := processBrowserAuth(cmd); err != nil {
			return err
		}

		// Process cookie authentication
		if err := processCookieAuth(cmd); err != nil {
			return err
		}
	}

	// Set verbose log output for feature probing
	if cfg.Verbose {
		adt.SetLogOutput(os.Stderr)
	}

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Starting vsp server\n")
		fmt.Fprintf(os.Stderr, "[VERBOSE] Mode: %s\n", cfg.Mode)
		if cfg.DisabledGroups != "" {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Disabled groups: %s (5/U=UI5, T=Tests, H=HANA, D=Debug)\n", cfg.DisabledGroups)
		}
		if strings.EqualFold(cfg.ConnectionMode, "rfc") {
			transport := cfg.SidecarTransport
			if transport == "" {
				transport = "http"
			}
			fmt.Fprintf(os.Stderr, "[VERBOSE] Connection: RFC mode (sidecar transport: %s)\n", transport)
			if cfg.SNC {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Auth: SNC/SSO (system ID: %s, %d JCo properties)\n", cfg.SysID, len(cfg.JcoProperties))
			} else if cfg.AsHost != "" {
				fmt.Fprintf(os.Stderr, "[VERBOSE] RFC: Direct connection to %s (sysnr: %s)\n", cfg.AsHost, cfg.SysNr)
			} else if cfg.MsHost != "" {
				fmt.Fprintf(os.Stderr, "[VERBOSE] RFC: Load balanced via %s (r3name: %s, group: %s)\n", cfg.MsHost, cfg.R3Name, cfg.Group)
			}
			fmt.Fprintf(os.Stderr, "[VERBOSE] JCo proxy JAR: %s\n", cfg.JcoProxyJar)
			fmt.Fprintf(os.Stderr, "[VERBOSE] JCo libs dir: %s\n", cfg.JcoLibsDir)
		} else {
			fmt.Fprintf(os.Stderr, "[VERBOSE] SAP URL: %s\n", cfg.URL)
		}
		fmt.Fprintf(os.Stderr, "[VERBOSE] SAP Client: %s\n", cfg.Client)
		fmt.Fprintf(os.Stderr, "[VERBOSE] SAP Language: %s\n", cfg.Language)
		if cfg.User != "" {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Auth: Basic (user: %s)\n", cfg.User)
		} else if cfg.SNC {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Auth: SNC/SSO\n")
		} else if len(cfg.Cookies) > 0 {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Auth: Cookie (%d cookies)\n", len(cfg.Cookies))
		}

		// Safety status
		if cfg.ReadOnly {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Safety: READ-ONLY mode enabled\n")
		}
		if cfg.BlockFreeSQL {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Safety: Free SQL queries BLOCKED\n")
		}
		if cfg.AllowedOps != "" {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Safety: Allowed operations: %s\n", cfg.AllowedOps)
		}
		if cfg.DisallowedOps != "" {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Safety: Disallowed operations: %s\n", cfg.DisallowedOps)
		}
		if len(cfg.AllowedPackages) > 0 {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Safety: Allowed packages: %v\n", cfg.AllowedPackages)
		}
		if cfg.EnableTransports {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Safety: Transport management ENABLED\n")
		}
		if cfg.AllowTransportableEdits {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Safety: Transportable edits ENABLED (can modify non-local objects)\n")
		}
		if !cfg.ReadOnly && !cfg.BlockFreeSQL && cfg.AllowedOps == "" && cfg.DisallowedOps == "" && len(cfg.AllowedPackages) == 0 {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Safety: UNRESTRICTED (no safety checks active)\n")
		}
		if cfg.KeepAliveInterval > 0 {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Session keep-alive: %s\n", cfg.KeepAliveInterval)
		}
	}

	// Load granular tool visibility from .vsp.json if present
	var systemsCfg *config.SystemsConfig
	var systemsConfigPath string
	{
		var err error
		if configFile != "" {
			systemsCfg, err = config.LoadSystemsFromFile(configFile)
			if err != nil {
				return fmt.Errorf("failed to load config from %s: %w", configFile, err)
			}
			systemsConfigPath = configFile
		} else {
			systemsCfg, systemsConfigPath, err = config.LoadSystems()
			if err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] Failed to load systems config: %v\n", err)
			}
		}
	}
	if systemsCfg != nil && systemsCfg.Tools != nil {
		cfg.ToolsConfig = systemsCfg.Tools
		if cfg.Verbose {
			enabled := 0
			disabled := 0
			for _, v := range systemsCfg.Tools {
				if v {
					enabled++
				} else {
					disabled++
				}
			}
			fmt.Fprintf(os.Stderr, "[VERBOSE] Tool config loaded from %s: %d enabled, %d disabled\n", systemsConfigPath, enabled, disabled)
		}
	}

	// Multi-system mode: create a multi-system server with per-system connections
	if multiSystem {
		if systemsCfg == nil {
			return fmt.Errorf("--multi-system requires a .vsp.json configuration file. Use --config to specify path or create one with 'vsp config init'")
		}
		if len(systemsCfg.Systems) == 0 {
			return fmt.Errorf("--multi-system: no systems defined in configuration file %s", systemsConfigPath)
		}

		// Resolve all systems from config
		multiSystems := make(map[string]*config.ResolvedConfig)
		rfcSystemCount := 0
		for sysID, sysDef := range systemsCfg.Systems {
			if sysDef.Disabled {
				if cfg.Verbose {
					fmt.Fprintf(os.Stderr, "[VERBOSE] Multi-system: skipping disabled system %q\n", sysID)
				}
				continue
			}
			sys, err := systemsCfg.GetSystem(sysID)
			if err != nil {
				return fmt.Errorf("--multi-system: failed to resolve system %q: %w", sysID, err)
			}

			resolved := sys.ToResolved()

			// Resolve SNC/SSO for this system if enabled
			if resolved.SNC {
				if resolved.SysID == "" {
					return fmt.Errorf("--multi-system: system %q has snc=true but no sysid configured", sysID)
				}
				jcoProps, err := adt.ResolveSNCJcoProperties(resolved.SysID, resolved.LandscapeFile, resolved.Client, resolved.Language)
				if err != nil {
					return fmt.Errorf("--multi-system: SNC resolution failed for system %q: %w", sysID, err)
				}
				resolved.JcoProperties = jcoProps
				resolved.ConnectionMode = "rfc" // SNC requires RFC mode
				if cfg.Verbose || resolved.Verbose {
					fmt.Fprintf(os.Stderr, "[VERBOSE] Multi-system: resolved SNC for %q (sysid: %s, %d JCo properties)\n",
						sysID, resolved.SysID, len(jcoProps))
				}
			}

			// Count RFC systems (explicit or via SNC)
			if strings.EqualFold(resolved.ConnectionMode, "rfc") {
				rfcSystemCount++
			}

			// Handle cookie auth from system config
			if sys.CookieFile != "" {
				cookies, err := adt.LoadCookiesFromFile(sys.CookieFile)
				if err == nil && len(cookies) > 0 {
					resolved.Cookies = cookies
				}
			}
			if sys.CookieString != "" {
				cookies := adt.ParseCookieString(sys.CookieString)
				if len(cookies) > 0 {
					resolved.Cookies = cookies
				}
			}

			// Handle browser-based SSO authentication for this system
			if sys.BrowserAuth && len(resolved.Cookies) == 0 {
				if resolved.URL == "" {
					return fmt.Errorf("--multi-system: system %q has browser_auth=true but no url configured", sysID)
				}

				browserTimeout := 120 * time.Second
				if sys.BrowserAuthTimeout != "" {
					if d, err := time.ParseDuration(sys.BrowserAuthTimeout); err == nil {
						browserTimeout = d
					}
				}

				if cfg.Verbose || resolved.Verbose {
					fmt.Fprintf(os.Stderr, "[BROWSER-AUTH] Multi-system: starting browser login for system %q (%s)\n", sysID, resolved.URL)
				}

				ctx := context.Background()
				cookies, err := adt.BrowserLogin(ctx, resolved.URL, resolved.Insecure, browserTimeout, sys.BrowserExec, cfg.Verbose || resolved.Verbose)
				if err != nil {
					return fmt.Errorf("--multi-system: browser authentication failed for system %q: %w", sysID, err)
				}
				resolved.Cookies = cookies

				// Save cookies if requested
				if sys.CookieSave != "" {
					if err := adt.SaveCookiesToFile(cookies, resolved.URL, sys.CookieSave); err != nil {
						fmt.Fprintf(os.Stderr, "[BROWSER-AUTH] Warning: failed to save cookies for system %q: %v\n", sysID, err)
					} else {
						fmt.Fprintf(os.Stderr, "[BROWSER-AUTH] Cookies for system %q saved to %s (reuse with cookie_file)\n", sysID, sys.CookieSave)
					}
				}
			}

			multiSystems[sysID] = resolved
		}

		// Enforce stdio transport when multiple systems use RFC mode.
		// Multiple HTTP-based sidecars would compete for ports; stdio is required.
		if rfcSystemCount > 1 {
			explicitHTTP := false
			if strings.EqualFold(cfg.SidecarTransport, "http") {
				explicitHTTP = true
			}
			for sysID, resolved := range multiSystems {
				if strings.EqualFold(resolved.ConnectionMode, "rfc") {
					if strings.EqualFold(resolved.SidecarTransport, "http") {
						explicitHTTP = true
					}
					if explicitHTTP {
						return fmt.Errorf("--multi-system: multiple systems use RFC mode (%d systems) — jco-sidecar-transport must be \"stdio\" (not \"http\"). "+
							"Each RFC system needs its own sidecar process, which is only supported via stdio transport. "+
							"Remove any explicit --jco-sidecar-transport=http or sidecar_transport settings", rfcSystemCount)
					}
					resolved.SidecarTransport = "stdio"
					multiSystems[sysID] = resolved
				}
			}
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Multi-system: %d RFC systems detected, enforcing stdio sidecar transport\n", rfcSystemCount)
			}
		}

		cfg.MultiSystem = true
		cfg.MultiSystems = multiSystems

		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Multi-system mode: %d systems loaded from %s\n", len(multiSystems), systemsConfigPath)
			for id := range multiSystems {
				fmt.Fprintf(os.Stderr, "[VERBOSE]   - %s\n", id)
			}
		}

		srv, err := mcp.NewMultiSystemServer(cfg)
		if err != nil {
			return fmt.Errorf("multi-system server creation failed: %w", err)
		}
		defer srv.Shutdown()
		return srv.ServeStdio()
	}

	// Single-system mode: create and start MCP server
	server := mcp.NewServer(cfg)
	return server.ServeStdio()
}

func resolveConfig(cmd *cobra.Command) {
	// If --system flag is specified, load system config from .vsp.json first.
	// System profile values act as base defaults; CLI flags can still override.
	if systemName != "" {
		sysCfg, configPath, err := config.LoadSystems()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] Failed to load systems config: %v\n", err)
		} else if sysCfg == nil {
			fmt.Fprintf(os.Stderr, "[WARN] --system '%s' specified but no .vsp.json found\n", systemName)
		} else {
			sys, err := sysCfg.GetSystem(systemName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] %v\n", err)
			} else {
				if cfg.Verbose {
					fmt.Fprintf(os.Stderr, "[VERBOSE] Loading system '%s' from %s\n", systemName, configPath)
				}
				// Apply system config as defaults (CLI flags already set on cfg take precedence)
				if cfg.URL == "" {
					cfg.URL = sys.URL
				}
				if cfg.User == "" {
					cfg.User = sys.User
				}
				if cfg.Password == "" {
					cfg.Password = sys.Password
				}
				if !cmd.Flags().Changed("client") && sys.Client != "" {
					cfg.Client = sys.Client
				}
				if !cmd.Flags().Changed("language") && sys.Language != "" {
					cfg.Language = sys.Language
				}
				if !cmd.Flags().Changed("insecure") && sys.Insecure {
					cfg.Insecure = true
				}
				// RFC connection settings from system profile
				if !cmd.Flags().Changed("connection-mode") && sys.ConnectionMode != "" {
					cfg.ConnectionMode = sys.ConnectionMode
				}
				if !cmd.Flags().Changed("ashost") && sys.AsHost != "" {
					cfg.AsHost = sys.AsHost
				}
				if !cmd.Flags().Changed("sysnr") && sys.SysNr != "" {
					cfg.SysNr = sys.SysNr
				}
				if !cmd.Flags().Changed("mshost") && sys.MsHost != "" {
					cfg.MsHost = sys.MsHost
				}
				if !cmd.Flags().Changed("msserv") && sys.MsServ != "" {
					cfg.MsServ = sys.MsServ
				}
				if !cmd.Flags().Changed("r3name") && sys.R3Name != "" {
					cfg.R3Name = sys.R3Name
				}
				if !cmd.Flags().Changed("group") && sys.Group != "" {
					cfg.Group = sys.Group
				}
				if !cmd.Flags().Changed("jco-proxy-jar") && sys.JcoProxyJar != "" {
					cfg.JcoProxyJar = sys.JcoProxyJar
				}
				if sys.JavaPath != "" && !cmd.Flags().Changed("java-path") {
					cfg.JavaPath = sys.JavaPath
				}
				// Cookie auth from system profile
				if sys.CookieFile != "" {
					cookies, err := adt.LoadCookiesFromFile(sys.CookieFile)
					if err == nil && len(cookies) > 0 {
						cfg.Cookies = cookies
					}
				}
				if sys.CookieString != "" {
					cookies := adt.ParseCookieString(sys.CookieString)
					if len(cookies) > 0 {
						cfg.Cookies = cookies
					}
				}
				// Safety settings from system profile
				if sys.ReadOnly {
					cfg.ReadOnly = true
				}
				if len(sys.AllowedPackages) > 0 && len(cfg.AllowedPackages) == 0 {
					cfg.AllowedPackages = sys.AllowedPackages
				}
			}
		}
	}

	// Check if cookie auth is explicitly requested via CLI flags OR env vars
	// If so, we should NOT load user/password from env/.env to avoid conflicts
	// Cookie auth takes precedence over basic auth since it's more explicit
	cookieAuthViaCLI := cmd.Flags().Changed("cookie-file") || cmd.Flags().Changed("cookie-string")
	cookieAuthViaEnv := viper.GetString("COOKIE_FILE") != "" || viper.GetString("COOKIE_STRING") != ""
	browserAuth, _ := cmd.Flags().GetBool("browser-auth")
	hasBrowserAuth := browserAuth || viper.GetBool("BROWSER_AUTH")
	hasCookieAuth := cookieAuthViaCLI || cookieAuthViaEnv || hasBrowserAuth || len(cfg.Cookies) > 0

	// URL: flag > system profile > SAP_URL env
	if cfg.URL == "" {
		cfg.URL = viper.GetString("URL")
	}
	if cfg.URL == "" {
		cfg.URL = viper.GetString("SERVICE_URL")
	}

	// Username: flag > system profile > SAP_USER env (skip if cookie auth is present)
	if cfg.User == "" && !hasCookieAuth {
		cfg.User = viper.GetString("USER")
	}
	if cfg.User == "" && !hasCookieAuth {
		cfg.User = viper.GetString("USERNAME")
	}

	// Password: flag > system profile > SAP_PASSWORD env (skip if cookie auth is present)
	if cfg.Password == "" && !hasCookieAuth {
		cfg.Password = viper.GetString("PASSWORD")
	}
	if cfg.Password == "" && !hasCookieAuth {
		cfg.Password = viper.GetString("PASS")
	}

	// Client: flag > system profile > SAP_CLIENT env > default
	if !cmd.Flags().Changed("client") && systemName == "" {
		if envClient := viper.GetString("CLIENT"); envClient != "" {
			cfg.Client = envClient
		}
	}

	// Language: flag > system profile > SAP_LANGUAGE env > default
	if !cmd.Flags().Changed("language") && systemName == "" {
		if envLang := viper.GetString("LANGUAGE"); envLang != "" {
			cfg.Language = envLang
		}
	}

	// Insecure: flag > system profile > SAP_INSECURE env
	if !cmd.Flags().Changed("insecure") && systemName == "" {
		cfg.Insecure = viper.GetBool("INSECURE")
	}

	// Mode: flag > SAP_MODE env > default (focused)
	if !cmd.Flags().Changed("mode") {
		if envMode := viper.GetString("MODE"); envMode != "" {
			cfg.Mode = envMode
		}
	}

	// DisabledGroups: flag > SAP_DISABLED_GROUPS env
	if !cmd.Flags().Changed("disabled-groups") {
		if envGroups := viper.GetString("DISABLED_GROUPS"); envGroups != "" {
			cfg.DisabledGroups = envGroups
		}
	}

	// Verbose: flag > SAP_VERBOSE env
	if !cmd.Flags().Changed("verbose") {
		cfg.Verbose = viper.GetBool("VERBOSE")
	}

	// Safety options: flag > SAP_* env
	if !cmd.Flags().Changed("read-only") {
		cfg.ReadOnly = viper.GetBool("READ_ONLY")
	}
	if !cmd.Flags().Changed("block-free-sql") {
		cfg.BlockFreeSQL = viper.GetBool("BLOCK_FREE_SQL")
	}
	if !cmd.Flags().Changed("allowed-ops") {
		cfg.AllowedOps = viper.GetString("ALLOWED_OPS")
	}
	if !cmd.Flags().Changed("disallowed-ops") {
		cfg.DisallowedOps = viper.GetString("DISALLOWED_OPS")
	}
	if !cmd.Flags().Changed("allowed-packages") {
		// Use GetString and split manually - GetStringSlice doesn't split comma-separated env vars
		if pkgStr := viper.GetString("ALLOWED_PACKAGES"); pkgStr != "" {
			cfg.AllowedPackages = splitCommaSeparated(pkgStr)
		}
	}
	if !cmd.Flags().Changed("enable-transports") {
		cfg.EnableTransports = viper.GetBool("ENABLE_TRANSPORTS")
	}
	if !cmd.Flags().Changed("transport-read-only") {
		cfg.TransportReadOnly = viper.GetBool("TRANSPORT_READ_ONLY")
	}
	if !cmd.Flags().Changed("allowed-transports") {
		// Use GetString and split manually - GetStringSlice doesn't split comma-separated env vars
		if transportStr := viper.GetString("ALLOWED_TRANSPORTS"); transportStr != "" {
			cfg.AllowedTransports = splitCommaSeparated(transportStr)
		}
	}
	if !cmd.Flags().Changed("allow-transportable-edits") {
		cfg.AllowTransportableEdits = viper.GetBool("ALLOW_TRANSPORTABLE_EDITS")
	}

	// Feature configuration: flag > SAP_FEATURE_* env
	if !cmd.Flags().Changed("feature-hana") {
		if v := viper.GetString("FEATURE_HANA"); v != "" {
			cfg.FeatureHANA = v
		}
	}
	if !cmd.Flags().Changed("feature-abapgit") {
		if v := viper.GetString("FEATURE_ABAPGIT"); v != "" {
			cfg.FeatureAbapGit = v
		}
	}
	if !cmd.Flags().Changed("feature-rap") {
		if v := viper.GetString("FEATURE_RAP"); v != "" {
			cfg.FeatureRAP = v
		}
	}
	if !cmd.Flags().Changed("feature-amdp") {
		if v := viper.GetString("FEATURE_AMDP"); v != "" {
			cfg.FeatureAMDP = v
		}
	}
	if !cmd.Flags().Changed("feature-ui5") {
		if v := viper.GetString("FEATURE_UI5"); v != "" {
			cfg.FeatureUI5 = v
		}
	}
	if !cmd.Flags().Changed("feature-transport") {
		if v := viper.GetString("FEATURE_TRANSPORT"); v != "" {
			cfg.FeatureTransport = v
		}
	}

	// Terminal ID for debugger: flag > SAP_TERMINAL_ID env
	if !cmd.Flags().Changed("terminal-id") {
		if v := viper.GetString("TERMINAL_ID"); v != "" {
			cfg.TerminalID = v
		}
	}

	// Keep-alive interval: flag > SAP_KEEPALIVE env
	if !cmd.Flags().Changed("keepalive") {
		if v := viper.GetString("KEEPALIVE"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				cfg.KeepAliveInterval = d
			}
		}
	} else {
		cfg.KeepAliveInterval, _ = cmd.Flags().GetDuration("keepalive")
	}

	// RFC settings: flag > system profile > SAP_* env
	// When --system is used, system profile already set these; don't let env vars override
	if !cmd.Flags().Changed("connection-mode") && cfg.ConnectionMode == "" {
		if v := viper.GetString("CONNECTION_MODE"); v != "" {
			cfg.ConnectionMode = v
		}
	}
	if !cmd.Flags().Changed("ashost") && cfg.AsHost == "" {
		if v := viper.GetString("ASHOST"); v != "" {
			cfg.AsHost = v
		}
	}
	if !cmd.Flags().Changed("sysnr") && cfg.SysNr == "" {
		if v := viper.GetString("SYSNR"); v != "" {
			cfg.SysNr = v
		}
	}
	if !cmd.Flags().Changed("mshost") && cfg.MsHost == "" {
		if v := viper.GetString("MSHOST"); v != "" {
			cfg.MsHost = v
		}
	}
	if !cmd.Flags().Changed("msserv") && cfg.MsServ == "" {
		if v := viper.GetString("MSSERV"); v != "" {
			cfg.MsServ = v
		}
	}
	if !cmd.Flags().Changed("r3name") && cfg.R3Name == "" {
		if v := viper.GetString("R3NAME"); v != "" {
			cfg.R3Name = v
		}
	}
	if !cmd.Flags().Changed("group") && cfg.Group == "" {
		if v := viper.GetString("GROUP"); v != "" {
			cfg.Group = v
		}
	}
	if !cmd.Flags().Changed("jco-proxy-jar") && cfg.JcoProxyJar == "" {
		if v := viper.GetString("JCO_PROXY_JAR"); v != "" {
			cfg.JcoProxyJar = v
		}
	}
	if !cmd.Flags().Changed("jco-libs-dir") && cfg.JcoLibsDir == "" {
		if v := viper.GetString("JCO_LIBS_DIR"); v != "" {
			cfg.JcoLibsDir = v
		}
	}
	if !cmd.Flags().Changed("java-path") && cfg.JavaPath == "" {
		if v := viper.GetString("JAVA_PATH"); v != "" {
			cfg.JavaPath = v
		}
	}
	if !cmd.Flags().Changed("rfc-proxy-port") && cfg.RfcProxyPort == 0 {
		if v := viper.GetInt("RFC_PROXY_PORT"); v != 0 {
			cfg.RfcProxyPort = v
		}
	}
	if !cmd.Flags().Changed("rfc-max-concurrent") && cfg.RfcMaxConcurrent == 0 {
		if v := viper.GetInt("RFC_MAX_CONCURRENT"); v != 0 {
			cfg.RfcMaxConcurrent = v
		}
	}
	if !cmd.Flags().Changed("jco-sidecar-transport") {
		if v := viper.GetString("JCO_SIDECAR_TRANSPORT"); v != "" {
			cfg.SidecarTransport = v
		}
	}

	// SNC/SSO settings: flag > SAP_* env
	if !cmd.Flags().Changed("snc") {
		cfg.SNC = viper.GetBool("SNC")
	}
	if !cmd.Flags().Changed("sysid") && cfg.SysID == "" {
		if v := viper.GetString("SYSID"); v != "" {
			cfg.SysID = v
		}
	}
	if !cmd.Flags().Changed("landscape-file") && cfg.LandscapeFile == "" {
		if v := viper.GetString("LANDSCAPE_FILE"); v != "" {
			cfg.LandscapeFile = v
		}
	}
}

func validateConfig() error {
	// In RFC mode, URL is not required; RFC connection params are
	if strings.EqualFold(cfg.ConnectionMode, "rfc") {
		if cfg.SNC {
			// SNC mode: connection params come from JcoProperties (resolved from landscape)
			if len(cfg.JcoProperties) == 0 {
				return fmt.Errorf("SNC mode enabled but no JCo properties resolved from landscape")
			}
		} else {
			// Standard RFC mode: need explicit connection params
			hasDirect := cfg.AsHost != ""
			hasLB := cfg.MsHost != ""
			if !hasDirect && !hasLB {
				return fmt.Errorf("RFC mode requires --ashost or --mshost")
			}
			if hasDirect && hasLB {
				return fmt.Errorf("cannot specify both --ashost (direct) and --mshost (load balancing)")
			}
			if hasDirect && cfg.SysNr == "" {
				return fmt.Errorf("--sysnr required for direct RFC connection")
			}
			if hasLB {
				if cfg.MsServ == "" {
					return fmt.Errorf("--msserv required for RFC load balancing")
				}
				if cfg.R3Name == "" {
					return fmt.Errorf("--r3name required for RFC load balancing")
				}
				if cfg.Group == "" {
					return fmt.Errorf("--group required for RFC load balancing")
				}
			}
		}
	} else {
		// HTTP mode requires URL
		if cfg.URL == "" {
			return fmt.Errorf("SAP URL is required. Use --url flag or SAP_URL environment variable")
		}
	}

	// Validate mode
	if cfg.Mode != "focused" && cfg.Mode != "expert" && cfg.Mode != "hyperfocused" {
		return fmt.Errorf("invalid mode: %s (must be 'focused', 'expert', or 'hyperfocused')", cfg.Mode)
	}

	// Check if we have either basic auth or cookies will be processed
	// Cookies are checked later in processCookieAuth
	return nil
}

func processBrowserAuth(cmd *cobra.Command) error {
	browserAuth, _ := cmd.Flags().GetBool("browser-auth")
	if !browserAuth && !viper.GetBool("BROWSER_AUTH") {
		return nil
	}

	if cfg.URL == "" {
		return fmt.Errorf("--browser-auth requires --url to be set")
	}

	// Determine timeout
	timeout, _ := cmd.Flags().GetDuration("browser-auth-timeout")
	if !cmd.Flags().Changed("browser-auth-timeout") {
		if v := viper.GetString("BROWSER_AUTH_TIMEOUT"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				timeout = d
			}
		}
	}

	// Determine browser executable
	browserExec, _ := cmd.Flags().GetString("browser-exec")
	if browserExec == "" {
		browserExec = viper.GetString("BROWSER_EXEC")
	}

	ctx := context.Background()
	cookies, err := adt.BrowserLogin(ctx, cfg.URL, cfg.Insecure, timeout, browserExec, cfg.Verbose)
	if err != nil {
		return fmt.Errorf("browser authentication failed: %w", err)
	}

	cfg.Cookies = cookies

	// Save cookies to file if requested
	cookieSave, _ := cmd.Flags().GetString("cookie-save")
	if cookieSave == "" {
		cookieSave = viper.GetString("COOKIE_SAVE")
	}
	if cookieSave != "" {
		if err := adt.SaveCookiesToFile(cookies, cfg.URL, cookieSave); err != nil {
			fmt.Fprintf(os.Stderr, "[BROWSER-AUTH] Warning: failed to save cookies: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "[BROWSER-AUTH] Cookies saved to %s (reuse with --cookie-file)\n", cookieSave)
		}
	}

	return nil
}

func processCookieAuth(cmd *cobra.Command) error {
	cookieFile, _ := cmd.Flags().GetString("cookie-file")
	cookieString, _ := cmd.Flags().GetString("cookie-string")

	// Check environment variables if flags not provided
	if cookieFile == "" {
		cookieFile = viper.GetString("COOKIE_FILE")
	}
	if cookieString == "" {
		cookieString = viper.GetString("COOKIE_STRING")
	}

	// Count authentication methods
	authMethods := 0
	if cfg.User != "" && cfg.Password != "" {
		authMethods++
	}
	if cookieFile != "" {
		authMethods++
	}
	if cookieString != "" {
		authMethods++
	}
	if cfg.SNC {
		authMethods++ // SNC uses OS-level SSO (Kerberos/SPNEGO), no user/password needed
	}
	// Browser auth already populated cfg.Cookies in processBrowserAuth
	if len(cfg.Cookies) > 0 {
		authMethods++
	}

	// In RFC mode, SSO is valid — no password or cookies needed
	isRFC := strings.EqualFold(cfg.ConnectionMode, "rfc")

	if authMethods > 1 {
		return fmt.Errorf("only one authentication method can be used at a time (basic auth, cookie-file, cookie-string, browser-auth, or SNC)")
	}

	if authMethods == 0 && !isRFC {
		return fmt.Errorf("authentication required. Use --user/--password, --cookie-file, --cookie-string, --browser-auth, or --snc/--sysid")
	}

	// If cookies already set by browser auth, we're done
	if len(cfg.Cookies) > 0 {
		return nil
	}

	// Process cookie file
	if cookieFile != "" {
		if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
			return fmt.Errorf("cookie file not found: %s", cookieFile)
		}

		cookies, err := adt.LoadCookiesFromFile(cookieFile)
		if err != nil {
			return fmt.Errorf("failed to load cookies from file: %w", err)
		}

		if len(cookies) == 0 {
			return fmt.Errorf("no cookies found in file: %s", cookieFile)
		}

		cfg.Cookies = cookies
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Loaded %d cookies from file: %s\n", len(cookies), cookieFile)
		}
	}

	// Process cookie string
	if cookieString != "" {
		cookies := adt.ParseCookieString(cookieString)
		if len(cookies) == 0 {
			return fmt.Errorf("failed to parse cookie string")
		}

		cfg.Cookies = cookies
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Parsed %d cookies from string\n", len(cookies))
		}
	}

	return nil
}

// splitCommaSeparated splits a comma-separated string into a slice, trimming whitespace.
// This is needed because viper.GetStringSlice doesn't properly split comma-separated env vars.
func splitCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
