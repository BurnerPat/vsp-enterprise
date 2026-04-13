//go:build testserver

package testserver

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// dispatchAction executes the named built-in action for a matched route.
func dispatchAction(w http.ResponseWriter, r *http.Request, rt *Route, vars map[string]string, state *State, csrfToken, guid string) {
	sub := func(s string) string {
		return substitute(s, vars, csrfToken, guid)
	}

	switch rt.Action {
	case "get-source":
		actionGetSource(w, r, rt, state, sub)
	case "store-source":
		actionStoreSource(w, r, rt, state)
	case "lock":
		actionLock(w, r, state)
	case "unlock":
		actionUnlock(w, r, rt, state)
	case "delete":
		actionDelete(w, r, rt, state)
	case "datapreview":
		actionDataPreview(w, r, state)
	default:
		http.Error(w, fmt.Sprintf("unknown action %q", rt.Action), http.StatusInternalServerError)
	}
}

// actionGetSource serves a source file. State (previously PUT) takes priority;
// falls back to response.body with placeholder substitution.
func actionGetSource(w http.ResponseWriter, r *http.Request, rt *Route, state *State, sub func(string) string) {
	ct := rt.Response.ContentType
	if ct == "" {
		ct = "text/plain; charset=utf-8"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)

	if src, ok := state.GetSource(r.URL.Path); ok {
		fmt.Fprint(w, src)
		return
	}
	fmt.Fprint(w, sub(rt.Response.Body))
}

// actionStoreSource reads the request body and stores it in state under the request path.
func actionStoreSource(w http.ResponseWriter, r *http.Request, rt *Route, state *State) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	state.SetSource(r.URL.Path, string(body))
	status := rt.Response.Status
	if status == 0 {
		status = http.StatusNoContent
	}
	w.WriteHeader(status)
}

// actionLock generates a deterministic lock handle, stores it in state, and returns
// the ADT lock result XML. The response body from the YAML route is ignored.
func actionLock(w http.ResponseWriter, r *http.Request, state *State) {
	objectURL := r.URL.Path
	handle := lockHandleFor(objectURL)
	state.AcquireLock(objectURL, handle)
	w.Header().Set("Content-Type", "application/vnd.sap.as+xml; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?>
<asx:abap xmlns:asx="http://www.sap.com/abapxml" version="1.0">
  <asx:values>
    <DATA>
      <LOCK_HANDLE>%s</LOCK_HANDLE>
      <CORRNR></CORRNR>
      <CORRUSER></CORRUSER>
      <CORRTEXT></CORRTEXT>
      <IS_LOCAL>X</IS_LOCAL>
      <IS_LINK_UP></IS_LINK_UP>
      <MODIFICATION_SUPPORT>WRITE</MODIFICATION_SUPPORT>
    </DATA>
  </asx:values>
</asx:abap>`, handle)
}

// actionUnlock releases the lock for the request path from state.
func actionUnlock(w http.ResponseWriter, r *http.Request, rt *Route, state *State) {
	state.ReleaseLock(r.URL.Path)
	status := rt.Response.Status
	if status == 0 {
		status = http.StatusNoContent
	}
	w.WriteHeader(status)
}

// actionDelete removes the source and lock for the request path from state.
func actionDelete(w http.ResponseWriter, r *http.Request, rt *Route, state *State) {
	objectURL := r.URL.Path
	state.DeleteSource(objectURL)
	state.ReleaseLock(objectURL)
	status := rt.Response.Status
	if status == 0 {
		status = http.StatusNoContent
	}
	w.WriteHeader(status)
}

// actionDataPreview executes the data-preview logic:
// 1. Exact SQL match against state
// 2. T000 fallback when the query mentions T000
// 3. Empty result set otherwise
func actionDataPreview(w http.ResponseWriter, r *http.Request, state *State) {
	body, _ := io.ReadAll(r.Body)
	sql := strings.TrimSpace(string(body))

	var rows []map[string]string
	if qrows, ok := state.QueryDataPreview(sql); ok {
		rows = qrows
	} else if strings.Contains(strings.ToUpper(sql), "T000") {
		rows = []map[string]string{
			{
				"MANDT":  state.Client,
				"MTEXT":  fmt.Sprintf("Test Client %s", state.Client),
				"LOGSYS": fmt.Sprintf("%sCLNT%s", state.SysID, state.Client),
			},
		}
	}
	// rows == nil → empty result

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, rowsToXML(rows))
}

// rowsToXML converts a slice of column→value maps into ADT table-data XML.
// Column order is derived from the first row's key insertion order (non-deterministic
// due to Go map iteration, but consistent within a single response).
func rowsToXML(rows []map[string]string) string {
	if len(rows) == 0 {
		return `<?xml version="1.0" encoding="UTF-8"?>
<dataPreview:tableData xmlns:dataPreview="http://www.sap.com/adt/dataPreview">
  <dataPreview:columns/>
</dataPreview:tableData>`
	}

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
	sb.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	sb.WriteString("<dataPreview:tableData xmlns:dataPreview=\"http://www.sap.com/adt/dataPreview\">\n")
	sb.WriteString("  <dataPreview:columns>\n")

	for _, col := range cols {
		sb.WriteString(fmt.Sprintf(
			"    <dataPreview:column><dataPreview:metadata dataPreview:name=%q dataPreview:type=\"CHAR\" dataPreview:length=\"40\" dataPreview:description=%q dataPreview:keyAttribute=\"false\"/>\n",
			col, col))
		sb.WriteString("      <dataPreview:dataSet>\n")
		for _, row := range rows {
			sb.WriteString(fmt.Sprintf("        <dataPreview:data>%s</dataPreview:data>\n", row[col]))
		}
		sb.WriteString("      </dataPreview:dataSet>\n")
		sb.WriteString("    </dataPreview:column>\n")
	}

	sb.WriteString("  </dataPreview:columns>\n")
	sb.WriteString("</dataPreview:tableData>")
	return sb.String()
}
