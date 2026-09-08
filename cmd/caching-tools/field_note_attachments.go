package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	fieldNoteAttachmentMaxBytes    int64 = 10 * 1024 * 1024
	fieldNoteAttachmentPreviewMax       = 64 * 1024
)

var allowedFieldNoteAttachmentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png": true,
	"image/webp": true,
	"image/gif": true,
	"application/pdf": true,
	"text/plain; charset=utf-8": true,
	"application/json": true,
	"text/csv": true,
	"application/gpx+xml": true,
	"application/xml": true,
	"text/xml; charset=utf-8": true,
}

type fieldNoteAttachment struct {
	ID          string `json:"id"`
	NoteID      string `json:"note_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	SHA256      string `json:"sha256,omitempty"`
	PreviewKind string `json:"preview_kind,omitempty"`
	TextLines   int    `json:"text_lines,omitempty"`
	PDFVersion  string `json:"pdf_version,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type fieldNoteAttachmentStore struct {
	mu      sync.Mutex
	baseDir string
}

func newFieldNoteAttachmentStore(dataDir string) *fieldNoteAttachmentStore {
	return &fieldNoteAttachmentStore{baseDir: filepath.Join(dataDir, "field-note-attachments")}
}

func (s *fieldNoteAttachmentStore) noteDir(noteID string) string {
	return filepath.Join(s.baseDir, noteID)
}

func sanitizeAttachmentFilename(name string) (string, error) {
	name = strings.TrimSpace(filepath.Base(strings.ReplaceAll(name, "\\", "/")))
	if name == "" || name == "." {
		return "", errors.New("attachment filename is required")
	}
	if len(name) > 255 {
		return "", errors.New("attachment filename is too long")
	}
	return name, nil
}

func safeAttachmentID(id string) bool {
	return strings.HasPrefix(id, "att-") && filepath.Base(id) == id && !strings.ContainsAny(id, `/\\`) && len(id) <= 80
}

func detectAllowedAttachmentType(data []byte) (string, error) {
	contentType := http.DetectContentType(data)
	if allowedFieldNoteAttachmentTypes[contentType] {
		return contentType, nil
	}
	return "", fmt.Errorf("unsupported attachment content type %q", contentType)
}

func attachmentPreviewKind(contentType string) string {
	switch {
	case strings.HasPrefix(contentType, "image/"):
		return "image"
	case contentType == "application/pdf":
		return "pdf"
	case strings.HasPrefix(contentType, "text/"), contentType == "application/json", contentType == "application/xml", contentType == "application/gpx+xml":
		return "text"
	default:
		return "none"
	}
}

func enrichAttachmentMetadata(item fieldNoteAttachment, data []byte) fieldNoteAttachment {
	sum := sha256.Sum256(data)
	item.SHA256 = fmt.Sprintf("%x", sum[:])
	item.Size = int64(len(data))
	item.PreviewKind = attachmentPreviewKind(item.ContentType)
	item.TextLines = 0
	item.PDFVersion = ""
	if item.PreviewKind == "text" && utf8.Valid(data) {
		item.TextLines = 1 + strings.Count(string(data), "\n")
		if len(data) > 0 && data[len(data)-1] == '\n' {
			item.TextLines--
		}
	}
	if item.PreviewKind == "pdf" && len(data) >= 8 && strings.HasPrefix(string(data[:8]), "%PDF-") {
		item.PDFVersion = strings.TrimSpace(string(data[5:8]))
	}
	return item
}

func readAttachment(file multipart.File) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(file, fieldNoteAttachmentMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("attachment is empty")
	}
	if int64(len(data)) > fieldNoteAttachmentMaxBytes {
		return nil, fmt.Errorf("attachment exceeds %d byte limit", fieldNoteAttachmentMaxBytes)
	}
	return data, nil
}

func (s *fieldNoteAttachmentStore) create(noteID, filename string, data []byte) (fieldNoteAttachment, error) {
	filename, err := sanitizeAttachmentFilename(filename)
	if err != nil {
		return fieldNoteAttachment{}, err
	}
	contentType, err := detectAllowedAttachmentType(data)
	if err != nil {
		return fieldNoteAttachment{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	dir := s.noteDir(noteID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fieldNoteAttachment{}, err
	}
	id := fmt.Sprintf("att-%x", time.Now().UnixNano())
	item := fieldNoteAttachment{ID: id, NoteID: noteID, Filename: filename, ContentType: contentType, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	item = enrichAttachmentMetadata(item, data)
	dataPath := filepath.Join(dir, id+".bin")
	metaPath := filepath.Join(dir, id+".json")
	dataTmp := dataPath + ".tmp"
	metaTmp := metaPath + ".tmp"
	if err := os.WriteFile(dataTmp, data, 0o600); err != nil {
		return fieldNoteAttachment{}, err
	}
	if err := os.Rename(dataTmp, dataPath); err != nil {
		_ = os.Remove(dataTmp)
		return fieldNoteAttachment{}, err
	}
	meta, err := json.MarshalIndent(item, "", "  ")
	if err != nil {
		_ = os.Remove(dataPath)
		return fieldNoteAttachment{}, err
	}
	meta = append(meta, '\n')
	if err := os.WriteFile(metaTmp, meta, 0o600); err != nil {
		_ = os.Remove(dataPath)
		return fieldNoteAttachment{}, err
	}
	if err := os.Rename(metaTmp, metaPath); err != nil {
		_ = os.Remove(metaTmp)
		_ = os.Remove(dataPath)
		return fieldNoteAttachment{}, err
	}
	return item, nil
}

func (s *fieldNoteAttachmentStore) list(noteID string) ([]fieldNoteAttachment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries, err := os.ReadDir(s.noteDir(noteID))
	if errors.Is(err, os.ErrNotExist) {
		return []fieldNoteAttachment{}, nil
	}
	if err != nil {
		return nil, err
	}
	items := []fieldNoteAttachment{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		meta, err := os.ReadFile(filepath.Join(s.noteDir(noteID), entry.Name()))
		if err != nil {
			return nil, err
		}
		var item fieldNoteAttachment
		if err := json.Unmarshal(meta, &item); err != nil {
			return nil, fmt.Errorf("read attachment metadata: %w", err)
		}
		if item.NoteID != noteID || !safeAttachmentID(item.ID) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(s.noteDir(noteID), item.ID+".bin"))
		if err != nil {
			return nil, err
		}
		items = append(items, enrichAttachmentMetadata(item, data))
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
	return items, nil
}

func (s *fieldNoteAttachmentStore) get(noteID, id string) (fieldNoteAttachment, []byte, error) {
	if !safeAttachmentID(id) {
		return fieldNoteAttachment{}, nil, os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	metaPath := filepath.Join(s.noteDir(noteID), id+".json")
	meta, err := os.ReadFile(metaPath)
	if errors.Is(err, os.ErrNotExist) {
		return fieldNoteAttachment{}, nil, os.ErrNotExist
	}
	if err != nil {
		return fieldNoteAttachment{}, nil, err
	}
	var item fieldNoteAttachment
	if err := json.Unmarshal(meta, &item); err != nil {
		return fieldNoteAttachment{}, nil, err
	}
	if item.NoteID != noteID || item.ID != id {
		return fieldNoteAttachment{}, nil, os.ErrNotExist
	}
	data, err := os.ReadFile(filepath.Join(s.noteDir(noteID), id+".bin"))
	if errors.Is(err, os.ErrNotExist) {
		return fieldNoteAttachment{}, nil, os.ErrNotExist
	}
	if err != nil {
		return fieldNoteAttachment{}, nil, err
	}
	return enrichAttachmentMetadata(item, data), data, nil
}

func (s *fieldNoteAttachmentStore) delete(noteID, id string) error {
	if !safeAttachmentID(id) {
		return os.ErrNotExist
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	metaPath := filepath.Join(s.noteDir(noteID), id+".json")
	if _, err := os.Stat(metaPath); errors.Is(err, os.ErrNotExist) {
		return os.ErrNotExist
	} else if err != nil {
		return err
	}
	if err := os.Remove(filepath.Join(s.noteDir(noteID), id+".bin")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.Remove(metaPath); err != nil {
		return err
	}
	return nil
}

func (s *fieldNoteAttachmentStore) deleteForNote(noteID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.RemoveAll(s.noteDir(noteID))
}

func fieldNoteExists(store *fieldNoteStore, id string) error {
	_, err := store.get(id)
	return err
}

func writeAttachmentPreview(w http.ResponseWriter, item fieldNoteAttachment, data []byte) error {
	if item.PreviewKind == "none" {
		return errors.New("attachment type does not support preview")
	}
	if item.PreviewKind == "text" && len(data) > fieldNoteAttachmentPreviewMax {
		data = data[:fieldNoteAttachmentPreviewMax]
		for len(data) > 0 && !utf8.Valid(data) {
			data = data[:len(data)-1]
		}
	}
	w.Header().Set("Content-Type", item.ContentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": item.Filename}))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=60")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write(data)
	return err
}

func registerFieldNoteAttachmentRoutes(mux *http.ServeMux, notes *fieldNoteStore, attachments *fieldNoteAttachmentStore) {
	mux.HandleFunc("GET /api/field-notes/{id}/attachments", func(w http.ResponseWriter, r *http.Request) {
		noteID := r.PathValue("id")
		if err := fieldNoteExists(notes, noteID); errors.Is(err, os.ErrNotExist) { writeError(w, 404, errors.New("field note not found")); return } else if err != nil { writeError(w, 500, err); return }
		items, err := attachments.list(noteID)
		if err != nil { writeError(w, 500, err); return }
		writeJSON(w, 200, items)
	})

	mux.HandleFunc("POST /api/field-notes/{id}/attachments", func(w http.ResponseWriter, r *http.Request) {
		noteID := r.PathValue("id")
		if err := fieldNoteExists(notes, noteID); errors.Is(err, os.ErrNotExist) { writeError(w, 404, errors.New("field note not found")); return } else if err != nil { writeError(w, 500, err); return }
		r.Body = http.MaxBytesReader(w, r.Body, fieldNoteAttachmentMaxBytes+1024*1024)
		if err := r.ParseMultipartForm(fieldNoteAttachmentMaxBytes + 1024*1024); err != nil { writeError(w, 400, errors.New("invalid attachment upload: "+err.Error())); return }
		file, header, err := r.FormFile("file")
		if err != nil { writeError(w, 400, errors.New("multipart field file is required")); return }
		defer file.Close()
		data, err := readAttachment(file)
		if err != nil { writeError(w, 400, err); return }
		item, err := attachments.create(noteID, header.Filename, data)
		if err != nil { writeError(w, 400, err); return }
		writeJSON(w, 201, item)
	})

	mux.HandleFunc("GET /api/field-notes/{id}/attachments/{attachment_id}", func(w http.ResponseWriter, r *http.Request) {
		noteID := r.PathValue("id")
		if err := fieldNoteExists(notes, noteID); errors.Is(err, os.ErrNotExist) { writeError(w, 404, errors.New("field note not found")); return } else if err != nil { writeError(w, 500, err); return }
		item, data, err := attachments.get(noteID, r.PathValue("attachment_id"))
		if errors.Is(err, os.ErrNotExist) { writeError(w, 404, errors.New("attachment not found")); return }
		if err != nil { writeError(w, 500, err); return }
		w.Header().Set("Content-Type", item.ContentType)
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": item.Filename}))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(200)
		_, _ = w.Write(data)
	})

	mux.HandleFunc("GET /api/field-notes/{id}/attachments/{attachment_id}/preview", func(w http.ResponseWriter, r *http.Request) {
		noteID := r.PathValue("id")
		if err := fieldNoteExists(notes, noteID); errors.Is(err, os.ErrNotExist) { writeError(w, 404, errors.New("field note not found")); return } else if err != nil { writeError(w, 500, err); return }
		item, data, err := attachments.get(noteID, r.PathValue("attachment_id"))
		if errors.Is(err, os.ErrNotExist) { writeError(w, 404, errors.New("attachment not found")); return }
		if err != nil { writeError(w, 500, err); return }
		if err := writeAttachmentPreview(w, item, data); err != nil { writeError(w, 415, err); return }
	})

	mux.HandleFunc("DELETE /api/field-notes/{id}/attachments/{attachment_id}", func(w http.ResponseWriter, r *http.Request) {
		noteID := r.PathValue("id")
		if err := fieldNoteExists(notes, noteID); errors.Is(err, os.ErrNotExist) { writeError(w, 404, errors.New("field note not found")); return } else if err != nil { writeError(w, 500, err); return }
		err := attachments.delete(noteID, r.PathValue("attachment_id"))
		if errors.Is(err, os.ErrNotExist) { writeError(w, 404, errors.New("attachment not found")); return }
		if err != nil { writeError(w, 500, err); return }
		w.WriteHeader(204)
	})

	registerFieldNoteArchiveRoutes(mux, notes, attachments)
}
