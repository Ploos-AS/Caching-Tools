package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFieldNoteBundleImportCreatesNewIDs(t *testing.T) {
	store := newFieldNoteStore(t.TempDir())
	original, err := store.create(fieldNoteRequest{Title: "Existing", OccurredAt: "2026-09-08T05:00:00Z"})
	if err != nil { t.Fatal(err) }
	bundle := fieldNoteBundle{
		Format: fieldNoteBundleFormat,
		Version: fieldNoteBundleVersion,
		Notes: []fieldNote{{
			ID: "note-source", Title: "Imported", Body: "Backup body", Status: "found", Type: "cache",
			WaypointID: "wp-source", WorkspaceID: "ws-source", OccurredAt: "2026-09-08T06:00:00Z",
			CreatedAt: "2000-01-01T00:00:00Z", UpdatedAt: "2000-01-01T00:00:00Z",
		}},
	}
	created, err := importFieldNoteBundle(bundle, store)
	if err != nil { t.Fatal(err) }
	if len(created) != 1 { t.Fatalf("created=%#v", created) }
	if created[0].ID == "note-source" || created[0].ID == original.ID { t.Fatalf("source/existing ID reused: %#v", created[0]) }
	if created[0].Title != "Imported" || created[0].WaypointID != "wp-source" || created[0].WorkspaceID != "ws-source" { t.Fatalf("created=%#v", created[0]) }
	if created[0].CreatedAt == "2000-01-01T00:00:00Z" || created[0].UpdatedAt == "2000-01-01T00:00:00Z" { t.Fatalf("source lifecycle timestamps reused: %#v", created[0]) }
	items, err := store.list(); if err != nil { t.Fatal(err) }
	if len(items) != 2 { t.Fatalf("items=%#v", items) }
}

func TestFieldNoteBundleImportIsAtomicOnInvalidNote(t *testing.T) {
	store := newFieldNoteStore(t.TempDir())
	if _, err := store.create(fieldNoteRequest{Title: "Existing", OccurredAt: "2026-09-08T05:00:00Z"}); err != nil { t.Fatal(err) }
	bundle := fieldNoteBundle{
		Format: fieldNoteBundleFormat, Version: fieldNoteBundleVersion,
		Notes: []fieldNote{
			{Title: "Valid", OccurredAt: "2026-09-08T06:00:00Z"},
			{Title: "", OccurredAt: "2026-09-08T07:00:00Z"},
		},
	}
	if _, err := importFieldNoteBundle(bundle, store); err == nil { t.Fatal("expected validation error") }
	items, err := store.list(); if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].Title != "Existing" { t.Fatalf("atomicity violated: %#v", items) }
}

func TestFieldNoteBundleImportAPI(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	bundle := fieldNoteBundle{
		Format: fieldNoteBundleFormat, Version: fieldNoteBundleVersion, ExportedAt: "2026-09-08T06:30:00Z",
		Notes: []fieldNote{{ID:"old-id", Title:"Imported via API", OccurredAt:"2026-09-08T06:00:00Z"}},
	}
	body, _ := json.Marshal(bundle)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/field-notes/import", bytes.NewReader(body)))
	if rec.Code != http.StatusCreated { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
	var result fieldNoteImportResult
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil { t.Fatal(err) }
	if result.Imported != 1 || len(result.Notes) != 1 || result.Notes[0].ID == "old-id" { t.Fatalf("result=%#v", result) }

	badFormat := []byte(`{"format":"wrong","version":1,"notes":[]}`)
	bad := httptest.NewRecorder()
	h.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, "/api/field-notes/import", bytes.NewReader(badFormat)))
	if bad.Code != http.StatusBadRequest { t.Fatalf("bad format status=%d", bad.Code) }

	badVersion := []byte(`{"format":"caching-tools-field-notes","version":2,"notes":[]}`)
	bad = httptest.NewRecorder()
	h.ServeHTTP(bad, httptest.NewRequest(http.MethodPost, "/api/field-notes/import", bytes.NewReader(badVersion)))
	if bad.Code != http.StatusBadRequest { t.Fatalf("bad version status=%d", bad.Code) }
}

func TestFieldNoteImportUIAsset(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/field-notes.js", nil))
	if rec.Code != http.StatusOK { t.Fatalf("status=%d", rec.Code) }
	for _, want := range []string{
		"Backup / restore", "field-note-import-file", "Import JSON backup", "field-note-import-json",
		"importFieldNotesJSON", "/api/field-notes/import", "file.text()", "JSON.parse", "new local IDs", "Caching Tools M1.33",
	} {
		if !bytes.Contains(rec.Body.Bytes(), []byte(want)) { t.Fatalf("missing %q", want) }
	}
}
