package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFinalCoordinateUIAssets(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	page := httptest.NewRecorder()
	h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK { t.Fatalf("status=%d", page.Code) }
	for _, want := range []string{`id="final-coordinate-form"`, `id="final-coordinate-save"`, `src="/formula.js"`, `N 59 54.[A+B][C][D]`} {
		if !strings.Contains(page.Body.String(), want) { t.Fatalf("missing %q", want) }
	}

	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/formula.js", nil))
	if asset.Code != http.StatusOK { t.Fatalf("formula.js status=%d", asset.Code) }
	for _, want := range []string{"/api/coordinates/final", "/api/waypoints", "parseVariableAssignments", "loadWaypoints", "refreshLocalMap", "type: 'final'"} {
		if !strings.Contains(asset.Body.String(), want) { t.Fatalf("missing %q in formula.js", want) }
	}
}
