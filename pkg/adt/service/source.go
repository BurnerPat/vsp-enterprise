package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/oisee/vibing-steampunk/pkg/adt/transport"
)

// SourceService provides read operations for ABAP repository objects.
type SourceService interface {
	SearchObject(ctx context.Context, query string, maxResults int) ([]SearchResult, error)
	GetProgram(ctx context.Context, programName string) (string, error)
	GetClass(ctx context.Context, className string) (map[string]string, error)
	GetClassSource(ctx context.Context, className string) (string, error)
	GetInterface(ctx context.Context, interfaceName string) (string, error)
	GetFunction(ctx context.Context, functionName, groupName string) (string, error)
	GetFunctionGroup(ctx context.Context, groupName string) (*FunctionGroup, error)
	GetInclude(ctx context.Context, includeName string) (string, error)
	GetDDLS(ctx context.Context, ddlsName string) (string, error)
	GetBDEF(ctx context.Context, bdefName string) (string, error)
	GetSRVD(ctx context.Context, srvdName string) (string, error)
	GetTable(ctx context.Context, tableName string) (string, error)
	GetView(ctx context.Context, viewName string) (string, error)
	GetStructure(ctx context.Context, structName string) (string, error)
	GetPackage(ctx context.Context, packageName string) (*PackageContent, error)
	GetMessageClass(ctx context.Context, msgClassName string) (*MessageClass, error)
	GetTransaction(ctx context.Context, tcode string) (*Transaction, error)
	GetTypeInfo(ctx context.Context, typeName string) (*TypeInfo, error)
	GetTableContents(ctx context.Context, tableName string, maxRows int, sqlFilter string) (*TableContentsResult, error)
	RunQuery(ctx context.Context, sqlQuery string, maxRows int) (*TableContentsResult, error)
}

// --------------------------------------------------------------------------
// Types used by SourceService (previously in client.go)
// --------------------------------------------------------------------------

// SearchResult represents a single search result.
type SearchResult struct {
	URI         string `json:"uri"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	PackageName string `json:"packageName,omitempty"`
	Description string `json:"description,omitempty"`
}

// FunctionGroup represents a function group.
type FunctionGroup struct {
	Name        string `xml:"name,attr" json:"name"`
	Description string `xml:"description,attr" json:"description"`
}

// PackageContent represents the contents of an ABAP package.
type PackageContent struct {
	Name        string          `json:"name"`
	Objects     []PackageObject `json:"objects"`
	SubPackages []string        `json:"subPackages"`
}

// PackageObject represents an object within a package.
type PackageObject struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	URI         string `json:"uri"`
	Description string `json:"description,omitempty"`
}

// MessageClassMessage represents a single message in a message class.
type MessageClassMessage struct {
	Number string `xml:"msgno,attr" json:"number"`
	Text   string `xml:"msgtext,attr" json:"text"`
}

// MessageClass represents an ABAP message class.
type MessageClass struct {
	Name        string                `xml:"name,attr" json:"name"`
	Description string                `xml:"description,attr" json:"description"`
	Messages    []MessageClassMessage `xml:"messages" json:"messages"`
}

// Transaction represents an SAP transaction.
type Transaction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Program     string `json:"program"`
}

// TypeInfo represents data type information.
type TypeInfo struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Length      int    `json:"length"`
	Decimals    int    `json:"decimals"`
}

// TableContentsResult represents the result of a table contents query.
type TableContentsResult struct {
	Columns []TableColumn            `json:"columns"`
	Rows    []map[string]interface{} `json:"rows"`
}

// TableColumn represents a column in table contents.
type TableColumn struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Length      int    `json:"length"`
	IsKey       bool   `json:"isKey"`
}

// --------------------------------------------------------------------------
// sourceService implementation
// --------------------------------------------------------------------------

type sourceService struct {
	baseService
}

// NewSourceService creates a new SourceService.
func NewSourceService(sender transport.Sender, safety SafetyChecker, cfg ServiceConfig) SourceService {
	return &sourceService{baseService{sender: sender, safety: safety, config: cfg}}
}

func (s *sourceService) SearchObject(ctx context.Context, query string, maxResults int) ([]SearchResult, error) {
	if maxResults <= 0 {
		maxResults = 100
	}

	params := url.Values{}
	params.Set("operation", "quickSearch")
	params.Set("query", query)
	params.Set("maxResults", fmt.Sprintf("%d", maxResults))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   "/sap/bc/adt/repository/informationsystem/search",
		Method: http.MethodGet,
		Query:  params,
		Accept: "application/xml",
	})
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}

	return parseSearchResults(resp.Body)
}

func (s *sourceService) GetProgram(ctx context.Context, programName string) (string, error) {
	programName = strings.ToUpper(programName)
	sourcePath := fmt.Sprintf("/sap/bc/adt/programs/programs/%s/source/main", url.PathEscape(programName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
	})
	if err != nil {
		return "", fmt.Errorf("getting program source: %w", err)
	}
	return string(resp.Body), nil
}

func (s *sourceService) GetClass(ctx context.Context, className string) (map[string]string, error) {
	className = strings.ToUpper(className)
	sourcePath := fmt.Sprintf("/sap/bc/adt/oo/classes/%s/source/main", url.PathEscape(className))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
	})
	if err != nil {
		return nil, fmt.Errorf("getting class source: %w", err)
	}

	sources := make(map[string]string)
	sources["main"] = string(resp.Body)
	return sources, nil
}

func (s *sourceService) GetClassSource(ctx context.Context, className string) (string, error) {
	sources, err := s.GetClass(ctx, className)
	if err != nil {
		return "", err
	}
	return sources["main"], nil
}

func (s *sourceService) GetInterface(ctx context.Context, interfaceName string) (string, error) {
	interfaceName = strings.ToUpper(interfaceName)
	sourcePath := fmt.Sprintf("/sap/bc/adt/oo/interfaces/%s/source/main", url.PathEscape(interfaceName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
	})
	if err != nil {
		return "", fmt.Errorf("getting interface source: %w", err)
	}
	return string(resp.Body), nil
}

func (s *sourceService) GetFunction(ctx context.Context, functionName, groupName string) (string, error) {
	functionName = strings.ToUpper(functionName)
	groupName = strings.ToUpper(groupName)
	sourcePath := fmt.Sprintf("/sap/bc/adt/functions/groups/%s/fmodules/%s/source/main",
		url.PathEscape(groupName), url.PathEscape(functionName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
		Accept: "text/plain",
	})
	if err != nil {
		return "", fmt.Errorf("getting function source: %w", err)
	}
	return string(resp.Body), nil
}

func (s *sourceService) GetFunctionGroup(ctx context.Context, groupName string) (*FunctionGroup, error) {
	groupName = strings.ToUpper(groupName)
	path := fmt.Sprintf("/sap/bc/adt/functions/groups/%s", url.PathEscape(groupName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   path,
		Method: http.MethodGet,
		Accept: "application/xml",
	})
	if err != nil {
		return nil, fmt.Errorf("getting function group: %w", err)
	}

	var fg FunctionGroup
	if err := xml.Unmarshal(resp.Body, &fg); err != nil {
		return nil, fmt.Errorf("parsing function group: %w", err)
	}
	return &fg, nil
}

func (s *sourceService) GetInclude(ctx context.Context, includeName string) (string, error) {
	includeName = strings.ToUpper(includeName)
	sourcePath := fmt.Sprintf("/sap/bc/adt/programs/includes/%s/source/main", url.PathEscape(includeName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
		Accept: "text/plain",
	})
	if err != nil {
		return "", fmt.Errorf("getting include source: %w", err)
	}
	return string(resp.Body), nil
}

func (s *sourceService) GetDDLS(ctx context.Context, ddlsName string) (string, error) {
	ddlsName = strings.ToUpper(ddlsName)
	sourcePath := fmt.Sprintf("/sap/bc/adt/ddic/ddl/sources/%s/source/main", url.PathEscape(ddlsName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
		Accept: "text/plain",
	})
	if err != nil {
		return "", fmt.Errorf("getting DDLS source: %w", err)
	}
	return string(resp.Body), nil
}

func (s *sourceService) GetBDEF(ctx context.Context, bdefName string) (string, error) {
	bdefName = strings.ToUpper(bdefName)
	sourcePath := fmt.Sprintf("/sap/bc/adt/bo/behaviordefinitions/%s/source/main", url.PathEscape(bdefName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
		Accept: "text/plain",
	})
	if err != nil {
		return "", fmt.Errorf("getting BDEF source: %w", err)
	}
	return string(resp.Body), nil
}

func (s *sourceService) GetSRVD(ctx context.Context, srvdName string) (string, error) {
	srvdName = strings.ToUpper(srvdName)
	sourcePath := fmt.Sprintf("/sap/bc/adt/ddic/srvd/sources/%s/source/main", url.PathEscape(srvdName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
		Accept: "text/plain",
	})
	if err != nil {
		return "", fmt.Errorf("getting SRVD source: %w", err)
	}
	return string(resp.Body), nil
}

func (s *sourceService) GetTable(ctx context.Context, tableName string) (string, error) {
	tableName = strings.ToUpper(tableName)
	sourcePath := fmt.Sprintf("/sap/bc/adt/ddic/tables/%s/source/main", url.PathEscape(tableName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
	})
	if err != nil {
		return "", fmt.Errorf("getting table source: %w", err)
	}
	return string(resp.Body), nil
}

func (s *sourceService) GetView(ctx context.Context, viewName string) (string, error) {
	viewName = strings.ToUpper(viewName)
	sourcePath := fmt.Sprintf("/sap/bc/adt/ddic/views/%s/source/main", url.PathEscape(viewName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
	})
	if err != nil {
		return "", fmt.Errorf("getting view source: %w", err)
	}
	return string(resp.Body), nil
}

func (s *sourceService) GetStructure(ctx context.Context, structName string) (string, error) {
	structName = strings.ToUpper(structName)
	sourcePath := fmt.Sprintf("/sap/bc/adt/ddic/structures/%s/source/main", url.PathEscape(structName))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   sourcePath,
		Method: http.MethodGet,
	})
	if err != nil {
		return "", fmt.Errorf("getting structure source: %w", err)
	}
	return string(resp.Body), nil
}

func (s *sourceService) GetPackage(ctx context.Context, packageName string) (*PackageContent, error) {
	packageName = strings.ToUpper(packageName)

	params := url.Values{}
	params.Set("parent_type", "DEVC/K")
	params.Set("parent_name", packageName)
	params.Set("withShortDescriptions", "true")

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   "/sap/bc/adt/repository/nodestructure",
		Method: http.MethodPost,
		Query:  params,
	})
	if err != nil {
		return nil, fmt.Errorf("getting package contents: %w", err)
	}

	return parsePackageNodeStructure(resp.Body, packageName)
}

func (s *sourceService) GetMessageClass(ctx context.Context, msgClassName string) (*MessageClass, error) {
	msgClassName = strings.ToUpper(msgClassName)
	path := fmt.Sprintf("/sap/bc/adt/messageclass/%s", url.PathEscape(strings.ToLower(msgClassName)))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   path,
		Method: http.MethodGet,
		Accept: "application/vnd.sap.adt.mc.messageclass+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("getting message class: %w", err)
	}

	var mc MessageClass
	if err := xml.Unmarshal(resp.Body, &mc); err != nil {
		return nil, fmt.Errorf("parsing message class XML: %w", err)
	}
	mc.Name = msgClassName
	return &mc, nil
}

func (s *sourceService) GetTransaction(ctx context.Context, tcode string) (*Transaction, error) {
	tcode = strings.ToUpper(tcode)
	path := fmt.Sprintf("/sap/bc/adt/vit/wb/object_type/TRAN/object_name/%s", tcode)

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   path,
		Method: http.MethodGet,
		Accept: "application/xml",
	})
	if err != nil {
		return nil, fmt.Errorf("getting transaction: %w", err)
	}

	type tranInfo struct {
		Name        string `xml:"name,attr"`
		Description string `xml:"description,attr"`
		Program     string `xml:"program,attr"`
	}
	var ti tranInfo
	if err := xml.Unmarshal(resp.Body, &ti); err != nil {
		return nil, fmt.Errorf("parsing transaction: %w", err)
	}
	return &Transaction{Name: ti.Name, Description: ti.Description, Program: ti.Program}, nil
}

func (s *sourceService) GetTypeInfo(ctx context.Context, typeName string) (*TypeInfo, error) {
	typeName = strings.ToUpper(typeName)
	path := fmt.Sprintf("/sap/bc/adt/ddic/dataelements/%s", typeName)

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:   path,
		Method: http.MethodGet,
		Accept: "application/xml",
	})
	if err != nil {
		return nil, fmt.Errorf("getting type info: %w", err)
	}

	type typeData struct {
		Name        string `xml:"name,attr"`
		Type        string `xml:"type,attr"`
		Description string `xml:"description,attr"`
		Length      int    `xml:"length,attr"`
		Decimals    int    `xml:"decimals,attr"`
	}
	var td typeData
	if err := xml.Unmarshal(resp.Body, &td); err != nil {
		return nil, fmt.Errorf("parsing type info: %w", err)
	}
	return &TypeInfo{Name: td.Name, Type: td.Type, Description: td.Description, Length: td.Length, Decimals: td.Decimals}, nil
}

func (s *sourceService) GetTableContents(ctx context.Context, tableName string, maxRows int, sqlFilter string) (*TableContentsResult, error) {
	tableName = strings.ToUpper(tableName)
	if maxRows <= 0 {
		maxRows = 100
	}

	params := url.Values{}
	params.Set("rowNumber", fmt.Sprintf("%d", maxRows))
	params.Set("ddicEntityName", tableName)

	req := &transport.AdtRequest{
		Path:   "/sap/bc/adt/datapreview/ddic",
		Method: http.MethodPost,
		Query:  params,
		Accept: "application/*",
	}
	if sqlFilter != "" {
		req.Body = []byte(sqlFilter)
		req.ContentType = "text/plain"
	}

	resp, err := s.sender.SendRequest(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("getting table contents: %w", err)
	}
	return parseTableContents(resp.Body)
}

func (s *sourceService) RunQuery(ctx context.Context, sqlQuery string, maxRows int) (*TableContentsResult, error) {
	if err := s.checkSafety(OpFreeSQL, "RunQuery"); err != nil {
		return nil, err
	}
	if sqlQuery == "" {
		return nil, fmt.Errorf("SQL query is required")
	}
	if maxRows <= 0 {
		maxRows = 100
	}

	params := url.Values{}
	params.Set("rowNumber", fmt.Sprintf("%d", maxRows))

	resp, err := s.sender.SendRequest(ctx, &transport.AdtRequest{
		Path:        "/sap/bc/adt/datapreview/freestyle",
		Method:      http.MethodPost,
		Query:       params,
		Accept:      "application/*",
		Body:        []byte(sqlQuery),
		ContentType: "text/plain",
	})
	if err != nil {
		return nil, fmt.Errorf("running query: %w", err)
	}
	return parseTableContents(resp.Body)
}

// --------------------------------------------------------------------------
// XML parsers (extracted from client.go)
// --------------------------------------------------------------------------

func parseSearchResults(data []byte) ([]SearchResult, error) {
	type objectRef struct {
		URI         string `xml:"uri,attr"`
		Type        string `xml:"type,attr"`
		Name        string `xml:"name,attr"`
		PackageName string `xml:"packageName,attr"`
		Description string `xml:"description,attr"`
	}
	type searchResults struct {
		Objects []objectRef `xml:"objectReference"`
	}

	// Strip namespace prefixes.
	xmlStr := strings.ReplaceAll(string(data), "adtcore:", "")

	var sr searchResults
	if err := xml.Unmarshal([]byte(xmlStr), &sr); err != nil {
		return nil, fmt.Errorf("parsing search results: %w", err)
	}

	results := make([]SearchResult, len(sr.Objects))
	for i, o := range sr.Objects {
		results[i] = SearchResult{
			URI:         o.URI,
			Type:        o.Type,
			Name:        o.Name,
			PackageName: o.PackageName,
			Description: o.Description,
		}
	}
	return results, nil
}

func parsePackageNodeStructure(data []byte, packageName string) (*PackageContent, error) {
	if len(data) == 0 {
		return &PackageContent{Name: packageName, Objects: []PackageObject{}, SubPackages: []string{}}, nil
	}

	type nodeData struct {
		TreeContent struct {
			Nodes []struct {
				ObjectType string `xml:"OBJECT_TYPE"`
				ObjectName string `xml:"OBJECT_NAME"`
				ObjectURI  string `xml:"OBJECT_URI"`
				Desc       string `xml:"DESCRIPTION"`
			} `xml:"SEU_ADT_REPOSITORY_OBJ_NODE"`
		} `xml:"TREE_CONTENT"`
	}
	type abapValues struct {
		Data nodeData `xml:"DATA"`
	}
	type abapResponse struct {
		Values abapValues `xml:"values"`
	}

	var resp abapResponse
	if err := xml.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parsing nodestructure: %w", err)
	}

	pkg := &PackageContent{Name: packageName, Objects: []PackageObject{}, SubPackages: []string{}}
	for _, node := range resp.Values.Data.TreeContent.Nodes {
		if node.ObjectName == "" {
			continue
		}
		if node.ObjectType == "DEVC/K" {
			pkg.SubPackages = append(pkg.SubPackages, node.ObjectName)
		} else {
			pkg.Objects = append(pkg.Objects, PackageObject{
				Type: node.ObjectType, Name: node.ObjectName, URI: node.ObjectURI, Description: node.Desc,
			})
		}
	}
	return pkg, nil
}

func parseTableContents(data []byte) (*TableContentsResult, error) {
	type tableData struct {
		Columns []struct {
			Metadata struct {
				Name        string `xml:"name,attr"`
				Type        string `xml:"type,attr"`
				Description string `xml:"description,attr"`
				Length      int    `xml:"length,attr"`
				IsKey       bool   `xml:"keyAttribute,attr"`
			} `xml:"metadata"`
			DataSet struct {
				Data []string `xml:"data"`
			} `xml:"dataSet"`
		} `xml:"columns"`
	}

	var td tableData
	if err := xml.Unmarshal(data, &td); err != nil {
		return nil, fmt.Errorf("parsing table data: %w", err)
	}

	result := &TableContentsResult{
		Columns: make([]TableColumn, len(td.Columns)),
		Rows:    []map[string]interface{}{},
	}

	maxRows := 0
	for i, col := range td.Columns {
		result.Columns[i] = TableColumn{
			Name: col.Metadata.Name, Type: col.Metadata.Type, Description: col.Metadata.Description,
			Length: col.Metadata.Length, IsKey: col.Metadata.IsKey,
		}
		if len(col.DataSet.Data) > maxRows {
			maxRows = len(col.DataSet.Data)
		}
	}

	for rowIdx := 0; rowIdx < maxRows; rowIdx++ {
		row := make(map[string]interface{})
		for _, col := range td.Columns {
			if rowIdx < len(col.DataSet.Data) {
				row[col.Metadata.Name] = col.DataSet.Data[rowIdx]
			}
		}
		result.Rows = append(result.Rows, row)
	}

	return result, nil
}
