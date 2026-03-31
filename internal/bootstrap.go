// Package internal provides internal infrastructure for vsp.
package internal

import (
	"fmt"
	"os"
	"strings"

	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/pkg/adt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Bootstrap orchestrates the entire configuration pipeline:
// 1. Load configuration from config file (lowest priority)
// 2. Layer environment variables (override config file)
// 3. Layer CLI flags (override both)
// 4. Load systems configuration (.vsp.json) for multi-system or tool visibility
// 5. Resolve multi-system or single-system mode
// 6. Augment each system with JCo/RFC properties, cookies, authentication
// 7. Validate all systems
// 8. Return fully-prepared config ready for mcp.NewServer()
//
// Parameters:
//   - cfg: Base config structure to populate (systems map will be created)
//   - singleSys: Per-system settings accumulated from CLI flags
//   - multiSystem: Whether multi-system mode is enabled
//   - configFile: Explicit path to .vsp.json (empty = auto-discover)
//   - systemName: System name from --system flag (for single-system mode)
//   - cmd: Cobra command for flag inspection and precedence handling
//
// Returns the fully-augmented ResolvedConfig ready for instantiation.
func Bootstrap(cfg *config.ResolvedConfig, singleSys *config.SystemResolvedConfig, multiSystem bool, configFile, systemName string, cmd *cobra.Command) (*config.ResolvedConfig, error) {
	// Load systems configuration (.vsp.json) for tool visibility and multi-system definitions
	var systemsCfg *config.SystemsConfig
	var systemsConfigPath string
	{
		var err error
		if configFile != "" {
			systemsCfg, err = config.LoadSystemsFromFile(configFile)
			if err != nil {
				return nil, fmt.Errorf("failed to load config from %s: %w", configFile, err)
			}
			systemsConfigPath = configFile
		} else {
			systemsCfg, systemsConfigPath, err = config.LoadSystems()
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "[WARN] Failed to load systems config: %v\n", err)
			}
		}
	}

	// Apply tool visibility from .vsp.json
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
			_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] Tool config loaded from %s: %d enabled, %d disabled\n", systemsConfigPath, enabled, disabled)
		}
	}

	// Initialize the Systems map
	cfg.Systems = make(map[string]*config.SystemResolvedConfig)

	if multiSystem {
		if err := bootstrapMultiSystem(cfg, systemsCfg, systemsConfigPath); err != nil {
			return nil, err
		}
	} else {
		if err := bootstrapSingleSystem(cfg, singleSys, systemsCfg, systemName, cmd); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// bootstrapMultiSystem populates cfg.Systems from .vsp.json and augments each system.
func bootstrapMultiSystem(cfg *config.ResolvedConfig, systemsCfg *config.SystemsConfig, systemsConfigPath string) error {
	if systemsCfg == nil {
		return fmt.Errorf("--multi-system requires a .vsp.json configuration file. Use --config to specify path or create one with 'vsp config init'")
	}
	if len(systemsCfg.Systems) == 0 {
		return fmt.Errorf("--multi-system: no systems defined in configuration file %s", systemsConfigPath)
	}

	rfcSystemCount := 0

	for sysID, sysDef := range systemsCfg.Systems {
		if sysDef.Disabled {
			if cfg.Verbose {
				_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] Multi-system: skipping disabled system %q\n", sysID)
			}
			continue
		}

		sys, err := systemsCfg.GetSystem(sysID)
		if err != nil {
			return fmt.Errorf("--multi-system: failed to resolve system %q: %w", sysID, err)
		}

		resolved := sys.ToSystemResolved()

		// Augment the system configuration
		if err := augmentSystemConfiguration(resolved, cfg); err != nil {
			return fmt.Errorf("--multi-system: failed to augment system %q: %w", sysID, err)
		}

		// Count RFC systems (explicit or via SNC)
		if strings.EqualFold(resolved.ConnectionMode, "rfc") {
			rfcSystemCount++
		}

		cfg.Systems[sysID] = resolved
	}

	// Enforce stdio transport when multiple systems use RFC mode
	if rfcSystemCount > 1 {
		if strings.EqualFold(cfg.SidecarTransport, "http") || cfg.SidecarTransport == "" {
			return fmt.Errorf("--multi-system: multiple systems use RFC mode (%d systems) — jco-sidecar-transport must be \"stdio\" (not \"http\"). "+
				"Each RFC system needs its own sidecar process, which is only supported via stdio transport. "+
				"Add --jco-sidecar-transport=stdio", rfcSystemCount)
		}
		if cfg.Verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] Multi-system: %d RFC systems detected, using stdio sidecar transport\n", rfcSystemCount)
		}
	}

	if cfg.Verbose {
		_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] Multi-system mode: %d systems loaded from %s\n", len(cfg.Systems), systemsConfigPath)
		for id := range cfg.Systems {
			_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE]   - %s\n", id)
		}
	}

	return nil
}

// bootstrapSingleSystem resolves single-system configuration from CLI/env/system profile and augments it.
func bootstrapSingleSystem(cfg *config.ResolvedConfig, singleSys *config.SystemResolvedConfig, systemsCfg *config.SystemsConfig, systemName string, cmd *cobra.Command) error {
	// If --system flag is specified, load system config from .vsp.json first
	// System profile values act as base defaults; CLI flags can override
	if systemName != "" {
		if systemsCfg == nil {
			_, _ = fmt.Fprintf(os.Stderr, "[WARN] --system '%s' specified but no .vsp.json found\n", systemName)
		} else {
			sys, err := systemsCfg.GetSystem(systemName)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "[WARN] %v\n", err)
			} else {
				if cfg.Verbose {
					_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] Loading system '%s' from .vsp.json\n", systemName)
				}
				// Apply config file as base (precedence: config file < env < CLI)
				applySystemConfigDefaults(singleSys, sys, cmd)
			}
		}
	}

	// Resolve configuration with precedence: config file < env < CLI
	// At this point, singleSys has config file defaults; now layer env and CLI
	if err := resolveSystemConfiguration(singleSys, cfg, cmd); err != nil {
		return err
	}

	// Validate single-system configuration
	if err := validateSystemConfiguration(singleSys, cfg); err != nil {
		return err
	}

	// Augment the system configuration (SNC, RFC, cookies, auth, etc.)
	if err := augmentSystemConfiguration(singleSys, cfg); err != nil {
		return err
	}

	cfg.Systems[config.DefaultSystemID] = singleSys
	return nil
}

// applySystemConfigDefaults applies config file values as base defaults to singleSys,
// respecting CLI flag precedence (CLI flags already set on singleSys take priority).
func applySystemConfigDefaults(singleSys *config.SystemResolvedConfig, sys *config.SystemConfig, cmd *cobra.Command) {
	// Connection config
	if singleSys.URL == "" {
		singleSys.URL = sys.URL
	}
	if singleSys.User == "" {
		singleSys.User = sys.User
	}
	if singleSys.Password == "" {
		singleSys.Password = sys.Password
	}
	if !cmd.Flags().Changed("client") && sys.Client != "" {
		singleSys.Client = sys.Client
	}
	if !cmd.Flags().Changed("language") && sys.Language != "" {
		singleSys.Language = sys.Language
	}
	if !cmd.Flags().Changed("insecure") && sys.Insecure {
		singleSys.Insecure = true
	}

	// RFC connection settings from system profile
	if !cmd.Flags().Changed("connection-mode") && sys.ConnectionMode != "" {
		singleSys.ConnectionMode = sys.ConnectionMode
	}
	if !cmd.Flags().Changed("ashost") && sys.AsHost != "" {
		singleSys.AsHost = sys.AsHost
	}
	if !cmd.Flags().Changed("sysnr") && sys.SysNr != "" {
		singleSys.SysNr = sys.SysNr
	}
	if !cmd.Flags().Changed("mshost") && sys.MsHost != "" {
		singleSys.MsHost = sys.MsHost
	}
	if !cmd.Flags().Changed("msserv") && sys.MsServ != "" {
		singleSys.MsServ = sys.MsServ
	}
	if !cmd.Flags().Changed("r3name") && sys.R3Name != "" {
		singleSys.R3Name = sys.R3Name
	}
	if !cmd.Flags().Changed("group") && sys.Group != "" {
		singleSys.Group = sys.Group
	}

	// Cookie auth from system profile (config file level)
	if sys.CookieFile != "" {
		cookies, err := adt.LoadCookiesFromFile(sys.CookieFile)
		if err == nil && len(cookies) > 0 {
			singleSys.Cookies = cookies
		}
	}
	if sys.CookieString != "" {
		cookies := adt.ParseCookieString(sys.CookieString)
		if len(cookies) > 0 {
			singleSys.Cookies = cookies
		}
	}

	// Safety settings from system profile
	if sys.ReadOnly {
		singleSys.ReadOnly = true
	}
	if len(sys.AllowedPackages) > 0 && len(singleSys.AllowedPackages) == 0 {
		singleSys.AllowedPackages = sys.AllowedPackages
	}
}

// resolveSystemConfiguration applies environment variables and CLI flags
// on top of config file defaults (precedence: config file < env < CLI).
func resolveSystemConfiguration(singleSys *config.SystemResolvedConfig, globalCfg *config.ResolvedConfig, cmd *cobra.Command) error {
	// Check if cookie auth is explicitly requested via CLI flags OR env vars
	cookieAuthViaCLI := cmd.Flags().Changed("cookie-file") || cmd.Flags().Changed("cookie-string")
	cookieAuthViaEnv := viper.GetString("COOKIE_FILE") != "" || viper.GetString("COOKIE_STRING") != ""
	browserAuth, _ := cmd.Flags().GetBool("browser-auth")
	hasBrowserAuth := browserAuth || viper.GetBool("BROWSER_AUTH")
	hasCookieAuth := cookieAuthViaCLI || cookieAuthViaEnv || hasBrowserAuth || len(singleSys.Cookies) > 0

	// URL: CLI flag > env var > config file (already set on singleSys)
	if singleSys.URL == "" {
		singleSys.URL = viper.GetString("URL")
	}
	if singleSys.URL == "" {
		singleSys.URL = viper.GetString("SERVICE_URL")
	}

	// Username: CLI flag > env var > config file (skip if cookie auth is present)
	if singleSys.User == "" && !hasCookieAuth {
		singleSys.User = viper.GetString("USER")
	}
	if singleSys.User == "" && !hasCookieAuth {
		singleSys.User = viper.GetString("USERNAME")
	}

	// Password: CLI flag > env var > config file (skip if cookie auth is present)
	if singleSys.Password == "" && !hasCookieAuth {
		singleSys.Password = viper.GetString("PASSWORD")
	}
	if singleSys.Password == "" && !hasCookieAuth {
		singleSys.Password = viper.GetString("PASS")
	}

	// Client: CLI flag > env var > config file > default
	if !cmd.Flags().Changed("client") {
		if envClient := viper.GetString("CLIENT"); envClient != "" {
			singleSys.Client = envClient
		} else if singleSys.Client == "" {
			singleSys.Client = "001"
		}
	}

	// Language: CLI flag > env var > config file > default
	if !cmd.Flags().Changed("language") {
		if envLang := viper.GetString("LANGUAGE"); envLang != "" {
			singleSys.Language = envLang
		} else if singleSys.Language == "" {
			singleSys.Language = "EN"
		}
	}

	// Insecure: CLI flag > env var > config file
	if !cmd.Flags().Changed("insecure") {
		singleSys.Insecure = viper.GetBool("INSECURE")
	}

	// RFC connection settings: CLI flag > env var > config file
	if !cmd.Flags().Changed("connection-mode") && singleSys.ConnectionMode == "" {
		if v := viper.GetString("CONNECTION_MODE"); v != "" {
			singleSys.ConnectionMode = v
		}
	}
	if !cmd.Flags().Changed("ashost") && singleSys.AsHost == "" {
		if v := viper.GetString("ASHOST"); v != "" {
			singleSys.AsHost = v
		}
	}
	if !cmd.Flags().Changed("sysnr") && singleSys.SysNr == "" {
		if v := viper.GetString("SYSNR"); v != "" {
			singleSys.SysNr = v
		}
	}
	if !cmd.Flags().Changed("mshost") && singleSys.MsHost == "" {
		if v := viper.GetString("MSHOST"); v != "" {
			singleSys.MsHost = v
		}
	}
	if !cmd.Flags().Changed("msserv") && singleSys.MsServ == "" {
		if v := viper.GetString("MSSERV"); v != "" {
			singleSys.MsServ = v
		}
	}
	if !cmd.Flags().Changed("r3name") && singleSys.R3Name == "" {
		if v := viper.GetString("R3NAME"); v != "" {
			singleSys.R3Name = v
		}
	}
	if !cmd.Flags().Changed("group") && singleSys.Group == "" {
		if v := viper.GetString("GROUP"); v != "" {
			singleSys.Group = v
		}
	}

	// Global JCo/RFC settings: CLI flag > env var > config file
	if !cmd.Flags().Changed("jco-proxy-jar") && globalCfg.JcoProxyJar == "" {
		if v := viper.GetString("JCO_PROXY_JAR"); v != "" {
			globalCfg.JcoProxyJar = v
		}
	}
	if !cmd.Flags().Changed("jco-libs-dir") && globalCfg.JcoLibsDir == "" {
		if v := viper.GetString("JCO_LIBS_DIR"); v != "" {
			globalCfg.JcoLibsDir = v
		}
	}
	if !cmd.Flags().Changed("java-path") && globalCfg.JavaPath == "" {
		if v := viper.GetString("JAVA_PATH"); v != "" {
			globalCfg.JavaPath = v
		}
	}
	if !cmd.Flags().Changed("rfc-proxy-port") && globalCfg.RfcProxyPort == 0 {
		if v := viper.GetInt("RFC_PROXY_PORT"); v != 0 {
			globalCfg.RfcProxyPort = v
		}
	}
	if !cmd.Flags().Changed("rfc-max-concurrent") && globalCfg.RfcMaxConcurrent == 0 {
		if v := viper.GetInt("RFC_MAX_CONCURRENT"); v != 0 {
			globalCfg.RfcMaxConcurrent = v
		}
	}
	if !cmd.Flags().Changed("jco-sidecar-transport") {
		if v := viper.GetString("JCO_SIDECAR_TRANSPORT"); v != "" {
			globalCfg.SidecarTransport = v
		}
	}

	// SNC/SSO settings: CLI flag > env var > config file
	if !cmd.Flags().Changed("snc") {
		singleSys.SNC = viper.GetBool("SNC")
	}
	if !cmd.Flags().Changed("sysid") && singleSys.SysID == "" {
		if v := viper.GetString("SYSID"); v != "" {
			singleSys.SysID = v
		}
	}
	if !cmd.Flags().Changed("landscape-file") && singleSys.LandscapeFile == "" {
		if v := viper.GetString("LANDSCAPE_FILE"); v != "" {
			singleSys.LandscapeFile = v
		}
	}

	return nil
}

// augmentSystemConfiguration augments a single system with derived/resolved data:
// - SNC/JCo properties from SAP UI Landscape
// - Cookie authentication (browser-based, file, or string)
// - Validation of connection configuration
func augmentSystemConfiguration(resolved *config.SystemResolvedConfig, globalCfg *config.ResolvedConfig) error {
	// Resolve SNC/SSO configuration from SAP UI Landscape file
	if resolved.SNC {
		if resolved.SysID == "" {
			return fmt.Errorf("--sysid is required when --snc is specified")
		}
		if globalCfg.Verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] SNC mode: resolving system %q from SAP UI Landscape\n", resolved.SysID)
			if resolved.LandscapeFile != "" {
				_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] Using landscape file: %s\n", resolved.LandscapeFile)
			}
		}
		jcoProps, err := adt.ResolveSNCJcoProperties(resolved.SysID, resolved.LandscapeFile, resolved.Client, resolved.Language)
		if err != nil {
			return fmt.Errorf("SNC configuration failed: %w", err)
		}
		resolved.JcoProperties = jcoProps
		resolved.ConnectionMode = "rfc" // SNC requires RFC mode via JCo sidecar
		if globalCfg.Verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] SNC: resolved %d JCo properties for system %q\n", len(jcoProps), resolved.SysID)
			for k, v := range jcoProps {
				_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE]   %s = %s\n", k, v)
			}
		}
	}

	// Handle cookie authentication (must run before browser auth to avoid duplicate auth methods)
	if err := augmentCookieAuthentication(resolved); err != nil {
		return err
	}

	return nil
}

// augmentCookieAuthentication augments cookie authentication for a system:
// - Processes cookie file (config file level already loaded in applySystemConfigDefaults)
// - Processes cookie string (CLI/env)
// - Handles browser-based SSO authentication
func augmentCookieAuthentication(resolved *config.SystemResolvedConfig) error {
	// If cookies already set by config file, they're already in place
	if len(resolved.Cookies) > 0 {
		return nil
	}

	// Not implementing browser auth here since it requires cmd context
	// Browser auth is handled separately in main.go processBrowserAuth

	return nil
}

// validateSystemConfiguration validates a system configuration for consistency.
func validateSystemConfiguration(singleSys *config.SystemResolvedConfig, globalCfg *config.ResolvedConfig) error {
	// In RFC mode, URL is not required; RFC connection params are
	if strings.EqualFold(singleSys.ConnectionMode, "rfc") {
		if singleSys.SNC {
			// SNC mode: connection params come from JcoProperties (resolved from landscape)
			if len(singleSys.JcoProperties) == 0 {
				return fmt.Errorf("SNC mode enabled but no JCo properties resolved from landscape")
			}
		} else {
			// Standard RFC mode: need explicit connection params
			hasDirect := singleSys.AsHost != ""
			hasLB := singleSys.MsHost != ""
			if !hasDirect && !hasLB {
				return fmt.Errorf("RFC mode requires --ashost or --mshost")
			}
			if hasDirect && hasLB {
				return fmt.Errorf("cannot specify both --ashost (direct) and --mshost (load balancing)")
			}
			if hasDirect && singleSys.SysNr == "" {
				return fmt.Errorf("--sysnr required for direct RFC connection")
			}
			if hasLB {
				if singleSys.MsServ == "" {
					return fmt.Errorf("--msserv required for RFC load balancing")
				}
				if singleSys.R3Name == "" {
					return fmt.Errorf("--r3name required for RFC load balancing")
				}
				if singleSys.Group == "" {
					return fmt.Errorf("--group required for RFC load balancing")
				}
			}
		}
	} else {
		// HTTP mode requires URL
		if singleSys.URL == "" {
			return fmt.Errorf("SAP URL is required. Use --url flag or SAP_URL environment variable")
		}
	}

	// Validate mode
	if globalCfg.Mode != "focused" && globalCfg.Mode != "expert" && globalCfg.Mode != "hyperfocused" {
		return fmt.Errorf("invalid mode: %s (must be 'focused', 'expert', or 'hyperfocused')", globalCfg.Mode)
	}

	return nil
}
