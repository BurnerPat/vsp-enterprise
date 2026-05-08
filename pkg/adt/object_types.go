package adt

import (
	"fmt"
	"net/url"
	"strings"
)

// ObjectType holds static metadata for a single SAP ADT object type.
// It is registered once at package init and never mutated.
type ObjectType struct {
	// ID is a human-readable identifier used in MCP tool parameters.
	// Examples: "class", "function_module", "cds_view"
	ID string

	// ShortCode is the 4-character SAP type abbreviation (uppercase).
	// Examples: "CLAS", "FUNC", "DDLS"
	ShortCode string

	// ADTType is the qualified ADT XML type (category/sub).
	// Examples: "CLAS/OC", "FUGR/FF", "DDLS/DF"
	ADTType string

	// BasePath is the URL path template for the object (with %s for name).
	// Examples: "/sap/bc/adt/oo/classes/%s", "/sap/bc/adt/ddic/ddl/sources/%s"
	BasePath string

	// LowercaseName indicates the object name must be lowercased in URL paths.
	LowercaseName bool

	// RequiresParent indicates the object lives under a parent (e.g. FUNC under FUGR).
	// When true, ParentBasePath is used instead of BasePath.
	RequiresParent bool

	// ParentBasePath is the URL path template when a parent is required.
	// First %s is parent name, second %s is child name.
	// Example: "/sap/bc/adt/functions/groups/%s/fmodules/%s"
	ParentBasePath string

	// SourceSuffix is appended to the base URI to get the source code endpoint.
	// Most types use "/source/main". Empty means no source endpoint (metadata-only types).
	SourceSuffix string

	// VersionSuffix is appended to the base URI to get the version feed.
	// Examples: "/source/main/versions", "/versions"
	// For classes and interfaces, this is handled specially via VersionIncludePattern.
	VersionSuffix string

	// VersionIncludePattern is used for types where version URL depends on an include parameter.
	// The %s is replaced with the include name (e.g., "main", "definitions").
	// Example: "/includes/%s/versions" (for CLAS/INTF)
	VersionIncludePattern string

	// CreationPath is the ADT endpoint for creating objects of this type.
	// Empty if creation is not supported.
	CreationPath string

	// CreationRootName is the XML root element name for creation requests.
	CreationRootName string

	// CreationNamespace is the XML namespace for creation requests.
	CreationNamespace string
}

// ObjectRef is a resolved, concrete reference to a specific ABAP object.
// It carries the type metadata and all identifiers needed to build any ADT URL.
// Pass *ObjectRef to ADT functions instead of raw (objectType, name string) pairs.
type ObjectRef struct {
	// Type is the resolved object type metadata (never nil after construction).
	Type *ObjectType

	// Name is the object name, normalized (uppercased unless LowercaseName is set).
	Name string

	// Parent is the parent object name (e.g., function group for FUNC).
	// Required when Type.RequiresParent is true.
	Parent string

	// Include specifies the class/interface include (e.g., "main", "definitions",
	// "implementations", "macros", "testclasses").
	// Only relevant for CLAS/INTF types.
	Include string

	// Method specifies a method name for method-level source extraction.
	// Only relevant for CLAS type.
	Method string
}

// RefOption is a functional option for NewObjectRef.
type RefOption func(*ObjectRef)

// WithParent sets the parent name on an ObjectRef.
func WithParent(parent string) RefOption {
	return func(r *ObjectRef) {
		r.Parent = strings.ToUpper(parent)
	}
}

// WithInclude sets the include type on an ObjectRef.
func WithInclude(include string) RefOption {
	return func(r *ObjectRef) {
		r.Include = strings.ToLower(include)
	}
}

// WithMethod sets the method name on an ObjectRef.
func WithMethod(method string) RefOption {
	return func(r *ObjectRef) {
		r.Method = strings.ToUpper(method)
	}
}

// NewObjectRef creates a new ObjectRef from a type identifier and object name.
// The typeID can be either a human-readable ID ("class", "cds_view") or a
// short code ("CLAS", "DDLS") for backward compatibility.
// Returns an error if the type is unknown.
func NewObjectRef(typeID, name string, opts ...RefOption) (*ObjectRef, error) {
	ot := ResolveObjectType(typeID)
	if ot == nil {
		return nil, fmt.Errorf("unknown object type %q (valid: %s)", typeID, validTypeList())
	}

	ref := &ObjectRef{
		Type: ot,
		Name: strings.ToUpper(name),
	}

	for _, opt := range opts {
		opt(ref)
	}

	if ot.RequiresParent && ref.Parent == "" {
		return nil, fmt.Errorf("object type %q requires a parent (use WithParent option)", ot.ID)
	}

	return ref, nil
}

// BaseURI returns the ADT base URI for this object.
func (r *ObjectRef) BaseURI() string {
	name := r.encodedName()

	if r.Type.RequiresParent {
		parent := url.PathEscape(r.Parent)
		return fmt.Sprintf(r.Type.ParentBasePath, parent, name)
	}
	return fmt.Sprintf(r.Type.BasePath, name)
}

// SourceURI returns the ADT source code URI for this object.
// For classes with Include set, returns the include source URL.
// For classes with Method set, still returns the main source URL (method extraction is done client-side).
func (r *ObjectRef) SourceURI() string {
	base := r.BaseURI()

	// Classes/interfaces with includes use a different source path
	if r.Include != "" && (r.Type.ShortCode == "CLAS" || r.Type.ShortCode == "INTF") {
		return base + "/includes/" + url.PathEscape(r.Include)
	}

	if r.Type.SourceSuffix == "" {
		return base
	}
	return base + r.Type.SourceSuffix
}

// VersionURI returns the ADT version feed URI for this object.
func (r *ObjectRef) VersionURI() string {
	base := r.BaseURI()

	// Types with include-based versioning (CLAS, INTF)
	if r.Type.VersionIncludePattern != "" {
		include := r.Include
		if include == "" {
			include = "main"
		}
		return base + fmt.Sprintf(r.Type.VersionIncludePattern, include)
	}

	if r.Type.VersionSuffix == "" {
		return ""
	}
	return base + r.Type.VersionSuffix
}

// String returns a human-readable representation like "class ZCL_MY_CLASS".
func (r *ObjectRef) String() string {
	if r.Parent != "" {
		return fmt.Sprintf("%s %s (parent: %s)", r.Type.ID, r.Name, r.Parent)
	}
	return fmt.Sprintf("%s %s", r.Type.ID, r.Name)
}

// encodedName returns the URL-path-escaped object name, respecting LowercaseName.
func (r *ObjectRef) encodedName() string {
	name := r.Name
	if r.Type.LowercaseName {
		name = strings.ToLower(name)
	}
	return url.PathEscape(name)
}

// --- Object Type Registry ---

// objectTypeRegistry is the central registry of all known object types.
// Keyed by human-readable ID.
var objectTypeRegistry = map[string]*ObjectType{}

// objectTypeByShortCode is an index for lookup by SAP short code.
var objectTypeByShortCode = map[string]*ObjectType{}

// objectTypeByADTType is an index for lookup by qualified ADT type.
var objectTypeByADTType = map[string]*ObjectType{}

func init() {
	types := []*ObjectType{
		{
			ID:                "program",
			ShortCode:         "PROG",
			ADTType:           "PROG/P",
			BasePath:          "/sap/bc/adt/programs/programs/%s",
			SourceSuffix:      "/source/main",
			VersionSuffix:     "/source/main/versions",
			CreationPath:      "/sap/bc/adt/programs/programs",
			CreationRootName:  "program:abapProgram",
			CreationNamespace: `xmlns:program="http://www.sap.com/adt/programs/programs"`,
		},
		{
			ID:                "include",
			ShortCode:         "INCL",
			ADTType:           "PROG/I",
			BasePath:          "/sap/bc/adt/programs/includes/%s",
			SourceSuffix:      "/source/main",
			VersionSuffix:     "/source/main/versions",
			CreationPath:      "/sap/bc/adt/programs/includes",
			CreationRootName:  "include:abapInclude",
			CreationNamespace: `xmlns:include="http://www.sap.com/adt/programs/includes"`,
		},
		{
			ID:                    "class",
			ShortCode:             "CLAS",
			ADTType:               "CLAS/OC",
			BasePath:              "/sap/bc/adt/oo/classes/%s",
			SourceSuffix:          "/source/main",
			VersionIncludePattern: "/includes/%s/versions",
			CreationPath:          "/sap/bc/adt/oo/classes",
			CreationRootName:      "class:abapClass",
			CreationNamespace:     `xmlns:class="http://www.sap.com/adt/oo/classes"`,
		},
		{
			ID:                    "interface",
			ShortCode:             "INTF",
			ADTType:               "INTF/OI",
			BasePath:              "/sap/bc/adt/oo/interfaces/%s",
			SourceSuffix:          "/source/main",
			VersionIncludePattern: "/includes/%s/versions",
			CreationPath:          "/sap/bc/adt/oo/interfaces",
			CreationRootName:      "intf:abapInterface",
			CreationNamespace:     `xmlns:intf="http://www.sap.com/adt/oo/interfaces"`,
		},
		{
			ID:                "function_group",
			ShortCode:         "FUGR",
			ADTType:           "FUGR/F",
			BasePath:          "/sap/bc/adt/functions/groups/%s",
			SourceSuffix:      "/source/main",
			CreationPath:      "/sap/bc/adt/functions/groups",
			CreationRootName:  "group:abapFunctionGroup",
			CreationNamespace: `xmlns:group="http://www.sap.com/adt/functions/groups"`,
		},
		{
			ID:                "function_module",
			ShortCode:         "FUNC",
			ADTType:           "FUGR/FF",
			RequiresParent:    true,
			ParentBasePath:    "/sap/bc/adt/functions/groups/%s/fmodules/%s",
			SourceSuffix:      "/source/main",
			VersionSuffix:     "/source/main/versions",
			CreationPath:      "/sap/bc/adt/functions/groups/%s/fmodules",
			CreationRootName:  "fmodule:abapFunctionModule",
			CreationNamespace: `xmlns:fmodule="http://www.sap.com/adt/functions/fmodules"`,
		},
		{
			ID:                "table",
			ShortCode:         "TABL",
			ADTType:           "TABL/DT",
			BasePath:          "/sap/bc/adt/ddic/tables/%s",
			SourceSuffix:      "/source/main",
			VersionSuffix:     "/source/main/versions",
			LowercaseName:     true,
			CreationPath:      "",
			CreationRootName:  "",
			CreationNamespace: "",
		},
		{
			ID:            "cds_view",
			ShortCode:     "DDLS",
			ADTType:       "DDLS/DF",
			BasePath:      "/sap/bc/adt/ddic/ddl/sources/%s",
			LowercaseName: true,
			SourceSuffix:  "/source/main",
			// DDLS uses /versions directly (no /source/main prefix)
			VersionSuffix:     "/versions",
			CreationPath:      "/sap/bc/adt/ddic/ddl/sources",
			CreationRootName:  "ddl:ddlSource",
			CreationNamespace: `xmlns:ddl="http://www.sap.com/adt/ddic/ddlsources"`,
		},
		{
			ID:        "data_element",
			ShortCode: "DTEL",
			ADTType:   "DTEL/DE",
			BasePath:  "/sap/bc/adt/ddic/dataelements/%s",
		},
		{
			ID:        "domain",
			ShortCode: "DOMA",
			ADTType:   "DOMA/DD",
			BasePath:  "/sap/bc/adt/ddic/domains/%s",
		},
		{
			ID:                "service_definition",
			ShortCode:         "SRVD",
			ADTType:           "SRVD/SRV",
			BasePath:          "/sap/bc/adt/ddic/srvd/sources/%s",
			LowercaseName:     true,
			SourceSuffix:      "/source/main",
			VersionSuffix:     "/source/main/versions",
			CreationPath:      "/sap/bc/adt/ddic/srvd/sources",
			CreationRootName:  "srvd:srvdSource",
			CreationNamespace: `xmlns:srvd="http://www.sap.com/adt/ddic/srvdsources"`,
		},
		{
			ID:                "behavior_definition",
			ShortCode:         "BDEF",
			ADTType:           "BDEF/BDO",
			BasePath:          "/sap/bc/adt/bo/behaviordefinitions/%s",
			LowercaseName:     true,
			SourceSuffix:      "/source/main",
			VersionSuffix:     "/source/main/versions",
			CreationPath:      "/sap/bc/adt/bo/behaviordefinitions",
			CreationRootName:  "bdef:behaviorDefinition",
			CreationNamespace: `xmlns:bdef="http://www.sap.com/adt/bo/behaviordefinitions"`,
		},
		{
			ID:                "service_binding",
			ShortCode:         "SRVB",
			ADTType:           "SRVB/SVB",
			BasePath:          "/sap/bc/adt/businessservices/bindings/%s",
			LowercaseName:     true,
			CreationPath:      "/sap/bc/adt/businessservices/bindings",
			CreationRootName:  "srvb:serviceBinding",
			CreationNamespace: `xmlns:srvb="http://www.sap.com/adt/ddic/ServiceBindings"`,
		},
		{
			ID:        "transformation",
			ShortCode: "XSLT",
			ADTType:   "XSLT/VT",
			BasePath:  "/sap/bc/adt/xslt/transformations/%s",
		},
		{
			ID:        "message_class",
			ShortCode: "MSAG",
			ADTType:   "MSAG/N",
			BasePath:  "/sap/bc/adt/messageclass/%s",
		},
		{
			ID:        "view",
			ShortCode: "VIEW",
			ADTType:   "",
			BasePath:  "/sap/bc/adt/ddic/views/%s",
		},
		{
			ID:                "package",
			ShortCode:         "DEVC",
			ADTType:           "DEVC/K",
			BasePath:          "/sap/bc/adt/packages/%s",
			CreationPath:      "/sap/bc/adt/packages",
			CreationRootName:  "pack:package",
			CreationNamespace: `xmlns:pack="http://www.sap.com/adt/packages"`,
		},
	}

	for _, t := range types {
		objectTypeRegistry[t.ID] = t
		objectTypeByShortCode[t.ShortCode] = t
		if t.ADTType != "" {
			objectTypeByADTType[t.ADTType] = t
		}
	}
}

// ResolveObjectType resolves a type identifier to an *ObjectType.
// Accepts human-readable IDs ("class", "cds_view"), short codes ("CLAS", "DDLS"),
// or qualified ADT types ("CLAS/OC", "DDLS/DF").
// Returns nil if not found.
func ResolveObjectType(typeID string) *ObjectType {
	// Try human-readable ID first (lowercase)
	if ot, ok := objectTypeRegistry[strings.ToLower(typeID)]; ok {
		return ot
	}
	// Try short code (uppercase)
	if ot, ok := objectTypeByShortCode[strings.ToUpper(typeID)]; ok {
		return ot
	}
	// Try qualified ADT type (e.g., "CLAS/OC")
	if ot, ok := objectTypeByADTType[strings.ToUpper(typeID)]; ok {
		return ot
	}
	return nil
}

// ObjectTypeByID looks up an object type by its human-readable ID.
// Returns nil if not found.
func ObjectTypeByID(id string) *ObjectType {
	return objectTypeRegistry[strings.ToLower(id)]
}

// ObjectTypeByShortCode looks up an object type by its SAP short code.
// Returns nil if not found.
func ObjectTypeByShortCode(code string) *ObjectType {
	return objectTypeByShortCode[strings.ToUpper(code)]
}

// ObjectTypeByADTType looks up an object type by its qualified ADT type string.
// Returns nil if not found.
func ObjectTypeByADTType(adtType string) *ObjectType {
	return objectTypeByADTType[strings.ToUpper(adtType)]
}

// AllObjectTypes returns all registered object types.
func AllObjectTypes() []*ObjectType {
	result := make([]*ObjectType, 0, len(objectTypeRegistry))
	for _, ot := range objectTypeRegistry {
		result = append(result, ot)
	}
	return result
}

// validTypeList returns a comma-separated list of valid type identifiers for error messages.
func validTypeList() string {
	ids := make([]string, 0, len(objectTypeRegistry))
	for id := range objectTypeRegistry {
		ids = append(ids, id)
	}
	return strings.Join(ids, ", ")
}

// SupportedTypeDescription returns a formatted string listing all supported types
// suitable for MCP tool parameter descriptions.
func SupportedTypeDescription() string {
	return "class, interface, program, include, function_module, function_group, " +
		"cds_view, table, data_element, domain, service_definition, behavior_definition, " +
		"service_binding, transformation, message_class, view, package"
}

// --- Unified Response Serialization ---

// RefOutput is the unified JSON-serializable representation of an ABAP object reference.
// Used in all API response structs instead of raw type/name/URI string fields.
type RefOutput struct {
	Type string `json:"type"`          // Human-readable ID: "class", "program", etc.
	Name string `json:"name"`          // Object name: "ZCL_FOO"
	URI  string `json:"uri,omitempty"` // ADT URI if available
}

// NewRefOutput creates a RefOutput by resolving an ADT type string to a human-readable ID.
// Accepts qualified ADT types ("CLAS/OC"), short codes ("CLAS"), or human-readable IDs ("class").
// If the type is not recognized, it falls through to use the raw type string.
func NewRefOutput(adtType, name, uri string) RefOutput {
	humanType := adtType
	if ot := ResolveObjectType(adtType); ot != nil {
		humanType = ot.ID
	}
	return RefOutput{Type: humanType, Name: name, URI: uri}
}

// RefOutputFromRef creates a RefOutput from an existing ObjectRef.
func RefOutputFromRef(ref *ObjectRef) RefOutput {
	return RefOutput{Type: ref.Type.ID, Name: ref.Name, URI: ref.BaseURI()}
}
