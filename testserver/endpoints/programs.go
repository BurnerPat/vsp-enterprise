package endpoints

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func defaultProgramSource(name string) string {
	return fmt.Sprintf(`REPORT %s.

START-OF-SELECTION.
  WRITE: / 'Hello from %s'.
`, strings.ToLower(name), name)
}

func defaultIncludeSource(name string) string {
	return fmt.Sprintf(`*----------------------------------------------------------------------*
* Include %s
*----------------------------------------------------------------------*
`, name)
}

func lockResultXML(handle string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
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

// handleSourceGet serves GET …/source/main, preferring state override over default.
func handleSourceGet(w http.ResponseWriter, r *http.Request, s *State, defaultFn func(string) string) {
	name := objectNameFromURL(r)
	if src, ok := s.GetSource(r.URL.Path); ok {
		textResponse(w, src)
		return
	}
	textResponse(w, defaultFn(name))
}

// handleSourcePut stores the request body as the source for the object.
func handleSourcePut(w http.ResponseWriter, r *http.Request, s *State) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}
	// Store under the exact path without query params
	s.SetSource(r.URL.Path, string(body))
	noContent(w)
}

// handleLockUnlock handles POST ?_action=LOCK and ?_action=UNLOCK.
// Returns false if the action was not lock/unlock (i.e. caller should handle it).
func handleLockUnlock(w http.ResponseWriter, r *http.Request, s *State) bool {
	action := r.URL.Query().Get("_action")
	switch strings.ToUpper(action) {
	case "LOCK":
		// strip /source/main suffix to get the object URL
		objectURL := strings.TrimSuffix(r.URL.Path, "/source/main")
		handle := lockHandleFor(objectURL)
		s.AcquireLock(objectURL, handle)
		w.Header().Set("Content-Type", "application/vnd.sap.as+xml; charset=UTF-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, lockResultXML(handle))
		return true
	case "UNLOCK":
		objectURL := strings.TrimSuffix(r.URL.Path, "/source/main")
		s.ReleaseLock(objectURL)
		noContent(w)
		return true
	}
	return false
}

// RegisterPrograms registers all /sap/bc/adt/programs/ endpoints.
func RegisterPrograms(mux *http.ServeMux, s *State) {
	// ---- Programs ----
	// GET source
	mux.HandleFunc("GET /sap/bc/adt/programs/programs/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourceGet(w, r, s, defaultProgramSource)
	})

	// PUT source
	mux.HandleFunc("PUT /sap/bc/adt/programs/programs/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})

	// POST: lock, unlock, or create
	mux.HandleFunc("POST /sap/bc/adt/programs/programs/{name}", func(w http.ResponseWriter, r *http.Request) {
		if handleLockUnlock(w, r, s) {
			return
		}
		// Create: return 201 with Location
		name := r.PathValue("name")
		created(w, "/sap/bc/adt/programs/programs/"+strings.ToUpper(name))
	})

	// POST collection = create
	mux.HandleFunc("POST /sap/bc/adt/programs/programs", func(w http.ResponseWriter, r *http.Request) {
		created(w, "/sap/bc/adt/programs/programs/ZNEW")
	})

	// DELETE
	mux.HandleFunc("DELETE /sap/bc/adt/programs/programs/{name}", func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		s.ReleaseLock("/sap/bc/adt/programs/programs/" + strings.ToUpper(name))
		noContent(w)
	})

	// ---- Includes ----
	mux.HandleFunc("GET /sap/bc/adt/programs/includes/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourceGet(w, r, s, defaultIncludeSource)
	})

	mux.HandleFunc("PUT /sap/bc/adt/programs/includes/{name}/source/main", func(w http.ResponseWriter, r *http.Request) {
		handleSourcePut(w, r, s)
	})

	mux.HandleFunc("POST /sap/bc/adt/programs/includes/{name}", func(w http.ResponseWriter, r *http.Request) {
		if handleLockUnlock(w, r, s) {
			return
		}
		name := r.PathValue("name")
		created(w, "/sap/bc/adt/programs/includes/"+strings.ToUpper(name))
	})
}
