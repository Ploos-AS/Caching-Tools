package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGPXInspectAPI(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	input := `<gpx version="1.1" creator="test"><rte><name>R1</name><rtept lat="59" lon="10"/><rtept lat="59.01" lon="10"/></rte><trk><name>T1</name><trkseg><trkpt lat="60" lon="11"><ele>100</ele><time>2026-09-07T08:00:00Z</time></trkpt><trkpt lat="60.01" lon="11"><ele>125</ele><time>2026-09-07T08:10:00Z</time></trkpt></trkseg></trk></gpx>`
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/gpx/inspect", strings.NewReader(input)))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, want := range []string{
		`"version":"1.1"`, `"name":"R1"`, `"name":"T1"`, `"segments":1`, `"points":2`,
		`"elevation_points":2`, `"min_elevation_m":100`, `"max_elevation_m":125`,
		`"elevation_gain_m":25`, `"timed_points":2`, `"duration_s":600`, `"average_speed_kmh":`, `"max_speed_kmh":`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in %s", want, body)
		}
	}
}

func TestGPXInspectAPIRejectsWrongVersion(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/gpx/inspect", strings.NewReader(`<gpx version="1.0"></gpx>`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
