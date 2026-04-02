package endpoints

import (
	"fmt"
	"net/http"
)

// ---- Default source generators per DDIC type ----

func defaultTableSource(name string) string {
	return fmt.Sprintf(`@EndUserText.label: '%s'
@AbapCatalog.enhancement.category: #NOT_EXTENSIBLE
define table %s {
  key mandt : mandt not null;
  key id    : numc10 not null;
  value     : char255;
}
`, name, name)
}

func defaultStructureSource(name string) string {
	return fmt.Sprintf(`@EndUserText.label: '%s'
define structure %s {
  id    : numc10;
  value : char255;
}
`, name, name)
}

func defaultViewSource(name string) string {
	return fmt.Sprintf(`@EndUserText.label: '%s'
define view %s as select from t000 {
  mandt,
  mtext
}
`, name, name)
}

func defaultDDLSource(name string) string {
	return fmt.Sprintf(`@EndUserText.label: '%s'
@AccessControl.authorizationCheck: #CHECK
define view entity %s
  as select from t000
{
  key mandt as Client,
      mtext as Description
}
`, name, name)
}

func defaultSRVDSource(name string) string {
	return fmt.Sprintf(`@EndUserText.label: '%s'
define service %s {
  expose ZI_DEMO as Demo;
}
`, name, name)
}

func defaultBDEFSource(name string) string {
	return fmt.Sprintf(`managed implementation in class ZBP_%s unique;

define behavior for %s alias Demo
persistent table zdemo
lock master
authorization master ( instance )
{
  create;
  update;
  delete;
}
`, name, name)
}

func defaultDataElementXML(name string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<dataElement xmlns="http://www.sap.com/adt/ddic"
  name="%s" type="CHAR" length="10" decimals="0"
  description="Data element %s (test server)"/>
`, name, name)
}

// RegisterDDIC registers all /sap/bc/adt/ddic/ and /sap/bc/adt/bo/behaviordefinitions/ endpoints.
func RegisterDDIC(mux *http.ServeMux, s *State) {
	// ---- Tables ----
	mux.HandleFunc("GET /sap/bc/adt/ddic/tables/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourceGet(w, r, s, defaultTableSource)
	})
	mux.HandleFunc("PUT /sap/bc/adt/ddic/tables/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})
	mux.HandleFunc("POST /sap/bc/adt/ddic/tables/{name}", func(w http.ResponseWriter, r *http.Request) {
		handleLockUnlock(w, r, s)
	})

	// ---- Structures ----
	mux.HandleFunc("GET /sap/bc/adt/ddic/structures/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourceGet(w, r, s, defaultStructureSource)
	})
	mux.HandleFunc("PUT /sap/bc/adt/ddic/structures/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})
	mux.HandleFunc("POST /sap/bc/adt/ddic/structures/{name}", func(w http.ResponseWriter, r *http.Request) {
		handleLockUnlock(w, r, s)
	})

	// ---- Views ----
	mux.HandleFunc("GET /sap/bc/adt/ddic/views/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourceGet(w, r, s, defaultViewSource)
	})
	mux.HandleFunc("PUT /sap/bc/adt/ddic/views/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})

	// ---- CDS DDL Sources ----
	mux.HandleFunc("GET /sap/bc/adt/ddic/ddl/sources/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourceGet(w, r, s, defaultDDLSource)
	})
	mux.HandleFunc("PUT /sap/bc/adt/ddic/ddl/sources/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})
	mux.HandleFunc("POST /sap/bc/adt/ddic/ddl/sources/{name}", func(w http.ResponseWriter, r *http.Request) {
		if handleLockUnlock(w, r, s) {
			return
		}
		name := r.PathValue("name")
		created(w, "/sap/bc/adt/ddic/ddl/sources/"+name)
	})

	// ---- Service Definitions (SRVD) ----
	mux.HandleFunc("GET /sap/bc/adt/ddic/srvd/sources/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourceGet(w, r, s, defaultSRVDSource)
	})
	mux.HandleFunc("PUT /sap/bc/adt/ddic/srvd/sources/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})

	// ---- Data Elements ----
	mux.HandleFunc("GET /sap/bc/adt/ddic/dataelements/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		xmlResponse(w, defaultDataElementXML(name))
	})

	// ---- Behavior Definitions (BDEF) ----
	mux.HandleFunc("GET /sap/bc/adt/bo/behaviordefinitions/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourceGet(w, r, s, defaultBDEFSource)
	})
	mux.HandleFunc("PUT /sap/bc/adt/bo/behaviordefinitions/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})
	mux.HandleFunc("POST /sap/bc/adt/bo/behaviordefinitions/{name}", func(w http.ResponseWriter, r *http.Request) {
		if handleLockUnlock(w, r, s) {
			return
		}
		name := r.PathValue("name")
		created(w, "/sap/bc/adt/bo/behaviordefinitions/"+name)
	})
}
