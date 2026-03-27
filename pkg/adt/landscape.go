package adt

import (
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// === XML types for SAP UI Landscape format ===
// See: SAP UI Landscape Configuration Guide (sap_ui_landscape_conf_guid.md)

// landscapeXML represents the root <Landscape> element.
type landscapeXML struct {
	XMLName        xml.Name                   `xml:"Landscape"`
	Version        string                     `xml:"version,attr"`
	Messageservers landscapeMessageserversXML `xml:"Messageservers"`
	Routers        landscapeRoutersXML        `xml:"Routers"`
	Services       landscapeServicesXML       `xml:"Services"`
}

type landscapeMessageserversXML struct {
	Items []landscapeMessageserverXML `xml:"Messageserver"`
}

type landscapeMessageserverXML struct {
	UUID        string `xml:"uuid,attr"`
	Name        string `xml:"name,attr"`
	Host        string `xml:"host,attr"`
	Port        string `xml:"port,attr"`
	Description string `xml:"description,attr"`
	RouterID    string `xml:"routerid,attr"`
}

type landscapeRoutersXML struct {
	Items []landscapeRouterXML `xml:"Router"`
}

type landscapeRouterXML struct {
	UUID   string `xml:"uuid,attr"`
	Name   string `xml:"name,attr"`
	Router string `xml:"router,attr"`
}

type landscapeServicesXML struct {
	Items []landscapeServiceXML `xml:"Service"`
}

type landscapeServiceXML struct {
	UUID     string `xml:"uuid,attr"`
	Name     string `xml:"name,attr"`
	Type     string `xml:"type,attr"`
	Mode     string `xml:"mode,attr"`
	MsID     string `xml:"msid,attr"`
	Server   string `xml:"server,attr"`
	RouterID string `xml:"routerid,attr"`
	SystemID string `xml:"systemid,attr"`
	SNCOp    string `xml:"sncop,attr"`
	SNCName  string `xml:"sncname,attr"`
	SNCNoSSO string `xml:"sncnosso,attr"`
	Client   string `xml:"client,attr"`
	User     string `xml:"user,attr"`
	Language string `xml:"language,attr"`
}

// === Parsed landscape types ===

// LandscapeService represents a SAPGUI service entry from the SAP UI Landscape XML.
type LandscapeService struct {
	UUID     string
	Name     string
	SystemID string
	Mode     int    // 0 = load balanced (message server), 1 = direct app server
	Server   string // group name (mode=0) or host:port (mode=1)
	MsID     string // UUID of referenced message server
	RouterID string
	Client   string
	Language string
	SNCOp    int    // SNC quality of protection: 0=off, 1=auth, 2=integrity, 3=encryption, 9=maximum
	SNCName  string // SNC partner name (e.g., "p:CN=SID,O=MyOrg,C=DE")
	SNCNoSSO bool   // If true, SNC connection without SSO (secure connection only)
}

// LandscapeMessageServer represents a message server entry from the landscape XML.
type LandscapeMessageServer struct {
	UUID     string
	Name     string // Usually the SAP System ID (SID)
	Host     string
	Port     int
	RouterID string
}

// LandscapeRouter represents a SAP router entry from the landscape XML.
type LandscapeRouter struct {
	UUID   string
	Name   string
	Router string // Full SAP router string (e.g., "/H/routerhost/S/3299")
}

// Landscape represents a parsed SAP UI Landscape configuration.
type Landscape struct {
	MessageServers map[string]*LandscapeMessageServer // keyed by UUID
	Routers        map[string]*LandscapeRouter        // keyed by UUID
	Services       []*LandscapeService
}

// FindLandscapeFiles locates SAP UI Landscape configuration files on the system.
// It follows the same discovery logic as the SAP ADT Eclipse client
// (SapUiLandscapeReader.java).
//
// Search order:
//  1. Explicit path (--landscape-file / SAP_LANDSCAPE_FILE)
//  2. SAPLOGON_LSXML_FILE environment variable
//  3. Windows registry (Windows only, via SAPLogon registry keys)
//  4. Platform-specific default paths
func FindLandscapeFiles(explicitPath string) []string {
	var files []string

	// 1. Explicit path
	if explicitPath != "" {
		if _, err := os.Stat(explicitPath); err == nil {
			files = append(files, explicitPath)
			if global := landscapeGlobalFilePath(explicitPath); global != "" {
				files = append(files, global)
			}
			return files
		}
		return nil
	}

	// 2. SAPLOGON_LSXML_FILE environment variable (same as SapUiLandscapeReader.java)
	if envPath := os.Getenv("SAPLOGON_LSXML_FILE"); envPath != "" {
		envPath = strings.TrimSpace(envPath)
		if _, err := os.Stat(envPath); err == nil {
			files = append(files, envPath)
			if global := landscapeGlobalFilePath(envPath); global != "" {
				files = append(files, global)
			}
			return files
		}
	}

	// 3. Platform-specific registry / config discovery (Windows registry)
	if regFiles := findLandscapeFilesFromRegistry(); len(regFiles) > 0 {
		return regFiles
	}

	// 4. Default paths based on OS
	home, _ := os.UserHomeDir()
	if home == "" {
		return nil
	}

	var candidates []string
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData != "" {
			candidates = append(candidates,
				filepath.Join(appData, "SAP", "Common", "SAPUILandscape.xml"),
			)
		}
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData != "" {
			// SAP GUI for Java on Windows stores files in LocalAppDataLow\SAPGUI
			candidates = append(candidates,
				filepath.Join(localAppData+"Low", "SAPGUI", "SAPGUILandscape.xml"),
				filepath.Join(localAppData, "SAPGUI", "SAPGUILandscape.xml"),
			)
		}
	case "linux":
		// SAP GUI for Java on Linux
		candidates = append(candidates,
			filepath.Join(home, ".SAPGUI", "SAPGUILandscape.xml"),
			filepath.Join(home, ".SAPGUI", "SAPUILandscape.xml"),
		)
	case "darwin":
		// SAP GUI for Java on macOS
		candidates = append(candidates,
			filepath.Join(home, "Library", "Preferences", "SAP", "SAPGUILandscape.xml"),
			filepath.Join(home, "Library", "Preferences", "SAP", "SAPUILandscape.xml"),
		)
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			files = append(files, candidate)
			if global := landscapeGlobalFilePath(candidate); global != "" {
				files = append(files, global)
			}
			break
		}
	}

	return files
}

// landscapeGlobalFilePath returns the path to the global landscape file if it exists.
// SAP convention: SAPUILandscape.xml has a companion SAPUILandscapeGlobal.xml.
func landscapeGlobalFilePath(localPath string) string {
	base := filepath.Base(localPath)
	var globalBase string
	switch base {
	case "SAPUILandscape.xml":
		globalBase = "SAPUILandscapeGlobal.xml"
	default:
		// SAPGUILandscape.xml uses a settings file for global path, not naming convention
		return ""
	}

	globalPath := filepath.Join(filepath.Dir(localPath), globalBase)
	if _, err := os.Stat(globalPath); err == nil {
		return globalPath
	}
	return ""
}

// ParseLandscapeFile parses a single SAP UI Landscape XML file.
func ParseLandscapeFile(path string) (*Landscape, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading landscape file %s: %w", path, err)
	}

	var lxml landscapeXML
	if err := xml.Unmarshal(data, &lxml); err != nil {
		return nil, fmt.Errorf("parsing landscape XML from %s: %w", path, err)
	}

	l := &Landscape{
		MessageServers: make(map[string]*LandscapeMessageServer),
		Routers:        make(map[string]*LandscapeRouter),
	}

	// Parse message servers
	for _, ms := range lxml.Messageservers.Items {
		port := 0
		if ms.Port != "" {
			port, _ = strconv.Atoi(ms.Port)
		}
		l.MessageServers[ms.UUID] = &LandscapeMessageServer{
			UUID:     ms.UUID,
			Name:     ms.Name,
			Host:     ms.Host,
			Port:     port,
			RouterID: ms.RouterID,
		}
	}

	// Parse routers
	for _, r := range lxml.Routers.Items {
		l.Routers[r.UUID] = &LandscapeRouter{
			UUID:   r.UUID,
			Name:   r.Name,
			Router: r.Router,
		}
	}

	// Parse SAPGUI services only
	for _, svc := range lxml.Services.Items {
		if !strings.EqualFold(svc.Type, "SAPGUI") {
			continue
		}

		mode := 0 // default: load balanced
		if svc.Mode != "" {
			mode, _ = strconv.Atoi(svc.Mode)
		}

		sncOp := 0
		if svc.SNCOp != "" {
			sncOp, _ = strconv.Atoi(svc.SNCOp)
		}

		sncNoSSO := svc.SNCNoSSO == "1" || strings.EqualFold(svc.SNCNoSSO, "true")

		l.Services = append(l.Services, &LandscapeService{
			UUID:     svc.UUID,
			Name:     svc.Name,
			SystemID: svc.SystemID,
			Mode:     mode,
			Server:   svc.Server,
			MsID:     svc.MsID,
			RouterID: svc.RouterID,
			Client:   svc.Client,
			Language: svc.Language,
			SNCOp:    sncOp,
			SNCName:  svc.SNCName,
			SNCNoSSO: sncNoSSO,
		})
	}

	return l, nil
}

// ParseLandscapeFiles parses multiple SAP UI Landscape files and merges them.
// Message servers, routers, and services from all files are combined.
func ParseLandscapeFiles(paths []string) (*Landscape, error) {
	merged := &Landscape{
		MessageServers: make(map[string]*LandscapeMessageServer),
		Routers:        make(map[string]*LandscapeRouter),
	}

	var parseErrors []string
	for _, path := range paths {
		l, err := ParseLandscapeFile(path)
		if err != nil {
			parseErrors = append(parseErrors, err.Error())
			continue
		}
		for k, v := range l.MessageServers {
			merged.MessageServers[k] = v
		}
		for k, v := range l.Routers {
			merged.Routers[k] = v
		}
		merged.Services = append(merged.Services, l.Services...)
	}

	if len(merged.Services) == 0 {
		if len(parseErrors) > 0 {
			return nil, fmt.Errorf("no SAPGUI services found; parse errors: %s", strings.Join(parseErrors, "; "))
		}
		return nil, fmt.Errorf("no SAPGUI services found in landscape files: %v", paths)
	}

	return merged, nil
}

// FindSystemByID finds a SAPGUI service matching the given SAP System ID.
// It searches:
//  1. The systemid attribute on SAPGUI services
//  2. The message server name (for load-balanced services)
//
// If multiple matches exist, it prefers SNC-enabled services.
func (l *Landscape) FindSystemByID(sysID string) (*LandscapeService, error) {
	sysID = strings.ToUpper(strings.TrimSpace(sysID))

	var matches []*LandscapeService
	for _, svc := range l.Services {
		// Match by systemid attribute on the service
		if strings.EqualFold(svc.SystemID, sysID) {
			matches = append(matches, svc)
			continue
		}

		// Match by message server name (for load-balanced services)
		if svc.MsID != "" {
			if ms, ok := l.MessageServers[svc.MsID]; ok {
				if strings.EqualFold(ms.Name, sysID) {
					matches = append(matches, svc)
				}
			}
		}
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("no SAPGUI service found for system ID %q in landscape", sysID)
	}

	// Prefer SNC-enabled service with SSO
	for _, m := range matches {
		if m.SNCOp > 0 && m.SNCName != "" && !m.SNCNoSSO {
			return m, nil
		}
	}

	// Fall back to any SNC-enabled service
	for _, m := range matches {
		if m.SNCOp > 0 && m.SNCName != "" {
			return m, nil
		}
	}

	// Return first match
	return matches[0], nil
}

// ResolveSNCJcoProperties resolves JCo connection properties for SNC SSO
// from the SAP UI Landscape configuration.
//
// It locates the landscape file, finds the system entry for the given System ID,
// validates SNC is configured, and returns the corresponding JCo properties
// suitable for passing to the JCo sidecar proxy.
//
// Parameters:
//   - sysID: Three-character SAP System ID (e.g., "S4H", "ERP")
//   - landscapeFile: Optional explicit path to landscape XML (empty = auto-discover)
//   - defaultClient: Default SAP client number if not specified in landscape
//   - defaultLanguage: Default logon language if not specified in landscape
func ResolveSNCJcoProperties(sysID, landscapeFile, defaultClient, defaultLanguage string) (map[string]string, error) {
	// Find landscape files
	files := FindLandscapeFiles(landscapeFile)
	if len(files) == 0 {
		return nil, fmt.Errorf("no SAP UI Landscape configuration file found; " +
			"use --landscape-file to specify the path or set SAPLOGON_LSXML_FILE environment variable")
	}

	// Parse all landscape files
	landscape, err := ParseLandscapeFiles(files)
	if err != nil {
		return nil, fmt.Errorf("parsing landscape: %w", err)
	}

	// Find the system
	svc, err := landscape.FindSystemByID(sysID)
	if err != nil {
		return nil, err
	}

	// Validate SNC is configured
	if svc.SNCOp <= 0 {
		return nil, fmt.Errorf("system %q found in landscape (%s) but SNC is not configured (sncop=%d)",
			sysID, svc.Name, svc.SNCOp)
	}
	if svc.SNCName == "" {
		return nil, fmt.Errorf("system %q has SNC enabled (sncop=%d) but no SNC partner name (sncname) configured",
			sysID, svc.SNCOp)
	}
	if svc.SNCNoSSO {
		return nil, fmt.Errorf("system %q has SNC SSO explicitly disabled (sncnosso=1); "+
			"only secure connection without SSO is configured", sysID)
	}

	// Build JCo properties map
	props := make(map[string]string)

	// SNC configuration
	props["jco.client.snc_mode"] = "1"
	props["jco.client.snc_partnername"] = svc.SNCName
	if svc.SNCOp > 0 {
		props["jco.client.snc_qop"] = strconv.Itoa(svc.SNCOp)
	}

	// SNC library: locate the cryptographic library on the system.
	// Eclipse and other IDEs may not inherit the full system PATH, causing
	// JCo to fail with "Unable to load GSS-API DLL". Providing the absolute
	// path via snc_lib ensures it works regardless of the parent process.
	if sncLib := findSNCLibrary(); sncLib != "" {
		props["jco.client.snc_lib"] = sncLib
	}

	// Connection parameters based on connection mode
	switch svc.Mode {
	case 0: // Load balanced — use message server
		if svc.MsID == "" {
			return nil, fmt.Errorf("load-balanced service %q has no message server reference", svc.Name)
		}
		ms, ok := landscape.MessageServers[svc.MsID]
		if !ok {
			return nil, fmt.Errorf("message server %q referenced by service %q not found in landscape",
				svc.MsID, svc.Name)
		}
		props["jco.client.mshost"] = ms.Host
		if ms.Port > 0 {
			props["jco.client.msserv"] = strconv.Itoa(ms.Port)
		}
		props["jco.client.r3name"] = ms.Name
		if svc.Server != "" {
			props["jco.client.group"] = svc.Server
		}

		// Router on message server
		if ms.RouterID != "" {
			if router, ok := landscape.Routers[ms.RouterID]; ok && router.Router != "" {
				props["jco.client.saprouter"] = router.Router
			}
		}

	case 1: // Direct connection — parse host:port
		host, sysNr := parseLandscapeServerAddress(svc.Server)
		if host != "" {
			props["jco.client.ashost"] = host
		}
		if sysNr != "" {
			props["jco.client.sysnr"] = sysNr
		}
	}

	// Router on service (overrides message server router)
	if svc.RouterID != "" {
		if router, ok := landscape.Routers[svc.RouterID]; ok && router.Router != "" {
			props["jco.client.saprouter"] = router.Router
		}
	}

	// Client (from landscape, falling back to CLI default)
	client := svc.Client
	if client == "" {
		client = defaultClient
	}
	if client != "" {
		props["jco.client.client"] = client
	}

	// Language (from landscape, falling back to CLI default)
	lang := svc.Language
	if lang == "" {
		lang = defaultLanguage
	}
	if lang != "" {
		props["jco.client.lang"] = lang
	}

	return props, nil
}

// findSNCLibrary attempts to locate the SNC cryptographic library on the system.
// Eclipse and other IDEs may not inherit the system PATH, so the JCo sidecar
// may fail to load the SNC library. This function finds the absolute path
// so it can be passed as jco.client.snc_lib.
//
// Search strategy:
//  1. Check if the library is already on PATH (exec.LookPath)
//  2. Scan well-known SAP installation directories
//
// Returns the absolute path to the library, or "" if not found.
func findSNCLibrary() string {
	// Determine library names based on OS and architecture
	names := sncLibraryNames()
	if len(names) == 0 {
		return ""
	}

	// 1. Check if the library is on PATH
	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			if absPath, err := filepath.Abs(path); err == nil {
				return absPath
			}
			return path
		}
	}

	// 2. Scan well-known SAP installation directories
	for _, dir := range sncLibraryScanDirs() {
		for _, name := range names {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}

	return ""
}

// sncLibraryNames returns the SNC library filenames to look for,
// ordered by preference. Platform-specific.
func sncLibraryNames() []string {
	switch runtime.GOOS {
	case "windows":
		if runtime.GOARCH == "amd64" {
			return []string{"sapcrypto.dll", "sncgss64.dll"}
		}
		return []string{"sapcrypto.dll", "sncgss32.dll"}
	case "linux":
		if runtime.GOARCH == "amd64" {
			return []string{"libsapcrypto.so", "libsncgss64.so"}
		}
		return []string{"libsapcrypto.so", "libsncgss.so"}
	case "darwin":
		return []string{"libsapcrypto.dylib"}
	default:
		return nil
	}
}

// parseLandscapeServerAddress parses a SAP GUI server address "host:port".
// For SAP GUI, ports in the form 32XX indicate system number XX.
func parseLandscapeServerAddress(server string) (host, sysNr string) {
	idx := strings.LastIndex(server, ":")
	if idx < 0 {
		return server, ""
	}
	host = server[:idx]
	port := server[idx+1:]

	// SAP GUI dispatcher port format: 32XX where XX is the system number
	if len(port) == 4 && strings.HasPrefix(port, "32") {
		sysNr = port[2:]
	}
	return host, sysNr
}
