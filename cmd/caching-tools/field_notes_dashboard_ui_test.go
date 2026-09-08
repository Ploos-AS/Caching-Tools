package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFieldNotesDashboardUI(t *testing.T) {
	h, err := newHandler()
	if err != nil { t.Fatal(err) }

	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/field-notes-dashboard.js", nil))
	if asset.Code != http.StatusOK { t.Fatalf("asset status=%d", asset.Code) }
	for _, want := range []string{
		"Logbook dashboard",
		"no telemetry is sent anywhere",
		"field-note-dashboard-refresh",
		"field-note-dashboard-summary",
		"dashboardCountBy",
		"dashboardTopLinks",
		"dashboardActivityMonths",
		"dashboardRecentCount",
		"renderLogbookDashboard",
		"refreshLogbookDashboard",
		"Total field notes",
		"Last 7 days",
		"Last 30 days",
		"Status distribution",
		"Type distribution",
		"Activity by month (UTC, last 6 months)",
		"Most-used linked waypoints",
		"Most-used linked mystery workspaces",
		"/api/field-notes",
		"/api/waypoints",
		"/api/mystery-workspaces",
		"MutationObserver",
		"Caching Tools M1.34",
	} {
		if !strings.Contains(asset.Body.String(), want) { t.Fatalf("missing %q", want) }
	}

	link := httptest.NewRecorder()
	h.ServeHTTP(link, httptest.NewRequest(http.MethodGet, "/map-link.js", nil))
	if link.Code != http.StatusOK { t.Fatalf("map-link status=%d", link.Code) }
	for _, want := range []string{"loadFieldNoteDashboard", "/field-notes-dashboard.js", "data-field-note-dashboard"} {
		if !strings.Contains(link.Body.String(), want) { t.Fatalf("missing %q from map-link.js", want) }
	}
}
