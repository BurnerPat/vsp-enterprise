// Package mcp provides the MCP server implementation for ABAP ADT tools.
// system.go contains the System struct representing a configured SAP system destination.
// It holds the ADT client, WebSocket clients, sidecar, feature prober, and per-system state.
// System implements the types.System interface consumed by all tool handlers.
package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// System represents a configured destination to an SAP system.
// It holds the connection (ADT client), WebSocket clients, sidecar manager,
// feature prober, and per-system state. System implements types.System.
type System struct {
	adtClient     *adt.Client
	amdpWSClient  *adt.AMDPWebSocketClient // WebSocket-based AMDP client (ZADT_VSP)
	config        *Config                  // Per-system configuration
	featureProber *adt.FeatureProber       // Feature detection system (safety network)
	sidecar       *adt.SidecarManager      // JCo sidecar (RFC mode only)
}

// Ensure System implements types.System at compile time.
var _ types.System = (*System)(nil)

// ADT implements types.System.
func (s *System) ADT() *adt.Client {
	return s.adtClient
}

// IsRfcMode implements types.System.
func (s *System) IsRfcMode() bool {
	return s.sidecar != nil
}

// FeatureProber implements types.System.
func (s *System) FeatureProber() *adt.FeatureProber {
	return s.featureProber
}

// EnsureWSConnected implements types.System.
func (s *System) EnsureWSConnected(ctx context.Context, toolName string) *mcp.CallToolResult {
	return s.ensureWSConnected(ctx, toolName)
}

// Shutdown gracefully stops system-level resources (sidecar, keep-alive, etc.).
func (s *System) Shutdown() {
	if s.sidecar != nil {
		_ = s.sidecar.Stop()
	}
}

// ensureWSConnected ensures the WebSocket client is connected, creating it if needed.
// Returns error result if connection fails, nil on success.
func (s *System) ensureWSConnected(ctx context.Context, toolName string) *mcp.CallToolResult {
	if s.amdpWSClient == nil || !s.amdpWSClient.IsConnected() {
		s.amdpWSClient = adt.NewAMDPWebSocketClient(
			s.config.URL, s.config.Client, s.config.User, s.config.Password, s.config.Insecure,
		)
		if err := s.amdpWSClient.Connect(ctx); err != nil {
			s.amdpWSClient = nil
			return newToolResultError(fmt.Sprintf("%s: WebSocket connect failed: %v", toolName, err))
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// System construction helpers
// ---------------------------------------------------------------------------

// newSystemInstance creates a System with an ADT client, feature prober, and optional sidecar.
// It does NOT create an MCP server or register tools — that is the Server's responsibility.
func newSystemInstance(cfg *Config) (*System, error) {
	opts := cfg.BuildADTOptions()
	adtClient, sidecar, err := createADTClient(cfg, opts)
	if err != nil {
		return nil, err
	}

	// Read global settings from the singleton
	globalCfg := config.GetInstance()

	// Set terminal ID for debugger operations
	if globalCfg.TerminalID != "" {
		adt.SetTerminalID(globalCfg.TerminalID)
	}
	adt.SetTerminalIDUser(cfg.User)

	featureConfig := cfg.BuildFeatureConfig()

	sys := &System{
		adtClient:     adtClient,
		config:        cfg,
		featureProber: adt.NewFeatureProber(adtClient, featureConfig, cfg.IsVerbose()),
		sidecar:       sidecar,
	}

	// Start session keep-alive if configured
	if globalCfg.KeepAliveInterval > 0 {
		adtClient.StartKeepAlive(globalCfg.KeepAliveInterval, cfg.IsVerbose())
	}

	return sys, nil
}

// ---------------------------------------------------------------------------
// ADT client & config builder helpers (package-level, used by newSystemInstance)
// ---------------------------------------------------------------------------

// createADTClient creates an ADT client and optional sidecar based on connection mode.
func createADTClient(cfg *Config, opts []adt.Option) (*adt.Client, *adt.SidecarManager, error) {
	if strings.EqualFold(cfg.ConnectionMode, "rfc") {
		return createRFCADTClient(cfg, opts)
	}
	return adt.NewClient(cfg.URL, cfg.User, cfg.Password, opts...), nil, nil
}

// createRFCADTClient creates an ADT client using RFC mode with a JCo sidecar.
func createRFCADTClient(cfg *Config, opts []adt.Option) (*adt.Client, *adt.SidecarManager, error) {
	adtCfg := adt.NewConfig("", cfg.User, cfg.Password, opts...)

	sidecarCfg := cfg.BuildSidecarConfig()
	sidecar := adt.NewSidecarManager(sidecarCfg)

	if err := sidecar.Start(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("failed to start JCo sidecar: %w", err)
	}

	maxConcurrent := 5
	if g := config.GetInstance(); g.RfcMaxConcurrent > 0 {
		maxConcurrent = g.RfcMaxConcurrent
	}

	var adtClient *adt.Client
	if sidecar.IsSTDIO() {
		stdioTransport := adt.NewStdioRfcTransport(sidecar, adtCfg, maxConcurrent)
		adtClient = adt.NewClientWithTransport(adtCfg, stdioTransport)
	} else {
		rfcTransport := adt.NewRfcTransport(sidecar.URL(), adtCfg, maxConcurrent)
		adtClient = adt.NewClientWithTransport(adtCfg, rfcTransport)
	}
	return adtClient, sidecar, nil
}

// newToolResultError creates an error result for tool execution failures.
func newToolResultError(message string) *mcp.CallToolResult {
	result := mcp.NewToolResultText(message)
	result.IsError = true
	return result
}
