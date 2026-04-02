package endpoints

import (
	"fmt"
	"net/http"
)

const discoveryXML = `<?xml version="1.0" encoding="utf-8"?>
<app:service xmlns:app="http://www.w3.org/2007/app" xmlns:atom="http://www.w3.org/2005/Atom">
  <app:workspace>
    <atom:title>ADT Test Server</atom:title>
    <app:collection href="/sap/bc/adt/programs/programs">
      <atom:title>Programs</atom:title>
    </app:collection>
    <app:collection href="/sap/bc/adt/oo/classes">
      <atom:title>Classes</atom:title>
    </app:collection>
    <app:collection href="/sap/bc/adt/oo/interfaces">
      <atom:title>Interfaces</atom:title>
    </app:collection>
    <app:collection href="/sap/bc/adt/functions/groups">
      <atom:title>Function Groups</atom:title>
    </app:collection>
    <app:collection href="/sap/bc/adt/ddic/tables">
      <atom:title>Tables</atom:title>
    </app:collection>
    <app:collection href="/sap/bc/adt/ddic/ddl/sources">
      <atom:title>CDS DDL Sources</atom:title>
    </app:collection>
    <app:collection href="/sap/bc/adt/repository/informationsystem/search">
      <atom:title>Repository Search</atom:title>
    </app:collection>
    <app:collection href="/sap/bc/adt/atc/runs">
      <atom:title>ATC Runs</atom:title>
    </app:collection>
    <app:collection href="/sap/bc/adt/abapunit/testruns">
      <atom:title>ABAP Unit Runs</atom:title>
    </app:collection>
  </app:workspace>
</app:service>`

// RegisterCore registers the /sap/bc/adt/core/discovery and /sap/bc/adt/discovery endpoints.
func RegisterCore(mux *http.ServeMux, _ *State, csrfToken string) {
	// CSRF token fetch — clients send HEAD with X-CSRF-Token: Fetch
	mux.HandleFunc("HEAD /sap/bc/adt/core/discovery", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-CSRF-Token", csrfToken)
		w.Header().Set("Set-Cookie", "sap-contextid=testserver-session-001; Path=/; HttpOnly")
		w.WriteHeader(http.StatusOK)
	})

	// Some clients also use GET for CSRF fetch
	mux.HandleFunc("GET /sap/bc/adt/core/discovery", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-CSRF-Token", csrfToken)
		w.Header().Set("Set-Cookie", "sap-contextid=testserver-session-001; Path=/; HttpOnly")
		xmlResponse(w, discoveryXML)
	})

	// Full discovery document
	mux.HandleFunc("GET /sap/bc/adt/discovery", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-CSRF-Token", csrfToken)
		xmlResponse(w, discoveryXML)
	})

	// Ping / keep-alive
	mux.HandleFunc("GET /sap/bc/adt/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sap/bc/adt/" {
			w.Header().Set("X-CSRF-Token", csrfToken)
			fmt.Fprint(w, "ADT Test Server OK")
		}
	})
}
