package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCRSUIAssetsAreServed(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	page := httptest.NewRecorder(); h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK { t.Fatalf("status=%d", page.Code) }
	if !strings.Contains(page.Body.String(), `src="/crs.js"`) { t.Fatal("missing CRS asset reference") }
	asset := httptest.NewRecorder(); h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/crs.js", nil))
	if asset.Code != http.StatusOK { t.Fatalf("crs.js status=%d", asset.Code) }
	for _, want := range []string{"Datum / CRS converter", "/api/coordinates/crs", "EPSG:4326", "EPSG:4258", "EPSG:25828", "EPSG:25832", "EPSG:25838", "25800 + zone", "Approximate datum relation", "Caching Tools M1.48"} {
		if !strings.Contains(asset.Body.String(), want) { t.Fatalf("missing %q from crs.js", want) }
	}
}
