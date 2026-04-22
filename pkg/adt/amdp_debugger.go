package adt

// AMDP Debugger types.

// AMDPDebugSession represents an AMDP debug session.
type AMDPDebugSession struct {
	MainID      string `json:"mainId"`
	HANASession string `json:"hanaSession,omitempty"`
	User        string `json:"user,omitempty"`
	CascadeMode string `json:"cascadeMode,omitempty"` // "NONE" or "FULL"
}

// AMDPDebugResponse represents the response from AMDP debugger operations.
type AMDPDebugResponse struct {
	Kind        string           `json:"kind"` // on_break, on_toggle_breakpoints, on_execution_end, etc.
	Position    *AMDPPosition    `json:"position,omitempty"`
	CallStack   []AMDPStackFrame `json:"callStack,omitempty"`
	Variables   []AMDPVariable   `json:"variables,omitempty"`
	Breakpoints []AMDPBreakpoint `json:"breakpoints,omitempty"`
	Message     string           `json:"message,omitempty"`
}

// AMDPPosition represents a position in AMDP source code.
type AMDPPosition struct {
	ObjectName string `json:"objectName"`
	Line       int    `json:"line"`
	Column     int    `json:"column,omitempty"`
}

// AMDPStackFrame represents a frame in the AMDP call stack.
type AMDPStackFrame struct {
	Name     string       `json:"name"`
	Position AMDPPosition `json:"position"`
	Level    int          `json:"level"`
}

// AMDPVariable represents a variable in AMDP debugging.
type AMDPVariable struct {
	Name  string `json:"name"`
	Type  string `json:"type"` // scalar, table, array
	Value string `json:"value,omitempty"`
	ID    string `json:"id,omitempty"`
	Rows  int    `json:"rows,omitempty"` // for table types
}

// AMDPBreakpoint represents an AMDP breakpoint.
type AMDPBreakpoint struct {
	ID         string `json:"id"`
	ObjectName string `json:"objectName"`
	Line       int    `json:"line"`
	Enabled    bool   `json:"enabled"`
}
