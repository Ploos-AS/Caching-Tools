package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMapLogbookUIAssets(t *testing.T) {
	h, err := newHandler()
	if err != nil { t.Fatal(err) }

	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/map-logbook.js", nil))
	if asset.Code != http.StatusOK { t.Fatalf("asset status=%d", asset.Code) }
	for _, want := range []string{
		"Map waypoint logbook",
		"New field note for selected waypoint",
		"caching-tools:map-select",
		"kind !== 'waypoint'",
		"waypoint_id === id",
		"showWaypointLogbook",
		"startWaypointFieldNote",
		"resetFieldNoteForm()",
		"fieldNoteForm.elements['waypoint-id'].value",
		"fieldNotesSection.scrollIntoView",
		"MutationObserver",
		"refreshMapWaypointLogbook",
		"Caching Tools M1.35",
	} {
		if !strings.Contains(asset.Body.String(), want) { t.Fatalf("missing %q from map-logbook.js", want) }
	}

	link := httptest.NewRecorder()
	h.ServeHTTP(link, httptest.NewRequest(http.MethodGet, "/map-link.js", nil))
	if link.Code != http.StatusOK { t.Fatalf("map-link status=%d", link.Code) }
	for _, want := range []string{
		"loadMapLogbook",
		"/map-logbook.js",
		"data-map-logbook",
		"loadFieldNoteDashboard",
		"/field-notes-dashboard.js",
	} {
		if !strings.Contains(link.Body.String(), want) { t.Fatalf("missing %q from map-link.js", want) }
	}
}
