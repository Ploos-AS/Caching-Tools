package main

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pngFixture() []byte {
	return append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, make([]byte, 32)...)
}

func TestFieldNoteAttachmentStoreRoundTrip(t *testing.T) {
	store := newFieldNoteAttachmentStore(t.TempDir())
	created, err := store.create("note-test", "../photo.png", pngFixture())
	if err != nil { t.Fatal(err) }
	if created.Filename != "photo.png" || created.ContentType != "image/png" || created.Size != int64(len(pngFixture())) { t.Fatalf("created=%#v", created) }

	items, err := store.list("note-test")
	if err != nil { t.Fatal(err) }
	if len(items) != 1 || items[0].ID != created.ID { t.Fatalf("items=%#v", items) }

	meta, data, err := store.get("note-test", created.ID)
	if err != nil { t.Fatal(err) }
	if meta.ID != created.ID || !bytes.Equal(data, pngFixture()) { t.Fatalf("meta=%#v data=%d", meta, len(data)) }

	if err := store.delete("note-test", created.ID); err != nil { t.Fatal(err) }
	items, err = store.list("note-test")
	if err != nil { t.Fatal(err) }
	if len(items) != 0 { t.Fatalf("items=%#v", items) }
}

func TestFieldNoteAttachmentValidation(t *testing.T) {
	store := newFieldNoteAttachmentStore(t.TempDir())
	if _, err := store.create("note-test", "bad.bin", []byte{0, 1, 2, 3, 4}); err == nil { t.Fatal("expected unsupported content type error") }
	if data, err := ioReadAttachmentForTest(bytes.NewReader(nil)); err == nil || data != nil { t.Fatal("expected empty attachment error") }
	tooLarge := bytes.NewReader(make([]byte, fieldNoteAttachmentMaxBytes+1))
	data, err := ioReadAttachmentForTest(tooLarge)
	if err == nil || data != nil { t.Fatal("expected size limit error") }
}

func ioReadAttachmentForTest(r *bytes.Reader) ([]byte, error) {
	return readAttachment(readerMultipartFile{Reader:r})
}

type readerMultipartFile struct{ *bytes.Reader }
func (f readerMultipartFile) Close() error { return nil }

func TestFieldNoteAttachmentHTTPRoundTripAndCascadeDelete(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("CACHING_TOOLS_DATA_DIR", dataDir)
	h, err := newHandler()
	if err != nil { t.Fatal(err) }

	createNote := httptest.NewRecorder()
	h.ServeHTTP(createNote, httptest.NewRequest(http.MethodPost, "/api/field-notes", strings.NewReader(`{"title":"Photo stop","occurred_at":"2026-09-08T14:00:00Z"}`)))
	if createNote.Code != http.StatusCreated { t.Fatalf("create note status=%d body=%s", createNote.Code, createNote.Body.String()) }
	var note fieldNote
	if err := json.Unmarshal(createNote.Body.Bytes(), &note); err != nil { t.Fatal(err) }

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "cache-photo.png")
	if err != nil { t.Fatal(err) }
	if _, err := part.Write(pngFixture()); err != nil { t.Fatal(err) }
	if err := writer.Close(); err != nil { t.Fatal(err) }

	uploadReq := httptest.NewRequest(http.MethodPost, "/api/field-notes/"+note.ID+"/attachments", &body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	upload := httptest.NewRecorder()
	h.ServeHTTP(upload, uploadReq)
	if upload.Code != http.StatusCreated { t.Fatalf("upload status=%d body=%s", upload.Code, upload.Body.String()) }
	var attachment fieldNoteAttachment
	if err := json.Unmarshal(upload.Body.Bytes(), &attachment); err != nil { t.Fatal(err) }
	if attachment.NoteID != note.ID || attachment.Filename != "cache-photo.png" { t.Fatalf("attachment=%#v", attachment) }

	list := httptest.NewRecorder()
	h.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/field-notes/"+note.ID+"/attachments", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), attachment.ID) { t.Fatalf("list status=%d body=%s", list.Code, list.Body.String()) }

	download := httptest.NewRecorder()
	h.ServeHTTP(download, httptest.NewRequest(http.MethodGet, "/api/field-notes/"+note.ID+"/attachments/"+attachment.ID, nil))
	if download.Code != http.StatusOK { t.Fatalf("download status=%d body=%s", download.Code, download.Body.String()) }
	if download.Header().Get("Content-Type") != "image/png" { t.Fatalf("content-type=%q", download.Header().Get("Content-Type")) }
	if download.Header().Get("X-Content-Type-Options") != "nosniff" { t.Fatal("missing nosniff") }
	if !bytes.Equal(download.Body.Bytes(), pngFixture()) { t.Fatal("downloaded data mismatch") }

	attachmentDir := filepath.Join(dataDir, "field-note-attachments", note.ID)
	if _, err := os.Stat(attachmentDir); err != nil { t.Fatal(err) }
	deleteNote := httptest.NewRecorder()
	h.ServeHTTP(deleteNote, httptest.NewRequest(http.MethodDelete, "/api/field-notes/"+note.ID, nil))
	if deleteNote.Code != http.StatusNoContent { t.Fatalf("delete note status=%d body=%s", deleteNote.Code, deleteNote.Body.String()) }
	if _, err := os.Stat(attachmentDir); !os.IsNotExist(err) { t.Fatalf("attachment directory still exists: %v", err) }
}

func TestFieldNoteAttachmentMissingNote(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler()
	if err != nil { t.Fatal(err) }
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/field-notes/missing/attachments", nil))
	if rec.Code != http.StatusNotFound { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
}
