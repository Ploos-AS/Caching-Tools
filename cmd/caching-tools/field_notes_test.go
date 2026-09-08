package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFieldNoteStoreRoundTrip(t *testing.T) {
	store := newFieldNoteStore(t.TempDir())
	created, err := store.create(fieldNoteRequest{Title:"Found cache", Body:"Dry container", Status:"found", Type:"cache", WaypointID:"wp-1", WorkspaceID:"ws-1", OccurredAt:"2026-09-08T05:00:00Z"})
	if err != nil { t.Fatal(err) }
	if created.ID == "" || created.Title != "Found cache" || created.WaypointID != "wp-1" || created.WorkspaceID != "ws-1" { t.Fatalf("unexpected created note: %#v", created) }
	if _, err := os.Stat(store.path); err != nil { t.Fatal(err) }
	items, err := store.list(); if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].ID != created.ID { t.Fatalf("items=%#v", items) }
	updated, err := store.update(created.ID, fieldNoteRequest{Title:"Found cache updated", Body:"Log text", Status:"found", Type:"cache", OccurredAt:"2026-09-08T05:01:00Z"})
	if err != nil { t.Fatal(err) }
	if updated.Title != "Found cache updated" || updated.WaypointID != "" { t.Fatalf("updated=%#v", updated) }
	if err := store.delete(created.ID); err != nil { t.Fatal(err) }
	items, err = store.list(); if err != nil { t.Fatal(err) }
	if len(items) != 0 { t.Fatalf("items=%#v", items) }
}

func TestFieldNoteValidation(t *testing.T) {
	store := newFieldNoteStore(t.TempDir())
	if _, err := store.create(fieldNoteRequest{}); err == nil { t.Fatal("expected title validation error") }
	if _, err := store.create(fieldNoteRequest{Title:"x", OccurredAt:"not-a-time"}); err == nil { t.Fatal("expected occurred_at validation error") }
}

func TestFieldNoteAPI(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("CACHING_TOOLS_DATA_DIR", dataDir)
	h, err := newHandler(); if err != nil { t.Fatal(err) }

	body := []byte(`{"title":"Trail note","body":"Bridge is slippery","status":"info","type":"trail","waypoint_id":"wp-example","occurred_at":"2026-09-08T05:00:00Z"}`)
	create := httptest.NewRecorder()
	h.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/field-notes", bytes.NewReader(body)))
	if create.Code != http.StatusCreated { t.Fatalf("create status=%d body=%s", create.Code, create.Body.String()) }
	var note fieldNote
	if err := json.Unmarshal(create.Body.Bytes(), &note); err != nil { t.Fatal(err) }
	if note.ID == "" || note.Status != "info" || note.WaypointID != "wp-example" { t.Fatalf("note=%#v", note) }

	list := httptest.NewRecorder()
	h.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/field-notes", nil))
	if list.Code != http.StatusOK { t.Fatalf("list status=%d", list.Code) }
	var notes []fieldNote
	if err := json.Unmarshal(list.Body.Bytes(), &notes); err != nil { t.Fatal(err) }
	if len(notes) != 1 || notes[0].ID != note.ID { t.Fatalf("notes=%#v", notes) }

	updateBody := []byte(`{"title":"Updated","body":"Done","status":"done","type":"trail","occurred_at":"2026-09-08T05:30:00Z"}`)
	update := httptest.NewRecorder()
	h.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/api/field-notes/"+note.ID, bytes.NewReader(updateBody)))
	if update.Code != http.StatusOK { t.Fatalf("update status=%d body=%s", update.Code, update.Body.String()) }

	missing := httptest.NewRecorder()
	h.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/field-notes/missing", nil))
	if missing.Code != http.StatusNotFound { t.Fatalf("missing status=%d", missing.Code) }

	deleteRec := httptest.NewRecorder()
	h.ServeHTTP(deleteRec, httptest.NewRequest(http.MethodDelete, "/api/field-notes/"+note.ID, nil))
	if deleteRec.Code != http.StatusNoContent { t.Fatalf("delete status=%d", deleteRec.Code) }

	if _, err := os.Stat(filepath.Join(dataDir, "field-notes.json")); err != nil { t.Fatal(err) }
}
