package endpoints

import (
	"fmt"
	"net/http"
	"strings"
)

const atcCustomizingXML = `<?xml version="1.0" encoding="UTF-8"?>
<atc:customizing xmlns:atc="http://www.sap.com/adt/atc">
  <atc:properties>
    <atc:property name="systemCheckVariant" value="DEFAULT"/>
  </atc:properties>
  <atc:exemptionReasons>
    <atc:exemptionReason id="FALS" title="False positive" justificationMandatory="true"/>
    <atc:exemptionReason id="OBSL" title="Obsolete" justificationMandatory="false"/>
  </atc:exemptionReasons>
</atc:customizing>`

func atcWorklistRunXML(worklistID string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<atcworklist:worklistRun xmlns:atcworklist="http://www.sap.com/adt/atc/worklist">
  <atcworklist:worklistId>%s</atcworklist:worklistId>
  <atcworklist:worklistTimestamp>2026-04-02T12:00:00Z</atcworklist:worklistTimestamp>
  <atcworklist:infos>
    <atcworklist:info type="info" description="Check completed successfully"/>
  </atcworklist:infos>
</atcworklist:worklistRun>`, worklistID)
}

func atcWorklistXML(worklistID string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<atcworklist:worklist xmlns:atcworklist="http://www.sap.com/adt/atc/worklist"
                      xmlns:atcobject="http://www.sap.com/adt/atc/object"
                      xmlns:atcfinding="http://www.sap.com/adt/atc/finding"
                      xmlns:adtcore="http://www.sap.com/adt/core"
                      id="%s"
                      timestamp="2026-04-02T12:00:00Z"
                      usedObjectSet="LAST_RUN"
                      objectSetIsComplete="true">
  <atcworklist:objectSets>
    <atcworklist:objectSet name="LAST_RUN" title="Last Run" kind="LAST_RUN"/>
  </atcworklist:objectSets>
  <atcworklist:objects/>
</atcworklist:worklist>`, worklistID)
}

// worklistIDFor creates a deterministic worklist ID from a request body hash.
func worklistIDFor(objectURL string) string {
	h := lockHandleFor(objectURL) // reuse the FNV hash helper
	return strings.ToUpper(h[:16])
}

// RegisterATC registers all /sap/bc/adt/atc/ endpoints.
func RegisterATC(mux *http.ServeMux, _ *State) {
	// POST start ATC run — returns worklistRun with a fake ID
	mux.HandleFunc("POST /sap/bc/adt/atc/runs", func(w http.ResponseWriter, r *http.Request) {
		id := worklistIDFor(r.URL.RawQuery)
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Location", "/sap/bc/adt/atc/worklists/"+id)
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, atcWorklistRunXML(id))
	})

	// GET worklist — empty findings
	mux.HandleFunc("GET /sap/bc/adt/atc/worklists/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		xmlResponse(w, atcWorklistXML(id))
	})

	// GET customizing
	mux.HandleFunc("GET /sap/bc/adt/atc/customizing", func(w http.ResponseWriter, r *http.Request) {
		xmlResponse(w, atcCustomizingXML)
	})
}
