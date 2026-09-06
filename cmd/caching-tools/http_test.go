package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConvertAPI(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/convert", strings.NewReader(`{"latitude":"N 58 7.404","longitude":"E 8 0 0"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"latitude":58.1234`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestNavigationAPIRejectsInvalidCoordinate(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/navigation", strings.NewReader(`{"from":{"latitude":"N 99","longitude":"E 10"},"to":{"latitude":"N 60","longitude":"E 5"}}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProjectionAPI(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/project", strings.NewReader(`{"from":{"latitude":"59.9139","longitude":"10.7522"},"bearing_deg":42,"distance_m":12345}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"distance_m":12345`) || !strings.Contains(rec.Body.String(), `"bearing_deg":42`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestProjectionAPIRejectsNegativeDistance(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/project", strings.NewReader(`{"from":{"latitude":"59.9","longitude":"10.7"},"bearing_deg":42,"distance_m":-1}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGridAPI(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/grid", strings.NewReader(`{"latitude":"59.9139","longitude":"10.7522"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"zone":32`) || !strings.Contains(body, `"mgrs":"32V NM 97979 43118"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestFromUTMAPI(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/from-utm", strings.NewReader(`{"zone":32,"hemisphere":"n","easting":597979.903,"northing":6643118.991}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"dd":"N 59.913900"`) || !strings.Contains(body, `"dd":"E 10.752200"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}
