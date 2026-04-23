package mcp

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/internal/log"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// PermissionManager manages all aspects of role-based permissions.
// It is created once at startup from the global configuration and provides:
//   - Startup-time tool filtering (which tools to register per system)
//   - Runtime object-level checks (which objects each tool can access)
type PermissionManager struct {
	// Resolved permissions per system
	systemPermissions map[string]*SystemPermissions

	// All tool definitions (reference)
	allTools []types.ToolDef
}

// EffectiveToolPermission holds the combined permission and availability state for a single tool on a system.
type EffectiveToolPermission struct {
	// Enabled indicates whether the tool is allowed by role-based permissions.
	Enabled bool

	// Available indicates whether the tool's required endpoints are present on the system.
	// Defaults to true until endpoint filtering is applied.
	Available bool

	// Resolved holds the object-level permission details (allowed/blocked objects, etc.).
	// May be nil if no specific permission was resolved for this tool.
	Resolved *config.ResolvedToolPermission

	// ToolDef points to the tool definition.
	ToolDef *types.ToolDef
}

// SystemPermissions holds resolved permissions for a single system.
type SystemPermissions struct {
	SystemID string

	// Tools holds all tools with their effective permission and availability state.
	Tools map[string]*EffectiveToolPermission

	// Role names for logging
	RoleNames []string
}

// NewPermissionManager creates and initializes a new permission manager.
// It validates roles, resolves permissions for each system, and determines
// which tools are globally enabled vs disabled.
func NewPermissionManager(cfg *config.GlobalConfig, allTools []types.ToolDef) (*PermissionManager, error) {
	pm := &PermissionManager{
		systemPermissions: make(map[string]*SystemPermissions),
		allTools:          allTools,
	}

	// Step 1: Validate role configuration
	warnings := config.ValidateRolesConfig(cfg.Roles)
	for _, w := range warnings {
		log.Warning("%s", w)
	}

	// Step 2: Build all tool names for permission resolution
	allToolNames := make([]string, len(allTools))
	for i, td := range allTools {
		allToolNames[i] = td.Tool.Name
	}

	// Step 3: Resolve permissions for each system
	for sysID, sysCfg := range cfg.Systems {
		sp, err := pm.resolveSystemPermissions(sysID, sysCfg, cfg.Roles, allToolNames)
		if err != nil {
			return nil, fmt.Errorf("permission resolution for system %q: %w", sysID, err)
		}
		pm.systemPermissions[strings.ToLower(sysID)] = sp
	}

	return pm, nil
}

// resolveSystemPermissions computes the full permission set for one system.
func (pm *PermissionManager) resolveSystemPermissions(
	sysID string,
	sysCfg config.SystemConfig,
	definedRoles map[string]config.RoleDefinition,
	allToolNames []string,
) (*SystemPermissions, error) {
	sp := &SystemPermissions{
		SystemID: sysID,
		Tools:    make(map[string]*EffectiveToolPermission),
	}

	// Step 1: Resolve which roles apply to this system
	roleDefs, err := config.ResolveRolesForSystem(sysID, sysCfg.Roles, definedRoles)
	if err != nil {
		return nil, err
	}

	// Log role resolution
	if len(sysCfg.Roles) == 0 {
		log.Warning("System %q: no roles specified, using built-in 'default' role", sysID)
		sp.RoleNames = []string{"default"}
	} else {
		sp.RoleNames = sysCfg.Roles
	}

	// Step 2: Expand nested roles
	roleNames := sysCfg.Roles
	if len(roleNames) == 0 {
		roleNames = []string{"default"}
	}

	allRoles, err := pm.expandAllRoles(roleNames, definedRoles)
	if err != nil {
		return nil, err
	}

	// Use all expanded roles if available, otherwise use the resolved ones
	if len(allRoles) > 0 {
		roleDefs = allRoles
	}

	// Step 3: Merge permissions from all roles
	merged := config.MergeRolePermissions(roleDefs, allToolNames)

	// Step 4: Build the unified tool map
	for i := range pm.allTools {
		td := &pm.allTools[i]
		toolName := td.Tool.Name
		resolved, exists := merged[toolName]
		enabled := exists && !resolved.GloballyDisabled
		sp.Tools[toolName] = &EffectiveToolPermission{
			Enabled:   enabled,
			Available: true, // until endpoint filtering is applied
			Resolved:  resolved,
			ToolDef:   td,
		}
	}

	return sp, nil
}

// expandAllRoles recursively expands all role names into flat RoleDefinitions.
func (pm *PermissionManager) expandAllRoles(
	roleNames []string,
	definedRoles map[string]config.RoleDefinition,
) ([]config.RoleDefinition, error) {
	return config.ExpandNestedRoles(roleNames, definedRoles)
}

// GetEnabledToolsForSystem returns tools enabled for a specific system.
func (pm *PermissionManager) GetEnabledToolsForSystem(systemID string) []*types.ToolDef {
	sp, exists := pm.systemPermissions[strings.ToLower(systemID)]
	if !exists {
		return nil
	}
	var result []*types.ToolDef
	for _, eff := range sp.Tools {
		if eff.Enabled && eff.Available {
			result = append(result, eff.ToolDef)
		}
	}
	return result
}

// GetEnabledToolNames returns sorted tool names enabled for a system (for discovery).
func (pm *PermissionManager) GetEnabledToolNames(systemID string) []string {
	sp, exists := pm.systemPermissions[strings.ToLower(systemID)]
	if !exists {
		return nil
	}
	var names []string
	for name, eff := range sp.Tools {
		if eff.Enabled && eff.Available {
			names = append(names, name)
		}
	}
	return names
}

// DiscoveryToolInfo describes a single tool's resolved object-level restrictions for discovery output.
type DiscoveryToolInfo struct {
	Name            string   `json:"name"`
	AllowedPackages []string `json:"allowed_packages,omitempty"`
	AllowedObjects  []string `json:"allowed_objects,omitempty"`
	BlockedObjects  []string `json:"blocked_objects,omitempty"`
}

// GetRestrictedTools returns the list of tools that have object-level restrictions for a system.
// Returns nil when no tools have restrictions.
func (pm *PermissionManager) GetRestrictedTools(systemID string) []DiscoveryToolInfo {
	sp, exists := pm.systemPermissions[strings.ToLower(systemID)]
	if !exists {
		return nil
	}

	var result []DiscoveryToolInfo
	for name, eff := range sp.Tools {
		if !eff.Enabled || !eff.Available {
			continue
		}
		if eff.Resolved == nil || !eff.Resolved.ObjectRestricted {
			continue
		}

		result = append(result, DiscoveryToolInfo{
			Name:            name,
			AllowedPackages: eff.Resolved.AllowedPackages,
			AllowedObjects:  eff.Resolved.AllowedObjects,
			BlockedObjects:  eff.Resolved.BlockedObjects,
		})
	}

	return result
}

// GetGloballyEnabledTools returns all tools that are enabled for at least one system.
// This is used to determine which tools to register with the MCP server.
func (pm *PermissionManager) GetGloballyEnabledTools() []*types.ToolDef {
	enabledSet := make(map[string]bool)
	for _, sp := range pm.systemPermissions {
		for name, eff := range sp.Tools {
			if eff.Enabled && eff.Available {
				enabledSet[name] = true
			}
		}
	}

	var result []*types.ToolDef
	for i := range pm.allTools {
		if enabledSet[pm.allTools[i].Tool.Name] {
			result = append(result, &pm.allTools[i])
		}
	}
	return result
}

// IsToolEnabledForSystem checks if a tool is globally enabled for a system.
func (pm *PermissionManager) IsToolEnabledForSystem(systemID, toolName string) bool {
	sp, exists := pm.systemPermissions[strings.ToLower(systemID)]
	if !exists {
		return false
	}
	eff, exists := sp.Tools[toolName]
	if !exists {
		return false
	}
	return eff.Enabled && eff.Available
}

// IsObjectAllowedForTool checks if an object is allowed for a tool at runtime.
// Returns nil if allowed, an error with context if blocked.
func (pm *PermissionManager) IsObjectAllowedForTool(
	systemID string,
	toolName string,
	objectName string,
	objectPackage string, //nolint:revive // reserved for future package-level restriction enforcement
) error {
	sp, exists := pm.systemPermissions[strings.ToLower(systemID)]
	if !exists {
		return nil // No permissions configured → allow
	}

	eff, exists := sp.Tools[toolName]
	if !exists || eff.Resolved == nil || !eff.Resolved.ObjectRestricted {
		return nil // No object restrictions → allow
	}

	resolved := eff.Resolved

	// Check blocked_objects (deny wins)
	for _, pattern := range resolved.BlockedObjects {
		if config.MatchObjectPattern(pattern, objectName) {
			return fmt.Errorf("object %q is blocked for tool %q on system %q (matched blocked pattern %q)",
				objectName, toolName, systemID, pattern)
		}
	}

	// Check allowed_objects (if specified, must match)
	if len(resolved.AllowedObjects) > 0 {
		matched := false
		for _, pattern := range resolved.AllowedObjects {
			if config.MatchObjectPattern(pattern, objectName) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("object %q is not in the allowed list for tool %q on system %q",
				objectName, toolName, systemID)
		}
	}

	// allowed_packages: parsed but not enforced yet
	// (warnings are already logged at startup)

	return nil
}

// ObjectNotAllowedMessage generates a helpful error message when an object is blocked.
func (pm *PermissionManager) ObjectNotAllowedMessage(
	systemID string,
	toolName string,
	objectName string,
) string {
	return fmt.Sprintf(
		"Access denied: object %q cannot be accessed by tool %q on system %q. "+
			"Check role configuration for allowed/blocked object patterns.",
		objectName, toolName, systemID)
}

// LogEffectivePermissions logs startup information about enabled/disabled tools per system.
func (pm *PermissionManager) LogEffectivePermissions() {
	for _, sp := range pm.systemPermissions {
		enabled := make([]string, 0)
		disabled := make([]string, 0)
		unavailable := make([]string, 0)

		roleStr := strings.Join(sp.RoleNames, ", ")
		log.Info("System %q (roles: [%s]):", sp.SystemID, roleStr)

		for name, eff := range sp.Tools {
			switch {
			case !eff.Enabled:
				disabled = append(disabled, name)
			case !eff.Available:
				unavailable = append(unavailable, name)
			case eff.Resolved != nil && eff.Resolved.ObjectRestricted:
				enabled = append(enabled, fmt.Sprintf("%s (object restrictions)", name))
			default:
				enabled = append(enabled, name)
			}
		}

		slices.Sort(enabled)
		slices.Sort(disabled)
		slices.Sort(unavailable)

		for _, name := range enabled {
			log.Info("  ✓ %s", name)
		}

		for _, name := range disabled {
			log.Info("  ✗ %s (disabled by role)", name)
		}

		for _, name := range unavailable {
			log.Info("  ✗ %s (endpoint unavailable)", name)
		}
	}
}

// FilterToolsByEndpoints removes tools whose declared Endpoints are not available
// on the target system according to its ADT discovery result.
// Tools with an empty Endpoints slice are never filtered (always pass).
// If discoveredEndpoints is nil or empty, no tools are filtered.
func FilterToolsByEndpoints(tools []*types.ToolDef, discoveredEndpoints adt.DiscoveredEndpoints) []*types.ToolDef {
	if len(discoveredEndpoints) == 0 {
		return tools
	}

	var result []*types.ToolDef
	for _, td := range tools {
		if len(td.Endpoints) == 0 {
			// No endpoint requirements declared → always available
			result = append(result, td)
			continue
		}

		allFound := true
		for _, ep := range td.Endpoints {
			if !discoveredEndpoints.HasEndpoint(ep) {
				allFound = false
				break
			}
		}

		if allFound {
			result = append(result, td)
		}
	}

	return result
}

// ApplyEndpointFilter marks tools as unavailable based on ADT discovery results.
// This is called after permission-based filtering and after all systems have connected.
func (pm *PermissionManager) ApplyEndpointFilter(systems map[string]types.System, verbose bool) {
	for sysID, sp := range pm.systemPermissions {
		sys, ok := systems[strings.ToLower(sysID)]
		if !ok {
			continue
		}

		endpoints := sys.DiscoveredEndpoints()
		if len(endpoints) == 0 {
			continue
		}

		filtered := 0
		for _, eff := range sp.Tools {
			if !eff.Enabled || len(eff.ToolDef.Endpoints) == 0 {
				continue
			}

			allFound := true
			for _, ep := range eff.ToolDef.Endpoints {
				if !endpoints.HasEndpoint(ep) {
					allFound = false
					break
				}
			}

			if !allFound {
				eff.Available = false
				filtered++
			}
		}

		if filtered > 0 && verbose {
			_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] System %q: endpoint filter marked %d tools as unavailable\n",
				sysID, filtered)
		}
	}
}
