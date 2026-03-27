//go:build !windows

package adt

// findLandscapeFilesFromRegistry is a stub on non-Windows platforms.
// On Linux and macOS, SAP landscape files are found via environment variables
// and well-known default paths, not the Windows registry.
func findLandscapeFilesFromRegistry() []string {
	return nil
}

// sncLibraryScanDirs returns well-known directories where SAP SNC libraries
// are typically installed on Linux and macOS.
func sncLibraryScanDirs() []string {
	return []string{
		"/usr/sap/snc/lib",
		"/usr/local/sap/snc/lib",
		"/opt/sap/snc/lib",
		"/usr/sap/sec",
		"/usr/local/sap/sec",
		"/opt/sap/sec",
	}
}
