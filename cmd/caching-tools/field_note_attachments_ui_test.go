package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFieldNoteAttachmentUIAssets(t *testing.T) {
	h, err := newHandler()
	if err != nil { t.Fatal(err) }

	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/field-note-attachments.js", nil))
	if asset.Code != http.StatusOK { t.Fatalf("asset status=%d", asset.Code) }
	for _, want := range []string{
		"Field-note attachments",
		"/data/field-note-attachments",
		"10 MiB",
		"field-note-attachment-note",
		"field-note-attachment-file",
		"field-note-attachment-preview",
		"Upload attachment",
		"Refresh attachments",
		"loadFieldNoteAttachments",
		"uploadFieldNoteAttachment",
		"showAttachmentPreview",
		"SHA-256:",
		"Text lines:",
		"PDF version:",
		"preview_kind",
		"/preview",
		"new FormData",
		"/attachments",
		"Delete attachment",
		"Download",
		"Preview",
		"Caching Tools M1.38",
	} {
		if !strings.Contains(asset.Body.String(), want) { t.Fatalf("missing %q", want) }
	}

	link := httptest.NewRecorder()
	h.ServeHTTP(link, httptest.NewRequest(http.MethodGet, "/map-link.js", nil))
	if link.Code != http.StatusOK { t.Fatalf("map-link status=%d", link.Code) }
	for _, want := range []string{"loadFieldNoteAttachments", "/field-note-attachments.js", "data-field-note-attachments", "loadMapLogbook"} {
		if !strings.Contains(link.Body.String(), want) { t.Fatalf("missing %q from map-link.js", want) }
	}
}
