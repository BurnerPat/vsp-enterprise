package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ConfigPaths returns the list of paths to search for systems config.
func ConfigPaths() []string {
	paths := []string{
		".vsp.json",         // Current directory (preferred)
		".vsp/systems.json", // Current directory .vsp folder
	}

	// Add home directory paths
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths,
			filepath.Join(home, ".vsp.json"),
			filepath.Join(home, ".vsp", "systems.json"),
		)
	}

	return paths
}

// LoadConfiguration loads systems configuration from the first found config file.
func LoadConfiguration() (*GlobalConfig, string, error) {
	for _, path := range ConfigPaths() {
		if _, err := os.Stat(path); err == nil {
			cfg, err := LoadConfigurationFromFile(path)
			if err != nil {
				return nil, path, err
			}

			return cfg, path, nil
		}
	}
	return nil, "", nil // No config file found (not an error)
}

// LoadConfigurationFromFile loads systems configuration from a specific file.
func LoadConfigurationFromFile(path string) (*GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfgJSON GlobalConfigJSON
	if err := json.Unmarshal(data, &cfgJSON); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	var cfg GlobalConfig
	cfg.GlobalConfigJSON = cfgJSON

	return &cfg, nil
}

// SaveToFile saves the configuration to a file.
func (c *GlobalConfig) SaveToFile(path string) error {
	data, err := json.MarshalIndent(c.GlobalConfigJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// ExampleConfig returns an example configuration for documentation.
func ExampleConfig() string {
	disabled := false
	example := GlobalConfigJSON{
		DefaultSystem: "dev",
		Roles: map[string]RoleDefinition{
			"read_only": {
				Description: "Read-only access to custom packages",
				Tools: map[string]ToolPermission{
					"Get*":    {},
					"Search*": {},
					"List*":   {},
				},
			},
			"cloud_production": {
				Description: "Production cloud-safe role",
				NestedRoles: []string{"read_only"},
				Tools: map[string]ToolPermission{
					"DataPreview": {
						BlockedObjects: []string{"T001", "T000", "USR*"},
					},
					"RunQuery": {Enabled: &disabled},
				},
			},
		},
		Systems: map[string]SystemConfig{
			"dev": {
				ConnectionConfig: ConnectionConfig{
					URL:    "http://dev.example.com:50000",
					User:   "DEVELOPER",
					Client: "001",
				},
				// No roles → uses built-in "default" role (all tools enabled)
			},
			"a4h": {
				ConnectionConfig: ConnectionConfig{
					URL:      "http://a4h.local:50000",
					User:     "ADMIN",
					Client:   "001",
					Insecure: true,
				},
				Roles: []string{"read_only"},
			},
			"prod": {
				ConnectionConfig: ConnectionConfig{
					URL:    "https://prod.example.com:44300",
					User:   "READONLY_USER",
					Client: "100",
				},
				Roles: []string{"cloud_production"},
			},
			"rfc-direct": {
				ConnectionConfig: ConnectionConfig{
					User:   "RFC_USER",
					Client: "001",
				},
				RfcConfig: RfcConfig{
					ConnectionMode: "rfc",
					AsHost:         "sap-app.example.com",
					SysNr:          "00",
				},
			},
		},
	}

	data, _ := json.MarshalIndent(example, "", "  ")
	return string(data)
}
