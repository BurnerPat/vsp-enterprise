// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_crud.go contains handlers for CRUD operations (lock, unlock, create, update, delete).
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// crudToolDefs returns tool definitions for CRUD operations.
func (s *Server) crudToolDefs() []toolDef {
	return []toolDef{
		{tool: mcp.NewTool("LockObject",
			mcp.WithDescription("Acquire an edit lock on an ABAP object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("access_mode", mcp.Description("Access mode: MODIFY (default) or READ")),
		), handler: s.handleLockObject, focused: true},

		{tool: mcp.NewTool("UnlockObject",
			mcp.WithDescription("Release an edit lock on an ABAP object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject")),
		), handler: s.handleUnlockObject, focused: true},

		{tool: mcp.NewTool("UpdateSource",
			mcp.WithDescription("Write source code to an ABAP object (requires lock)"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code to write")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleUpdateSource},

		{tool: mcp.NewTool("CreateObject",
			mcp.WithDescription("Create a new ABAP object. Supports: PROG/P (program), CLAS/OC (class), INTF/OI (interface), PROG/I (include), FUGR/F (function group), FUGR/FF (function module), DEVC/K (package), DDLS/DF (CDS view), BDEF/BDO (behavior definition), SRVD/SRV (service definition), SRVB/SVB (service binding)"),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("Object type: PROG/P, CLAS/OC, INTF/OI, PROG/I, FUGR/F, FUGR/FF, DEVC/K, DDLS/DF, BDEF/BDO, SRVD/SRV, SRVB/SVB")),
			mcp.WithString("name", mcp.Required(), mcp.Description("Object name (e.g., ZTEST_PROGRAM)")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Object description")),
			mcp.WithString("package_name", mcp.Required(), mcp.Description("Package name (e.g., $TMP for local, ZPACKAGE for transportable)")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for non-local packages)")),
			mcp.WithString("parent_name", mcp.Description("Parent name (required for function modules - the function group name)")),
			mcp.WithString("service_definition", mcp.Description("For SRVB: the service definition name to bind")),
			mcp.WithString("binding_version", mcp.Description("For SRVB: OData version 'V2' or 'V4' (default: V2)")),
			mcp.WithString("binding_category", mcp.Description("For SRVB: '0' for Web API, '1' for UI (default: 0)")),
		), handler: s.handleCreateObject},

		{tool: mcp.NewTool("CreatePackage",
			mcp.WithDescription("Create a new ABAP package. Local packages ($*) work by default. Transportable packages require --enable-transports flag and transport parameter."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Package name (e.g., $ZTEST for local, ZPRODUCTION for transportable)")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Package description")),
			mcp.WithString("parent", mcp.Description("Parent package name (optional, e.g., $TMP, ZPROD). If not specified, creates a root-level package.")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for transportable packages, e.g., 'A4HK900114')")),
			mcp.WithString("software_component", mcp.Description("Software component name (required for transportable packages, e.g., 'HOME', 'ZLOCAL'). Use GetInstalledComponents to list available components.")),
		), handler: s.handleCreatePackage, focused: true},

		{tool: mcp.NewTool("CreateTable",
			mcp.WithDescription("Create a DDIC transparent table from a simple JSON definition. Handles full workflow: create → set source → activate. Supports common ABAP types: CHAR, NUMC, INT4, DEC, STRING, TIMESTAMPL, UUID, etc."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Table name (uppercase, max 30 chars, must start with Z/Y)")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Short description of the table")),
			mcp.WithString("package", mcp.Description("Target package (default: $TMP)")),
			mcp.WithString("fields", mcp.Required(), mcp.Description("JSON array of fields: [{\"name\":\"ID\",\"type\":\"CHAR32\",\"key\":true},{\"name\":\"VALUE\",\"type\":\"STRING\"}]. Types: CHAR/CHARnn, NUMC/NUMCnn, INT4, DEC, STRING, TIMESTAMPL, UUID, DATS, TIMS, or data element name.")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for $TMP)")),
			mcp.WithString("delivery_class", mcp.Description("Delivery class: A=Application (default), C=Customizing, L=Temporary")),
		), handler: s.handleCreateTable, focused: true},

		{tool: mcp.NewTool("CompareSource",
			mcp.WithDescription("Compare source code of two objects and return unified diff. Supports all object types from GetSource."),
			mcp.WithString("type1", mcp.Required(), mcp.Description("Object type of first object: PROG, CLAS, INTF, FUNC, FUGR, INCL, DDLS, BDEF, SRVD")),
			mcp.WithString("name1", mcp.Required(), mcp.Description("Name of first object")),
			mcp.WithString("type2", mcp.Required(), mcp.Description("Object type of second object (can be same or different)")),
			mcp.WithString("name2", mcp.Required(), mcp.Description("Name of second object")),
			mcp.WithString("include1", mcp.Description("Class include type for first object if CLAS: definitions, implementations, macros, testclasses")),
			mcp.WithString("include2", mcp.Description("Class include type for second object if CLAS")),
			mcp.WithString("parent1", mcp.Description("Function group for first object if FUNC")),
			mcp.WithString("parent2", mcp.Description("Function group for second object if FUNC")),
		), handler: s.handleCompareSource, readOnly: true, focused: true},

		{tool: mcp.NewTool("CloneObject",
			mcp.WithDescription("Copy an ABAP object to a new name. Replaces object name in source. Supports PROG, CLAS, INTF."),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("Object type: PROG, CLAS, INTF")),
			mcp.WithString("source_name", mcp.Required(), mcp.Description("Name of object to copy")),
			mcp.WithString("target_name", mcp.Required(), mcp.Description("Name for the new object")),
			mcp.WithString("package", mcp.Required(), mcp.Description("Target package (e.g., $TMP)")),
		), handler: s.handleCloneObject, focused: true},

		{tool: mcp.NewTool("GetClassInfo",
			mcp.WithDescription("Get class metadata without full source: methods, attributes, interfaces, superclass, abstract/final status."),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
		), handler: s.handleGetClassInfo, readOnly: true, focused: true},

		{tool: mcp.NewTool("DeleteObject",
			mcp.WithDescription("Delete an ABAP object (requires lock)"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), handler: s.handleDeleteObject},

		{tool: mcp.NewTool("GetUserTransports",
			mcp.WithDescription("Get all transport requests for a user (requires --enable-transports flag). Returns both workbench and customizing requests grouped by target system."),
			mcp.WithString("user_name", mcp.Required(), mcp.Description("SAP user name (will be converted to uppercase)")),
		), handler: s.handleGetUserTransports, readOnly: true},

		{tool: mcp.NewTool("GetTransportInfo",
			mcp.WithDescription("Get transport information for an ABAP object (requires --enable-transports flag). Returns available transports and lock status."),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("dev_class", mcp.Required(), mcp.Description("Development class/package of the object")),
		), handler: s.handleGetTransportInfo, readOnly: true},
	}
}

// routeCRUDAction routes "edit" for LOCK/UNLOCK/UPDATE_SOURCE, "create" for OBJECT/DEVC/TABL/CLONE, "delete" for OBJECT.
func (s *Server) routeCRUDAction(ctx context.Context, action, objectType, objectName string, params map[string]any) (*mcp.CallToolResult, bool, error) {
	if action == "edit" {
		switch objectType {
		case "LOCK":
			return s.callHandler(ctx, s.handleLockObject, params)
		case "UNLOCK":
			return s.callHandler(ctx, s.handleUnlockObject, params)
		case "UPDATE_SOURCE":
			return s.callHandler(ctx, s.handleUpdateSource, params)
		case "MOVE":
			return s.callHandler(ctx, s.handleMoveObject, params)
		case "COMPARE_SOURCE":
			return s.callHandler(ctx, s.handleCompareSource, params)
		}
	}

	if action == "create" {
		switch objectType {
		case "OBJECT":
			return s.callHandler(ctx, s.handleCreateObject, params)
		case "DEVC":
			return s.callHandler(ctx, s.handleCreatePackage, params)
		case "TABL":
			return s.callHandler(ctx, s.handleCreateTable, params)
		case "CLONE":
			return s.callHandler(ctx, s.handleCloneObject, params)
		}
	}

	if action == "delete" {
		switch objectType {
		case "OBJECT", "":
			if getStringParam(params, "object_url") != "" {
				return s.callHandler(ctx, s.handleDeleteObject, params)
			}
		}
	}

	// read CLASS_INFO
	if action == "read" && objectType == "CLASS_INFO" {
		return s.callHandler(ctx, s.handleGetClassInfo, map[string]any{"class_name": objectName})
	}

	return nil, false, nil
}

// --- CRUD Handlers ---

func (s *Server) handleLockObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	accessMode := "MODIFY"
	if am, ok := request.Params.Arguments["access_mode"].(string); ok && am != "" {
		accessMode = am
	}

	result, err := s.adtClient.LockObject(ctx, objectURL, accessMode)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to lock object: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleUnlockObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	lockHandle, ok := request.Params.Arguments["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	err := s.adtClient.UnlockObject(ctx, objectURL, lockHandle)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to unlock object: %v", err)), nil
	}

	return mcp.NewToolResultText("Object unlocked successfully"), nil
}

func (s *Server) handleUpdateSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	source, ok := request.Params.Arguments["source"].(string)
	if !ok || source == "" {
		return newToolResultError("source is required"), nil
	}

	lockHandle, ok := request.Params.Arguments["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	// Append /source/main to object URL if not already present
	sourceURL := objectURL
	if !strings.HasSuffix(sourceURL, "/source/main") {
		sourceURL = objectURL + "/source/main"
	}

	err := s.adtClient.UpdateSource(ctx, sourceURL, source, lockHandle, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to update source: %v", err)), nil
	}

	return mcp.NewToolResultText("Source updated successfully"), nil
}

func (s *Server) handleCreateObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.Params.Arguments["object_type"].(string)
	if !ok || objectType == "" {
		return newToolResultError("object_type is required"), nil
	}

	name, ok := request.Params.Arguments["name"].(string)
	if !ok || name == "" {
		return newToolResultError("name is required"), nil
	}

	description, ok := request.Params.Arguments["description"].(string)
	if !ok || description == "" {
		return newToolResultError("description is required"), nil
	}

	packageName, ok := request.Params.Arguments["package_name"].(string)
	if !ok || packageName == "" {
		return newToolResultError("package_name is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	parentName := ""
	if p, ok := request.Params.Arguments["parent_name"].(string); ok {
		parentName = p
	}

	// RAP-specific options
	serviceDefinition := ""
	if sd, ok := request.Params.Arguments["service_definition"].(string); ok {
		serviceDefinition = sd
	}
	bindingVersion := ""
	if bv, ok := request.Params.Arguments["binding_version"].(string); ok {
		bindingVersion = bv
	}
	bindingCategory := ""
	if bc, ok := request.Params.Arguments["binding_category"].(string); ok {
		bindingCategory = bc
	}

	opts := adt.CreateObjectOptions{
		ObjectType:        adt.CreatableObjectType(objectType),
		Name:              name,
		Description:       description,
		PackageName:       packageName,
		Transport:         transport,
		ParentName:        parentName,
		ServiceDefinition: serviceDefinition,
		BindingVersion:    bindingVersion,
		BindingCategory:   bindingCategory,
	}

	err := s.adtClient.CreateObject(ctx, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to create object: %v", err)), nil
	}

	// Return the object URL for convenience
	objURL := adt.GetObjectURL(opts.ObjectType, opts.Name, opts.ParentName)
	result := map[string]string{
		"status":     "created",
		"object_url": objURL,
	}
	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCreatePackage(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.Params.Arguments["name"].(string)
	if !ok || name == "" {
		return newToolResultError("name is required"), nil
	}

	name = strings.ToUpper(name)

	description, ok := request.Params.Arguments["description"].(string)
	if !ok || description == "" {
		return newToolResultError("description is required"), nil
	}

	parent := ""
	if p, ok := request.Params.Arguments["parent"].(string); ok && p != "" {
		parent = strings.ToUpper(p)
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok && t != "" {
		transport = t
	}

	softwareComponent := ""
	if sc, ok := request.Params.Arguments["software_component"].(string); ok && sc != "" {
		softwareComponent = strings.ToUpper(sc)
	}

	// Transportable packages require transport parameter
	if !strings.HasPrefix(name, "$") && transport == "" {
		return newToolResultError("transport is required for creating transportable packages (non-$ packages). Use --enable-transports flag."), nil
	}

	opts := adt.CreateObjectOptions{
		ObjectType:        adt.ObjectTypePackage,
		Name:              name,
		Description:       description,
		PackageName:       parent, // Parent package
		Transport:         transport,
		SoftwareComponent: softwareComponent,
	}

	err := s.adtClient.CreateObject(ctx, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to create package: %v", err)), nil
	}

	result := map[string]string{
		"status":      "created",
		"package":     name,
		"description": description,
	}
	if parent != "" {
		result["parent"] = parent
	}
	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCreateTable(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.Params.Arguments["name"].(string)
	if !ok || name == "" {
		return newToolResultError("name is required"), nil
	}

	description, ok := request.Params.Arguments["description"].(string)
	if !ok || description == "" {
		return newToolResultError("description is required"), nil
	}

	fieldsJSON, ok := request.Params.Arguments["fields"].(string)
	if !ok || fieldsJSON == "" {
		return newToolResultError("fields is required (JSON array)"), nil
	}

	// Parse fields JSON
	var fields []adt.TableField
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return newToolResultError(fmt.Sprintf("Invalid fields JSON: %v", err)), nil
	}

	if len(fields) == 0 {
		return newToolResultError("At least one field is required"), nil
	}

	// Optional parameters
	pkg := "$TMP"
	if p, ok := request.Params.Arguments["package"].(string); ok && p != "" {
		pkg = strings.ToUpper(p)
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok && t != "" {
		transport = t
	}

	deliveryClass := "A"
	if dc, ok := request.Params.Arguments["delivery_class"].(string); ok && dc != "" {
		deliveryClass = strings.ToUpper(dc)
	}

	opts := adt.CreateTableOptions{
		Name:          name,
		Description:   description,
		Package:       pkg,
		Fields:        fields,
		Transport:     transport,
		DeliveryClass: deliveryClass,
	}

	err := s.adtClient.CreateTable(ctx, opts)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to create table: %v", err)), nil
	}

	result := map[string]interface{}{
		"status":      "created",
		"table":       strings.ToUpper(name),
		"package":     pkg,
		"description": description,
		"fields":      len(fields),
	}
	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCompareSource(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	type1, _ := request.Params.Arguments["type1"].(string)
	name1, _ := request.Params.Arguments["name1"].(string)
	type2, _ := request.Params.Arguments["type2"].(string)
	name2, _ := request.Params.Arguments["name2"].(string)

	if type1 == "" || name1 == "" || type2 == "" || name2 == "" {
		return newToolResultError("type1, name1, type2, and name2 are all required"), nil
	}

	// Build options for first object
	opts1 := &adt.GetSourceOptions{}
	if inc, ok := request.Params.Arguments["include1"].(string); ok && inc != "" {
		opts1.Include = inc
	}
	if parent, ok := request.Params.Arguments["parent1"].(string); ok && parent != "" {
		opts1.Parent = parent
	}

	// Build options for second object
	opts2 := &adt.GetSourceOptions{}
	if inc, ok := request.Params.Arguments["include2"].(string); ok && inc != "" {
		opts2.Include = inc
	}
	if parent, ok := request.Params.Arguments["parent2"].(string); ok && parent != "" {
		opts2.Parent = parent
	}

	diff, err := s.adtClient.CompareSource(ctx, type1, name1, type2, name2, opts1, opts2)
	if err != nil {
		return newToolResultError(fmt.Sprintf("CompareSource failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(diff, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleCloneObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, _ := request.Params.Arguments["object_type"].(string)
	sourceName, _ := request.Params.Arguments["source_name"].(string)
	targetName, _ := request.Params.Arguments["target_name"].(string)
	pkg, _ := request.Params.Arguments["package"].(string)

	if objectType == "" || sourceName == "" || targetName == "" || pkg == "" {
		return newToolResultError("object_type, source_name, target_name, and package are all required"), nil
	}

	result, err := s.adtClient.CloneObject(ctx, objectType, sourceName, targetName, pkg)
	if err != nil {
		return newToolResultError(fmt.Sprintf("CloneObject failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleGetClassInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, _ := request.Params.Arguments["class_name"].(string)
	if className == "" {
		return newToolResultError("class_name is required"), nil
	}

	info, err := s.adtClient.GetClassInfo(ctx, className)
	if err != nil {
		return newToolResultError(fmt.Sprintf("GetClassInfo failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(info, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func (s *Server) handleDeleteObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.Params.Arguments["object_url"].(string)
	if !ok || objectURL == "" {
		return newToolResultError("object_url is required"), nil
	}

	lockHandle, ok := request.Params.Arguments["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return newToolResultError("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.Params.Arguments["transport"].(string); ok {
		transport = t
	}

	err := s.adtClient.DeleteObject(ctx, objectURL, lockHandle, transport)
	if err != nil {
		return newToolResultError(fmt.Sprintf("Failed to delete object: %v", err)), nil
	}

	return mcp.NewToolResultText("Object deleted successfully"), nil
}

func (s *Server) handleMoveObject(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.Params.Arguments["object_type"].(string)
	if !ok || objectType == "" {
		return newToolResultError("object_type is required"), nil
	}

	objectName, ok := request.Params.Arguments["object_name"].(string)
	if !ok || objectName == "" {
		return newToolResultError("object_name is required"), nil
	}

	newPackage, ok := request.Params.Arguments["new_package"].(string)
	if !ok || newPackage == "" {
		return newToolResultError("new_package is required"), nil
	}

	// RFC mode: MoveObject requires ZADT_VSP WebSocket
	if s.isRfcMode() {
		return s.rfcModeWSUnavailable("MoveObject"), nil
	}

	// Ensure WebSocket client is connected
	if err := s.ensureDebugWSClient(ctx); err != nil {
		return newToolResultError(fmt.Sprintf("Failed to connect to ZADT_VSP WebSocket: %v. Ensure ZADT_VSP is deployed and SAPC/SICF are configured.", err)), nil
	}

	result, err := s.debugWSClient.MoveObject(ctx, objectType, objectName, newPackage)
	if err != nil {
		return newToolResultError(fmt.Sprintf("MoveObject failed: %v", err)), nil
	}

	// Format result
	if result.Success {
		return mcp.NewToolResultText(fmt.Sprintf("Object moved successfully.\n\nObject: %s %s\nNew Package: %s\nMessage: %s",
			result.Object, result.ObjName, result.NewPackage, result.Message)), nil
	}
	return newToolResultError(fmt.Sprintf("Move failed: %s", result.Message)), nil
}
