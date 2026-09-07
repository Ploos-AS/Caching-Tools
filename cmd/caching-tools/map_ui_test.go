package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLocalMapAssetsAreServed(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}

	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK {
		t.Fatalf("page status=%d body=%s", page.Code, page.Body.String())
	}
	for _, want := range []string{
		`id="local-map"`, `id="map-refresh"`, `id="map-zoom-in"`, `id="map-zoom-out"`,
		`id="map-fit-all"`, `id="map-fit-selection"`, `id="map-selection"`,
		`src="/map-link.js"`, `src="/map.js"`, `Caching Tools M1.13`,
	} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("missing %q from page", want)
		}
	}

	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if asset.Code != http.StatusOK {
		t.Fatalf("map.js status=%d", asset.Code)
	}
	for _, want := range []string{
		"refreshLocalMap", "/api/waypoints", "/api/paths", "createElementNS",
		"segment.Points", "point.Latitude", "point.Longitude", "pointerdown", "wheel",
		"map-fit-selection", "caching-tools:map-select", "tabindex",
	} {
		if !strings.Contains(asset.Body.String(), want) {
			t.Fatalf("missing %q from map.js", want)
		}
	}

	link := httptest.NewRecorder()
	h.ServeHTTP(link, httptest.NewRequest(http.MethodGet, "/map-link.js", nil))
	if link.Code != http.StatusOK {
		t.Fatalf("map-link.js status=%d", link.Code)
	}
	for _, want := range []string{"caching-tools:map-select", "list-selected", "scrollIntoView"} {
		if !strings.Contains(link.Body.String(), want) {
			t.Fatalf("missing %q from map-link.js", want)
		}
	}
}
