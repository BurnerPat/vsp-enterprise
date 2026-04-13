package config

import (
	"fmt"
	"strings"

	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// ---------------------------------------------------------------------------
// Built-in roles
// ---------------------------------------------------------------------------

// GetBuiltInRoles returns the built-in roles that are always available.
// The "default" role enables all tools with no restrictions.
func GetBuiltInRoles() map[string]RoleDefinition {
	return map[string]RoleDefinition{
		"default": {
			Description: "Built-in default role: unrestricted access to all tools and objects",
			Tools: map[string]ToolPermission{
				"*": {}, // All tools enabled, no restrictions
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Role resolution for a system
// ---------------------------------------------------------------------------

// ResolveRolesForSystem returns the role definitions to apply to a system.
// If system has no explicit roles, returns the built-in or user-overridden "default" role.
// If system has explicit roles, returns those roles.
func ResolveRolesForSystem(
	systemID string,
	systemRoles []string,
	definedRoles map[string]RoleDefinition,
) ([]RoleDefinition, error) {
	builtInRoles := GetBuiltInRoles()

	if len(systemRoles) == 0 {
		// No explicit roles → use "default" (user-defined overrides built-in)
		if roleDef, exists := definedRoles["default"]; exists {
			return []RoleDefinition{roleDef}, nil
		}
		return []RoleDefinition{builtInRoles["default"]}, nil
	}

	// System has explicit roles → resolve them
	result := make([]RoleDefinition, 0, len(systemRoles))
	for _, roleName := range systemRoles {
		// Check user-defined roles first (allows override of built-in)
		if roleDef, exists := definedRoles[roleName]; exists {
			result = append(result, roleDef)
			continue
		}
		// Fall back to built-in roles
		if roleDef, exists := builtInRoles[roleName]; exists {
			result = append(result, roleDef)
			continue
		}
		return nil, fmt.Errorf("system %q: undefined role %q", systemID, roleName)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Role expansion (nested roles)
// ---------------------------------------------------------------------------

// ExpandNestedRoles recursively expands composite roles into a flat list.
// Detects cycles and returns an error if one is found.
func ExpandNestedRoles(roles []string, roleMap map[string]RoleDefinition) ([]RoleDefinition, error) {
	var result []RoleDefinition
	visited := make(map[string]bool)
	inStack := make(map[string]bool)

	var expand func(roleName string) error
	expand = func(roleName string) error {
		if inStack[roleName] {
			return fmt.Errorf("cyclic role reference detected: %q", roleName)
		}
		if visited[roleName] {
			return nil // Already processed
		}

		visited[roleName] = true
		inStack[roleName] = true

		roleDef, exists := roleMap[roleName]
		if !exists {
			// Check built-in roles
			builtIn := GetBuiltInRoles()
			if biDef, biExists := builtIn[roleName]; biExists {
				roleDef = biDef
			} else {
				return fmt.Errorf("undefined role: %q", roleName)
			}
		}

		// Expand nested roles first (depth-first)
		for _, nestedName := range roleDef.NestedRoles {
			if err := expand(nestedName); err != nil {
				return err
			}
		}

		result = append(result, roleDef)
		inStack[roleName] = false
		return nil
	}

	for _, roleName := range roles {
		if err := expand(roleName); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// Permission merging
// ---------------------------------------------------------------------------

// MergeRolePermissions combines multiple role definitions into a resolved permission set per tool.
// Semantics:
//   - If ANY role enables a tool pattern → tool is enabled
//   - If ANY role sets "enabled": false → tool is GLOBALLY DISABLED (deny wins)
//   - blocked_objects from ANY role → object is blocked (deny wins)
//   - allowed_objects from ANY role → union of allowed objects
//   - allowed_packages from ANY role → union of allowed packages
func MergeRolePermissions(roles []RoleDefinition, allToolNames []string) map[string]*ResolvedToolPermission {
	result := make(map[string]*ResolvedToolPermission)

	// Collect patterns from all roles
	type patternPerm struct {
		pattern string
		perm    ToolPermission
		safety  *SafetyConfigJSON
	}
	var allPatterns []patternPerm

	for _, role := range roles {
		var safetyCfg *SafetyConfigJSON
		if role.Safety != nil {
			s := SafetyConfigJSON(*role.Safety)
			safetyCfg = &s
		}
		for pattern, perm := range role.Tools {
			allPatterns = append(allPatterns, patternPerm{
				pattern: pattern,
				perm:    perm,
				safety:  safetyCfg,
			})
		}
	}

	// Phase 2: For each actual tool, resolve permissions by matching against patterns
	for _, toolName := range allToolNames {
		resolved := &ResolvedToolPermission{}
		matched := false
		explicitlyDenied := false

		for _, pp := range allPatterns {
			if !MatchToolPattern(pp.pattern, toolName) {
				continue
			}

			// Check explicit deny
			if pp.perm.Enabled != nil && !*pp.perm.Enabled {
				explicitlyDenied = true
				continue
			}

			matched = true

			// Merge object restrictions (union semantics)
			if len(pp.perm.AllowedPackages) > 0 {
				resolved.AllowedPackages = appendUnique(resolved.AllowedPackages, pp.perm.AllowedPackages...)
				resolved.ObjectRestricted = true
			}
			if len(pp.perm.AllowedObjects) > 0 {
				resolved.AllowedObjects = appendUnique(resolved.AllowedObjects, pp.perm.AllowedObjects...)
				resolved.ObjectRestricted = true
			}
			if len(pp.perm.BlockedObjects) > 0 {
				resolved.BlockedObjects = appendUnique(resolved.BlockedObjects, pp.perm.BlockedObjects...)
				resolved.ObjectRestricted = true
			}

			// Merge safety (most restrictive wins)
			if pp.safety != nil {
				if resolved.Safety == nil {
					safetyCopy := *pp.safety
					resolved.Safety = &safetyCopy
				}
				// For boolean safety flags, OR semantics = most restrictive wins
			}
		}

		if explicitlyDenied {
			resolved.GloballyDisabled = true
		} else if !matched {
			resolved.GloballyDisabled = true
		}

		result[toolName] = resolved
	}

	return result
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

// ValidateRolesConfig checks for invalid role references, cycles, etc.
func ValidateRolesConfig(roles map[string]RoleDefinition) []string {
	var warnings []string

	if len(roles) == 0 {
		return warnings
	}

	builtIn := GetBuiltInRoles()

	for roleName, roleDef := range roles {
		// Check for undefined nested role references
		for _, nestedRole := range roleDef.NestedRoles {
			if _, exists := roles[nestedRole]; !exists {
				if _, biExists := builtIn[nestedRole]; !biExists {
					warnings = append(warnings, fmt.Sprintf("role %q: references undefined nested role %q", roleName, nestedRole))
				}
			}
		}

		// Check for package-level restrictions (not yet implemented)
		for toolPattern, toolPerm := range roleDef.Tools {
			if len(toolPerm.AllowedPackages) > 0 {
				warnings = append(warnings, fmt.Sprintf(
					"role %q tool %q: package-level restrictions not yet implemented (ignored)",
					roleName, toolPattern))
			}
		}
	}

	// Check for cycles
	for roleName := range roles {
		_, err := ExpandNestedRoles([]string{roleName}, roles)
		if err != nil && strings.Contains(err.Error(), "cyclic") {
			warnings = append(warnings, fmt.Sprintf("role %q: %s", roleName, err.Error()))
		}
	}

	return warnings
}

// ---------------------------------------------------------------------------
// Tool pattern matching
// ---------------------------------------------------------------------------

// MatchToolPattern checks if a tool name matches a glob-like pattern.
// Supports '*' as a wildcard character.
func MatchToolPattern(pattern, toolName string) bool {
	if pattern == "*" {
		return true
	}

	parts := strings.Split(pattern, "*")

	remaining := toolName
	for i, part := range parts {
		if part == "" {
			continue
		}
		idx := strings.Index(remaining, part)
		if idx == -1 {
			return false
		}
		// First part must match at the beginning if pattern doesn't start with *
		if i == 0 && !strings.HasPrefix(pattern, "*") && idx != 0 {
			return false
		}
		remaining = remaining[idx+len(part):]
	}

	// If pattern doesn't end with *, remaining must be empty
	return strings.HasSuffix(pattern, "*") || remaining == ""
}

// MatchObjectPattern checks if an object name matches a glob-like pattern.
// Uses case-insensitive matching. Supports '*' as a wildcard character.
func MatchObjectPattern(pattern, objectName string) bool {
	return MatchToolPattern(strings.ToUpper(pattern), strings.ToUpper(objectName))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// SafetyConfigJSON is an alias for adt.SafetyConfig used during JSON processing.
type SafetyConfigJSON = adt.SafetyConfig

func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool, len(slice))
	for _, s := range slice {
		seen[s] = true
	}
	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}
	return slice
}
