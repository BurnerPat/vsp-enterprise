// Package mcp provides the MCP server implementation for ABAP ADT tools.
// tool_crud.go contains handlers for CRUD operations (lock, unlock, create, update, delete).
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// CRUDToolDefs returns tool definitions for CRUD operations.
func CRUDToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("LockObject",
			mcp.WithDescription("Acquire an edit lock on an ABAP object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("access_mode", mcp.Description("Access mode: MODIFY (default) or READ")),
		), Handler: HandleLockObject, Focused: true},

		{Tool: mcp.NewTool("UnlockObject",
			mcp.WithDescription("Release an edit lock on an ABAP object"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject")),
		), Handler: HandleUnlockObject, Focused: true},

		{Tool: mcp.NewTool("UpdateSource",
			mcp.WithDescription("Write source code to an ABAP object (requires lock)"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("source", mcp.Required(), mcp.Description("ABAP source code to write")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("Lock handle from LockObject")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), Handler: HandleUpdateSource},

		{Tool: mcp.NewTool("CreateObject",
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
		), Handler: HandleCreateObject},

		{Tool: mcp.NewTool("CreatePackage",
			mcp.WithDescription("Create a new ABAP package. Local packages ($*) work by default. Transportable packages require --enable-transports flag and transport parameter."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Package name (e.g., $ZTEST for local, ZPRODUCTION for transportable)")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Package description")),
			mcp.WithString("parent", mcp.Description("Parent package name (optional, e.g., $TMP, ZPROD). If not specified, creates a root-level package.")),
			mcp.WithString("transport", mcp.Description("Transport request number (required for transportable packages, e.g., 'A4HK900114')")),
			mcp.WithString("software_component", mcp.Description("Software component name (required for transportable packages, e.g., 'HOME', 'ZLOCAL'). Use GetInstalledComponents to list available components.")),
		), Handler: HandleCreatePackage, Focused: true},

		{Tool: mcp.NewTool("CreateTable",
			mcp.WithDescription("Create a DDIC transparent table from a simple JSON definition. Handles full workflow: create → set source → activate. Supports common ABAP types: CHAR, NUMC, INT4, DEC, STRING, TIMESTAMPL, UUID, etc."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Table name (uppercase, max 30 chars, must start with Z/Y)")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Short description of the table")),
			mcp.WithString("package", mcp.Description("Target package (default: $TMP)")),
			mcp.WithString("fields", mcp.Required(), mcp.Description("JSON array of fields: [{\"name\":\"ID\",\"type\":\"CHAR32\",\"key\":true},{\"name\":\"VALUE\",\"type\":\"STRING\"}]. Types: CHAR/CHARnn, NUMC/NUMCnn, INT4, DEC, STRING, TIMESTAMPL, UUID, DATS, TIMS, or data element name.")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for $TMP)")),
			mcp.WithString("delivery_class", mcp.Description("Delivery class: A=Application (default), C=Customizing, L=Temporary")),
		), Handler: HandleCreateTable, Focused: true},

		{Tool: mcp.NewTool("CompareSource",
			mcp.WithDescription("Compare source code of two objects and return unified diff. Supports all object types from GetSource."),
			mcp.WithString("type1", mcp.Required(), mcp.Description("Object type of first object: PROG, CLAS, INTF, FUNC, FUGR, INCL, DDLS, BDEF, SRVD")),
			mcp.WithString("name1", mcp.Required(), mcp.Description("Name of first object")),
			mcp.WithString("type2", mcp.Required(), mcp.Description("Object type of second object (can be same or different)")),
			mcp.WithString("name2", mcp.Required(), mcp.Description("Name of second object")),
			mcp.WithString("include1", mcp.Description("Class include type for first object if CLAS: definitions, implementations, macros, testclasses")),
			mcp.WithString("include2", mcp.Description("Class include type for second object if CLAS")),
			mcp.WithString("parent1", mcp.Description("Function group for first object if FUNC")),
			mcp.WithString("parent2", mcp.Description("Function group for second object if FUNC")),
		), Handler: HandleCompareSource, ReadOnly: true, Focused: true},

		{Tool: mcp.NewTool("CloneObject",
			mcp.WithDescription("Copy an ABAP object to a new name. Replaces object name in source. Supports PROG, CLAS, INTF."),
			mcp.WithString("object_type", mcp.Required(), mcp.Description("Object type: PROG, CLAS, INTF")),
			mcp.WithString("source_name", mcp.Required(), mcp.Description("Name of object to copy")),
			mcp.WithString("target_name", mcp.Required(), mcp.Description("Name for the new object")),
			mcp.WithString("package", mcp.Required(), mcp.Description("Target package (e.g., $TMP)")),
		), Handler: HandleCloneObject, Focused: true},

		{Tool: mcp.NewTool("GetClassInfo",
			mcp.WithDescription("Get class metadata without full source: methods, attributes, interfaces, superclass, abstract/final status."),
			mcp.WithString("class_name", mcp.Required(), mcp.Description("Name of the ABAP class")),
		), Handler: HandleGetClassInfo, ReadOnly: true, Focused: true},

		{Tool: mcp.NewTool("DeleteObject",
			mcp.WithDescription("Delete an ABAP object (requires lock)"),
			mcp.WithString("object_url", mcp.Required(), mcp.Description("ADT URL of the object (e.g., /sap/bc/adt/programs/programs/ZTEST)")),
			mcp.WithString("lock_handle", mcp.Required(), mcp.Description("lock_handle from LockObject")),
			mcp.WithString("transport", mcp.Description("Transport request number (optional for local packages)")),
		), Handler: HandleDeleteObject},
	}
}

// --- CRUD Handlers ---

func HandleLockObject(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	accessMode := "MODIFY"
	if am, ok := request.GetArguments()["access_mode"].(string); ok && am != "" {
		accessMode = am
	}

	result, err := sys.ADT().LockObject(ctx, objectURL, accessMode)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to lock object: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleUnlockObject(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	lockHandle, ok := request.GetArguments()["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return types.ErrorResult("lock_handle is required"), nil
	}

	err := sys.ADT().UnlockObject(ctx, objectURL, lockHandle)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to unlock object: %v", err)), nil
	}

	return mcp.NewToolResultText("Object unlocked successfully"), nil
}

func HandleUpdateSource(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	source, ok := request.GetArguments()["source"].(string)
	if !ok || source == "" {
		return types.ErrorResult("source is required"), nil
	}

	lockHandle, ok := request.GetArguments()["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return types.ErrorResult("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.GetArguments()["transport"].(string); ok {
		transport = t
	}

	// Append /source/main to object URL if not already present
	sourceURL := objectURL
	if !strings.HasSuffix(sourceURL, "/source/main") {
		sourceURL = objectURL + "/source/main"
	}

	err := sys.ADT().UpdateSource(ctx, sourceURL, source, lockHandle, transport)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to update source: %v", err)), nil
	}

	return mcp.NewToolResultText("Source updated successfully"), nil
}

func HandleCreateObject(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}

	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	description, ok := request.GetArguments()["description"].(string)
	if !ok || description == "" {
		return types.ErrorResult("description is required"), nil
	}

	packageName, ok := request.GetArguments()["package_name"].(string)
	if !ok || packageName == "" {
		return types.ErrorResult("package_name is required"), nil
	}

	transport := ""
	if t, ok := request.GetArguments()["transport"].(string); ok {
		transport = t
	}

	parentName := ""
	if p, ok := request.GetArguments()["parent_name"].(string); ok {
		parentName = p
	}

	// RAP-specific options
	serviceDefinition := ""
	if sd, ok := request.GetArguments()["service_definition"].(string); ok {
		serviceDefinition = sd
	}
	bindingVersion := ""
	if bv, ok := request.GetArguments()["binding_version"].(string); ok {
		bindingVersion = bv
	}
	bindingCategory := ""
	if bc, ok := request.GetArguments()["binding_category"].(string); ok {
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

	err := sys.ADT().CreateObject(ctx, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to create object: %v", err)), nil
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

func HandleCreatePackage(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	name = strings.ToUpper(name)

	description, ok := request.GetArguments()["description"].(string)
	if !ok || description == "" {
		return types.ErrorResult("description is required"), nil
	}

	parent := ""
	if p, ok := request.GetArguments()["parent"].(string); ok && p != "" {
		parent = strings.ToUpper(p)
	}

	transport := ""
	if t, ok := request.GetArguments()["transport"].(string); ok && t != "" {
		transport = t
	}

	softwareComponent := ""
	if sc, ok := request.GetArguments()["software_component"].(string); ok && sc != "" {
		softwareComponent = strings.ToUpper(sc)
	}

	// Transportable packages require transport parameter
	if !strings.HasPrefix(name, "$") && transport == "" {
		return types.ErrorResult("transport is required for creating transportable packages (non-$ packages). Use --enable-transports flag."), nil
	}

	opts := adt.CreateObjectOptions{
		ObjectType:        adt.ObjectTypePackage,
		Name:              name,
		Description:       description,
		PackageName:       parent, // Parent package
		Transport:         transport,
		SoftwareComponent: softwareComponent,
	}

	err := sys.ADT().CreateObject(ctx, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to create package: %v", err)), nil
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

func HandleCreateTable(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	description, ok := request.GetArguments()["description"].(string)
	if !ok || description == "" {
		return types.ErrorResult("description is required"), nil
	}

	fieldsJSON, ok := request.GetArguments()["fields"].(string)
	if !ok || fieldsJSON == "" {
		return types.ErrorResult("fields is required (JSON array)"), nil
	}

	// Parse fields JSON
	var fields []adt.TableField
	if err := json.Unmarshal([]byte(fieldsJSON), &fields); err != nil {
		return types.ErrorResult(fmt.Sprintf("Invalid fields JSON: %v", err)), nil
	}

	if len(fields) == 0 {
		return types.ErrorResult("At least one field is required"), nil
	}

	// Optional parameters
	pkg := "$TMP"
	if p, ok := request.GetArguments()["package"].(string); ok && p != "" {
		pkg = strings.ToUpper(p)
	}

	transport := ""
	if t, ok := request.GetArguments()["transport"].(string); ok && t != "" {
		transport = t
	}

	deliveryClass := "A"
	if dc, ok := request.GetArguments()["delivery_class"].(string); ok && dc != "" {
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

	err := sys.ADT().CreateTable(ctx, opts)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to create table: %v", err)), nil
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

func HandleCompareSource(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	type1, _ := request.GetArguments()["type1"].(string)
	name1, _ := request.GetArguments()["name1"].(string)
	type2, _ := request.GetArguments()["type2"].(string)
	name2, _ := request.GetArguments()["name2"].(string)

	if type1 == "" || name1 == "" || type2 == "" || name2 == "" {
		return types.ErrorResult("type1, name1, type2, and name2 are all required"), nil
	}

	// Build options for first object
	opts1 := &adt.GetSourceOptions{}
	if inc, ok := request.GetArguments()["include1"].(string); ok && inc != "" {
		opts1.Include = inc
	}
	if parent, ok := request.GetArguments()["parent1"].(string); ok && parent != "" {
		opts1.Parent = parent
	}

	// Build options for second object
	opts2 := &adt.GetSourceOptions{}
	if inc, ok := request.GetArguments()["include2"].(string); ok && inc != "" {
		opts2.Include = inc
	}
	if parent, ok := request.GetArguments()["parent2"].(string); ok && parent != "" {
		opts2.Parent = parent
	}

	diff, err := sys.ADT().CompareSource(ctx, type1, name1, type2, name2, opts1, opts2)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("CompareSource failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(diff, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleCloneObject(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, _ := request.GetArguments()["object_type"].(string)
	sourceName, _ := request.GetArguments()["source_name"].(string)
	targetName, _ := request.GetArguments()["target_name"].(string)
	pkg, _ := request.GetArguments()["package"].(string)

	if objectType == "" || sourceName == "" || targetName == "" || pkg == "" {
		return types.ErrorResult("object_type, source_name, target_name, and package are all required"), nil
	}

	result, err := sys.ADT().CloneObject(ctx, objectType, sourceName, targetName, pkg)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("CloneObject failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleGetClassInfo(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	className, _ := request.GetArguments()["class_name"].(string)
	if className == "" {
		return types.ErrorResult("class_name is required"), nil
	}

	info, err := sys.ADT().GetClassInfo(ctx, className)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetClassInfo failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(info, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleDeleteObject(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectURL, ok := request.GetArguments()["object_url"].(string)
	if !ok || objectURL == "" {
		return types.ErrorResult("object_url is required"), nil
	}

	lockHandle, ok := request.GetArguments()["lock_handle"].(string)
	if !ok || lockHandle == "" {
		return types.ErrorResult("lock_handle is required"), nil
	}

	transport := ""
	if t, ok := request.GetArguments()["transport"].(string); ok {
		transport = t
	}

	err := sys.ADT().DeleteObject(ctx, objectURL, lockHandle, transport)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("Failed to delete object: %v", err)), nil
	}

	return mcp.NewToolResultText("Object deleted successfully"), nil
}

func HandleMoveObject(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}

	objectName, ok := request.GetArguments()["object_name"].(string)
	if !ok || objectName == "" {
		return types.ErrorResult("object_name is required"), nil
	}

	newPackage, ok := request.GetArguments()["new_package"].(string)
	if !ok || newPackage == "" {
		return types.ErrorResult("new_package is required"), nil
	}

	// MoveObject requires ZADT_VSP WebSocket
	if sys.IsRfcMode() {
		return types.ErrorResult("MoveObject is not available in RFC mode"), nil
	}

	// Ensure WebSocket client is connected
	if errResult := sys.EnsureWSConnected(ctx, "MoveObject"); errResult != nil {
		return errResult, nil
	}

	// This is a placeholder since sys doesn't have a DebugWSClient yet.
	// We'll need to add it to the System interface if needed.
	return types.ErrorResult("MoveObject is not yet implemented in the new architecture"), nil
}
