// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_fileio.go contains handlers for file-based deployment operations.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// fileToolDefs returns tool definitions for file-based deployment tools.
func (s *Server) fileToolDefs() []toolDef {
	defs := []toolDef{
		{tool: mcp.NewTool("DeployFromFile",
			mcp.WithDescription("✅ RECOMMENDED - Smart deploy from file: auto-detects if object exists and creates/updates accordingly. Solves token limit problem for large generated files (ML models, 3948+ lines). Example: DeployFromFile(file_path=\"/path/to/zcl_ml_iris.clas.abap\", package_name=\"$ZAML_IRIS\") deploys any size file. Workflow: Parse → Check existence → Create or Update → Lock → SyntaxCheck → Write → Unlock → Activate. Supports .clas.abap, .prog.abap, .intf.abap, .fugr.abap, .func.abap. Use this for all file-based deployments."),
			mcp.WithString("file_path", mcp.Required(), mcp.Description("Absolute path to ABAP source file")),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (required for new objects, e.g., $ZAML_IRIS)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleDeployFromFile},

		{tool: mcp.NewTool("SaveToFile",
			mcp.WithDescription("Save ABAP object source to local file (SAP → File). Enables BIDIRECTIONAL SYNC WORKFLOW: (1) SaveToFile downloads object from SAP, (2) edit locally with vim/VS Code/AI assistants, (3) DeployFromFile uploads changes back to SAP. Example: SaveToFile(objType=\"CLAS/OC\", objectName=\"ZCL_ML_IRIS\", outputPath=\"./src/\") creates ./src/zcl_ml_iris.clas.abap. Then edit locally and use DeployFromFile to sync back. Recommended for iterative development. Auto-determines file extension."),
			mcp.WithString("objType", mcp.Required(), mcp.Description("Object type: CLAS/OC (class), PROG/P (program), INTF/OI (interface), FUGR/F (function group), FUGR/FF (function module)")),
			mcp.WithString("objectName", mcp.Required(), mcp.Description("Object name (e.g., ZCL_ML_IRIS, ZAML_IRIS_DEMO)")),
			mcp.WithString("outputPath", mcp.Description("Output file path or directory. If directory, filename is auto-generated with correct extension. If omitted, saves to current directory.")),
		), handler: s.handleSaveToFile, readOnly: true},

		{tool: mcp.NewTool("RenameObject",
			mcp.WithDescription("Rename ABAP object by creating copy with new name and deleting old one. Useful for fixing naming conventions. Workflow: GetSource → Replace names → CreateNew → ActivateNew → DeleteOld"),
			mcp.WithString("objType", mcp.Required(), mcp.Description("Object type: CLAS/OC (class), PROG/P (program), INTF/OI (interface), FUGR/F (function group)")),
			mcp.WithString("oldName", mcp.Required(), mcp.Description("Current object name")),
			mcp.WithString("newName", mcp.Required(), mcp.Description("New object name")),
			mcp.WithString("packageName", mcp.Required(), mcp.Description("Package name for new object (e.g., $ZAML_IRIS)")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleRenameObject},
	}
	// Append ImportFromFile/ExportToFile from tool_source.go
	defs = append(defs, s.fileSourceToolDefs()...)
	return defs
}

// editToolDefs returns tool definitions for surgical edit tools.
func (s *Server) editToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("EditSource",
			mcp.WithDescription("Surgical string replacement on ABAP source code. Matches the Edit tool pattern for local files. Workflow: GetSource → FindReplace → SyntaxCheck → Lock → Update → Unlock → Activate. Example: EditSource(object_url=\"/sap/bc/adt/programs/programs/ZTEST\", old_string=\"METHOD foo.\\n  ENDMETHOD.\", new_string=\"METHOD foo.\\n  rv_result = 42.\\n  ENDMETHOD.\", replace_all=false, syntax_check=true). Requires unique match if replace_all=false. Use this for incremental edits between syntax checks - no need to download/upload full source!"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of object (e.g., /sap/bc/adt/programs/programs/ZTEST, /sap/bc/adt/oo/classes/zcl_test)")),
			mcp.WithString("old_string", mcp.Required(), mcp.Description("Exact string to find and replace. Must be unique in source if replace_all=false. Include enough context (surrounding lines) to ensure uniqueness.")),
			mcp.WithString("new_string", mcp.Required(), mcp.Description("Replacement string. Can be multiline (use \\n). Length can differ from old_string.")),
			mcp.WithBoolean("replace_all", mcp.Description("If true, replace all occurrences. If false (default), require unique match. Default: false")),
			mcp.WithBoolean("syntax_check", mcp.Description("If true (default), validate syntax before saving. If syntax errors found, changes are NOT saved. Default: true")),
			mcp.WithBoolean("case_insensitive", mcp.Description("If true, ignore case when matching old_string. Useful for renaming variables regardless of case. Default: false")),
			mcp.WithString("method", mcp.Description("For CLAS only: constrain search/replace to this method only. Prevents accidental edits in other methods. (optional)")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for objects not in $TMP package)")),
		), handler: s.handleEditSource, focused: true},
	}
}

// routeFileIOAction routes file I/O operations.
func (s *Server) routeFileIOAction(ctx context.Context, action, objectType, objectName string, params map[string]any) (*mcp.CallToolResult, bool, error) {
	if action == "system" || action == "edit" {
		fileType := getStringParam(params, "type")
		switch fileType {
		case "deploy_from_file":
			return s.callHandler(ctx, s.handleDeployFromFile, params)
		case "save_to_file":
			return s.callHandler(ctx, s.handleSaveToFile, params)
		case "rename":
			return s.callHandler(ctx, s.handleRenameObject, params)
		}
	}
	return nil, false, nil
}

// --- File-Based Deployment Handlers ---

// Note: CreateFromFile and UpdateFromFile handlers removed - use DeployFromFile instead

func (s *Server) handleDeployFromFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	filePath, ok := request.Params.Arguments["file_path"].(string)
	if !ok || filePath == "" {
		return newToolResultError("file_path is required"), nil
	}

	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	result, err := s.adtClient.DeployFromFile(ctx, filePath, packageName, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("DeployFromFile failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleSaveToFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// Support both old (objType/objectName/outputPath) and new (object_type/object_name/output_dir) parameter names
	objTypeStr, ok := request.Params.Arguments["objType"].(string)
	if !ok || objTypeStr == "" {
		objTypeStr, ok = request.Params.Arguments["object_type"].(string)
		if !ok || objTypeStr == "" {
			return newToolResultError("object_type is required (e.g., PROG, CLAS, INTF, FUGR, FUNC, DDLS, BDEF, SRVD)"), nil
		}
	}

	objectName, ok := request.Params.Arguments["objectName"].(string)
	if !ok || objectName == "" {
		objectName, ok = request.Params.Arguments["object_name"].(string)
		if !ok || objectName == "" {
			return newToolResultError("object_name is required"), nil
		}
	}

	outputPath := ""
	if p, ok := request.Params.Arguments["outputPath"].(string); ok {
		outputPath = p
	} else if p, ok := request.Params.Arguments["output_dir"].(string); ok {
		outputPath = p
	}

	// Check for include parameter (for class includes)
	includeStr := ""
	if inc, ok := request.Params.Arguments["include"].(string); ok {
		includeStr = strings.ToLower(inc)
	}

	// Check for parent/function_group parameter (required for FUNC type)
	parentName := ""
	if p, ok := request.Params.Arguments["parent"].(string); ok {
		parentName = p
	} else if p, ok := request.Params.Arguments["function_group"].(string); ok {
		parentName = p
	} else if p, ok := request.Params.Arguments["parentName"].(string); ok {
		parentName = p
	}

	// Parse object type - support both short (PROG) and full (PROG/P) format
	var objType adt.CreatableObjectType
	switch strings.ToUpper(objTypeStr) {
	case "PROG", "PROG/P":
		objType = adt.ObjectTypeProgram
	case "CLAS", "CLAS/OC":
		objType = adt.ObjectTypeClass
	case "INTF", "INTF/OI":
		objType = adt.ObjectTypeInterface
	case "FUGR", "FUGR/F":
		objType = adt.ObjectTypeFunctionGroup
	case "FUNC", "FUGR/FF":
		objType = adt.ObjectTypeFunctionMod
	case "INCL", "PROG/I":
		objType = adt.ObjectTypeInclude
	// RAP types
	case "DDLS", "DDLS/DF":
		objType = adt.ObjectTypeDDLS
	case "BDEF", "BDEF/BDO":
		objType = adt.ObjectTypeBDEF
	case "SRVD", "SRVD/SRV":
		objType = adt.ObjectTypeSRVD
	default:
		objType = adt.CreatableObjectType(objTypeStr)
	}

	// Handle class includes
	if objType == adt.ObjectTypeClass && includeStr != "" && includeStr != "main" {
		var includeType adt.ClassIncludeType
		switch includeStr {
		case "testclasses":
			includeType = adt.ClassIncludeTestClasses
		case "definitions":
			includeType = adt.ClassIncludeDefinitions
		case "implementations":
			includeType = adt.ClassIncludeImplementations
		case "macros":
			includeType = adt.ClassIncludeMacros
		default:
			return newToolResultError(fmt.Sprintf("invalid include type: %s (expected: main, testclasses, definitions, implementations, macros)", includeStr)), nil
		}

		result, err := s.adtClient.SaveClassIncludeToFile(ctx, objectName, includeType, outputPath)
		if err != nil {
			return newToolResultError(fmt.Sprintf("SaveClassIncludeToFile failed: %v", err)), nil
		}
		output, _ := json.MarshalIndent(result, "", "  ")
		return mcp.NewToolResultText(string(output)), nil
	}

	result, err := s.adtClient.SaveToFile(ctx, objType, objectName, parentName, outputPath)
	if err != nil {
		return newToolResultError(fmt.Sprintf("SaveToFile failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleRenameObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objTypeStr, ok := request.Params.Arguments["objType"].(string)
	if !ok || objTypeStr == "" {
		return newToolResultError("objType is required (e.g., CLAS/OC, PROG/P, INTF/OI, FUGR/F)"), nil
	}

	oldName, ok := request.Params.Arguments["oldName"].(string)
	if !ok || oldName == "" {
		return newToolResultError("oldName is required"), nil
	}

	newName, ok := request.Params.Arguments["newName"].(string)
	if !ok || newName == "" {
		return newToolResultError("newName is required"), nil
	}

	packageName, ok := request.Params.Arguments["packageName"].(string)
	if !ok || packageName == "" {
		return newToolResultError("packageName is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	// Parse object type
	objType := adt.CreatableObjectType(objTypeStr)

	result, err := s.adtClient.RenameObject(ctx, objType, oldName, newName, packageName, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("RenameObject failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleEditSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	oldString, ok := request.Params.Arguments["old_string"].(string)
	if !ok || oldString == "" {
		return newToolResultError("old_string is required"), nil
	}

	newString, ok := request.Params.Arguments["new_string"].(string)
	if !ok {
		return newToolResultError("new_string is required"), nil
	}

	replaceAll := false
	if r, ok := request.Params.Arguments["replace_all"].(bool); ok {
		replaceAll = r
	}

	syntaxCheck := true
	if sc, ok := request.Params.Arguments["syntax_check"].(bool); ok {
		syntaxCheck = sc
	}

	caseInsensitive := false
	if ci, ok := request.Params.Arguments["case_insensitive"].(bool); ok {
		caseInsensitive = ci
	}

	method := ""
	if m, ok := request.Params.Arguments["method"].(string); ok {
		method = m
	}

	ignoreWarnings := false
	if iw, ok := request.Params.Arguments["ignore_warnings"].(bool); ok {
		ignoreWarnings = iw
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	opts := &adt.EditSourceOptions{
		ReplaceAll:      replaceAll,
		SyntaxCheck:     syntaxCheck,
		IgnoreWarnings:  ignoreWarnings,
		CaseInsensitive: caseInsensitive,
		Method:          method,
		Transport:       transport,
	}

	result, err := s.adtClient.EditSourceWithOptions(ctx, objectURL, oldString, newString, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("EditSource failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}
