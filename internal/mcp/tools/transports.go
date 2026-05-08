package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/aymanbagabas/go-udiff"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/oisee/vibing-steampunk/internal/mcp/types"
	"github.com/oisee/vibing-steampunk/pkg/adt"
)

// --- Transport Diff Types ---

// transportObjectDiff holds the diff result for a single object (or class include) in a transport.
type transportObjectDiff struct {
	Object     adt.RefOutput      `json:"object"`
	PgmID      string             `json:"pgmid"`
	WBType     string             `json:"wbtype,omitempty"`
	Info       string             `json:"info,omitempty"`
	Include    string             `json:"include,omitempty"`    // class include if applicable
	MethodName string             `json:"methodName,omitempty"` // specific method if LIMU/METH
	Message    string             `json:"message,omitempty"`    // set when diff cannot be computed
	Diffs      []*adt.VersionDiff `json:"diffs,omitempty"`      // one or more diffs (multiple if foreign versions intervene)
}

// transportDiffResult is the complete result returned by GetTransportDiff.
type transportDiffResult struct {
	Transport   string                `json:"transport"`
	Description string                `json:"description"`
	Owner       string                `json:"owner"`
	Status      string                `json:"status"`
	Objects     []transportObjectDiff `json:"objects"`
}

// versionQuery represents a unique version-history query (used for deduplication).
type versionQuery struct {
	ObjectType string
	Name       string
	Include    string
	MethodName string // non-empty for LIMU/METH — triggers method-level extraction
	// original transport objects that map to this query
	Sources []adt.TransportObjectV2
}

// TransportToolDefs returns tool definitions for transport request tools.
func TransportToolDefs() []types.ToolDef {
	return []types.ToolDef{
		{
			Tool: mcp.NewTool("GetTransport",
				mcp.WithDescription("Get detailed information about a single transport request by its number. Returns owner, description, status, target system, tasks with their objects."),
				mcp.WithString("number", mcp.Required(),
					mcp.Description("Transport request or task number (e.g. DEVK900001, D61K907178)")),
			),
			Handler:   HandleGetTransport,
			ReadOnly:  true,
			Focused:   true,
			Endpoints: []string{"/sap/bc/adt/cts/transportrequests"},
		},
		{
			Tool: mcp.NewTool("ListTransports",
				mcp.WithDescription("List transport requests for a user. Returns modifiable workbench and customizing requests with their status, target system, and description."),
				mcp.WithString("user",
					mcp.Description("SAP user name to list transports for. Defaults to the connected user if omitted. Use `*` to list transports for all users (requires appropriate permissions).")),
			),
			Handler:   HandleListTransports,
			ReadOnly:  true,
			Focused:   true,
			Endpoints: []string{"/sap/bc/adt/cts/transportrequests"},
		},
		{
			Tool: mcp.NewTool("GetTransportDiff",
				mcp.WithDescription(
					"Get git-style unified diffs for all objects in a transport request. "+
						"Designed for automated code reviews: returns a flat list of diffs showing exactly what changed in each object. "+
						"Automatically includes objects from all tasks within the transport. "+
						"Each object diff uses `--- base/...` and `+++ target/...` headers (oldest vs newest version in the transport). "+
						"If other transports modified an object between the versions in this transport, multiple diffs are returned for that object (one per contiguous segment). "+
						"Objects without source code (data elements, domains, etc.) are listed with a message indicating no diff is available. "+
						"Supports: PROG, CLAS (all includes), INTF, DDLS, BDEF, SRVD, INCL, TABL."),
				mcp.WithString("number", mcp.Required(),
					mcp.Description("Transport request number (e.g. DEVK900001, D61K907178). Can be a workbench request, customizing request, or task.")),
			),
			Handler:   HandleGetTransportDiff,
			ReadOnly:  true,
			Focused:   true,
			Endpoints: []string{"/sap/bc/adt/cts/transportrequests"},
		},
	}
}

// --- Transport Handlers ---

func HandleGetTransport(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	number, _ := request.GetArguments()["number"].(string)
	if number == "" {
		return types.ErrorResult("number is required"), nil
	}

	details, err := sys.ADT().GetTransport(ctx, number)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetTransport failed: %v", err)), nil
	}

	output, _ := json.MarshalIndent(details, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

func HandleListTransports(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	user, _ := request.GetArguments()["user"].(string)

	transports, err := sys.ADT().ListTransports(ctx, user)
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("ListTransports failed: %v", err)), nil
	}

	if len(transports) == 0 {
		return mcp.NewToolResultText("No transport requests found."), nil
	}

	output, _ := json.MarshalIndent(transports, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// HandleGetTransportDiff produces git-style diffs for all objects in a transport.
func HandleGetTransportDiff(ctx context.Context, sys types.System, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	number, _ := request.GetArguments()["number"].(string)
	if number == "" {
		return types.ErrorResult("number is required"), nil
	}

	// 1. Fetch transport details
	details, err := sys.ADT().GetTransport(ctx, strings.ToUpper(number))
	if err != nil {
		return types.ErrorResult(fmt.Sprintf("GetTransport failed: %v", err)), nil
	}

	// 2. Build set of transport/task numbers (for matching versions)
	transportNumbers := make(map[string]bool)
	transportNumbers[details.Number] = true
	for _, task := range details.Tasks {
		transportNumbers[task.Number] = true
	}

	// 3. Collect all objects and deduplicate by pgmid+type+name
	type objKey struct{ pgmid, typ, name string }
	seen := make(map[objKey]bool)
	var allObjects []adt.TransportObjectV2

	addObject := func(obj adt.TransportObjectV2) {
		key := objKey{obj.PgmID, obj.Type, obj.Object.Name}
		if !seen[key] {
			seen[key] = true
			allObjects = append(allObjects, obj)
		}
	}

	// Objects from the request level (includes all_objects if present)
	for _, obj := range details.Objects {
		addObject(obj)
	}
	// Objects from tasks (fallback / complement)
	for _, task := range details.Tasks {
		for _, obj := range task.Objects {
			addObject(obj)
		}
	}

	// 4. Map objects to version queries and deduplicate at query level
	type queryKey struct{ objectType, name, include, method string }
	queryMap := make(map[queryKey]*versionQuery)
	var queryOrder []queryKey // preserve order

	for _, obj := range allObjects {
		queries := mapTransportObjectToVersionQueries(obj)
		for _, q := range queries {
			key := queryKey{q.ObjectType, q.Name, q.Include, q.MethodName}
			if existing, ok := queryMap[key]; ok {
				existing.Sources = append(existing.Sources, obj)
			} else {
				queryMap[key] = &q
				queryOrder = append(queryOrder, key)
			}
		}
	}

	// 5. For each unique query, fetch versions and compute diffs
	result := transportDiffResult{
		Transport:   details.Number,
		Description: details.Description,
		Owner:       details.Owner,
		Status:      details.StatusText,
	}

	for _, key := range queryOrder {
		q := queryMap[key]
		src := q.Sources[0] // representative source object

		objDiff := transportObjectDiff{
			Object:     src.Object,
			PgmID:      src.PgmID,
			WBType:     src.WBType,
			Info:       src.Info,
			Include:    q.Include,
			MethodName: q.MethodName,
		}

		// Check if this was mapped as unsupported
		if q.ObjectType == "" {
			objDiff.Message = "No diff available: object type not supported for source comparison"
			result.Objects = append(result.Objects, objDiff)
			continue
		}

		ref, refErr := resolveRef(q.ObjectType, q.Name, "", q.Include, "")
		if refErr != nil {
			objDiff.Message = fmt.Sprintf("Cannot resolve object reference: %v", refErr)
			result.Objects = append(result.Objects, objDiff)
			continue
		}
		revisions, err := sys.ADT().GetObjectVersions(ctx, ref)
		if err != nil {
			objDiff.Message = fmt.Sprintf("Cannot retrieve version history: %v", err)
			result.Objects = append(result.Objects, objDiff)
			continue
		}

		if len(revisions) == 0 {
			objDiff.Message = "No version history available for this object"
			result.Objects = append(result.Objects, objDiff)
			continue
		}

		// Revisions are returned newest-first; reverse to chronological order (oldest first)
		chronological := make([]adt.Revision, len(revisions))
		for i, rev := range revisions {
			chronological[len(revisions)-1-i] = rev
		}

		// Find indices of versions that belong to this transport
		var matchingIndices []int
		for i, rev := range chronological {
			if transportNumbers[rev.Transport] {
				matchingIndices = append(matchingIndices, i)
			}
		}

		// Method-level diff path: when we have a specific method to extract
		if q.MethodName != "" {
			diff := computeMethodDiff(ctx, sys, chronological, matchingIndices, transportNumbers, q.MethodName)
			if diff != nil {
				objDiff.Diffs = append(objDiff.Diffs, diff)
			} else {
				objDiff.Message = "Could not compute method diff: method not found in version history or no changes detected"
			}
			result.Objects = append(result.Objects, objDiff)
			continue
		}

		// Standard include/program-level diff path
		if len(matchingIndices) == 0 {
			objDiff.Message = "Object has no version entries matching this transport or its tasks"
			result.Objects = append(result.Objects, objDiff)
			continue
		}

		firstIdx := matchingIndices[0]
		lastIdx := matchingIndices[len(matchingIndices)-1]

		// Determine the base version: the version immediately before the first matching one
		// If firstIdx == 0, there's no predecessor — diff shows full content as added
		var baseURI string
		if firstIdx > 0 {
			baseURI = chronological[firstIdx-1].URI
		}

		// Check for foreign versions between first and last matching indices
		hasForeign := false
		for i := firstIdx; i <= lastIdx; i++ {
			if !transportNumbers[chronological[i].Transport] {
				hasForeign = true
				break
			}
		}

		if !hasForeign {
			// Simple case: single diff from base to latest
			targetURI := chronological[lastIdx].URI
			if baseURI == "" {
				// No predecessor — compare first version against empty (show all as added)
				// Use the first matching version's source directly as a "new file" diff
				diff, diffErr := buildNewFileDiff(ctx, sys, chronological[firstIdx], chronological[lastIdx])
				if diffErr != nil {
					objDiff.Message = fmt.Sprintf("Cannot compute diff: %v", diffErr)
				} else {
					objDiff.Diffs = append(objDiff.Diffs, diff)
				}
			} else {
				diff, diffErr := sys.ADT().CompareObjectVersions(ctx, baseURI, targetURI)
				if diffErr != nil {
					objDiff.Message = fmt.Sprintf("Cannot compute diff: %v", diffErr)
				} else {
					objDiff.Diffs = append(objDiff.Diffs, diff)
				}
			}
		} else {
			// Complex case: foreign versions intervene — produce multiple segment diffs
			diffs := computeSegmentDiffs(ctx, sys, chronological, matchingIndices, transportNumbers)
			if len(diffs) == 0 {
				objDiff.Message = "Could not compute any diffs (errors in all segments)"
			} else {
				objDiff.Diffs = diffs
			}
		}

		result.Objects = append(result.Objects, objDiff)
	}

	output, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(output)), nil
}

// --- Transport Diff Helpers ---

// mapTransportObjectToVersionQueries maps a transport object to zero or more version queries.
// Returns queries with ObjectType="" if the object type is not supported (caller sets message).
func mapTransportObjectToVersionQueries(obj adt.TransportObjectV2) []versionQuery {
	switch {
	// CORR/RELE = release comment entries — not diffable
	case obj.PgmID == "CORR":
		return []versionQuery{{ObjectType: "", Sources: []adt.TransportObjectV2{obj}}}

	// LIMU/METH = individual class method
	case obj.PgmID == "LIMU" && obj.Type == "METH":
		className := extractClassName(obj.Object.Name)
		methodName := extractMethodName(obj.Object.Name)
		if className == "" || methodName == "" {
			return []versionQuery{{ObjectType: "", Sources: []adt.TransportObjectV2{obj}}}
		}
		// The "main" include contains the global class definition and ALL method
		// implementations (the "Global Class" tab in ADT). Method implementations
		// are NOT in the "implementations" include — that one holds local types.
		return []versionQuery{{
			ObjectType: "CLAS",
			Name:       className,
			Include:    "main",
			MethodName: methodName,
			Sources:    []adt.TransportObjectV2{obj},
		}}

	// LIMU/FUNC = function module (needs parent function group — not available in transport metadata)
	case obj.PgmID == "LIMU" && obj.Type == "FUNC":
		return []versionQuery{{ObjectType: "", Sources: []adt.TransportObjectV2{obj}}}

	// LIMU/REPS, LIMU/DYNP, etc. — other LIMU sub-objects
	case obj.PgmID == "LIMU":
		return []versionQuery{{ObjectType: "", Sources: []adt.TransportObjectV2{obj}}}

	// R3TR = main object level
	case obj.PgmID == "R3TR":
		return mapR3TRObject(obj)

	default:
		return []versionQuery{{ObjectType: "", Sources: []adt.TransportObjectV2{obj}}}
	}
}

// mapR3TRObject maps R3TR-level objects to version queries.
// ABAP class includes in ADT correspond to the editor tabs:
//   - "main"            = Global Class (class definition + all method implementations)
//   - "definitions"     = Class-relevant Local Types (local type definitions)
//   - "implementations" = Local Types (local class implementations, helpers)
//   - "testclasses"     = Test Classes (ABAP Unit)
//   - "macros"          = Macros
func mapR3TRObject(obj adt.TransportObjectV2) []versionQuery {
	switch obj.Type {
	case "PROG":
		return []versionQuery{{ObjectType: "PROG", Name: obj.Object.Name, Sources: []adt.TransportObjectV2{obj}}}
	case "CLAS":
		// Full class: diff all five source includes (matching the ADT editor tabs)
		return []versionQuery{
			{ObjectType: "CLAS", Name: obj.Object.Name, Include: "main", Sources: []adt.TransportObjectV2{obj}},
			{ObjectType: "CLAS", Name: obj.Object.Name, Include: "definitions", Sources: []adt.TransportObjectV2{obj}},
			{ObjectType: "CLAS", Name: obj.Object.Name, Include: "implementations", Sources: []adt.TransportObjectV2{obj}},
			{ObjectType: "CLAS", Name: obj.Object.Name, Include: "testclasses", Sources: []adt.TransportObjectV2{obj}},
			{ObjectType: "CLAS", Name: obj.Object.Name, Include: "macros", Sources: []adt.TransportObjectV2{obj}},
		}
	case "INTF":
		return []versionQuery{{ObjectType: "INTF", Name: obj.Object.Name, Sources: []adt.TransportObjectV2{obj}}}
	case "DDLS":
		return []versionQuery{{ObjectType: "DDLS", Name: obj.Object.Name, Sources: []adt.TransportObjectV2{obj}}}
	case "BDEF":
		return []versionQuery{{ObjectType: "BDEF", Name: obj.Object.Name, Sources: []adt.TransportObjectV2{obj}}}
	case "SRVD":
		return []versionQuery{{ObjectType: "SRVD", Name: obj.Object.Name, Sources: []adt.TransportObjectV2{obj}}}
	case "INCL":
		return []versionQuery{{ObjectType: "INCL", Name: obj.Object.Name, Sources: []adt.TransportObjectV2{obj}}}
	case "TABL":
		return []versionQuery{{ObjectType: "TABL", Name: obj.Object.Name, Sources: []adt.TransportObjectV2{obj}}}
	case "FUGR":
		// Function group: cannot diff as a whole (individual function modules need parent)
		return []versionQuery{{ObjectType: "", Sources: []adt.TransportObjectV2{obj}}}
	default:
		// DOMA, DTEL, TTYP, VIEW, ENQU, MSAG, etc. — no source code to diff
		return []versionQuery{{ObjectType: "", Sources: []adt.TransportObjectV2{obj}}}
	}
}

// extractClassName parses the class name from a LIMU/METH transport object name.
// Format: "CLASSNAME              METHODNAME" (class name padded with spaces to 30 chars).
func extractClassName(methodName string) string {
	// The name format is: CLASS_NAME followed by spaces then METHOD_NAME
	// Total class field is 30 characters wide
	if len(methodName) < 31 {
		// Try splitting on multiple spaces
		parts := strings.Fields(methodName)
		if len(parts) >= 1 {
			return strings.TrimSpace(parts[0])
		}
		return ""
	}
	return strings.TrimSpace(methodName[:30])
}

// extractMethodName parses the method name from a LIMU/METH transport object name.
// Format: "CLASSNAME              METHODNAME" (class name padded with spaces to 30 chars).
// For interface methods: "ZIF_FOO~BAR" — the full qualified name is returned.
func extractMethodName(fullName string) string {
	if len(fullName) < 31 {
		parts := strings.Fields(fullName)
		if len(parts) >= 2 {
			return strings.TrimSpace(parts[len(parts)-1])
		}
		return ""
	}
	return strings.TrimSpace(fullName[30:])
}

// buildNewFileDiff creates a diff for an object that has no predecessor version.
// If firstIdx == lastIdx, we just show the single version as all-new content.
// If they differ, we compare first vs last within the transport's versions.
func buildNewFileDiff(ctx context.Context, sys types.System, first, last adt.Revision) (*adt.VersionDiff, error) {
	// Get the latest version's source
	targetSource, err := sys.ADT().GetObjectVersionSource(ctx, last.URI)
	if err != nil {
		return nil, fmt.Errorf("getting target version source: %w", err)
	}

	// Create a diff showing everything as added (compare against empty string)
	result := &adt.VersionDiff{
		BaseVersionURI:   "(new)",
		TargetVersionURI: last.URI,
		Identical:        targetSource == "",
	}

	if result.Identical {
		result.Diff = "Empty source"
		return result, nil
	}

	// Build unified diff from empty to target
	lines := strings.Split(targetSource, "\n")
	var sb strings.Builder
	sb.WriteString("--- /dev/null\n")
	sb.WriteString(fmt.Sprintf("+++ target/%s\n", last.Transport))
	sb.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
	for _, line := range lines {
		sb.WriteString("+" + line + "\n")
	}
	result.Diff = sb.String()
	result.AddedLines = len(lines)

	return result, nil
}

// computeSegmentDiffs handles the case where foreign versions exist between transport versions.
// It produces a diff for each contiguous segment of versions belonging to this transport.
func computeSegmentDiffs(ctx context.Context, sys types.System, chronological []adt.Revision, matchingIndices []int, transportNumbers map[string]bool) []*adt.VersionDiff {
	var diffs []*adt.VersionDiff

	// Group matching indices into contiguous segments.
	// A segment is a maximal run of consecutive matching indices without foreign versions in between.
	type segment struct {
		firstMatchIdx int // index into chronological
		lastMatchIdx  int
	}

	var segments []segment
	segStart := matchingIndices[0]
	prevIdx := matchingIndices[0]

	for i := 1; i < len(matchingIndices); i++ {
		currIdx := matchingIndices[i]
		// Check if there are any foreign versions between prevIdx and currIdx
		hasForeignBetween := false
		for j := prevIdx + 1; j < currIdx; j++ {
			if !transportNumbers[chronological[j].Transport] {
				hasForeignBetween = true
				break
			}
		}
		if hasForeignBetween {
			// End current segment, start new one
			segments = append(segments, segment{segStart, prevIdx})
			segStart = currIdx
		}
		prevIdx = currIdx
	}
	segments = append(segments, segment{segStart, prevIdx})

	// For each segment, produce a diff: version-before-first → last
	for _, seg := range segments {
		var baseURI string
		if seg.firstMatchIdx > 0 {
			baseURI = chronological[seg.firstMatchIdx-1].URI
		}
		targetURI := chronological[seg.lastMatchIdx].URI

		if baseURI == "" {
			// No predecessor — new file diff
			diff, err := buildNewFileDiff(ctx, sys, chronological[seg.firstMatchIdx], chronological[seg.lastMatchIdx])
			if err == nil {
				diffs = append(diffs, diff)
			}
		} else {
			diff, err := sys.ADT().CompareObjectVersions(ctx, baseURI, targetURI)
			if err == nil {
				diffs = append(diffs, diff)
			}
		}
	}

	return diffs
}

// computeMethodDiff produces a diff for a specific method within a class implementations include.
// It handles the case where version history entries may not have transport links for class includes.
// Strategy:
//  1. If transport matching works: use matched versions and extract the method from each.
//  2. If transport matching fails: compare the oldest version vs the newest version,
//     extracting the method from both.
func computeMethodDiff(ctx context.Context, sys types.System, chronological []adt.Revision, matchingIndices []int, transportNumbers map[string]bool, methodName string) *adt.VersionDiff {
	var baseIdx, targetIdx int

	if len(matchingIndices) > 0 {
		// Transport matching succeeded: use version before first match as base, last match as target
		firstMatchIdx := matchingIndices[0]
		lastMatchIdx := matchingIndices[len(matchingIndices)-1]
		if firstMatchIdx > 0 {
			baseIdx = firstMatchIdx - 1
		} else {
			baseIdx = -1 // no predecessor
		}
		targetIdx = lastMatchIdx
	} else {
		// Transport matching failed: fall back to oldest vs newest version
		// This is the common case for class includes where transport links are not populated
		if len(chronological) < 2 {
			// Only one version — try comparing against empty
			baseIdx = -1
			targetIdx = len(chronological) - 1
		} else {
			baseIdx = 0
			targetIdx = len(chronological) - 1
		}
	}

	// Fetch target version source and extract method
	targetSource, err := sys.ADT().GetObjectVersionSource(ctx, chronological[targetIdx].URI)
	if err != nil {
		return nil
	}
	targetMethod := extractMethodFromSource(targetSource, methodName)
	if targetMethod == "" {
		return nil
	}

	// Fetch base version source and extract method (or empty if no predecessor)
	var baseMethod string
	if baseIdx >= 0 {
		baseSource, err := sys.ADT().GetObjectVersionSource(ctx, chronological[baseIdx].URI)
		if err != nil {
			return nil
		}
		baseMethod = extractMethodFromSource(baseSource, methodName)
		// If method doesn't exist in base version, treat as new method (base = "")
	}

	// If sources are identical, no diff needed
	if baseMethod == targetMethod {
		return nil // no changes to this method
	}

	// Build unified diff
	baseLabel := fmt.Sprintf("base/%s", methodName)
	targetLabel := fmt.Sprintf("target/%s", methodName)

	result := &adt.VersionDiff{
		Identical: false,
	}
	if baseIdx >= 0 {
		result.BaseVersionURI = chronological[baseIdx].URI
	} else {
		result.BaseVersionURI = "(new)"
	}
	result.TargetVersionURI = chronological[targetIdx].URI

	if baseMethod == "" {
		// New method: show all lines as added
		lines := strings.Split(targetMethod, "\n")
		var sb strings.Builder
		sb.WriteString("--- /dev/null\n")
		sb.WriteString(fmt.Sprintf("+++ %s\n", targetLabel))
		sb.WriteString(fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines)))
		for _, line := range lines {
			sb.WriteString("+" + line + "\n")
		}
		result.Diff = sb.String()
		result.AddedLines = len(lines)
	} else {
		result.Diff = udiff.Unified(baseLabel, targetLabel, baseMethod, targetMethod)
		for _, line := range strings.Split(result.Diff, "\n") {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				result.AddedLines++
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				result.RemovedLines++
			}
		}
	}

	return result
}

// extractMethodFromSource extracts a METHOD...ENDMETHOD block for the given method name from ABAP source.
// Uses case-insensitive regex matching. Returns empty string if method not found.
// Handles interface method names with ~ (e.g., ZIF_FOO~METHOD_NAME).
func extractMethodFromSource(source, methodName string) string {
	// Escape special regex characters in method name (~ is not special in regex, but be safe)
	escaped := regexp.QuoteMeta(methodName)

	// Match: METHOD method_name followed by optional "." or whitespace, then content, then ENDMETHOD.
	// ABAP is case-insensitive; METHOD/ENDMETHOD can have varying indentation.
	// The dot after METHOD name is required in ABAP syntax.
	pattern := `(?ims)^[ \t]*METHOD\s+` + escaped + `\s*\..*?^[ \t]*ENDMETHOD\s*\.`
	re, err := regexp.Compile(pattern)
	if err != nil {
		return ""
	}

	match := re.FindString(source)
	return match
}
