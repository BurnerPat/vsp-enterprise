package mcp

import (
	"fmt"
	"strings"

	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/internal/log"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
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

// SystemPermissions holds resolved permissions for a single system.
type SystemPermissions struct {
	SystemID string

	// Tools globally disabled for this system (not in any role)
	DisabledTools map[string]bool

	// Per-tool resolved permissions (includes object restrictions)
	ToolPermissions map[string]*config.ResolvedToolPermission

	// Tools enabled for this system (pointers into allTools)
	EnabledTools []*types.ToolDef

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
		log.LogWarning("%s", w)
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
		SystemID:      sysID,
		DisabledTools: make(map[string]bool),
	}

	// Step 1: Resolve which roles apply to this system
	roleDefs, err := config.ResolveRolesForSystem(sysID, sysCfg.Roles, definedRoles)
	if err != nil {
		return nil, err
	}

	// Log role resolution
	if len(sysCfg.Roles) == 0 {
		log.LogWarning("System %q: no roles specified, using built-in 'default' role", sysID)
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
	sp.ToolPermissions = merged

	// Step 4: Build enabled/disabled tool lists
	for i := range pm.allTools {
		td := &pm.allTools[i]
		toolName := td.Tool.Name
		if resolved, exists := merged[toolName]; exists && !resolved.GloballyDisabled {
			sp.EnabledTools = append(sp.EnabledTools, td)
		} else {
			sp.DisabledTools[toolName] = true
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
	return sp.EnabledTools
}

// GetGloballyEnabledTools returns all tools that are enabled for at least one system.
// This is used to determine which tools to register with the MCP server.
func (pm *PermissionManager) GetGloballyEnabledTools() []*types.ToolDef {
	enabledSet := make(map[string]bool)
	for _, sp := range pm.systemPermissions {
		for _, td := range sp.EnabledTools {
			enabledSet[td.Tool.Name] = true
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
	return !sp.DisabledTools[toolName]
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

	resolved, exists := sp.ToolPermissions[toolName]
	if !exists || !resolved.ObjectRestricted {
		return nil // No object restrictions → allow
	}

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
		roleStr := strings.Join(sp.RoleNames, ", ")
		log.LogInfo("System %q (roles: [%s]):", sp.SystemID, roleStr)

		for _, td := range sp.EnabledTools {
			toolName := td.Tool.Name
			resolved := sp.ToolPermissions[toolName]
			if resolved != nil && resolved.ObjectRestricted {
				log.LogInfo("  ✓ %s (with object restrictions)", toolName)
			} else {
				log.LogInfo("  ✓ %s", toolName)
			}
		}

		// Log disabled tools
		for toolName := range sp.DisabledTools {
			log.LogInfo("  ✗ %s", toolName)
		}
	}
}
