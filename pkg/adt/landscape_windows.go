//go:build windows

package adt

import (
	"os"
	"path/filepath"
	"runtime"

	"golang.org/x/sys/windows/registry"
)

// findLandscapeFilesFromRegistry attempts to locate SAP UI Landscape files
// from the Windows registry, following the same approach as the SAP ADT Eclipse
// client (SapUiLandscapeReader.java).
//
// Search order:
//  1. HKCU\Software\SAP\SAPLogon\LandscapeFilesLastUsed (most reliable)
//  2. HKCU/HKLM\Software\SAP\SAPLogon\Options\PathConfigFilesLocal
func findLandscapeFilesFromRegistry() []string {
	var files []string

	// 1. Try "last used" paths — these reflect what SAP GUI actually loaded
	if local, global := readLastUsedLandscapePaths(); local != "" || global != "" {
		if local != "" {
			files = append(files, local)
		}
		if global != "" {
			files = append(files, global)
		}
		return files
	}

	// 2. Try config directory from registry options
	if localFile := readLandscapeConfigPath(); localFile != "" {
		files = append(files, localFile)
		if global := landscapeGlobalFilePath(localFile); global != "" {
			files = append(files, global)
		}
		return files
	}

	return nil
}

// readLastUsedLandscapePaths reads landscape file paths from
// HKCU\Software\SAP\SAPLogon\LandscapeFilesLastUsed.
// These are stored by SAP GUI and reflect the last successfully used files.
func readLastUsedLandscapePaths() (local, global string) {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\SAP\SAPLogon\LandscapeFilesLastUsed`, registry.QUERY_VALUE)
	if err != nil {
		return "", ""
	}
	defer k.Close()

	// Global file: try multiple registry values in priority order
	for _, name := range []string{"LandscapeFileOnServer", "CoreLandscapeFileOnServer", "LandscapeFileGlobal"} {
		if val, _, err := k.GetStringValue(name); err == nil && val != "" {
			global = val
			break
		}
	}

	// Local file
	if val, _, err := k.GetStringValue("LandscapeFile"); err == nil && val != "" {
		local = val
	}

	return local, global
}

// readLandscapeConfigPath reads the landscape config directory from registry
// and returns the full path to SAPUILandscape.xml.
// Checks HKCU first, then HKLM (both 64-bit and 32-bit paths).
func readLandscapeConfigPath() string {
	regPaths := []struct {
		root registry.Key
		path string
	}{
		{registry.CURRENT_USER, `Software\SAP\SAPLogon\Options`},
		{registry.LOCAL_MACHINE, `Software\SAP\SAPLogon\Options`},
		{registry.LOCAL_MACHINE, `SOFTWARE\Wow6432Node\SAP\SAPLogon\Options`},
	}

	for _, rp := range regPaths {
		k, err := registry.OpenKey(rp.root, rp.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		val, valType, err := k.GetStringValue("PathConfigFilesLocal")
		k.Close()
		if err != nil || val == "" {
			continue
		}

		// Expand environment variables for REG_EXPAND_SZ values (e.g., %APPDATA%)
		if valType == registry.EXPAND_SZ {
			if expanded, err := registry.ExpandString(val); err == nil {
				val = expanded
			}
		}

		localFile := val + `\SAPUILandscape.xml`
		return localFile
	}

	// Also check for global/server landscape file paths
	for _, rp := range regPaths {
		k, err := registry.OpenKey(rp.root, rp.path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		// CoreLandscapeFileOnServer overrides LandscapeFileOnServer
		for _, attrName := range []string{"CoreLandscapeFileOnServer", "LandscapeFileOnServer"} {
			val, valType, err := k.GetStringValue(attrName)
			if err != nil || val == "" {
				continue
			}
			if valType == registry.EXPAND_SZ {
				if expanded, err := registry.ExpandString(val); err == nil {
					val = expanded
				}
			}
			k.Close()
			return val
		}
		k.Close()
	}

	return ""
}

// sncLibraryScanDirs returns well-known Windows directories where SAP SNC
// libraries (sapcrypto.dll / sncgss64.dll) are typically installed.
// Uses ProgramFiles / ProgramFiles(x86) environment variables to handle
// both 64-bit and 32-bit installations.
func sncLibraryScanDirs() []string {
	var dirs []string

	// Determine program files directories
	var programFiles, programFilesX86 string
	if runtime.GOARCH == "amd64" {
		programFiles = os.Getenv("ProgramFiles")         // C:\Program Files
		programFilesX86 = os.Getenv("ProgramFiles(x86)") // C:\Program Files (x86)
	} else {
		programFiles = os.Getenv("ProgramFiles") // On 32-bit this is the x86 path
	}

	// SAP Secure Login Client (most common for SNC/SSO)
	if programFiles != "" {
		dirs = append(dirs,
			filepath.Join(programFiles, "SAP", "FrontEnd", "SecureLogin", "lib"),
			filepath.Join(programFiles, "SAP", "FrontEnd", "SecureLogin"),
		)
	}
	if programFilesX86 != "" {
		dirs = append(dirs,
			filepath.Join(programFilesX86, "SAP", "FrontEnd", "SecureLogin", "lib"),
			filepath.Join(programFilesX86, "SAP", "FrontEnd", "SecureLogin"),
		)
	}

	// SAP CommonCryptoLib (standalone installation)
	if programFiles != "" {
		dirs = append(dirs,
			filepath.Join(programFiles, "SAP", "CommonCryptoLib"),
		)
	}
	if programFilesX86 != "" {
		dirs = append(dirs,
			filepath.Join(programFilesX86, "SAP", "CommonCryptoLib"),
		)
	}

	// SAP GUI installation
	if programFiles != "" {
		dirs = append(dirs,
			filepath.Join(programFiles, "SAP", "FrontEnd", "SAPgui"),
		)
	}
	if programFilesX86 != "" {
		dirs = append(dirs,
			filepath.Join(programFilesX86, "SAP", "FrontEnd", "SAPgui"),
		)
	}

	// System32 / SysWOW64
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot != "" {
		dirs = append(dirs,
			filepath.Join(systemRoot, "System32"),
		)
		if runtime.GOARCH == "amd64" {
			dirs = append(dirs,
				filepath.Join(systemRoot, "SysWOW64"),
			)
		}
	}

	return dirs
}
