package endpoints

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// rowsToXML converts a slice of column→value maps into ADT table-data XML.
func rowsToXML(rows []map[string]string) string {
	if len(rows) == 0 {
		return `<?xml version="1.0" encoding="UTF-8"?>
<dataPreview:tableData xmlns:dataPreview="http://www.sap.com/adt/dataPreview">
  <dataPreview:columns/>
</dataPreview:tableData>`
	}

	// Collect column names preserving insertion order from first row.
	seen := map[string]bool{}
	var cols []string
	for _, row := range rows {
		for k := range row {
			if !seen[k] {
				seen[k] = true
				cols = append(cols, k)
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<dataPreview:tableData xmlns:dataPreview="http://www.sap.com/adt/dataPreview">` + "\n")
	sb.WriteString(`  <dataPreview:columns>` + "\n")

	for _, col := range cols {
		sb.WriteString(fmt.Sprintf(
			`    <dataPreview:column><dataPreview:metadata dataPreview:name="%s" dataPreview:type="CHAR" dataPreview:length="40" dataPreview:description="%s" dataPreview:keyAttribute="false"/>`+"\n",
			col, col))
		sb.WriteString(`      <dataPreview:dataSet>` + "\n")
		for _, row := range rows {
			sb.WriteString(fmt.Sprintf(`        <dataPreview:data>%s</dataPreview:data>`+"\n", row[col]))
		}
		sb.WriteString(`      </dataPreview:dataSet>` + "\n")
		sb.WriteString(`    </dataPreview:column>` + "\n")
	}

	sb.WriteString(`  </dataPreview:columns>` + "\n")
	sb.WriteString(`</dataPreview:tableData>`)
	return sb.String()
}

// t000Rows returns a synthetic T000 row for the configured system.
func t000Rows(s *State) []map[string]string {
	return []map[string]string{
		{
			"MANDT":  s.Client,
			"MTEXT":  fmt.Sprintf("Test Client %s", s.Client),
			"LOGSYS": fmt.Sprintf("%sCLNT%s", s.SysID, s.Client),
		},
	}
}

// handleDataPreview is shared by ddic and freestyle endpoints.
func handleDataPreview(w http.ResponseWriter, r *http.Request, s *State) {
	body, _ := io.ReadAll(r.Body)
	sql := strings.TrimSpace(string(body))

	// 1. Exact fixture match
	if rows, ok := s.QueryDataPreview(sql); ok {
		xmlResponse(w, rowsToXML(rows))
		return
	}

	// 2. T000 fallback regardless of exact query text
	if strings.Contains(strings.ToUpper(sql), "T000") {
		xmlResponse(w, rowsToXML(t000Rows(s)))
		return
	}

	// 3. Generic empty result
	xmlResponse(w, rowsToXML(nil))
}

// RegisterDatapreview registers the /sap/bc/adt/datapreview/ endpoints.
func RegisterDatapreview(mux *http.ServeMux, s *State) {
	mux.HandleFunc("POST /sap/bc/adt/datapreview/ddic", func(w http.ResponseWriter, r *http.Request) {
		handleDataPreview(w, r, s)
	})

	mux.HandleFunc("POST /sap/bc/adt/datapreview/freestyle", func(w http.ResponseWriter, r *http.Request) {
		handleDataPreview(w, r, s)
	})
}
