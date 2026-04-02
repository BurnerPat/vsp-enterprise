package endpoints

import (
	"net/http"
)

// Empty body = successful activation (the client treats an empty 200 response as success).
const inactiveObjectsXML = `<?xml version="1.0" encoding="UTF-8"?>
<ioc:inactiveObjects xmlns:ioc="http://www.sap.com/adt/activation/inactivectsobjects/v1"
                     xmlns:adtcore="http://www.sap.com/adt/core">
</ioc:inactiveObjects>`

// RegisterActivation registers the /sap/bc/adt/activation endpoints.
func RegisterActivation(mux *http.ServeMux, _ *State) {
	// POST activate — empty body signals success to the client
	mux.HandleFunc("POST /sap/bc/adt/activation", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// GET inactive objects — empty list means nothing is pending activation
	mux.HandleFunc("GET /sap/bc/adt/activation/inactiveobjects", func(w http.ResponseWriter, r *http.Request) {
		xmlResponse(w, inactiveObjectsXML)
	})
}
