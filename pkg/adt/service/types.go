// Package service contains domain-grouped service interfaces and implementations
// for the ADT client. Each service groups related operations (source, debugger,
// code intelligence, …) behind a focused interface.
//
// Service implementations depend only on transport.Sender (to send requests) and
// on SafetyChecker (for safety policy enforcement). They do NOT import the root
// adt package, which avoids circular dependencies.
package service

import (
	"github.com/oisee/vibing-steampunk/pkg/adt/transport"
)

// SafetyChecker is the interface service implementations use to enforce safety
// policies. The root adt package provides an adapter from *SafetyConfig.
type SafetyChecker interface {
	// CheckOp verifies that an operation type (rune constant, e.g., 'R', 'U') is allowed.
	CheckOp(op rune, opName string) error
	// CheckPkg verifies that operations on the given package are allowed.
	CheckPkg(pkg string) error
	// CheckTransportEdit verifies that editing objects with the given transport is allowed.
	CheckTransportEdit(transport, opName string) error
	// CheckTransport verifies that a transport management operation is allowed.
	CheckTransport(number, opName string, isWrite bool) error
	// IsPkgAllowed checks if a package is in the allowed list.
	IsPkgAllowed(pkg string) bool
}

// Operation type constants that match adt.OperationType rune values.
// They are redefined here so the service package does not import adt.
const (
	OpRead         rune = 'R'
	OpSearch       rune = 'S'
	OpQuery        rune = 'Q'
	OpFreeSQL      rune = 'F'
	OpCreate       rune = 'C'
	OpUpdate       rune = 'U'
	OpDelete       rune = 'D'
	OpActivate     rune = 'A'
	OpTest         rune = 'T'
	OpLock         rune = 'L'
	OpIntelligence rune = 'I'
	OpWorkflow     rune = 'W'
	OpTransport    rune = 'X'
)

// ServiceConfig holds the non-safety configuration values that services need.
// These are simple scalar values extracted from the root adt.Config so the
// service package does not depend on it.
type ServiceConfig struct {
	Username string // SAP username (used by some operations)
	Client   string // SAP client number
	Language string // SAP language
}

// baseService is embedded by all concrete service implementations.
// It provides the shared request sender and safety checker.
type baseService struct {
	sender transport.Sender
	safety SafetyChecker
	config ServiceConfig
}

// checkSafety is a convenience wrapper around safety.CheckOp.
func (b *baseService) checkSafety(op rune, opName string) error {
	if b.safety == nil {
		return nil
	}
	return b.safety.CheckOp(op, opName)
}

// checkPkg is a convenience wrapper around safety.CheckPkg.
func (b *baseService) checkPkg(pkg string) error {
	if b.safety == nil {
		return nil
	}
	return b.safety.CheckPkg(pkg)
}

// checkTransportEdit is a convenience wrapper around safety.CheckTransportEdit.
func (b *baseService) checkTransportEdit(tr, opName string) error {
	if b.safety == nil {
		return nil
	}
	return b.safety.CheckTransportEdit(tr, opName)
}

// NoopSafety is a SafetyChecker that allows everything (no restrictions).
type NoopSafety struct{}

func (NoopSafety) CheckOp(rune, string) error                { return nil }
func (NoopSafety) CheckPkg(string) error                     { return nil }
func (NoopSafety) CheckTransportEdit(string, string) error   { return nil }
func (NoopSafety) CheckTransport(string, string, bool) error { return nil }
func (NoopSafety) IsPkgAllowed(string) bool                  { return true }
