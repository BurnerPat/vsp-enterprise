package endpoints

import (
	"fmt"
	"net/http"
	"strings"
)

func defaultFunctionSource(name string) string {
	return fmt.Sprintf(`FUNCTION %s.
*"----------------------------------------------------------------------
*"*"Local Interface:
*"  IMPORTING
*"     VALUE(IV_INPUT) TYPE  STRING OPTIONAL
*"  EXPORTING
*"     VALUE(EV_OUTPUT) TYPE  STRING
*"----------------------------------------------------------------------

  EV_OUTPUT = |Result from { '%s' }|.

ENDFUNCTION.
`, name, name)
}

func functionGroupXML(group string) string {
	lower := strings.ToLower(group)
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<fugr:functionGroup xmlns:fugr="http://www.sap.com/adt/functions"
  xmlns:adtcore="http://www.sap.com/adt/core"
  adtcore:name="%s"
  adtcore:description="Function group %s (test server)">
  <fugr:functionModules>
    <fugr:functionModule adtcore:uri="/sap/bc/adt/functions/groups/%s/fmodules/Z%s_MAIN"
      adtcore:name="Z%s_MAIN"
      adtcore:description="Main function module"/>
  </fugr:functionModules>
</fugr:functionGroup>`, group, group, lower, group, group)
}

// RegisterFunctions registers all /sap/bc/adt/functions/ endpoints.
func RegisterFunctions(mux *http.ServeMux, s *State) {
	// GET function group descriptor
	mux.HandleFunc("GET /sap/bc/adt/functions/groups/{group}", func(w http.ResponseWriter, r *http.Request) {
		group := strings.ToUpper(r.PathValue("group"))
		xmlResponse(w, functionGroupXML(group))
	})

	// GET function module source
	mux.HandleFunc("GET /sap/bc/adt/functions/groups/{group}/fmodules/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		name := strings.ToUpper(r.PathValue("name"))
		if src, ok := s.GetSource(r.URL.Path); ok {
			textResponse(w, src)
			return
		}
		textResponse(w, defaultFunctionSource(name))
	})

	// PUT function module source
	mux.HandleFunc("PUT /sap/bc/adt/functions/groups/{group}/fmodules/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})

	// POST lock/unlock on function module
	mux.HandleFunc("POST /sap/bc/adt/functions/groups/{group}/fmodules/{name}", func(w http.ResponseWriter, r *http.Request) {
		if handleLockUnlock(w, r, s) {
			return
		}
		name := r.PathValue("name")
		group := r.PathValue("group")
		created(w, fmt.Sprintf("/sap/bc/adt/functions/groups/%s/fmodules/%s",
			strings.ToUpper(group), strings.ToUpper(name)))
	})

	// POST lock/unlock on function group
	mux.HandleFunc("POST /sap/bc/adt/functions/groups/{group}", func(w http.ResponseWriter, r *http.Request) {
		if handleLockUnlock(w, r, s) {
			return
		}
		group := r.PathValue("group")
		created(w, "/sap/bc/adt/functions/groups/"+strings.ToUpper(group))
	})
}
