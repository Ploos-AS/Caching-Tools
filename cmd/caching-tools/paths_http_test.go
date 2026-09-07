package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPersistentPathImportListGetDelete(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	gpx := `<gpx version="1.1" creator="test"><rte><name>R1</name><rtept lat="59" lon="10"/><rtept lat="59.01" lon="10"/></rte><trk><name>T1</name><trkseg><trkpt lat="60" lon="11"><ele>10</ele><time>2026-09-07T08:00:00Z</time></trkpt><trkpt lat="60.01" lon="11"><ele>20</ele><time>2026-09-07T08:10:00Z</time></trkpt></trkseg></trk></gpx>`

	imported := httptest.NewRecorder()
	h.ServeHTTP(imported, httptest.NewRequest(http.MethodPost, "/api/gpx/paths/import", strings.NewReader(gpx)))
	if imported.Code != http.StatusCreated {
		t.Fatalf("import status=%d body=%s", imported.Code, imported.Body.String())
	}
	if !strings.Contains(imported.Body.String(), `"imported":2`) || !strings.Contains(imported.Body.String(), `"kind":"track"`) {
		t.Fatalf("unexpected import body: %s", imported.Body.String())
	}

	list := httptest.NewRecorder()
	h.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/paths", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"name":"R1"`) || !strings.Contains(list.Body.String(), `"name":"T1"`) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}

	body := imported.Body.String()
	marker := `"id":"`
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatal("missing path id")
	}
	start += len(marker)
	end := strings.Index(body[start:], `"`)
	id := body[start : start+end]

	get := httptest.NewRecorder()
	h.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/paths/"+id, nil))
	if get.Code != http.StatusOK || !strings.Contains(get.Body.String(), `"id":"`+id+`"`) {
		t.Fatalf("get status=%d body=%s", get.Code, get.Body.String())
	}

	deleted := httptest.NewRecorder()
	h.ServeHTTP(deleted, httptest.NewRequest(http.MethodDelete, "/api/paths/"+id, nil))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
}

func TestPersistentPathImportRejectsInvalidBeforeWrite(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	gpx := `<gpx version="1.1"><rte><rtept lat="59" lon="10"/></rte><trk><trkseg><trkpt lat="99" lon="11"/></trkseg></trk></gpx>`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/gpx/paths/import", strings.NewReader(gpx)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	list := httptest.NewRecorder()
	h.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/paths", nil))
	if strings.TrimSpace(list.Body.String()) != "[]" {
		t.Fatalf("expected empty store, got %s", list.Body.String())
	}
}
