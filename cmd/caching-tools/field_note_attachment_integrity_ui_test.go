package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFieldNoteAttachmentIntegrityUI(t *testing.T) {
	h, err := newHandler()
	if err != nil { t.Fatal(err) }
	asset := httptest.NewRecorder()
	h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/field-note-integrity.js", nil))
	if asset.Code != http.StatusOK { t.Fatalf("asset status=%d", asset.Code) }
	for _, want := range []string{"Attachment integrity", "Verify all attachments", "Download integrity report", "/api/field-note-attachments/integrity", "Mismatch", "Missing", "Unrecorded checksum", "Caching Tools M1.39"} {
		if !strings.Contains(asset.Body.String(), want) { t.Fatalf("missing %q", want) }
	}
	link := httptest.NewRecorder()
	h.ServeHTTP(link, httptest.NewRequest(http.MethodGet, "/map-link.js", nil))
	if link.Code != http.StatusOK { t.Fatalf("map-link status=%d", link.Code) }
	for _, want := range []string{"loadFieldNoteIntegrity", "/field-note-integrity.js", "data-field-note-integrity", "loadFieldNotesAsset", "loadFieldNoteAttachmentsAsset"} {
		if !strings.Contains(link.Body.String(), want) { t.Fatalf("missing %q", want) }
	}
}
