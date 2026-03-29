package mcp

// SystemConfigResolved is a configuration for a single SAP system.
type SystemConfigResolved struct {
	BaseURL            string
	Username           string
	Password           string
	Client             string
	Language           string
	InsecureSkipVerify bool
	Cookies            map[string]string
	ReadOnly           bool
	AllowedPackages    []string
	ConnectionMode     string
	AsHost             string
	SysNr              string
	MsHost             string
	MsServ             string
	R3Name             string
	Group              string
	JcoProxyJar        string
	JavaPath           string
	RfcProxyPort       int
	RfcMaxConcurrent   int
	SidecarTransport   string
	SNC                bool
	SysID              string
	LandscapeFile      string
	JcoProperties      map[string]string
	Verbose            bool
	BrowserAuth        bool
	BrowserAuthTimeout string
	BrowserExec        string
	CookieSave         string
}

// systemConfigForMCP converts a config.SystemConfig to an mcp.Config for creating per-system servers.
func systemConfigForMCP(sysID string, sysCfg *SystemConfigResolved, globalCfg *Config) *Config {
	c := &Config{
		// Per-system connection settings
		BaseURL:            sysCfg.BaseURL,
		Username:           sysCfg.Username,
		Password:           sysCfg.Password,
		Client:             sysCfg.Client,
		Language:           sysCfg.Language,
		InsecureSkipVerify: sysCfg.InsecureSkipVerify,
		Cookies:            sysCfg.Cookies,

		// Per-system safety settings
		ReadOnly:        sysCfg.ReadOnly,
		AllowedPackages: sysCfg.AllowedPackages,

		// Per-system RFC settings
		ConnectionMode:   sysCfg.ConnectionMode,
		AsHost:           sysCfg.AsHost,
		SysNr:            sysCfg.SysNr,
		MsHost:           sysCfg.MsHost,
		MsServ:           sysCfg.MsServ,
		R3Name:           sysCfg.R3Name,
		Group:            sysCfg.Group,
		JcoProxyJar:      sysCfg.JcoProxyJar,
		JcoLibsDir:       globalCfg.JcoLibsDir,
		JavaPath:         sysCfg.JavaPath,
		RfcProxyPort:     sysCfg.RfcProxyPort,
		RfcMaxConcurrent: sysCfg.RfcMaxConcurrent,
		SidecarTransport: sysCfg.SidecarTransport,

		// Per-system SNC settings
		SNC:           sysCfg.SNC,
		SysID:         sysCfg.SysID,
		LandscapeFile: sysCfg.LandscapeFile,
		JcoProperties: sysCfg.JcoProperties,

		// Global settings (shared across all systems)
		Verbose:        globalCfg.Verbose || sysCfg.Verbose,
		Mode:           globalCfg.Mode,
		DisabledGroups: globalCfg.DisabledGroups,
		ToolsConfig:    globalCfg.ToolsConfig,

		// Global safety settings (can be overridden per-system)
		BlockFreeSQL:            globalCfg.BlockFreeSQL,
		AllowedOps:              globalCfg.AllowedOps,
		DisallowedOps:           globalCfg.DisallowedOps,
		EnableTransports:        globalCfg.EnableTransports,
		TransportReadOnly:       globalCfg.TransportReadOnly,
		AllowedTransports:       globalCfg.AllowedTransports,
		AllowTransportableEdits: globalCfg.AllowTransportableEdits,

		// Global feature settings
		FeatureHANA:      globalCfg.FeatureHANA,
		FeatureAbapGit:   globalCfg.FeatureAbapGit,
		FeatureRAP:       globalCfg.FeatureRAP,
		FeatureAMDP:      globalCfg.FeatureAMDP,
		FeatureUI5:       globalCfg.FeatureUI5,
		FeatureTransport: globalCfg.FeatureTransport,

		// Global debugger settings
		TerminalID: globalCfg.TerminalID,

		// Global keep-alive (applies to all systems)
		KeepAliveInterval: globalCfg.KeepAliveInterval,
	}

	return c
}
