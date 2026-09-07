package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBearingBearingIntersectionAPI(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/coordinates/intersection/bearing-bearing", strings.NewReader(`{"a":{"latitude":"0","longitude":"0"},"bearing_a_deg":45,"b":{"latitude":"0","longitude":"1"},"bearing_b_deg":315}`)))
	if rec.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
	if !strings.Contains(rec.Body.String(), `"points":[`) { t.Fatalf("body=%s", rec.Body.String()) }
}

func TestBearingDistanceIntersectionAPI(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/coordinates/intersection/bearing-distance", strings.NewReader(`{"from":{"latitude":"0","longitude":"0"},"bearing_deg":90,"center":{"latitude":"0","longitude":"0.09"},"distance_m":5000}`)))
	if rec.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
}

func TestCircleIntersectionAPIRejectsNoSolution(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/coordinates/intersection/circle-circle", strings.NewReader(`{"a":{"latitude":"0","longitude":"0"},"radius_a_m":1000,"b":{"latitude":"0","longitude":"1"},"radius_b_m":1000}`)))
	if rec.Code != http.StatusBadRequest { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
}
