package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// ObjectInfoToolDefs returns tool definitions for ABAP object information tools.
func ObjectInfoToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{Tool: mcp.NewTool("GetObjectProperties",
			mcp.WithDescription("Get metadata about an ABAP object: creator, package hierarchy, creation date, language, system, API release state, and description."),
			mcp.WithString("object_type", mcp.Required(),
				mcp.Description("Object type: CLAS, INTF, PROG, FUGR, FUNC, TABL, DDLS, DTEL, DOMA, SRVD, BDEF, SRVB")),
			mcp.WithString("name", mcp.Required(),
				mcp.Description("Object name (e.g., ZCL_MY_CLASS, Z_MY_PROGRAM)")),
		), Handler: HandleGetObjectProperties, ReadOnly: true, Focused: true,
			Endpoints: []string{"/sap/bc/adt/repository/informationsystem/objectproperties"},
			Routes: []types.UniversalRoute{{Action: "analyze", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["object_type"] = ot
				p["name"] = on
				return p
			}}}},

		{Tool: mcp.NewTool("GetObjectOutline",
			mcp.WithDescription("Get the structural outline of an ABAP object: methods, attributes, events, types, and other components with their visibility and properties. For classes/interfaces, optionally includes inherited members."),
			mcp.WithString("object_type", mcp.Required(),
				mcp.Description("Object type: CLAS, INTF, PROG, FUGR, FUNC, TABL, DDLS, DTEL, DOMA, SRVD, BDEF, SRVB")),
			mcp.WithString("name", mcp.Required(),
				mcp.Description("Object name (e.g., ZCL_MY_CLASS)")),
			mcp.WithBoolean("include_inherited",
				mcp.Description("Include inherited members in the outline (default: false). Only relevant for CLAS/INTF.")),
		), Handler: HandleGetObjectOutline, ReadOnly: true, Focused: true,
			Endpoints: []string{"/sap/bc/adt/oo/classes"},
			Routes: []types.UniversalRoute{{Action: "analyze", TargetType: "STRUCTURE", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["object_type"] = ot
				p["name"] = on
				return p
			}}}},

		{Tool: mcp.NewTool("GetObjectNetwork",
			mcp.WithDescription("Get the dependency network of an ABAP object: all directly used objects (classes, interfaces, tables, data elements, etc.) with their types, descriptions, and packages."),
			mcp.WithString("object_type", mcp.Required(),
				mcp.Description("Object type: CLAS, INTF, PROG, FUGR, FUNC, TABL, DDLS, DTEL, DOMA, SRVD, BDEF, SRVB")),
			mcp.WithString("name", mcp.Required(),
				mcp.Description("Object name (e.g., ZCL_MY_CLASS)")),
		), Handler: HandleGetObjectNetwork, ReadOnly: true, Focused: true,
			Endpoints: []string{"/sap/bc/adt/objectrelations/network"},
			Routes: []types.UniversalRoute{{Action: "analyze", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["object_type"] = ot
				p["name"] = on
				return p
			}}}},

		{Tool: mcp.NewTool("GetWhereUsed",
			mcp.WithDescription("Get where-used list: find all objects that reference a given ABAP object or one of its members. Optionally includes code snippets showing exact usage locations."),
			mcp.WithString("object_type", mcp.Required(),
				mcp.Description("Object type: CLAS, INTF, PROG, FUGR, FUNC, TABL, DDLS, DTEL, DOMA, SRVD, BDEF, SRVB")),
			mcp.WithString("name", mcp.Required(),
				mcp.Description("Object name (e.g., ZCL_MY_CLASS)")),
			mcp.WithString("member_uri",
				mcp.Description("ADT URI of a specific member to search for (e.g., from GetObjectOutline href). If omitted, searches for the object itself.")),
			mcp.WithBoolean("include_snippets",
				mcp.Description("Fetch code snippets showing the exact usage locations (default: false). Requires an additional server request.")),
		), Handler: HandleGetWhereUsed, ReadOnly: true, Focused: true,
			Endpoints: []string{"/sap/bc/adt/repository/informationsystem/usageReferences"},
			Routes: []types.UniversalRoute{{Action: "analyze", MapArgs: func(ot, on string, p map[string]any) map[string]any {
				p["object_type"] = ot
				p["name"] = on
				return p
			}}}},
	}
}

// --- Object Info Handlers ---

func HandleGetObjectProperties(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	props, err := sys.ADT().GetObjectProperties(ctx, objectType, name)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetObjectProperties failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatObjectProperties(props)), nil
}

func HandleGetObjectOutline(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	includeInherited := false
	if v, ok := request.GetArguments()["include_inherited"].(bool); ok {
		includeInherited = v
	}

	outline, err := sys.ADT().GetObjectOutline(ctx, objectType, name, includeInherited)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetObjectOutline failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatObjectOutline(outline)), nil
}

func HandleGetObjectNetwork(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	network, err := sys.ADT().GetObjectNetwork(ctx, objectType, name)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetObjectNetwork failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatObjectNetwork(network)), nil
}

func HandleGetWhereUsed(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	objectType, ok := request.GetArguments()["object_type"].(string)
	if !ok || objectType == "" {
		return types.ErrorResult("object_type is required"), nil
	}
	name, ok := request.GetArguments()["name"].(string)
	if !ok || name == "" {
		return types.ErrorResult("name is required"), nil
	}

	memberURI, _ := request.GetArguments()["member_uri"].(string)

	includeSnippets := false
	if v, ok := request.GetArguments()["include_snippets"].(bool); ok {
		includeSnippets = v
	}

	result, err := sys.ADT().GetWhereUsed(ctx, objectType, name, memberURI, includeSnippets)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetWhereUsed failed: %v", err)), nil
	}

	return mcp.NewToolResultText(formatWhereUsed(result)), nil
}

// --- Formatters ---

func formatObjectProperties(props *adt.ObjectProperties) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Object: %s\n", props.Name)
	if props.Description != "" {
		fmt.Fprintf(&sb, "Description: %s\n", props.Description)
	}
	if props.ObjectType != "" {
		fmt.Fprintf(&sb, "Type: %s\n", props.ObjectType)
	}
	if props.Package != "" {
		fmt.Fprintf(&sb, "Package: %s\n", props.Package)
	}
	if props.Owner != "" {
		fmt.Fprintf(&sb, "Owner: %s\n", props.Owner)
	}
	if props.CreatedYear != "" {
		fmt.Fprintf(&sb, "Created: %s\n", props.CreatedYear)
	}
	if props.Language != "" {
		fmt.Fprintf(&sb, "Language: %s\n", props.Language)
	}
	if props.System != "" {
		fmt.Fprintf(&sb, "System: %s\n", props.System)
	}
	if props.APIState != "" {
		fmt.Fprintf(&sb, "API State: %s\n", props.APIState)
	}

	// Show full package hierarchy
	var packages []string
	for _, p := range props.Properties {
		if p.Facet == "PACKAGE" {
			entry := p.Name
			if p.Text != "" {
				entry += " (" + p.Text + ")"
			}
			packages = append(packages, entry)
		}
	}
	if len(packages) > 1 {
		sb.WriteString("\nPackage Hierarchy:\n")
		for _, pkg := range packages {
			fmt.Fprintf(&sb, "  - %s\n", pkg)
		}
	}

	return sb.String()
}

func formatObjectOutline(outline *adt.OutlineElement) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s (%s)\n", outline.Name, outline.Type)
	if outline.Description != "" {
		fmt.Fprintf(&sb, "  %s\n", outline.Description)
	}
	sb.WriteString("\n")

	// Group children by type category
	methods := []*adt.OutlineElement{}
	attributes := []*adt.OutlineElement{}
	events := []*adt.OutlineElement{}
	typeElements := []*adt.OutlineElement{}
	others := []*adt.OutlineElement{}

	for i := range outline.Children {
		child := &outline.Children[i]
		t := strings.ToUpper(child.Type)
		switch {
		case strings.Contains(t, "METH") || strings.Contains(t, "/OM"):
			methods = append(methods, child)
		case strings.Contains(t, "DATA") || strings.Contains(t, "ATTR") || strings.Contains(t, "/OA") || strings.Contains(t, "/PD"):
			attributes = append(attributes, child)
		case strings.Contains(t, "EVNT") || strings.Contains(t, "EVENT"):
			events = append(events, child)
		case strings.Contains(t, "TYPE") || strings.Contains(t, "/OT"):
			typeElements = append(typeElements, child)
		default:
			others = append(others, child)
		}
	}

	if len(methods) > 0 {
		fmt.Fprintf(&sb, "## Methods (%d)\n", len(methods))
		for _, m := range methods {
			fmt.Fprintf(&sb, "  - %s%s\n", m.Name, outlineFlags(m))
			if m.Description != "" {
				fmt.Fprintf(&sb, "    %s\n", m.Description)
			}
		}
		sb.WriteString("\n")
	}

	if len(attributes) > 0 {
		fmt.Fprintf(&sb, "## Attributes (%d)\n", len(attributes))
		for _, a := range attributes {
			fmt.Fprintf(&sb, "  - %s%s\n", a.Name, outlineFlags(a))
			if a.Description != "" {
				fmt.Fprintf(&sb, "    %s\n", a.Description)
			}
		}
		sb.WriteString("\n")
	}

	if len(events) > 0 {
		fmt.Fprintf(&sb, "## Events (%d)\n", len(events))
		for _, e := range events {
			fmt.Fprintf(&sb, "  - %s%s\n", e.Name, outlineFlags(e))
		}
		sb.WriteString("\n")
	}

	if len(typeElements) > 0 {
		fmt.Fprintf(&sb, "## Types (%d)\n", len(typeElements))
		for _, t := range typeElements {
			fmt.Fprintf(&sb, "  - %s%s\n", t.Name, outlineFlags(t))
		}
		sb.WriteString("\n")
	}

	if len(others) > 0 {
		fmt.Fprintf(&sb, "## Other Components (%d)\n", len(others))
		for _, o := range others {
			fmt.Fprintf(&sb, "  - %s (%s)%s\n", o.Name, o.Type, outlineFlags(o))
		}
	}

	return sb.String()
}

func outlineFlags(e *adt.OutlineElement) string {
	var flags []string
	if e.Visibility != "" {
		flags = append(flags, e.Visibility)
	}
	if e.IsStatic {
		flags = append(flags, "static")
	}
	if e.IsFinal {
		flags = append(flags, "final")
	}
	if e.IsAbstract {
		flags = append(flags, "abstract")
	}
	if e.IsReadOnly {
		flags = append(flags, "read-only")
	}
	if e.IsConstant {
		flags = append(flags, "constant")
	}
	if len(flags) > 0 {
		return " [" + strings.Join(flags, ", ") + "]"
	}
	return ""
}

func formatObjectNetwork(network *adt.ObjectNetwork) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Dependencies (%d objects)\n\n", len(network.References))

	// Group references by type
	groups := map[string][]adt.ObjectReference{}
	for _, ref := range network.References {
		category := categorizeADTType(ref.Type)
		groups[category] = append(groups[category], ref)
	}

	order := []string{"Classes", "Interfaces", "Tables", "Structures", "Data Elements", "Type Groups", "Transformations", "Other"}
	for _, category := range order {
		refs, ok := groups[category]
		if !ok {
			continue
		}
		fmt.Fprintf(&sb, "## %s (%d)\n", category, len(refs))
		for _, ref := range refs {
			desc := ref.Description
			if desc == "" {
				desc = ref.Name
			}
			fmt.Fprintf(&sb, "  - %s: %s", ref.Name, desc)
			if ref.PackageName != "" {
				fmt.Fprintf(&sb, " [%s]", ref.PackageName)
			}
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func categorizeADTType(adtType string) string {
	switch {
	case strings.Contains(adtType, "CLAS"):
		return "Classes"
	case strings.Contains(adtType, "INTF"):
		return "Interfaces"
	case adtType == "TABL/DT":
		return "Tables"
	case adtType == "TABL/DS":
		return "Structures"
	case strings.Contains(adtType, "DTEL"):
		return "Data Elements"
	case strings.Contains(adtType, "TYPE"):
		return "Type Groups"
	case strings.Contains(adtType, "XSLT"):
		return "Transformations"
	default:
		return "Other"
	}
}

func formatWhereUsed(result *adt.WhereUsedResult) string {
	var sb strings.Builder
	if result.ResultDescription != "" {
		fmt.Fprintf(&sb, "%s\n", result.ResultDescription)
	}
	fmt.Fprintf(&sb, "Results: %d\n\n", result.NumberOfResults)

	// Group references by parent object
	grouped := map[string][]adt.WhereUsedReference{}
	var order []string
	for _, ref := range result.References {
		// Skip container entries (they have no objectIdentifier and no usage info)
		if ref.ObjectIdentifier == "" && ref.UsageInformation == "" && ref.Type != "" {
			key := ref.Name
			if _, exists := grouped[key]; !exists {
				order = append(order, key)
				grouped[key] = nil
			}
			continue
		}
		if ref.ObjectIdentifier == "" {
			continue
		}
		// Find or create parent group
		parentKey := ""
		for _, k := range order {
			if strings.Contains(ref.URI, strings.ToLower(k)) || strings.Contains(ref.URI, url_encode(k)) {
				parentKey = k
				break
			}
		}
		if parentKey == "" {
			parentKey = ref.Name
			if _, exists := grouped[parentKey]; !exists {
				order = append(order, parentKey)
			}
		}
		grouped[parentKey] = append(grouped[parentKey], ref)
	}

	for _, key := range order {
		refs := grouped[key]
		fmt.Fprintf(&sb, "## %s\n", key)
		for _, ref := range refs {
			usage := ""
			if ref.UsageInformation != "" {
				usage = " (" + ref.UsageInformation + ")"
			}
			fmt.Fprintf(&sb, "  - %s%s\n", ref.Name, usage)
		}
		sb.WriteString("\n")
	}

	// Append snippets if present
	if len(result.Snippets) > 0 {
		sb.WriteString("## Code Snippets\n\n")
		for _, s := range result.Snippets {
			if s.Description != "" {
				fmt.Fprintf(&sb, "### %s\n", s.Description)
			}
			fmt.Fprintf(&sb, "```abap\n%s\n```\n\n", strings.TrimSpace(s.Content))
		}
	}

	return sb.String()
}

// url_encode is a simple helper for URL-safe lowercase matching.
func url_encode(s string) string {
	return strings.ToLower(strings.ReplaceAll(s, " ", "%20"))
}
