package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFieldNoteArchiveUI(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/field-note-archive.js", nil))
	if asset.Code != http.StatusOK { t.Fatalf("asset status=%d", asset.Code) }
	for _, want := range []string{
		"Field-note archive backup / restore",
		"Download ZIP backup",
		"Restore ZIP backup",
		"/api/field-notes/archive",
		"application/zip",
		"128 * 1024 * 1024",
		"imported_notes",
		"imported_attachments",
		"refreshFieldNotes",
		"refreshFieldNoteAttachments",
		"refreshLogbookDashboard",
		"Caching Tools M1.37",
	} {
		if !strings.Contains(asset.Body.String(), want) { t.Fatalf("missing %q", want) }
	}

	loader := httptest.NewRecorder()
	h.ServeHTTP(loader, httptest.NewRequest(http.MethodGet, "/map-link.js", nil))
	if loader.Code != http.StatusOK { t.Fatalf("loader status=%d", loader.Code) }
	for _, want := range []string{"loadFieldNoteArchive", "/field-note-archive.js", "data-field-note-archive", "loadFieldNoteAttachments"} {
		if !strings.Contains(loader.Body.String(), want) { t.Fatalf("loader missing %q", want) }
	}
}
