// Package internal provides internal infrastructure for vsp.
package internal

import (
	"fmt"
	"os"
	"strings"

	"dario.cat/mergo"
	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/pkg/adt"
	"github.com/spf13/cobra"
)

// Bootstrap orchestrates the entire configuration pipeline:
// 1. Viper/Cobra have already resolved CLI flags and environment variables into cfg
// 2. Load configuration file (.vsp.json) if available
// 3. Merge config file into cfg with precedence: CLI/Env wins > Config File
// 4. Augment configuration with derived data (SNC, JCo properties, cookies, etc.)
// 5. Validate final configuration
// 6. Return fully-prepared config ready for mcp.NewServer()
//
// Parameters:
//   - cfg: Base config structure (already populated by Viper/Cobra from CLI/ENV)
//   - singleSys: Per-system settings from CLI/ENV (already populated)
//   - multiSystem: Whether multi-system mode is enabled
//   - configFile: Explicit path to .vsp.json (empty = auto-discover)
//   - systemName: System name from --system flag (for single-system mode)
//   - cmd: Cobra command for flag inspection
//
// Returns the fully-augmented ResolvedConfig ready for instantiation.
func Bootstrap(cfg *config.GlobalConfig, singleSys *config.SystemConfig, multiSystem bool, configFile, systemName string, cmd *cobra.Command) (*config.GlobalConfig, error) {
	// Step 1: Load configuration file (.vsp.json) if available
	// Config file is OPTIONAL - if not found, just continue with CLI/ENV config
	var systemsCfg *config.SystemsConfig
	var systemsConfigPath string

	if configFile != "" {
		var err error
		systemsCfg, err = config.LoadSystemsFromFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config from %s: %w", configFile, err)
		}
		systemsConfigPath = configFile
	} else {
		var err error
		systemsCfg, systemsConfigPath, err = config.LoadSystems()
		if err != nil {
			// Config file not found - that's OK, continue with CLI/ENV config
			if cfg.Verbose {
				_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] No config file found, using CLI/ENV configuration\n")
			}
		}
	}

	// Apply tool visibility from .vsp.json (if it exists)
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

	// Step 2: Initialize the Systems map
	cfg.Systems = make(map[string]*config.SystemConfig)

	// Step 3: Route to multi-system or single-system bootstrap
	if multiSystem {
		if err := bootstrapMultiSystem(cfg, systemsCfg, systemsConfigPath); err != nil {
			return nil, err
		}
	} else {
		if err := bootstrapSingleSystem(cfg, singleSys, systemsCfg, systemName); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

// bootstrapMultiSystem populates cfg.Systems from .vsp.json and augments each system.
func bootstrapMultiSystem(cfg *config.GlobalConfig, systemsCfg *config.SystemsConfig, systemsConfigPath string) error {
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

		// GetSystem returns *config.SystemConfig directly — no conversion needed
		sys, err := systemsCfg.GetSystem(sysID)
		if err != nil {
			return fmt.Errorf("--multi-system: failed to resolve system %q: %w", sysID, err)
		}

		if err := augmentSystemConfiguration(sys, cfg); err != nil {
			return fmt.Errorf("--multi-system: failed to augment system %q: %w", sysID, err)
		}

		// Count RFC systems (explicit or via SNC)
		if strings.EqualFold(sys.ConnectionMode, "rfc") {
			rfcSystemCount++
		}

		cfg.Systems[sysID] = sys
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

// bootstrapSingleSystem resolves single-system configuration from CLI/ENV and .vsp.json profile.
func bootstrapSingleSystem(cfg *config.GlobalConfig, singleSys *config.SystemConfig, systemsCfg *config.SystemsConfig, systemName string) error {
	// If --system flag is specified, load system config from .vsp.json first
	// System profile values act as base defaults; CLI/ENV values override them via merging
	if systemName != "" {
		if systemsCfg == nil {
			_, _ = fmt.Fprintf(os.Stderr, "[WARN] --system '%s' specified but no .vsp.json found\n", systemName)
		} else {
			// GetSystem returns *config.SystemConfig directly — no conversion needed
			sys, err := systemsCfg.GetSystem(systemName)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "[WARN] %v\n", err)
			} else {
				if cfg.Verbose {
					_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] Loading system '%s' from .vsp.json\n", systemName)
				}
				// Merge config file profile into singleSys; CLI/ENV values win
				if err := mergeSystemConfiguration(singleSys, sys); err != nil {
					return fmt.Errorf("failed to merge system profile: %w", err)
				}
			}
		}
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

// mergeSystemConfiguration merges fileConfig into cliEnv with CLI/ENV taking precedence.
// Uses mergo: zero-value fields in cliEnv are filled from fileConfig; non-zero fields are kept.
func mergeSystemConfiguration(cliEnv, fileConfig *config.SystemConfig) error {
	if err := mergo.Merge(cliEnv, fileConfig); err != nil {
		return fmt.Errorf("failed to merge system configuration: %w", err)
	}
	return nil
}

// augmentSystemConfiguration augments a system with derived runtime data:
// SNC/JCo properties from SAP UI Landscape, and cookie authentication.
func augmentSystemConfiguration(sys *config.SystemConfig, globalCfg *config.GlobalConfig) error {
	// Resolve SNC/SSO configuration from SAP UI Landscape file
	if sys.SNC {
		if sys.SysID == "" {
			return fmt.Errorf("--sysid is required when --snc is specified")
		}
		if globalCfg.Verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] SNC mode: resolving system %q from SAP UI Landscape\n", sys.SysID)
			if sys.LandscapeFile != "" {
				_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] Using landscape file: %s\n", sys.LandscapeFile)
			}
		}
		jcoProps, err := adt.ResolveSNCJcoProperties(sys.SysID, sys.LandscapeFile, sys.Client, sys.Language)
		if err != nil {
			return fmt.Errorf("SNC configuration failed: %w", err)
		}
		sys.JcoProperties = jcoProps
		sys.ConnectionMode = "rfc" // SNC requires RFC mode via JCo sidecar
		if globalCfg.Verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] SNC: resolved %d JCo properties for system %q\n", len(jcoProps), sys.SysID)
			for k, v := range jcoProps {
				_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE]   %s = %s\n", k, v)
			}
		}
	}

	return nil
}

// validateSystemConfiguration validates a system configuration for consistency.
func validateSystemConfiguration(sys *config.SystemConfig, globalCfg *config.GlobalConfig) error {
	// In RFC mode, URL is not required; RFC connection params are
	if strings.EqualFold(sys.ConnectionMode, "rfc") {
		if sys.SNC {
			// SNC mode: connection params come from JcoProperties (resolved from landscape)
			if len(sys.JcoProperties) == 0 {
				return fmt.Errorf("SNC mode enabled but no JCo properties resolved from landscape")
			}
		} else {
			// Standard RFC mode: need explicit connection params
			hasDirect := sys.AsHost != ""
			hasLB := sys.MsHost != ""
			if !hasDirect && !hasLB {
				return fmt.Errorf("RFC mode requires --ashost or --mshost")
			}
			if hasDirect && hasLB {
				return fmt.Errorf("cannot specify both --ashost (direct) and --mshost (load balancing)")
			}
			if hasDirect && sys.SysNr == "" {
				return fmt.Errorf("--sysnr required for direct RFC connection")
			}
			if hasLB {
				if sys.MsServ == "" {
					return fmt.Errorf("--msserv required for RFC load balancing")
				}
				if sys.R3Name == "" {
					return fmt.Errorf("--r3name required for RFC load balancing")
				}
				if sys.Group == "" {
					return fmt.Errorf("--group required for RFC load balancing")
				}
			}
		}
	} else {
		// HTTP mode requires URL
		if sys.URL == "" {
			return fmt.Errorf("SAP URL is required. Use --url flag or SAP_URL environment variable")
		}
	}

	// Validate mode
	if globalCfg.Mode != "focused" && globalCfg.Mode != "expert" && globalCfg.Mode != "hyperfocused" {
		return fmt.Errorf("invalid mode: %s (must be 'focused', 'expert', or 'hyperfocused')", globalCfg.Mode)
	}

	return nil
}
