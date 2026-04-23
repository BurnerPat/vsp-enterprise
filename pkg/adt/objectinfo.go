package adt

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/oisee/vibing-steampunk/pkg/adt/connection"
)

// objectURIFromType maps a short object type string and name to an ADT URI path.
func objectURIFromType(objectType, name string) (string, error) {
	objectType = strings.ToUpper(objectType)
	name = strings.ToUpper(name)
	encoded := url.PathEscape(name)

	switch objectType {
	case "CLAS":
		return fmt.Sprintf("/sap/bc/adt/oo/classes/%s", encoded), nil
	case "INTF":
		return fmt.Sprintf("/sap/bc/adt/oo/interfaces/%s", encoded), nil
	case "PROG":
		return fmt.Sprintf("/sap/bc/adt/programs/programs/%s", encoded), nil
	case "FUGR":
		return fmt.Sprintf("/sap/bc/adt/functions/groups/%s", encoded), nil
	case "FUNC":
		return fmt.Sprintf("/sap/bc/adt/functions/groups/%s", encoded), nil
	case "TABL":
		return fmt.Sprintf("/sap/bc/adt/ddic/tables/%s", encoded), nil
	case "DDLS":
		return fmt.Sprintf("/sap/bc/adt/ddic/ddl/sources/%s", url.PathEscape(strings.ToLower(name))), nil
	case "DTEL":
		return fmt.Sprintf("/sap/bc/adt/ddic/dataelements/%s", encoded), nil
	case "DOMA":
		return fmt.Sprintf("/sap/bc/adt/ddic/domains/%s", encoded), nil
	case "SRVD":
		return fmt.Sprintf("/sap/bc/adt/ddic/srvd/sources/%s", url.PathEscape(strings.ToLower(name))), nil
	case "BDEF":
		return fmt.Sprintf("/sap/bc/adt/bo/behaviordefinitions/%s", url.PathEscape(strings.ToLower(name))), nil
	case "SRVB":
		return fmt.Sprintf("/sap/bc/adt/businessservices/bindings/%s", url.PathEscape(strings.ToLower(name))), nil
	case "XSLT":
		return fmt.Sprintf("/sap/bc/adt/xslt/transformations/%s", encoded), nil
	case "MSAG":
		return fmt.Sprintf("/sap/bc/adt/messageclass/%s", encoded), nil
	default:
		return "", fmt.Errorf("unsupported object type: %s", objectType)
	}
}

// adtTypeFromShort maps a short object type to the ADT type identifier used in XML bodies.
func adtTypeFromShort(objectType string) string {
	switch strings.ToUpper(objectType) {
	case "CLAS":
		return "CLAS/OC"
	case "INTF":
		return "INTF/OI"
	case "PROG":
		return "PROG/P"
	case "FUGR":
		return "FUGR/F"
	case "FUNC":
		return "FUGR/FF"
	case "TABL":
		return "TABL/DT"
	case "DDLS":
		return "DDLS/DF"
	case "DTEL":
		return "DTEL/DE"
	case "DOMA":
		return "DOMA/DD"
	case "SRVD":
		return "SRVD/SRV"
	case "BDEF":
		return "BDEF/BDO"
	case "SRVB":
		return "SRVB/SVB"
	case "XSLT":
		return "XSLT/VT"
	case "MSAG":
		return "MSAG/N"
	default:
		return objectType
	}
}

// --- Object Properties ---

// ObjectProperty represents a single facet-value pair from the object properties response.
type ObjectProperty struct {
	Facet       string `json:"facet"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	URI         string `json:"uri,omitempty"`
	Text        string `json:"text,omitempty"`
}

// ObjectProperties contains metadata about an ABAP object.
type ObjectProperties struct {
	URI         string           `json:"uri"`
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Package     string           `json:"package,omitempty"`
	Owner       string           `json:"owner,omitempty"`
	CreatedYear string           `json:"createdYear,omitempty"`
	Language    string           `json:"language,omitempty"`
	System      string           `json:"system,omitempty"`
	APIState    string           `json:"apiState,omitempty"`
	ObjectType  string           `json:"objectType,omitempty"`
	Properties  []ObjectProperty `json:"properties,omitempty"`
}

// GetObjectProperties retrieves general metadata about an ABAP object.
func (c *Client) GetObjectProperties(ctx context.Context, objectType, name string) (*ObjectProperties, error) {
	objectURI, err := objectURIFromType(objectType, name)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("uri", objectURI)

	resp, err := c.sendRequest(ctx, "/sap/bc/adt/repository/informationsystem/objectproperties/values", &connection.Request{
		Method: http.MethodGet,
		Query:  query,
		Accept: "application/vnd.sap.adt.repository.objproperties.result.v1+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("get object properties failed: %w", err)
	}

	return parseObjectProperties(resp.Body)
}

func parseObjectProperties(data []byte) (*ObjectProperties, error) {
	xmlStr := string(data)
	xmlStr = strings.ReplaceAll(xmlStr, "opr:", "")
	xmlStr = strings.ReplaceAll(xmlStr, "atom:", "")

	type xmlProperty struct {
		Facet       string `xml:"facet,attr"`
		Name        string `xml:"name,attr"`
		DisplayName string `xml:"displayName,attr"`
		URI         string `xml:"uri,attr"`
		Text        string `xml:"text,attr"`
	}
	type xmlObject struct {
		URI     string `xml:"uri,attr"`
		Name    string `xml:"name,attr"`
		Text    string `xml:"text,attr"`
		Package string `xml:"package,attr"`
		Type    string `xml:"type,attr"`
	}
	type xmlResponse struct {
		URI        string        `xml:"uri,attr"`
		Name       string        `xml:"name,attr"`
		Properties []xmlProperty `xml:"property"`
		Object     xmlObject     `xml:"object"`
	}

	var resp xmlResponse
	if err := xml.Unmarshal([]byte(xmlStr), &resp); err != nil {
		return nil, fmt.Errorf("parsing object properties: %w", err)
	}

	result := &ObjectProperties{
		URI:         resp.URI,
		Name:        resp.Name,
		Description: resp.Object.Text,
		ObjectType:  resp.Object.Type,
	}

	for _, p := range resp.Properties {
		result.Properties = append(result.Properties, ObjectProperty{
			Facet:       p.Facet,
			Name:        p.Name,
			DisplayName: p.DisplayName,
			URI:         p.URI,
			Text:        p.Text,
		})

		switch p.Facet {
		case "PACKAGE":
			if result.Package == "" {
				result.Package = p.Name
			}
		case "OWNER":
			result.Owner = p.Name
		case "CREATED":
			result.CreatedYear = p.Name
		case "LANGUAGE":
			result.Language = p.DisplayName
		case "SYSTEM":
			result.System = p.Name
		case "API":
			result.APIState = p.Name
		}
	}

	return result, nil
}

// --- Object Outline ---

// OutlineElement represents a node in the object structure tree.
type OutlineElement struct {
	Name        string           `json:"name"`
	Type        string           `json:"type"`
	Description string           `json:"description,omitempty"`
	Href        string           `json:"href,omitempty"`
	Visibility  string           `json:"visibility,omitempty"`
	IsStatic    bool             `json:"isStatic,omitempty"`
	IsFinal     bool             `json:"isFinal,omitempty"`
	IsAbstract  bool             `json:"isAbstract,omitempty"`
	IsConstant  bool             `json:"isConstant,omitempty"`
	IsReadOnly  bool             `json:"isReadOnly,omitempty"`
	Children    []OutlineElement `json:"children,omitempty"`
}

// GetObjectOutline retrieves the structural outline (methods, fields, components) of an ABAP object.
func (c *Client) GetObjectOutline(ctx context.Context, objectType, name string, includeInherited bool) (*OutlineElement, error) {
	objectURI, err := objectURIFromType(objectType, name)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf("%s/objectstructure", objectURI)

	query := url.Values{}
	query.Set("version", "active")
	query.Set("withShortDescriptions", "true")
	if includeInherited {
		query.Set("inheritedMembers", "true")
	}

	resp, err := c.sendRequest(ctx, endpoint, &connection.Request{
		Method: http.MethodGet,
		Query:  query,
		Accept: "application/vnd.sap.adt.objectstructure.v2+xml, application/vnd.sap.adt.objectstructure+xml;q=0.9, application/xml;q=0.8",
	})
	if err != nil {
		return nil, fmt.Errorf("get object outline failed: %w", err)
	}

	return parseObjectOutline(resp.Body, endpoint)
}

// resolveHref resolves a potentially relative href against a base path.
func resolveHref(href, basePath string) string {
	if href == "" || strings.HasPrefix(href, "/") {
		return href
	}
	// Parse base to get the directory part (strip last segment)
	base, err := url.Parse(basePath)
	if err != nil {
		return href
	}
	ref, err := url.Parse(href)
	if err != nil {
		return href
	}
	return base.ResolveReference(ref).String()
}

func parseObjectOutline(data []byte, basePath string) (*OutlineElement, error) {
	xmlStr := string(data)
	xmlStr = strings.ReplaceAll(xmlStr, "abapsource:", "")
	xmlStr = strings.ReplaceAll(xmlStr, "adtcore:", "")
	xmlStr = strings.ReplaceAll(xmlStr, "atom:", "")

	type xmlLink struct {
		Href string `xml:"href,attr"`
		Rel  string `xml:"rel,attr"`
	}
	type xmlElement struct {
		Name        string       `xml:"name,attr"`
		Type        string       `xml:"type,attr"`
		Description string       `xml:"description,attr"`
		Visibility  string       `xml:"visibility,attr"`
		IsStatic    bool         `xml:"isStatic,attr"`
		IsFinal     bool         `xml:"isFinal,attr"`
		IsAbstract  bool         `xml:"isAbstract,attr"`
		IsConstant  bool         `xml:"constant,attr"`
		IsReadOnly  bool         `xml:"readOnly,attr"`
		Links       []xmlLink    `xml:"link"`
		Children    []xmlElement `xml:"objectStructureElement"`
	}

	var root xmlElement
	if err := xml.Unmarshal([]byte(xmlStr), &root); err != nil {
		return nil, fmt.Errorf("parsing object outline: %w", err)
	}

	var convert func(*xmlElement) *OutlineElement
	convert = func(e *xmlElement) *OutlineElement {
		elem := &OutlineElement{
			Name:        e.Name,
			Type:        e.Type,
			Description: e.Description,
			Visibility:  e.Visibility,
			IsStatic:    e.IsStatic,
			IsFinal:     e.IsFinal,
			IsAbstract:  e.IsAbstract,
			IsConstant:  e.IsConstant,
			IsReadOnly:  e.IsReadOnly,
		}
		for _, l := range e.Links {
			if strings.Contains(l.Rel, "definitionIdentifier") {
				elem.Href = resolveHref(l.Href, basePath)
				break
			}
		}
		if elem.Href == "" && len(e.Links) > 0 {
			elem.Href = resolveHref(e.Links[0].Href, basePath)
		}
		for _, child := range e.Children {
			childCopy := child
			elem.Children = append(elem.Children, *convert(&childCopy))
		}
		return elem
	}

	return convert(&root), nil
}

// --- Object Network (Relations) ---

// ObjectRelation represents a directed relationship between two objects.
type ObjectRelation struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	RelationType string `json:"relationType"`
	State        string `json:"state"`
}

// ObjectNetwork contains the dependency graph for an ABAP object.
type ObjectNetwork struct {
	Relations  []ObjectRelation  `json:"relations"`
	References []ObjectReference `json:"references"`
}

// GetObjectNetwork retrieves the dependency network (used objects) for an ABAP object.
func (c *Client) GetObjectNetwork(ctx context.Context, objectType, name string) (*ObjectNetwork, error) {
	objectURI, err := objectURIFromType(objectType, name)
	if err != nil {
		return nil, err
	}

	adtType := adtTypeFromShort(objectType)

	body := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<oro:request xmlns:adtcore="http://www.sap.com/adt/core" xmlns:oro="http://www.sap.com/adt/objectrelations">
  <oro:reference adtcore:name="%s" adtcore:type="%s" adtcore:uri="%s"/>
</oro:request>`, strings.ToUpper(name), adtType, objectURI)

	resp, err := c.sendRequest(ctx, "/sap/bc/adt/objectrelations/network", &connection.Request{
		Method:      http.MethodPost,
		Body:        []byte(body),
		ContentType: "application/vnd.sap.adt.objectrelations.request.v1+xml",
		Accept:      "application/vnd.sap.adt.objectrelations.response.network.v1+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("get object network failed: %w", err)
	}

	return parseObjectNetwork(resp.Body)
}

func parseObjectNetwork(data []byte) (*ObjectNetwork, error) {
	xmlStr := string(data)
	xmlStr = strings.ReplaceAll(xmlStr, "oro:", "")
	xmlStr = strings.ReplaceAll(xmlStr, "adtcore:", "")

	type xmlRelation struct {
		RelationType string `xml:"relationType,attr"`
		State        string `xml:"state,attr"`
		Object1      string `xml:"object1"`
		Object2      string `xml:"object2"`
	}
	type xmlObjRef struct {
		URI         string `xml:"uri,attr"`
		Type        string `xml:"type,attr"`
		Name        string `xml:"name,attr"`
		PackageName string `xml:"packageName,attr"`
		Description string `xml:"description,attr"`
	}
	type xmlResponse struct {
		Relations  []xmlRelation `xml:"relation"`
		References []xmlObjRef   `xml:"objectReference"`
	}

	var resp xmlResponse
	if err := xml.Unmarshal([]byte(xmlStr), &resp); err != nil {
		return nil, fmt.Errorf("parsing object network: %w", err)
	}

	result := &ObjectNetwork{}
	for _, r := range resp.Relations {
		result.Relations = append(result.Relations, ObjectRelation{
			Source:       r.Object1,
			Target:       r.Object2,
			RelationType: r.RelationType,
			State:        r.State,
		})
	}
	for _, ref := range resp.References {
		result.References = append(result.References, ObjectReference{
			URI:         ref.URI,
			Type:        ref.Type,
			Name:        ref.Name,
			Description: ref.Description,
			PackageName: ref.PackageName,
		})
	}

	return result, nil
}

// --- Where-Used List ---

// WhereUsedReference represents an object that references the searched target.
type WhereUsedReference struct {
	URI              string            `json:"uri"`
	ParentURI        string            `json:"parentUri,omitempty"`
	Name             string            `json:"name"`
	Type             string            `json:"type,omitempty"`
	Description      string            `json:"description,omitempty"`
	PackageName      string            `json:"packageName,omitempty"`
	ObjectIdentifier string            `json:"objectIdentifier,omitempty"`
	UsageInformation string            `json:"usageInformation,omitempty"`
	Snippet          *WhereUsedSnippet `json:"snippet,omitempty"`
}

// WhereUsedSnippet represents a code snippet showing a usage location.
type WhereUsedSnippet struct {
	URI     string `json:"uri"`
	Content string `json:"content"`
}

// WhereUsedResult contains the where-used list and optional code snippets.
type WhereUsedResult struct {
	NumberOfResults   int                  `json:"numberOfResults"`
	ResultDescription string               `json:"resultDescription"`
	References        []WhereUsedReference `json:"references"`
}

// rawSnippet is an internal type for snippet data before merging into references.
type rawSnippet struct {
	objectIdentifier string
	URI              string
	Content          string
}

// GetWhereUsed retrieves the where-used list for an ABAP object or member.
// If memberURI is non-empty, it is used as the search URI (e.g., from outline href with position).
// If includeSnippets is true, code snippets for each usage are fetched in a second request.
func (c *Client) GetWhereUsed(ctx context.Context, objectType, name string, memberURI string, includeSnippets bool) (*WhereUsedResult, error) {
	var searchURI string
	if memberURI != "" {
		searchURI = memberURI
	} else {
		uri, err := objectURIFromType(objectType, name)
		if err != nil {
			return nil, err
		}
		searchURI = uri
	}

	body := `<?xml version="1.0" encoding="ASCII"?>
<usagereferences:usageReferenceRequest xmlns:usagereferences="http://www.sap.com/adt/ris/usageReferences">
  <usagereferences:affectedObjects/>
</usagereferences:usageReferenceRequest>`

	endpoint := fmt.Sprintf("/sap/bc/adt/repository/informationsystem/usageReferences?uri=%s",
		url.QueryEscape(searchURI))

	resp, err := c.sendRequest(ctx, endpoint, &connection.Request{
		Method:      http.MethodPost,
		Body:        []byte(body),
		ContentType: "application/vnd.sap.adt.repository.usagereferences.request.v1+xml",
		Accept:      "application/vnd.sap.adt.repository.usagereferences.result.v1+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("get where-used list failed: %w", err)
	}

	result, err := parseWhereUsedResult(resp.Body)
	if err != nil {
		return nil, err
	}

	if includeSnippets && len(result.References) > 0 {
		// Collect objectIdentifiers for snippet request
		var identifiers []string
		for _, ref := range result.References {
			if ref.ObjectIdentifier != "" {
				identifiers = append(identifiers, ref.ObjectIdentifier)
			}
		}
		if len(identifiers) > 0 {
			snippets, err := c.getWhereUsedSnippets(ctx, identifiers)
			if err == nil {
				// Merge snippets into their matching references by objectIdentifier
				snipMap := make(map[string]*WhereUsedSnippet, len(snippets))
				for i := range snippets {
					snipMap[snippets[i].objectIdentifier] = &WhereUsedSnippet{
						URI:     snippets[i].URI,
						Content: snippets[i].Content,
					}
				}
				for i := range result.References {
					if s, ok := snipMap[result.References[i].ObjectIdentifier]; ok {
						result.References[i].Snippet = s
					}
				}
			}
		}
	}

	return result, nil
}

func parseWhereUsedResult(data []byte) (*WhereUsedResult, error) {
	xmlStr := string(data)
	xmlStr = strings.ReplaceAll(xmlStr, "usageReferences:", "")
	xmlStr = strings.ReplaceAll(xmlStr, "adtcore:", "")

	type xmlPackageRef struct {
		URI  string `xml:"uri,attr"`
		Name string `xml:"name,attr"`
	}
	type xmlAdtObject struct {
		Name        string        `xml:"name,attr"`
		Type        string        `xml:"type,attr"`
		Responsible string        `xml:"responsible,attr"`
		Description string        `xml:"description,attr"`
		PackageRef  xmlPackageRef `xml:"packageRef"`
	}
	type xmlRefObject struct {
		URI              string       `xml:"uri,attr"`
		ParentURI        string       `xml:"parentUri,attr"`
		UsageInformation string       `xml:"usageInformation,attr"`
		AdtObject        xmlAdtObject `xml:"adtObject"`
		ObjectIdentifier string       `xml:"objectIdentifier"`
	}
	type xmlRefObjects struct {
		Objects []xmlRefObject `xml:"referencedObject"`
	}
	type xmlResponse struct {
		NumberOfResults   int           `xml:"numberOfResults,attr"`
		ResultDescription string        `xml:"resultDescription,attr"`
		ReferencedObjects xmlRefObjects `xml:"referencedObjects"`
	}

	var resp xmlResponse
	if err := xml.Unmarshal([]byte(xmlStr), &resp); err != nil {
		return nil, fmt.Errorf("parsing where-used result: %w", err)
	}

	result := &WhereUsedResult{
		NumberOfResults:   resp.NumberOfResults,
		ResultDescription: resp.ResultDescription,
	}

	for _, obj := range resp.ReferencedObjects.Objects {
		ref := WhereUsedReference{
			URI:              obj.URI,
			ParentURI:        obj.ParentURI,
			Name:             obj.AdtObject.Name,
			Type:             obj.AdtObject.Type,
			Description:      obj.AdtObject.Description,
			PackageName:      obj.AdtObject.PackageRef.Name,
			ObjectIdentifier: obj.ObjectIdentifier,
			UsageInformation: obj.UsageInformation,
		}
		result.References = append(result.References, ref)
	}

	return result, nil
}

func (c *Client) getWhereUsedSnippets(ctx context.Context, identifiers []string) ([]rawSnippet, error) {
	var idElements strings.Builder
	for _, id := range identifiers {
		fmt.Fprintf(&idElements, `    <usagereferences:objectIdentifier optional="false">%s</usagereferences:objectIdentifier>`+"\n", id)
	}

	body := fmt.Sprintf(`<?xml version="1.0" encoding="ASCII"?>
<usagereferences:usageSnippetRequest xmlns:usagereferences="http://www.sap.com/adt/ris/usageReferences">
  <usagereferences:objectIdentifiers>
%s  </usagereferences:objectIdentifiers>
  <usagereferences:affectedObjects/>
</usagereferences:usageSnippetRequest>`, idElements.String())

	resp, err := c.sendRequest(ctx, "/sap/bc/adt/repository/informationsystem/usageSnippets", &connection.Request{
		Method:      http.MethodPost,
		Body:        []byte(body),
		ContentType: "application/vnd.sap.adt.repository.usagesnippets.request.v1+xml",
		Accept:      "application/vnd.sap.adt.repository.usagesnippets.result.v1+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("get usage snippets failed: %w", err)
	}

	return parseWhereUsedSnippets(resp.Body)
}

func parseWhereUsedSnippets(data []byte) ([]rawSnippet, error) {
	xmlStr := string(data)
	xmlStr = strings.ReplaceAll(xmlStr, "usageReferences:", "")

	type xmlCodeSnippet struct {
		URI         string `xml:"uri,attr"`
		Content     string `xml:"content"`
		Description string `xml:"description"`
	}
	type xmlSnippetObject struct {
		ObjectIdentifier string           `xml:"objectIdentifier"`
		CodeSnippets     []xmlCodeSnippet `xml:"codeSnippets>codeSnippet"`
	}
	type xmlResponse struct {
		SnippetObjects []xmlSnippetObject `xml:"codeSnippetObjects>codeSnippetObject"`
	}

	var resp xmlResponse
	if err := xml.Unmarshal([]byte(xmlStr), &resp); err != nil {
		return nil, fmt.Errorf("parsing usage snippets: %w", err)
	}

	var snippets []rawSnippet
	for _, obj := range resp.SnippetObjects {
		for _, s := range obj.CodeSnippets {
			snippets = append(snippets, rawSnippet{
				objectIdentifier: obj.ObjectIdentifier,
				URI:              s.URI,
				Content:          s.Content,
			})
		}
	}

	return snippets, nil
}
