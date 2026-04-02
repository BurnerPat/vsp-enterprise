package endpoints

import (
	"net/http"
)

// Empty runResult = no test classes found; client treats it as success.
const unitTestResultXML = `<?xml version="1.0" encoding="UTF-8"?>
<aunit:runResult xmlns:aunit="http://www.sap.com/adt/aunit"
                 xmlns:adtcore="http://www.sap.com/adt/core">
  <program adtcore:uri="" adtcore:type="CLAS/OC" adtcore:name="">
    <testClasses/>
  </program>
</aunit:runResult>`

// RegisterAbapunit registers the POST /sap/bc/adt/abapunit/testruns endpoint.
func RegisterAbapunit(mux *http.ServeMux, _ *State) {
	mux.HandleFunc("POST /sap/bc/adt/abapunit/testruns", func(w http.ResponseWriter, r *http.Request) {
		xmlResponse(w, unitTestResultXML)
	})
}
