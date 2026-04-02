package endpoints

import (
	"net/http"
)

const cleanCheckRunXML = `<?xml version="1.0" encoding="UTF-8"?>
<chkrun:checkRunReports xmlns:chkrun="http://www.sap.com/adt/checkrun">
  <chkrun:checkReport chkrun:reporter="abapCheckRun" chkrun:status="finished">
    <chkrun:checkMessageList/>
  </chkrun:checkReport>
</chkrun:checkRunReports>`

// RegisterCheckruns registers the POST /sap/bc/adt/checkruns endpoint.
func RegisterCheckruns(mux *http.ServeMux, _ *State) {
	mux.HandleFunc("POST /sap/bc/adt/checkruns", func(w http.ResponseWriter, r *http.Request) {
		xmlResponse(w, cleanCheckRunXML)
	})
}
