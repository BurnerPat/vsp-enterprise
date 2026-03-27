// Package deps provides embedded dependencies for the vsp binary.
package deps

import (
	_ "embed"
)

// Embedded JCo proxy JAR (shaded, ~6MB). Our code, not SAP proprietary.
// Build with: cd sidecar/jco-proxy && mvn package && cp target/jco-proxy-1.0.0.jar ../../embedded/deps/jco-proxy.jar
//
//go:embed jco-proxy.jar
var JcoProxyJar []byte

// GetEmbeddedProxyJar returns the embedded jco-proxy.jar bytes, or nil if not available.
func GetEmbeddedProxyJar() []byte {
	if len(JcoProxyJar) == 0 {
		return nil
	}
	return JcoProxyJar
}
