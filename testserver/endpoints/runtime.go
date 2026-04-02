package endpoints

import (
	"fmt"
	"net/http"
)

const dumpsFeedXML = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Runtime Errors</title>
  <entry>
    <id>/sap/bc/adt/vit/runtime/dumps/TST001</id>
    <title>COMPUTE_INT_ZERODIVIDE</title>
    <updated>2026-04-02T10:00:00Z</updated>
    <link href="/sap/bc/adt/vit/runtime/dumps/TST001"/>
    <category term="ABAP"/>
    <content type="application/xml">
      <source program="ZTEST_PROGRAM" include="" line="42"/>
      <exception type="CX_SY_ZERODIVIDE"/>
      <runtime user="DEVELOPER" client="001" host="testserver"/>
    </content>
  </entry>
</feed>`

func dumpDetailHTML(id string) string {
	return fmt.Sprintf(`<html><head><title>Runtime Error %s</title></head>
<body><h1>Runtime Error: COMPUTE_INT_ZERODIVIDE</h1>
<p>Program: ZTEST_PROGRAM, Line: 42</p>
<p>Error ID: %s</p>
</body></html>`, id, id)
}

const tracesFeedXML = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>ABAP Traces</title>
  <entry>
    <id>TS_TRACE_001</id>
    <title>Trace 2026-04-02 10:00:00</title>
    <updated>2026-04-02T10:00:00Z</updated>
    <link href="/sap/bc/adt/runtime/traces/abaptraces/TS_TRACE_001"/>
    <author><name>DEVELOPER</name></author>
    <content type="application/xml">
      <trace startTime="2026-04-02T10:00:00Z" endTime="2026-04-02T10:00:01Z"
             duration="1000000" processType="DIALOG" objectType="PROG/P" status="finished"/>
    </content>
  </entry>
</feed>`

func traceAnalysisXML(id, toolType string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<traceAnalysis xmlns="http://www.sap.com/adt/runtime/trace"
               traceId="%s" toolType="%s" totalTime="1000000" totalCalls="42">
  <entries>
    <entry program="ZTEST_PROGRAM" event="MODULE" line="10"
           grossTime="500000" netTime="400000" calls="1" percentage="50.0"/>
    <entry program="ZTEST_PROGRAM" event="PERFORM" line="25"
           grossTime="200000" netTime="200000" calls="5" percentage="20.0"/>
  </entries>
</traceAnalysis>`, id, toolType)
}

// RegisterRuntime registers all /sap/bc/adt/runtime/ endpoints.
func RegisterRuntime(mux *http.ServeMux, _ *State) {
	// Short dumps list
	mux.HandleFunc("GET /sap/bc/adt/runtime/dumps", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, dumpsFeedXML)
	})

	// Single dump detail (HTML)
	mux.HandleFunc("GET /sap/bc/adt/runtime/dump/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, dumpDetailHTML(id))
	})

	// Traces list
	mux.HandleFunc("GET /sap/bc/adt/runtime/traces/abaptraces", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, tracesFeedXML)
	})

	// Trace analysis (hitlist / statements / dbAccesses)
	mux.HandleFunc("GET /sap/bc/adt/runtime/traces/abaptraces/{id}/{toolType}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		toolType := r.PathValue("toolType")
		xmlResponse(w, traceAnalysisXML(id, toolType))
	})
}
