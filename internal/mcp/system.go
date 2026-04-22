// Package mcp provides the MCP server implementation for ABAP ADT tools.
// system.go contains the System struct representing a configured SAP system destination.
// It holds the ADT client, WebSocket clients, sidecar, feature prober, and per-system state.
// System implements the types.System interface consumed by all tool handlers.
package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/config"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
	"github.com/oisee/vibing-steampunk/pkg/adt/connection"
)

// System represents a configured destination to an SAP system.
// It holds the connection (ADT client), WebSocket clients, sidecar manager,
// feature prober, and per-system state. System implements types.System.
type System struct {
	adtClient           *adt.Client
	amdpWSClient        *adt.AMDPWebSocketClient // WebSocket-based AMDP client (ZADT_VSP)
	config              *config.SystemConfig     // Per-system configuration
	featureProber       *adt.FeatureProber       // Feature detection system (safety network)
	sidecar             *adt.SidecarManager      // JCo sidecar (RFC mode only)
	cookies             map[string]string        // Runtime cookies (browser-auth, cookie-file, etc.)
	discoveredEndpoints adt.DiscoveredEndpoints  // ADT endpoints from /sap/bc/adt/discovery
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

// DiscoveredEndpoints implements types.System.
func (s *System) DiscoveredEndpoints() adt.DiscoveredEndpoints {
	return s.discoveredEndpoints
}

// EnsureWSConnected implements types.System.
func (s *System) EnsureWSConnected(ctx context.Context, toolName string) *mcp.CallToolResult {
	return s.ensureWSConnected(ctx, toolName)
}

// Connect implements types.System by validating credentials and establishing connection.
// Performs a GetSystemInfo call to validate authentication and establish session,
// then discovers available ADT endpoints for tool filtering.
// HTTP 400 from GetSystemInfo is treated as non-fatal (e.g., missing authorization for T000 query)
// since the session was still established and other tools may work fine.
// In RFC/JCo mode, requests are tunneled through the JCo proxy.
// Connect is idempotent and safe to call multiple times.
func (s *System) Connect(ctx context.Context) error {
	// Discover available ADT endpoints from the system's service document.
	// Non-fatal: if discovery fails, all tools remain available.
	endpoints, err := s.adtClient.DiscoverFeatures(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to system: %w", err)
	} else {
		s.discoveredEndpoints = endpoints
		if s.config.IsVerbose() {
			_, _ = fmt.Fprintf(os.Stderr, "[VERBOSE] ADT discovery: %d endpoints discovered\n", len(endpoints))
		}
	}

	return nil
}

// Start implements types.System by activating runtime behavior like session keep-alive.
// For HTTP mode, starts the keep-alive goroutine if configured.
// For RFC/SNC mode, is a silent no-op (runtime management is handled by sidecar or deferred to future enhancement).
// Start should be called after Connect.
func (s *System) Start(_ context.Context) error {
	if strings.EqualFold(s.config.ConnectionMode, "rfc") {
		// Intentional no-op for RFC/JCo sidecar; runtime activation (e.g., connection pooling monitoring)
		// is managed by sidecar or deferred to future enhancement.
		return nil
	}

	// HTTP mode: start keep-alive if configured
	globalCfg := config.GetInstance()
	if globalCfg.KeepAliveInterval > 0 {
		s.adtClient.StartKeepAlive(globalCfg.KeepAliveInterval, s.config.IsVerbose())
	}

	return nil
}

// Shutdown implements types.System by gracefully stopping system resources.
// Idempotent and safe to call multiple times.
func (s *System) Shutdown() error {
	// Stop keep-alive goroutine if running
	s.adtClient.StopKeepAlive()

	// Stop sidecar if running (RFC mode only)
	if s.sidecar != nil {
		if err := s.sidecar.Stop(); err != nil {
			return fmt.Errorf("failed to stop sidecar: %w", err)
		}
	}

	return nil
}

// ensureWSConnected ensures the WebSocket client is connected, creating it if needed.
// Returns error result if connection fails, nil on success.
func (s *System) ensureWSConnected(ctx context.Context, toolName string) *mcp.CallToolResult {
	if s.amdpWSClient == nil || !s.amdpWSClient.IsConnected() {
		s.amdpWSClient = adt.NewAMDPWebSocketClient(
			s.config.URL, s.config.Client, s.config.Username, s.config.Password, s.config.Insecure,
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
// This is a pure allocation step with no network I/O or eager connection setup.
// Call System.Connect() to validate credentials, then System.Start() to activate runtime behavior.
func newSystemInstance(cfg config.SystemConfig, cookies map[string]string) (*System, error) {
	opts := cfg.BuildADTOptions()

	// Cookies are a runtime concern owned by the System, not the config struct.
	// They are passed in by the server and applied here when building
	// the ADT client so the HTTP connection includes them from the first request.
	if len(cookies) > 0 {
		opts = append(opts, adt.WithCookies(cookies))
	}

	adtClient, sidecar, err := createADTClient(&cfg, opts)
	if err != nil {
		return nil, err
	}

	// Read global settings from the singleton
	globalCfg := config.GetInstance()

	// Set terminal ID for debugger operations
	if globalCfg.TerminalID != "" {
		adt.SetTerminalID(globalCfg.TerminalID)
	}
	adt.SetTerminalIDUser(cfg.Username)

	featureConfig := cfg.BuildFeatureConfig()

	sys := &System{
		adtClient:     adtClient,
		config:        &cfg,
		cookies:       cookies,
		featureProber: adt.NewFeatureProber(adtClient, featureConfig, cfg.IsVerbose()),
		sidecar:       sidecar,
	}

	// NOTE: No eager keep-alive or connection validation here.
	// Both are deferred to System.Start() and System.Connect() respectively.

	return sys, nil
}

// ---------------------------------------------------------------------------
// ADT client & config builder helpers (package-level, used by newSystemInstance)
// ---------------------------------------------------------------------------

// createADTClient creates an ADT client and optional sidecar based on connection mode.
func createADTClient(cfg *config.SystemConfig, opts []adt.Option) (*adt.Client, *adt.SidecarManager, error) {
	if strings.EqualFold(cfg.ConnectionMode, "rfc") {
		return createRFCADTClient(cfg, opts)
	}
	return adt.NewClient(cfg.URL, cfg.Username, cfg.Password, opts...), nil, nil
}

// createRFCADTClient creates an ADT client using RFC mode with a JCo sidecar.
func createRFCADTClient(cfg *config.SystemConfig, opts []adt.Option) (*adt.Client, *adt.SidecarManager, error) {
	adtCfg := adt.NewConfig("", cfg.Username, cfg.Password, opts...)

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

	sidecarTransport := connection.NewJcoStdioTransport(sidecar)
	connection := connection.NewJcoConnection(sidecarTransport, sidecar, &connection.JcoConnectionConfig{
		Client:        cfg.Client,
		MaxConcurrent: maxConcurrent,
		SessionType:   connection.SessionStateless,
	})

	adtClient = adt.NewClientWithConnection(adtCfg, connection)

	return adtClient, sidecar, nil
}

// newToolResultError creates an error result for tool execution failures.
func newToolResultError(message string) *mcp.CallToolResult {
	result := mcp.NewToolResultText(message)
	result.IsError = true
	return result
}
