package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPathRenameAndGPXExport(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	gpx := `<gpx version="1.1" creator="test"><trk><name>Old track</name><trkseg><trkpt lat="60" lon="11"><ele>10</ele><time>2026-09-07T08:00:00Z</time></trkpt><trkpt lat="60.01" lon="11"><ele>20</ele><time>2026-09-07T08:10:00Z</time></trkpt></trkseg><trkseg><trkpt lat="60.02" lon="11"><ele>15</ele></trkpt></trkseg></trk></gpx>`

	imported := httptest.NewRecorder()
	h.ServeHTTP(imported, httptest.NewRequest(http.MethodPost, "/api/gpx/paths/import", strings.NewReader(gpx)))
	if imported.Code != http.StatusCreated {
		t.Fatalf("import status=%d body=%s", imported.Code, imported.Body.String())
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

	renamed := httptest.NewRecorder()
	h.ServeHTTP(renamed, httptest.NewRequest(http.MethodPut, "/api/paths/"+id, strings.NewReader(`{"name":"Sunday hike"}`)))
	if renamed.Code != http.StatusOK || !strings.Contains(renamed.Body.String(), `"name":"Sunday hike"`) {
		t.Fatalf("rename status=%d body=%s", renamed.Code, renamed.Body.String())
	}

	exported := httptest.NewRecorder()
	h.ServeHTTP(exported, httptest.NewRequest(http.MethodGet, "/api/paths/"+id+"/export", nil))
	if exported.Code != http.StatusOK {
		t.Fatalf("export status=%d body=%s", exported.Code, exported.Body.String())
	}
	if got := exported.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/gpx+xml") {
		t.Fatalf("content-type=%q", got)
	}
	text := exported.Body.String()
	for _, want := range []string{`version="1.1"`, `<name>Sunday hike</name>`, `<trkseg>`, `<ele>10</ele>`, `<time>2026-09-07T08:00:00Z</time>`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
	if strings.Count(text, "<trkseg>") != 2 {
		t.Fatalf("expected two track segments: %s", text)
	}

	doc, err := decodeGPX(strings.NewReader(text))
	if err != nil {
		t.Fatalf("export did not round-trip as GPX 1.1: %v", err)
	}
	if len(doc.Tracks) != 1 || doc.Tracks[0].Name != "Sunday hike" || len(doc.Tracks[0].Segments) != 2 {
		t.Fatalf("unexpected round-trip track: %+v", doc.Tracks)
	}
}

func TestPathRenameValidationAndMissingExport(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}

	renamed := httptest.NewRecorder()
	h.ServeHTTP(renamed, httptest.NewRequest(http.MethodPut, "/api/paths/missing", strings.NewReader(`{"name":"x"}`)))
	if renamed.Code != http.StatusNotFound {
		t.Fatalf("rename missing status=%d body=%s", renamed.Code, renamed.Body.String())
	}

	exported := httptest.NewRecorder()
	h.ServeHTTP(exported, httptest.NewRequest(http.MethodGet, "/api/paths/missing/export", nil))
	if exported.Code != http.StatusNotFound {
		t.Fatalf("export missing status=%d body=%s", exported.Code, exported.Body.String())
	}
}
