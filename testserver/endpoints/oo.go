package endpoints

import (
	"fmt"
	"net/http"
	"strings"
)

func defaultClassSource(name string) string {
	return fmt.Sprintf(`CLASS %s DEFINITION
  PUBLIC
  FINAL
  CREATE PUBLIC.

  PUBLIC SECTION.
    METHODS: constructor,
             do_something RETURNING VALUE(rv_result) TYPE string.

  PROTECTED SECTION.
  PRIVATE SECTION.
ENDCLASS.

CLASS %s IMPLEMENTATION.

  METHOD constructor.
  ENDMETHOD.

  METHOD do_something.
    rv_result = '%s works'.
  ENDMETHOD.

ENDCLASS.
`, name, name, name)
}

func defaultInterfaceSource(name string) string {
	return fmt.Sprintf(`INTERFACE %s
  PUBLIC.

  METHODS: do_something RETURNING VALUE(rv_result) TYPE string.

ENDINTERFACE.
`, name)
}

func objectStructureXML(name string) string {
	lower := strings.ToLower(name)
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<adtcore:objectStructure xmlns:adtcore="http://www.sap.com/adt/core"
  xmlns:adtobj="http://www.sap.com/adt/objectstructure/v2"
  adtcore:uri="/sap/bc/adt/oo/classes/%s"
  adtcore:type="CLAS/OC"
  adtcore:name="%s">
  <adtobj:methodList>
    <adtobj:method adtobj:name="CONSTRUCTOR"
      adtobj:visibility="public"
      adtobj:level="instance"
      adtobj:definitionStart="7" adtobj:definitionEnd="7"
      adtobj:implementationStart="21" adtobj:implementationEnd="22"/>
    <adtobj:method adtobj:name="DO_SOMETHING"
      adtobj:visibility="public"
      adtobj:level="instance"
      adtobj:definitionStart="8" adtobj:definitionEnd="8"
      adtobj:implementationStart="25" adtobj:implementationEnd="27"/>
  </adtobj:methodList>
</adtcore:objectStructure>`, lower, name)
}

// RegisterOO registers all /sap/bc/adt/oo/ endpoints (classes and interfaces).
func RegisterOO(mux *http.ServeMux, s *State) {
	// ---- Classes ----
	mux.HandleFunc("GET /sap/bc/adt/oo/classes/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourceGet(w, r, s, defaultClassSource)
	})

	mux.HandleFunc("PUT /sap/bc/adt/oo/classes/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})

	mux.HandleFunc("GET /sap/bc/adt/oo/classes/{name}/objectstructure", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		xmlResponse(w, objectStructureXML(strings.ToUpper(name)))
	})

	mux.HandleFunc("POST /sap/bc/adt/oo/classes/{name}", func(w http.ResponseWriter, r *http.Request) {
		if handleLockUnlock(w, r, s) {
			return
		}
		name := r.PathValue("name")
		created(w, "/sap/bc/adt/oo/classes/"+strings.ToUpper(name))
	})

	mux.HandleFunc("POST /sap/bc/adt/oo/classes", func(w http.ResponseWriter, r *http.Request) {
		created(w, "/sap/bc/adt/oo/classes/ZCL_NEW")
	})

	mux.HandleFunc("DELETE /sap/bc/adt/oo/classes/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		s.ReleaseLock("/sap/bc/adt/oo/classes/" + strings.ToUpper(name))
		noContent(w)
	})

	// ---- Interfaces ----
	mux.HandleFunc("GET /sap/bc/adt/oo/interfaces/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourceGet(w, r, s, defaultInterfaceSource)
	})

	mux.HandleFunc("PUT /sap/bc/adt/oo/interfaces/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})

	mux.HandleFunc("POST /sap/bc/adt/oo/interfaces/{name}", func(w http.ResponseWriter, r *http.Request) {
		if handleLockUnlock(w, r, s) {
			return
		}
		name := r.PathValue("name")
		created(w, "/sap/bc/adt/oo/interfaces/"+strings.ToUpper(name))
	})

	mux.HandleFunc("POST /sap/bc/adt/oo/interfaces", func(w http.ResponseWriter, r *http.Request) {
		created(w, "/sap/bc/adt/oo/interfaces/ZIF_NEW")
	})

	mux.HandleFunc("DELETE /sap/bc/adt/oo/interfaces/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		s.ReleaseLock("/sap/bc/adt/oo/interfaces/" + strings.ToUpper(name))
		noContent(w)
	})
}
