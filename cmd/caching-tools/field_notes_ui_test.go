package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFieldNotesSearchFilterExportUI(t *testing.T) {
	h, err := newHandler()
	if err != nil { t.Fatal(err) }

	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/field-notes.js", nil))
	if asset.Code != http.StatusOK { t.Fatalf("asset status=%d", asset.Code) }

	for _, want := range []string{
		"Search / filter",
		"field-note-search",
		"field-note-filter-status",
		"field-note-filter-type",
		"field-note-filter-waypoint",
		"field-note-filter-workspace",
		"Clear filters",
		"Export filtered JSON",
		"Export filtered CSV",
		"filteredFieldNotes",
		"applyFieldNoteFilters",
		"noteFilterValues",
		"fieldNotesCSV",
		"csvCell",
		"exportFilteredFieldNotesJSON",
		"exportFilteredFieldNotesCSV",
		"downloadFieldNoteExport",
		"caching-tools-field-notes",
		"application/json;charset=utf-8",
		"text/csv;charset=utf-8",
		"URL.createObjectURL",
		"URL.revokeObjectURL",
	} {
		if !strings.Contains(asset.Body.String(), want) { t.Fatalf("missing %q", want) }
	}
}
