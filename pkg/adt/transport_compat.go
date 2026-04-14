package adt

import "github.com/oisee/vibing-steampunk/pkg/adt/transport"

// --------------------------------------------------------------------------
// Backward-compatible re-exports from transport package.
//
// Existing code using adt.APIError, adt.IsNotFoundError, etc. continues to
// work unchanged. New code should import from pkg/adt/transport directly.
// --------------------------------------------------------------------------

// TransportAPIError is the transport-layer API error type.
// The legacy adt.APIError in http.go remains for backward compatibility.
// New code should use transport.APIError.
type TransportAPIError = transport.APIError

// AdtRequest is re-exported for convenience so callers don't need a separate import.
type AdtRequest = transport.AdtRequest

// AdtResponse is re-exported for convenience.
type AdtResponse = transport.AdtResponse

// AdtConnection is re-exported for convenience.
type AdtConnection = transport.AdtConnection

// HttpConnectionConfig is re-exported for convenience.
type HttpConnectionConfig = transport.HttpConnectionConfig

// JcoConnectionConfig is re-exported for convenience.
type JcoConnectionConfig = transport.JcoConnectionConfig
