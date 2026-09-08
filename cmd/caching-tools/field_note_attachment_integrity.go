package main

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type attachmentIntegrityItem struct {
	NoteID         string `json:"note_id"`
	AttachmentID   string `json:"attachment_id"`
	Filename       string `json:"filename"`
	Status         string `json:"status"`
	ExpectedSHA256 string `json:"expected_sha256,omitempty"`
	ActualSHA256   string `json:"actual_sha256,omitempty"`
	Size           int64  `json:"size,omitempty"`
}

type attachmentIntegrityReport struct {
	GeneratedAt string                    `json:"generated_at"`
	Total       int                       `json:"total"`
	OK          int                       `json:"ok"`
	Mismatch    int                       `json:"mismatch"`
	Missing     int                       `json:"missing"`
	Unrecorded  int                       `json:"unrecorded"`
	Items       []attachmentIntegrityItem `json:"items"`
}

func (s *fieldNoteAttachmentStore) verifyNote(noteID string) ([]attachmentIntegrityItem, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dir := s.noteDir(noteID)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return []attachmentIntegrityItem{}, nil
	}
	if err != nil {
		return nil, err
	}

	items := []attachmentIntegrityItem{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		meta, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var attachment fieldNoteAttachment
		if err := json.Unmarshal(meta, &attachment); err != nil {
			return nil, fmt.Errorf("read attachment metadata %s: %w", entry.Name(), err)
		}
		if attachment.NoteID != noteID || !safeAttachmentID(attachment.ID) {
			continue
		}
		result := attachmentIntegrityItem{
			NoteID: noteID, AttachmentID: attachment.ID, Filename: attachment.Filename,
			ExpectedSHA256: attachment.SHA256,
		}
		data, err := os.ReadFile(filepath.Join(dir, attachment.ID+".bin"))
		if errors.Is(err, os.ErrNotExist) {
			result.Status = "missing"
			items = append(items, result)
			continue
		}
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		result.ActualSHA256 = fmt.Sprintf("%x", sum[:])
		result.Size = int64(len(data))
		switch {
		case attachment.SHA256 == "":
			result.Status = "unrecorded"
		case attachment.SHA256 == result.ActualSHA256:
			result.Status = "ok"
		default:
			result.Status = "mismatch"
		}
		items = append(items, result)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].NoteID != items[j].NoteID { return items[i].NoteID < items[j].NoteID }
		return items[i].AttachmentID < items[j].AttachmentID
	})
	return items, nil
}

func buildAttachmentIntegrityReport(items []attachmentIntegrityItem) attachmentIntegrityReport {
	report := attachmentIntegrityReport{GeneratedAt: time.Now().UTC().Format(time.RFC3339Nano), Total: len(items), Items: items}
	for _, item := range items {
		switch item.Status {
		case "ok": report.OK++
		case "mismatch": report.Mismatch++
		case "missing": report.Missing++
		case "unrecorded": report.Unrecorded++
		}
	}
	return report
}

func verifyAllAttachments(notes *fieldNoteStore, attachments *fieldNoteAttachmentStore) (attachmentIntegrityReport, error) {
	noteItems, err := notes.list()
	if err != nil { return attachmentIntegrityReport{}, err }
	all := []attachmentIntegrityItem{}
	for _, note := range noteItems {
		items, err := attachments.verifyNote(note.ID)
		if err != nil { return attachmentIntegrityReport{}, err }
		all = append(all, items...)
	}
	return buildAttachmentIntegrityReport(all), nil
}

func registerFieldNoteAttachmentIntegrityRoutes(mux *http.ServeMux, notes *fieldNoteStore, attachments *fieldNoteAttachmentStore) {
	mux.HandleFunc("GET /api/field-notes/{id}/attachments/integrity", func(w http.ResponseWriter, r *http.Request) {
		noteID := r.PathValue("id")
		if err := fieldNoteExists(notes, noteID); errors.Is(err, os.ErrNotExist) { writeError(w, 404, errors.New("field note not found")); return } else if err != nil { writeError(w, 500, err); return }
		items, err := attachments.verifyNote(noteID)
		if err != nil { writeError(w, 500, err); return }
		writeJSON(w, 200, buildAttachmentIntegrityReport(items))
	})
	mux.HandleFunc("GET /api/field-note-attachments/integrity", func(w http.ResponseWriter, r *http.Request) {
		report, err := verifyAllAttachments(notes, attachments)
		if err != nil { writeError(w, 500, err); return }
		writeJSON(w, 200, report)
	})
}
