package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

const (
	fieldNoteArchiveFormat         = "caching-tools-field-note-archive"
	fieldNoteArchiveVersion        = 1
	fieldNoteArchiveMaxBytes int64 = 128 * 1024 * 1024
	fieldNoteArchiveManifestMax    = 2 * 1024 * 1024
	fieldNoteArchiveMaxEntries     = 20000
)

type fieldNoteArchiveAttachment struct {
	SourceID    string `json:"source_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Path        string `json:"path"`
}

type fieldNoteArchiveNote struct {
	SourceID    string                       `json:"source_id"`
	Note        fieldNoteRequest             `json:"note"`
	Attachments []fieldNoteArchiveAttachment `json:"attachments,omitempty"`
}

type fieldNoteArchiveManifest struct {
	Format     string                 `json:"format"`
	Version    int                    `json:"version"`
	ExportedAt string                 `json:"exported_at"`
	Notes      []fieldNoteArchiveNote `json:"notes"`
}

type fieldNoteArchiveImportResult struct {
	ImportedNotes       int `json:"imported_notes"`
	ImportedAttachments int `json:"imported_attachments"`
}

type validatedArchiveAttachment struct {
	Filename string
	Data     []byte
}

type validatedArchiveNote struct {
	Request     fieldNoteRequest
	Attachments []validatedArchiveAttachment
}

func buildFieldNoteArchive(notes *fieldNoteStore, attachments *fieldNoteAttachmentStore) ([]byte, error) {
	items, err := notes.list()
	if err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	manifest := fieldNoteArchiveManifest{Format: fieldNoteArchiveFormat, Version: fieldNoteArchiveVersion, ExportedAt: time.Now().UTC().Format(time.RFC3339Nano), Notes: make([]fieldNoteArchiveNote, 0, len(items))}
	fileIndex := 0
	for _, note := range items {
		archiveNote := fieldNoteArchiveNote{SourceID: note.ID, Note: fieldNoteRequest{Title: note.Title, Body: note.Body, Status: note.Status, Type: note.Type, WaypointID: note.WaypointID, WorkspaceID: note.WorkspaceID, OccurredAt: note.OccurredAt}}
		linked, err := attachments.list(note.ID)
		if err != nil {
			_ = writer.Close()
			return nil, err
		}
		for _, attachment := range linked {
			meta, data, err := attachments.get(note.ID, attachment.ID)
			if err != nil {
				_ = writer.Close()
				return nil, err
			}
			fileIndex++
			entryPath := fmt.Sprintf("attachments/%06d.bin", fileIndex)
			part, err := writer.Create(entryPath)
			if err != nil {
				_ = writer.Close()
				return nil, err
			}
			if _, err := part.Write(data); err != nil {
				_ = writer.Close()
				return nil, err
			}
			archiveNote.Attachments = append(archiveNote.Attachments, fieldNoteArchiveAttachment{SourceID: meta.ID, Filename: meta.Filename, ContentType: meta.ContentType, Size: meta.Size, Path: entryPath})
		}
		manifest.Notes = append(manifest.Notes, archiveNote)
	}
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		_ = writer.Close()
		return nil, err
	}
	manifestData = append(manifestData, '\n')
	part, err := writer.Create("manifest.json")
	if err != nil {
		_ = writer.Close()
		return nil, err
	}
	if _, err := part.Write(manifestData); err != nil {
		_ = writer.Close()
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func safeArchiveEntryName(name string) bool {
	return name != "" && path.Clean(name) == name && !strings.HasPrefix(name, "/") && !strings.HasPrefix(name, "../") && !strings.Contains(name, "\\")
}

func readZipEntry(file *zip.File, limit int64) ([]byte, error) {
	if file.FileInfo().IsDir() {
		return nil, errors.New("archive directories are not supported")
	}
	if file.UncompressedSize64 > uint64(limit) {
		return nil, errors.New("archive entry exceeds size limit")
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("archive entry exceeds size limit")
	}
	return data, nil
}

func validateFieldNoteArchive(data []byte) ([]validatedArchiveNote, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, errors.New("invalid field note archive")
	}
	if len(reader.File) == 0 || len(reader.File) > fieldNoteArchiveMaxEntries {
		return nil, errors.New("invalid field note archive entry count")
	}
	entries := make(map[string]*zip.File, len(reader.File))
	var total uint64
	for _, file := range reader.File {
		if !safeArchiveEntryName(file.Name) {
			return nil, errors.New("unsafe archive entry path")
		}
		if _, exists := entries[file.Name]; exists {
			return nil, errors.New("duplicate archive entry")
		}
		total += file.UncompressedSize64
		if total > uint64(fieldNoteArchiveMaxBytes) {
			return nil, errors.New("archive expands beyond size limit")
		}
		entries[file.Name] = file
	}
	manifestFile := entries["manifest.json"]
	if manifestFile == nil {
		return nil, errors.New("archive manifest is missing")
	}
	manifestData, err := readZipEntry(manifestFile, fieldNoteArchiveManifestMax)
	if err != nil {
		return nil, err
	}
	var manifest fieldNoteArchiveManifest
	decoder := json.NewDecoder(bytes.NewReader(manifestData))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return nil, errors.New("invalid archive manifest")
	}
	if manifest.Format != fieldNoteArchiveFormat {
		return nil, errors.New("unsupported field note archive format")
	}
	if manifest.Version != fieldNoteArchiveVersion {
		return nil, fmt.Errorf("unsupported field note archive version %d", manifest.Version)
	}
	used := map[string]bool{"manifest.json": true}
	validated := make([]validatedArchiveNote, 0, len(manifest.Notes))
	for i, archived := range manifest.Notes {
		normalized, err := normalizeFieldNoteRequest(archived.Note)
		if err != nil {
			return nil, fmt.Errorf("note %d: %w", i+1, err)
		}
		item := validatedArchiveNote{Request: normalized}
		for j, attachment := range archived.Attachments {
			if !safeArchiveEntryName(attachment.Path) || attachment.Path == "manifest.json" {
				return nil, fmt.Errorf("note %d attachment %d: invalid path", i+1, j+1)
			}
			if used[attachment.Path] {
				return nil, fmt.Errorf("note %d attachment %d: duplicate path", i+1, j+1)
			}
			file := entries[attachment.Path]
			if file == nil {
				return nil, fmt.Errorf("note %d attachment %d: file missing", i+1, j+1)
			}
			attachmentData, err := readZipEntry(file, fieldNoteAttachmentMaxBytes)
			if err != nil {
				return nil, fmt.Errorf("note %d attachment %d: %w", i+1, j+1, err)
			}
			if attachment.Size != 0 && attachment.Size != int64(len(attachmentData)) {
				return nil, fmt.Errorf("note %d attachment %d: size mismatch", i+1, j+1)
			}
			filename, err := sanitizeAttachmentFilename(attachment.Filename)
			if err != nil {
				return nil, fmt.Errorf("note %d attachment %d: %w", i+1, j+1, err)
			}
			if _, err := detectAllowedAttachmentType(attachmentData); err != nil {
				return nil, fmt.Errorf("note %d attachment %d: %w", i+1, j+1, err)
			}
			used[attachment.Path] = true
			item.Attachments = append(item.Attachments, validatedArchiveAttachment{Filename: filename, Data: attachmentData})
		}
		validated = append(validated, item)
	}
	if len(used) != len(entries) {
		return nil, errors.New("archive contains unreferenced entries")
	}
	return validated, nil
}

func restoreFieldNoteArchive(data []byte, notes *fieldNoteStore, attachments *fieldNoteAttachmentStore) (fieldNoteArchiveImportResult, error) {
	validated, err := validateFieldNoteArchive(data)
	if err != nil {
		return fieldNoteArchiveImportResult{}, err
	}
	requests := make([]fieldNoteRequest, len(validated))
	for i := range validated {
		requests[i] = validated[i].Request
	}
	created, err := notes.importMany(requests)
	if err != nil {
		return fieldNoteArchiveImportResult{}, err
	}
	rollback := func() {
		for _, note := range created {
			_ = attachments.deleteForNote(note.ID)
			_ = notes.delete(note.ID)
		}
	}
	result := fieldNoteArchiveImportResult{ImportedNotes: len(created)}
	for i, note := range created {
		for _, attachment := range validated[i].Attachments {
			if _, err := attachments.create(note.ID, attachment.Filename, attachment.Data); err != nil {
				rollback()
				return fieldNoteArchiveImportResult{}, err
			}
			result.ImportedAttachments++
		}
	}
	return result, nil
}

func registerFieldNoteArchiveRoutes(mux *http.ServeMux, notes *fieldNoteStore, attachments *fieldNoteAttachmentStore) {
	mux.HandleFunc("GET /api/field-notes/archive", func(w http.ResponseWriter, _ *http.Request) {
		data, err := buildFieldNoteArchive(notes, attachments)
		if err != nil { writeError(w, 500, err); return }
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": "caching-tools-field-notes.zip"}))
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(200)
		_, _ = w.Write(data)
	})
	mux.HandleFunc("POST /api/field-notes/archive", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, fieldNoteArchiveMaxBytes)
		defer r.Body.Close()
		data, err := io.ReadAll(r.Body)
		if err != nil { writeError(w, 400, errors.New("invalid field note archive upload")); return }
		if len(data) == 0 { writeError(w, 400, errors.New("field note archive is empty")); return }
		result, err := restoreFieldNoteArchive(data, notes, attachments)
		if err != nil { writeError(w, 400, err); return }
		writeJSON(w, 201, result)
	})
}
