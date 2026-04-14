package service

import (
	"github.com/oisee/vibing-steampunk/pkg/adt/transport"
)

// --------------------------------------------------------------------------
// DevToolsService — syntax check, activation, unit tests, ATC, formatting
// --------------------------------------------------------------------------

// DevToolsService groups development-tool operations.
type DevToolsService interface {
	// TODO: migrate interfaces for SyntaxCheck, Activate, ActivatePackage,
	// ActivatePackageIterative, GetInactiveObjects, RunUnitTests,
	// RunATCCheck, GetATCCustomizing, GetATCCheckVariant, CreateATCRun,
	// GetATCWorklist, PrettyPrint, GetPrettyPrinterSettings, SetPrettyPrinterSettings
}

type devToolsService struct{ baseService }

func NewDevToolsService(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) DevToolsService {
	return &devToolsService{baseService{sender: sender, safety: safety, config: cfg}}
}

// --------------------------------------------------------------------------
// CrudService — lock, unlock, create, update, delete ABAP objects
// --------------------------------------------------------------------------

// CrudService groups CRUD lifecycle operations on ABAP objects.
type CrudService interface {
	// TODO: migrate interfaces for LockObject, UnlockObject, UpdateSource,
	// CreateObject, DeleteObject, CreateTestInclude, GetClassInclude,
	// UpdateClassInclude, PublishServiceBinding, UnpublishServiceBinding, CreateTable
}

type crudService struct{ baseService }

func NewCrudService(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) CrudService {
	return &crudService{baseService{sender: sender, safety: safety, config: cfg}}
}

// --------------------------------------------------------------------------
// CodeIntelService — navigation, completion, references, hierarchy
// --------------------------------------------------------------------------

// CodeIntelService groups code-intelligence operations.
type CodeIntelService interface {
	// TODO: migrate interfaces for FindDefinition, FindReferences,
	// CodeCompletion, CodeCompletionFull, GetClassComponents, GetTypeHierarchy
}

type codeIntelService struct{ baseService }

func NewCodeIntelService(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) CodeIntelService {
	return &codeIntelService{baseService{sender: sender, safety: safety, config: cfg}}
}

// --------------------------------------------------------------------------
// DebuggerService — breakpoints, listen, attach, step, variables
// --------------------------------------------------------------------------

// DebuggerService groups debugger operations.
type DebuggerService interface {
	// TODO: migrate interfaces for SetExternalBreakpoint, GetExternalBreakpoints,
	// DeleteExternalBreakpoint, DeleteAllExternalBreakpoints,
	// ValidateBreakpointCondition, DebuggerListen, DebuggerCheckListener,
	// DebuggerStopListener, DebuggerAttach, DebuggerDetach, DebuggerResetSession,
	// DebuggerStep, DebuggerGetStack, DebuggerGetVariables,
	// DebuggerGetChildVariables, DebuggerSetVariableValue, DebuggerGoToStack,
	// DebuggerBatchRequest, DebuggerStepWithBatch
}

type debuggerService struct{ baseService }

func NewDebuggerService(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) DebuggerService {
	return &debuggerService{baseService{sender: sender, safety: safety, config: cfg}}
}

// --------------------------------------------------------------------------
// TransportService — CTS transport request management
// --------------------------------------------------------------------------

// TransportService groups CTS transport-management operations.
type TransportService interface {
	// TODO: migrate interfaces for GetUserTransports, ListTransports,
	// GetTransport, GetTransportInfo, CreateTransport, CreateTransportV2,
	// ReleaseTransport, ReleaseTransportV2, DeleteTransport
}

type transportService struct{ baseService }

func NewTransportService(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) TransportService {
	return &transportService{baseService{sender: sender, safety: safety, config: cfg}}
}

// --------------------------------------------------------------------------
// AnalysisService — call graph, object explorer, CDS dependencies
// --------------------------------------------------------------------------

// AnalysisService groups code-analysis and dependency operations.
type AnalysisService interface {
	// TODO: migrate interfaces for GetCallGraph, GetCallersOf, GetCalleesOf,
	// GetObjectStructureCAI, GetObjectChildren, GetObjectEntryPoints,
	// TraceExecution, GetCDSDependencies
}

type analysisService struct{ baseService }

func NewAnalysisService(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) AnalysisService {
	return &analysisService{baseService{sender: sender, safety: safety, config: cfg}}
}

// --------------------------------------------------------------------------
// TraceService — runtime traces, dumps, SQL traces
// --------------------------------------------------------------------------

// TraceService groups trace and dump operations.
type TraceService interface {
	// TODO: migrate interfaces for ListTraces, GetTrace, GetDumps, GetDump,
	// GetSQLTraceState, ListSQLTraces
}

type traceService struct{ baseService }

func NewTraceService(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) TraceService {
	return &traceService{baseService{sender: sender, safety: safety, config: cfg}}
}

// --------------------------------------------------------------------------
// UI5Service — BSP/Fiori application management
// --------------------------------------------------------------------------

// UI5Service groups UI5/Fiori BSP operations.
type UI5Service interface {
	// TODO: migrate interfaces for UI5ListApps, UI5GetApp, UI5GetFileContent,
	// UI5UploadFile, UI5DeleteFile, UI5CreateApp, UI5DeleteApp
}

type ui5Service struct{ baseService }

func NewUI5Service(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) UI5Service {
	return &ui5Service{baseService{sender: sender, safety: safety, config: cfg}}
}

// --------------------------------------------------------------------------
// WorkflowService — composite write operations, grep, execute, deploy
// --------------------------------------------------------------------------

// WorkflowService groups high-level composite operations.
type WorkflowService interface {
	// TODO: migrate interfaces for WriteProgram, WriteClass,
	// CreateAndActivateProgram, CreateClassWithTests, EditSourceWithOptions,
	// GrepObject, GrepPackage, GrepPackages, ExecuteABAP, ExecuteABAPMultiple,
	// CreateFromFile, UpdateFromFile, DeployFromFile, RenameObject,
	// SaveToFile, SaveClassIncludeToFile, GetSource, WriteSource,
	// CompareSource, CloneObject, GetClassInfo
}

type workflowService struct{ baseService }

func NewWorkflowService(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) WorkflowService {
	return &workflowService{baseService{sender: sender, safety: safety, config: cfg}}
}
