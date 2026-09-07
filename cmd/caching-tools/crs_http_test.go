package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCRSAPIWGS84ToETRS89UTM(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	body := `{"source":"EPSG:4326","target":"ETRS89-UTM","latitude":"59.9139","longitude":"10.7522","zone":32}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/coordinates/crs", strings.NewReader(body)))
	if rr.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String()) }
	var result crsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil { t.Fatal(err) }
	if result.Grid == nil || result.Grid.CRS != "EPSG:25832" || !result.Approximate { t.Fatalf("result=%+v", result) }
}

func TestCRSAPIRejectsUnsupportedZone(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	body := `{"source":"EPSG:4258","target":"ETRS89-UTM","latitude":"59","longitude":"10","zone":27}`
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/api/coordinates/crs", strings.NewReader(body)))
	if rr.Code != http.StatusBadRequest { t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String()) }
}
