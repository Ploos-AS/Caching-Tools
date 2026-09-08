package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFieldNoteArchiveRoundTrip(t *testing.T) {
	sourceDir := t.TempDir()
	sourceNotes := newFieldNoteStore(sourceDir)
	sourceAttachments := newFieldNoteAttachmentStore(sourceDir)
	note, err := sourceNotes.create(fieldNoteRequest{Title:"Cache visit", Body:"Photo attached", Status:"found", Type:"cache", WaypointID:"wp-old", WorkspaceID:"ws-old", OccurredAt:"2026-09-08T14:00:00Z"})
	if err != nil { t.Fatal(err) }
	attachment, err := sourceAttachments.create(note.ID, "cache.png", pngFixture())
	if err != nil { t.Fatal(err) }

	archive, err := buildFieldNoteArchive(sourceNotes, sourceAttachments)
	if err != nil { t.Fatal(err) }
	zr, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil { t.Fatal(err) }
	if len(zr.File) != 2 { t.Fatalf("entries=%d", len(zr.File)) }
	var manifest fieldNoteArchiveManifest
	for _, file := range zr.File {
		if file.Name != "manifest.json" { continue }
		r, err := file.Open(); if err != nil { t.Fatal(err) }
		if err := json.NewDecoder(r).Decode(&manifest); err != nil { _ = r.Close(); t.Fatal(err) }
		_ = r.Close()
	}
	if manifest.Format != fieldNoteArchiveFormat || manifest.Version != 1 || len(manifest.Notes) != 1 { t.Fatalf("manifest=%#v", manifest) }
	if manifest.Notes[0].SourceID != note.ID || manifest.Notes[0].Attachments[0].SourceID != attachment.ID { t.Fatalf("manifest note=%#v", manifest.Notes[0]) }

	destDir := t.TempDir()
	destNotes := newFieldNoteStore(destDir)
	destAttachments := newFieldNoteAttachmentStore(destDir)
	result, err := restoreFieldNoteArchive(archive, destNotes, destAttachments)
	if err != nil { t.Fatal(err) }
	if result.ImportedNotes != 1 || result.ImportedAttachments != 1 { t.Fatalf("result=%#v", result) }

	restored, err := destNotes.list()
	if err != nil { t.Fatal(err) }
	if len(restored) != 1 { t.Fatalf("notes=%#v", restored) }
	if restored[0].ID == note.ID { t.Fatal("source note ID was reused") }
	if restored[0].CreatedAt == note.CreatedAt { t.Fatal("source created_at was reused") }
	if restored[0].WaypointID != "wp-old" || restored[0].WorkspaceID != "ws-old" { t.Fatalf("soft refs lost: %#v", restored[0]) }
	linked, err := destAttachments.list(restored[0].ID)
	if err != nil { t.Fatal(err) }
	if len(linked) != 1 || linked[0].ID == attachment.ID || linked[0].Filename != "cache.png" { t.Fatalf("attachments=%#v", linked) }
	_, restoredData, err := destAttachments.get(restored[0].ID, linked[0].ID)
	if err != nil { t.Fatal(err) }
	if !bytes.Equal(restoredData, pngFixture()) { t.Fatal("attachment data mismatch") }
}

func TestFieldNoteArchiveRejectsUnreferencedEntryBeforeImport(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	manifest := fieldNoteArchiveManifest{Format:fieldNoteArchiveFormat, Version:1, ExportedAt:"2026-09-08T14:00:00Z", Notes:[]fieldNoteArchiveNote{{SourceID:"note-old", Note:fieldNoteRequest{Title:"Valid", OccurredAt:"2026-09-08T14:00:00Z"}}}}
	part, _ := writer.Create("manifest.json")
	data, _ := json.Marshal(manifest)
	_, _ = part.Write(data)
	extra, _ := writer.Create("unexpected.bin")
	_, _ = extra.Write(pngFixture())
	_ = writer.Close()

	dir := t.TempDir()
	notes := newFieldNoteStore(dir)
	attachments := newFieldNoteAttachmentStore(dir)
	if _, err := restoreFieldNoteArchive(buffer.Bytes(), notes, attachments); err == nil || !strings.Contains(err.Error(), "unreferenced") { t.Fatalf("err=%v", err) }
	items, err := notes.list(); if err != nil { t.Fatal(err) }
	if len(items) != 0 { t.Fatalf("partial import: %#v", items) }
}

func TestFieldNoteArchiveHTTPExportRestore(t *testing.T) {
	sourceDir := t.TempDir()
	t.Setenv("CACHING_TOOLS_DATA_DIR", sourceDir)
	h, err := newHandler(); if err != nil { t.Fatal(err) }

	create := httptest.NewRecorder()
	h.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/field-notes", strings.NewReader(`{"title":"Archive me","occurred_at":"2026-09-08T14:00:00Z"}`)))
	if create.Code != http.StatusCreated { t.Fatalf("create=%d %s", create.Code, create.Body.String()) }
	var note fieldNote
	if err := json.Unmarshal(create.Body.Bytes(), &note); err != nil { t.Fatal(err) }

	var uploadBody bytes.Buffer
	mw := multipartWriterForArchiveTest(t, &uploadBody, "proof.png", pngFixture())
	req := httptest.NewRequest(http.MethodPost, "/api/field-notes/"+note.ID+"/attachments", &uploadBody)
	req.Header.Set("Content-Type", mw)
	upload := httptest.NewRecorder(); h.ServeHTTP(upload, req)
	if upload.Code != http.StatusCreated { t.Fatalf("upload=%d %s", upload.Code, upload.Body.String()) }

	export := httptest.NewRecorder(); h.ServeHTTP(export, httptest.NewRequest(http.MethodGet, "/api/field-notes/archive", nil))
	if export.Code != http.StatusOK { t.Fatalf("export=%d %s", export.Code, export.Body.String()) }
	if export.Header().Get("Content-Type") != "application/zip" || export.Header().Get("X-Content-Type-Options") != "nosniff" { t.Fatalf("headers=%v", export.Header()) }

	destDir := t.TempDir()
	t.Setenv("CACHING_TOOLS_DATA_DIR", destDir)
	dest, err := newHandler(); if err != nil { t.Fatal(err) }
	restore := httptest.NewRecorder(); restoreReq := httptest.NewRequest(http.MethodPost, "/api/field-notes/archive", bytes.NewReader(export.Body.Bytes()))
	restoreReq.Header.Set("Content-Type", "application/zip")
	dest.ServeHTTP(restore, restoreReq)
	if restore.Code != http.StatusCreated { t.Fatalf("restore=%d %s", restore.Code, restore.Body.String()) }
	if !strings.Contains(restore.Body.String(), `"imported_notes":1`) || !strings.Contains(restore.Body.String(), `"imported_attachments":1`) { t.Fatalf("restore body=%s", restore.Body.String()) }
}

func multipartWriterForArchiveTest(t *testing.T, body *bytes.Buffer, filename string, data []byte) string {
	t.Helper()
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename); if err != nil { t.Fatal(err) }
	if _, err := part.Write(data); err != nil { t.Fatal(err) }
	if err := writer.Close(); err != nil { t.Fatal(err) }
	return writer.FormDataContentType()
}
