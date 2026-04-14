package adt

import (
	"context"

	"github.com/oisee/vibing-steampunk/pkg/adt/service"
	"github.com/oisee/vibing-steampunk/pkg/adt/transport"
)

// AdtClient is the primary interface for interacting with an SAP ADT backend.
// It provides:
//   - Connect / Close for lifecycle management
//   - SendRequest for low-level request dispatch
//   - Domain-grouped service accessors (Source, DevTools, …)
//
// The concrete implementation is *Client (in client.go).
type AdtClient interface {
	// Connect eagerly validates credentials and establishes the session.
	// For HTTP connections this fetches a CSRF token; for JCo it is currently a no-op.
	Connect(ctx context.Context) error

	// Close releases all resources: stops the keep-alive goroutine, closes the
	// underlying AdtConnection (which in turn stops the JCo sidecar if applicable).
	Close() error

	// SendRequest dispatches a single request through the underlying connection.
	// Most callers should use the service accessors instead.
	SendRequest(ctx context.Context, req *transport.AdtRequest) (*transport.AdtResponse, error)

	// Connection returns the underlying AdtConnection.
	// Useful for advanced callers that need transport-level access.
	Connection() transport.AdtConnection

	// GetConfig returns the client configuration.
	GetConfig() *Config

	// Safety returns the safety configuration.
	Safety() *SafetyConfig

	// --- Service Accessors ---

	// Source returns the service for reading ABAP source objects.
	Source() service.SourceService
	// System returns the service for system-level operations.
	System() service.SystemService
	// DevTools returns the service for development tools (syntax check, activation, …).
	DevTools() service.DevToolsService
	// Crud returns the service for CRUD lifecycle operations.
	Crud() service.CrudService
	// CodeIntel returns the service for code-intelligence operations.
	CodeIntel() service.CodeIntelService
	// Debugger returns the service for debugger operations.
	Debugger() service.DebuggerService
	// Transport returns the service for CTS transport management.
	Transport() service.TransportService
	// Analysis returns the service for code analysis and dependency operations.
	Analysis() service.AnalysisService
	// Trace returns the service for runtime traces and dumps.
	Trace() service.TraceService
	// UI5 returns the service for UI5/Fiori BSP management.
	UI5() service.UI5Service
	// Workflow returns the service for composite write operations.
	Workflow() service.WorkflowService
}

// safetyCheckerAdapter adapts *SafetyConfig to the service.SafetyChecker interface.
// This bridges the root adt package types with the service sub-package without
// the service package needing to import adt.
type safetyCheckerAdapter struct {
	s *SafetyConfig
}

var _ service.SafetyChecker = (*safetyCheckerAdapter)(nil)

func (a *safetyCheckerAdapter) CheckOp(op rune, opName string) error {
	return a.s.CheckOperation(OperationType(op), opName)
}

func (a *safetyCheckerAdapter) CheckPkg(pkg string) error {
	return a.s.CheckPackage(pkg)
}

func (a *safetyCheckerAdapter) CheckTransportEdit(tr, opName string) error {
	return a.s.CheckTransportableEdit(tr, opName)
}

func (a *safetyCheckerAdapter) CheckTransport(number, opName string, isWrite bool) error {
	return a.s.CheckTransport(number, opName, isWrite)
}

func (a *safetyCheckerAdapter) IsPkgAllowed(pkg string) bool {
	return a.s.IsPackageAllowed(pkg)
}

// NewSafetyChecker wraps a *SafetyConfig into a service.SafetyChecker.
func NewSafetyChecker(s *SafetyConfig) service.SafetyChecker {
	if s == nil {
		return service.NoopSafety{}
	}
	return &safetyCheckerAdapter{s: s}
}

// HttpConfigFromConfig converts an adt.Config into a transport.HttpConnectionConfig.
func HttpConfigFromConfig(cfg *Config) *transport.HttpConnectionConfig {
	return &transport.HttpConnectionConfig{
		BaseURL:            cfg.BaseURL,
		Username:           cfg.Username,
		Password:           cfg.Password,
		Client:             cfg.Client,
		Language:           cfg.Language,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
		SessionType:        transport.SessionType(cfg.SessionType),
		Timeout:            cfg.Timeout,
		Cookies:            cfg.Cookies,
	}
}

// ServiceConfigFromConfig converts an adt.Config into a service.ServiceConfig.
func ServiceConfigFromConfig(cfg *Config) service.ServiceConfig {
	return service.ServiceConfig{
		Username: cfg.Username,
		Client:   cfg.Client,
		Language: cfg.Language,
	}
}
