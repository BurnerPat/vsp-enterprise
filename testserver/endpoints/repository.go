package endpoints

import (
	"fmt"
	"net/http"
	"strings"
)

func searchResultsXML(query string) string {
	// Echo a few plausible results based on the query term.
	name := strings.ToUpper(strings.NewReplacer("*", "", "?", "").Replace(query))
	if name == "" {
		name = "ZTEST"
	}
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<adtcore:objectReferences xmlns:adtcore="http://www.sap.com/adt/core">
  <adtcore:objectReference
    adtcore:uri="/sap/bc/adt/programs/programs/%s"
    adtcore:type="PROG/P"
    adtcore:name="%s"
    adtcore:packageName="$TMP"
    adtcore:description="Test program matching %s"/>
  <adtcore:objectReference
    adtcore:uri="/sap/bc/adt/oo/classes/ZCL_%s"
    adtcore:type="CLAS/OC"
    adtcore:name="ZCL_%s"
    adtcore:packageName="$TMP"
    adtcore:description="Test class matching %s"/>
</adtcore:objectReferences>`, name, name, query, name, name, query)
}

func nodeStructureXML(packageName string) string {
	lower := strings.ToLower(packageName)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<asx:abap xmlns:asx="http://www.sap.com/abapxml" version="1.0">
  <asx:values>
    <DATA>
      <TREE_CONTENT>
        <SEU_ADT_REPOSITORY_OBJ_NODE>
          <OBJECT_TYPE>PROG/P</OBJECT_TYPE>
          <OBJECT_NAME>Z%s_MAIN</OBJECT_NAME>
          <OBJECT_URI>/sap/bc/adt/programs/programs/z%s_main</OBJECT_URI>
          <DESCRIPTION>Main program of %s</DESCRIPTION>
        </SEU_ADT_REPOSITORY_OBJ_NODE>
        <SEU_ADT_REPOSITORY_OBJ_NODE>
          <OBJECT_TYPE>CLAS/OC</OBJECT_TYPE>
          <OBJECT_NAME>ZCL_%s</OBJECT_NAME>
          <OBJECT_URI>/sap/bc/adt/oo/classes/zcl_%s</OBJECT_URI>
          <DESCRIPTION>Main class of %s</DESCRIPTION>
        </SEU_ADT_REPOSITORY_OBJ_NODE>
      </TREE_CONTENT>
    </DATA>
  </asx:values>
</asx:abap>`,
		strings.ToUpper(packageName), lower,
		packageName,
		strings.ToUpper(packageName), lower,
		packageName)
}

// RegisterRepository registers all /sap/bc/adt/repository/ endpoints.
func RegisterRepository(mux *http.ServeMux, _ *State) {
	// Object search
	mux.HandleFunc("GET /sap/bc/adt/repository/informationsystem/search", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		xmlResponse(w, searchResultsXML(query))
	})

	// Package node structure
	mux.HandleFunc("POST /sap/bc/adt/repository/nodestructure", func(w http.ResponseWriter, r *http.Request) {
		pkg := r.URL.Query().Get("parent_name")
		if pkg == "" {
			pkg = "ZTEST"
		}
		xmlResponse(w, nodeStructureXML(pkg))
	})
}
