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
	for _, want := range []string{`id="local-map"`, `id="map-refresh"`, `src="/map.js"`, `Caching Tools M1.9`} {
		if !strings.Contains(page.Body.String(), want) {
			t.Fatalf("missing %q from page", want)
		}
	}

	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if asset.Code != http.StatusOK {
		t.Fatalf("map.js status=%d", asset.Code)
	}
	for _, want := range []string{"refreshLocalMap", "/api/waypoints", "/api/paths", "createElementNS"} {
		if !strings.Contains(asset.Body.String(), want) {
			t.Fatalf("missing %q from map.js", want)
		}
	}
}
