package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntersectionWaypointSaveUIAssets(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	page := httptest.NewRecorder(); h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK { t.Fatalf("page status=%d body=%s", page.Code, page.Body.String()) }
	for _, want := range []string{`id="intersection-bearing-result"`,`id="intersection-distance-result"`,`id="intersection-circle-result"`,`Caching Tools M1.16`,"saved directly as a local waypoint"} { if !strings.Contains(page.Body.String(), want) { t.Fatalf("missing %q from page", want) } }
	asset := httptest.NewRecorder(); h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/intersections.js", nil)); if asset.Code != http.StatusOK { t.Fatalf("intersections.js status=%d", asset.Code) }
	for _, want := range []string{"Save solution","/api/waypoints","type: 'intersection'","loadWaypoints","refreshLocalMap","intersection-save-status"} { if !strings.Contains(asset.Body.String(), want) { t.Fatalf("missing %q from intersections.js", want) } }
}
