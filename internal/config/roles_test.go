package config

import (
	"testing"
)

// ---------------------------------------------------------------------------
// MatchToolPattern
// ---------------------------------------------------------------------------

func TestMatchToolPattern_Exact(t *testing.T) {
	if !MatchToolPattern("ReadTable", "ReadTable") {
		t.Error("exact match should succeed")
	}
	if MatchToolPattern("ReadTable", "ReadTableX") {
		t.Error("exact match should not match longer name")
	}
}

func TestMatchToolPattern_Wildcard(t *testing.T) {
	if !MatchToolPattern("*", "AnyTool") {
		t.Error("* should match anything")
	}
}

func TestMatchToolPattern_Prefix(t *testing.T) {
	if !MatchToolPattern("Get*", "GetClass") {
		t.Error("prefix match should succeed")
	}
	if MatchToolPattern("Get*", "SetClass") {
		t.Error("prefix match should not match different prefix")
	}
}

func TestMatchToolPattern_Suffix(t *testing.T) {
	if !MatchToolPattern("*Source", "GetSource") {
		t.Error("suffix match should succeed")
	}
	if MatchToolPattern("*Source", "GetSources") {
		t.Error("suffix match should not match with extra chars")
	}
}

func TestMatchToolPattern_Contains(t *testing.T) {
	if !MatchToolPattern("*Test*", "RunUnitTests") {
		t.Error("contains match should succeed")
	}
	if MatchToolPattern("*Test*", "RunQuery") {
		t.Error("contains match should not match unrelated")
	}
}

// ---------------------------------------------------------------------------
// MatchObjectPattern (case-insensitive)
// ---------------------------------------------------------------------------

func TestMatchObjectPattern_CaseInsensitive(t *testing.T) {
	if !MatchObjectPattern("Z*", "ZAPP") {
		t.Error("should match uppercase")
	}
	if !MatchObjectPattern("z*", "ZAPP") {
		t.Error("should match case-insensitively")
	}
	if !MatchObjectPattern("USR*", "usrt001") {
		t.Error("should match case-insensitively")
	}
}

// ---------------------------------------------------------------------------
// ExpandNestedRoles
// ---------------------------------------------------------------------------

func TestExpandNestedRoles_Simple(t *testing.T) {
	roles := map[string]RoleDefinition{
		"reader": {
			Tools: map[string]ToolPermission{
				"Get*": {},
			},
		},
	}

	result, err := ExpandNestedRoles([]string{"reader"}, roles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 role, got %d", len(result))
	}
}

func TestExpandNestedRoles_Nested(t *testing.T) {
	roles := map[string]RoleDefinition{
		"base": {
			Tools: map[string]ToolPermission{
				"Get*": {},
			},
		},
		"extended": {
			NestedRoles: []string{"base"},
			Tools: map[string]ToolPermission{
				"Search*": {},
			},
		},
	}

	result, err := ExpandNestedRoles([]string{"extended"}, roles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 roles (base + extended), got %d", len(result))
	}
}

func TestExpandNestedRoles_CycleDetection(t *testing.T) {
	roles := map[string]RoleDefinition{
		"a": {NestedRoles: []string{"b"}},
		"b": {NestedRoles: []string{"a"}},
	}

	_, err := ExpandNestedRoles([]string{"a"}, roles)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
}

func TestExpandNestedRoles_UndefinedRole(t *testing.T) {
	roles := map[string]RoleDefinition{
		"a": {NestedRoles: []string{"nonexistent"}},
	}

	_, err := ExpandNestedRoles([]string{"a"}, roles)
	if err == nil {
		t.Fatal("expected error for undefined role")
	}
}

func TestExpandNestedRoles_BuiltInDefault(t *testing.T) {
	roles := map[string]RoleDefinition{
		"custom": {
			NestedRoles: []string{"default"},
			Tools: map[string]ToolPermission{
				"RunQuery": {Enabled: boolPtr(false)},
			},
		},
	}

	result, err := ExpandNestedRoles([]string{"custom"}, roles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 roles (default + custom), got %d", len(result))
	}
}

// ---------------------------------------------------------------------------
// MergeRolePermissions
// ---------------------------------------------------------------------------

func TestMergeRolePermissions_AllowSemantics(t *testing.T) {
	roles := []RoleDefinition{
		{
			Tools: map[string]ToolPermission{
				"Get*":    {},
				"Search*": {},
			},
		},
	}

	allTools := []string{"GetClass", "GetProgram", "SearchObject", "RunQuery"}
	merged := MergeRolePermissions(roles, allTools)

	// Get* and Search* should be enabled
	if merged["GetClass"].GloballyDisabled {
		t.Error("GetClass should be enabled")
	}
	if merged["SearchObject"].GloballyDisabled {
		t.Error("SearchObject should be enabled")
	}
	// RunQuery should be disabled (not matched by any pattern)
	if !merged["RunQuery"].GloballyDisabled {
		t.Error("RunQuery should be disabled")
	}
}

func TestMergeRolePermissions_DenyWins(t *testing.T) {
	roles := []RoleDefinition{
		{
			Tools: map[string]ToolPermission{
				"ReadTable": {
					AllowedObjects: []string{"Z*", "Y*"},
				},
			},
		},
		{
			Tools: map[string]ToolPermission{
				"ReadTable": {
					BlockedObjects: []string{"Z001"},
				},
			},
		},
	}

	allTools := []string{"ReadTable"}
	merged := MergeRolePermissions(roles, allTools)

	rt := merged["ReadTable"]
	if rt.GloballyDisabled {
		t.Error("ReadTable should be enabled")
	}
	if !rt.ObjectRestricted {
		t.Error("ReadTable should have object restrictions")
	}
	if len(rt.AllowedObjects) != 2 {
		t.Errorf("expected 2 allowed objects, got %d", len(rt.AllowedObjects))
	}
	if len(rt.BlockedObjects) != 1 {
		t.Errorf("expected 1 blocked object, got %d", len(rt.BlockedObjects))
	}
}

func TestMergeRolePermissions_ExplicitDenyOverridesAllow(t *testing.T) {
	roles := []RoleDefinition{
		{
			Tools: map[string]ToolPermission{
				"*": {}, // Allow all
			},
		},
		{
			Tools: map[string]ToolPermission{
				"RunQuery": {Enabled: boolPtr(false)},
			},
		},
	}

	allTools := []string{"GetClass", "RunQuery"}
	merged := MergeRolePermissions(roles, allTools)

	if merged["GetClass"].GloballyDisabled {
		t.Error("GetClass should be enabled")
	}
	if !merged["RunQuery"].GloballyDisabled {
		t.Error("RunQuery should be disabled (explicit deny)")
	}
}

func TestMergeRolePermissions_WildcardMatching(t *testing.T) {
	roles := []RoleDefinition{
		{
			Tools: map[string]ToolPermission{
				"Get*":    {},
				"Search*": {},
				"DataPreview": {
					BlockedObjects: []string{"T001"},
				},
			},
		},
	}

	allTools := []string{"GetClass", "GetProgram", "SearchObject", "DataPreview", "RunQuery"}
	merged := MergeRolePermissions(roles, allTools)

	if merged["GetClass"].GloballyDisabled {
		t.Error("GetClass should be enabled")
	}
	if merged["GetProgram"].GloballyDisabled {
		t.Error("GetProgram should be enabled")
	}
	if merged["DataPreview"].GloballyDisabled {
		t.Error("DataPreview should be enabled")
	}
	if !merged["DataPreview"].ObjectRestricted {
		t.Error("DataPreview should have object restrictions")
	}
	if !merged["RunQuery"].GloballyDisabled {
		t.Error("RunQuery should be disabled")
	}
}

// ---------------------------------------------------------------------------
// ResolveRolesForSystem
// ---------------------------------------------------------------------------

func TestResolveRolesForSystem_DefaultRoleFallback(t *testing.T) {
	roles, err := ResolveRolesForSystem("dev", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(roles))
	}
	// Should have * pattern (all tools)
	if _, exists := roles[0].Tools["*"]; !exists {
		t.Error("default role should have '*' pattern")
	}
}

func TestResolveRolesForSystem_DefaultRoleOverride(t *testing.T) {
	definedRoles := map[string]RoleDefinition{
		"default": {
			Description: "Custom default",
			Tools: map[string]ToolPermission{
				"Get*": {},
			},
		},
	}

	roles, err := ResolveRolesForSystem("dev", nil, definedRoles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(roles))
	}
	if roles[0].Description != "Custom default" {
		t.Error("should use user-defined default, not built-in")
	}
}

func TestResolveRolesForSystem_ExplicitRoles(t *testing.T) {
	definedRoles := map[string]RoleDefinition{
		"reader": {
			Tools: map[string]ToolPermission{
				"Get*": {},
			},
		},
	}

	roles, err := ResolveRolesForSystem("prod", []string{"reader"}, definedRoles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(roles) != 1 {
		t.Fatalf("expected 1 role, got %d", len(roles))
	}
}

func TestResolveRolesForSystem_UndefinedRole(t *testing.T) {
	_, err := ResolveRolesForSystem("prod", []string{"nonexistent"}, nil)
	if err == nil {
		t.Fatal("expected error for undefined role")
	}
}

// ---------------------------------------------------------------------------
// ValidateRolesConfig
// ---------------------------------------------------------------------------

func TestValidateRolesConfig_UndefinedNestedRole(t *testing.T) {
	roles := map[string]RoleDefinition{
		"parent": {
			NestedRoles: []string{"nonexistent"},
		},
	}

	warnings := ValidateRolesConfig(roles)
	if len(warnings) == 0 {
		t.Fatal("expected warnings for undefined nested role")
	}
}

func TestValidateRolesConfig_PackageRestrictionWarning(t *testing.T) {
	roles := map[string]RoleDefinition{
		"reader": {
			Tools: map[string]ToolPermission{
				"Get*": {AllowedPackages: []string{"Z*"}},
			},
		},
	}

	warnings := ValidateRolesConfig(roles)
	found := false
	for _, w := range warnings {
		if contains(w, "package-level restrictions not yet implemented") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected package restriction warning")
	}
}

func TestValidateRolesConfig_CycleWarning(t *testing.T) {
	roles := map[string]RoleDefinition{
		"a": {NestedRoles: []string{"b"}},
		"b": {NestedRoles: []string{"a"}},
	}

	warnings := ValidateRolesConfig(roles)
	found := false
	for _, w := range warnings {
		if contains(w, "cyclic") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected cycle warning")
	}
}

func TestValidateRolesConfig_Empty(t *testing.T) {
	warnings := ValidateRolesConfig(nil)
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for empty config, got %d", len(warnings))
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func boolPtr(v bool) *bool {
	return &v
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
