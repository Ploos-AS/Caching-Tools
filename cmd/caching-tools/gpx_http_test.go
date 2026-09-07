package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGPXImportAndExportAPI(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}

	gpx := `<?xml version="1.0"?><gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1"><wpt lat="59.9139" lon="10.7522"><name>Trailhead</name><cmt>Start here</cmt><type>parking</type></wpt><wpt lat="60.3913" lon="5.3221"><name>Bergen</name></wpt></gpx>`
	importReq := httptest.NewRequest(http.MethodPost, "/api/gpx/waypoints/import", strings.NewReader(gpx))
	importReq.Header.Set("Content-Type", "application/gpx+xml")
	importRec := httptest.NewRecorder()
	h.ServeHTTP(importRec, importReq)
	if importRec.Code != http.StatusCreated {
		t.Fatalf("import status=%d body=%s", importRec.Code, importRec.Body.String())
	}
	var result gpxImportResult
	if err := json.Unmarshal(importRec.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Imported != 2 || len(result.Waypoints) != 2 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	listRec := httptest.NewRecorder()
	h.ServeHTTP(listRec, httptest.NewRequest(http.MethodGet, "/api/waypoints", nil))
	if listRec.Code != http.StatusOK || !strings.Contains(listRec.Body.String(), `"name":"Trailhead"`) || !strings.Contains(listRec.Body.String(), `"name":"Bergen"`) {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}

	exportRec := httptest.NewRecorder()
	h.ServeHTTP(exportRec, httptest.NewRequest(http.MethodGet, "/api/gpx/waypoints/export", nil))
	if exportRec.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", exportRec.Code, exportRec.Body.String())
	}
	if got := exportRec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/gpx+xml") {
		t.Fatalf("unexpected content type: %s", got)
	}
	body := exportRec.Body.String()
	for _, want := range []string{`version="1.1"`, `<name>Trailhead</name>`, `<name>Bergen</name>`, `<type>parking</type>`} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
}

func TestGPXImportIsAtomicOnInvalidWaypoint(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	gpx := `<gpx version="1.1"><wpt lat="59" lon="10"><name>Good</name></wpt><wpt lat="99" lon="10"><name>Bad</name></wpt></gpx>`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/gpx/waypoints/import", strings.NewReader(gpx)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	list := httptest.NewRecorder()
	h.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/waypoints", nil))
	if strings.TrimSpace(list.Body.String()) != "[]" {
		t.Fatalf("import was not atomic: %s", list.Body.String())
	}
}
