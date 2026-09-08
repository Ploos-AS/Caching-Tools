package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	fieldNoteBundleFormat  = "caching-tools-field-notes"
	fieldNoteBundleVersion = 1
)

type fieldNote struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Body        string `json:"body,omitempty"`
	Status      string `json:"status,omitempty"`
	Type        string `json:"type,omitempty"`
	WaypointID  string `json:"waypoint_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	OccurredAt  string `json:"occurred_at"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type fieldNoteRequest struct {
	Title       string `json:"title"`
	Body        string `json:"body,omitempty"`
	Status      string `json:"status,omitempty"`
	Type        string `json:"type,omitempty"`
	WaypointID  string `json:"waypoint_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	OccurredAt  string `json:"occurred_at,omitempty"`
}

type fieldNoteBundle struct {
	Format     string      `json:"format"`
	Version    int         `json:"version"`
	ExportedAt string      `json:"exported_at,omitempty"`
	Notes      []fieldNote `json:"notes"`
}

type fieldNoteImportResult struct {
	Imported int         `json:"imported"`
	Notes    []fieldNote `json:"notes"`
}

type fieldNoteStore struct {
	mu   sync.Mutex
	path string
}

func newFieldNoteStore(dataDir string) *fieldNoteStore {
	return &fieldNoteStore{path: filepath.Join(dataDir, "field-notes.json")}
}

func normalizeFieldNoteRequest(req fieldNoteRequest) (fieldNoteRequest, error) {
	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)
	req.Status = strings.TrimSpace(req.Status)
	req.Type = strings.TrimSpace(req.Type)
	req.WaypointID = strings.TrimSpace(req.WaypointID)
	req.WorkspaceID = strings.TrimSpace(req.WorkspaceID)
	req.OccurredAt = strings.TrimSpace(req.OccurredAt)
	if req.Title == "" {
		return req, errors.New("title is required")
	}
	if req.OccurredAt == "" {
		req.OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
	} else if parsed, err := time.Parse(time.RFC3339Nano, req.OccurredAt); err != nil {
		return req, errors.New("occurred_at must be RFC3339")
	} else {
		req.OccurredAt = parsed.UTC().Format(time.RFC3339Nano)
	}
	return req, nil
}

func (s *fieldNoteStore) list() ([]fieldNote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].OccurredAt > items[j].OccurredAt })
	return items, nil
}

func (s *fieldNoteStore) create(req fieldNoteRequest) (fieldNote, error) {
	req, err := normalizeFieldNoteRequest(req)
	if err != nil {
		return fieldNote{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return fieldNote{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	item := fieldNote{
		ID: fmt.Sprintf("note-%x", time.Now().UnixNano()), Title: req.Title, Body: req.Body,
		Status: req.Status, Type: req.Type, WaypointID: req.WaypointID, WorkspaceID: req.WorkspaceID,
		OccurredAt: req.OccurredAt, CreatedAt: now, UpdatedAt: now,
	}
	items = append(items, item)
	if err := s.saveLocked(items); err != nil {
		return fieldNote{}, err
	}
	return item, nil
}

func (s *fieldNoteStore) importMany(requests []fieldNoteRequest) ([]fieldNote, error) {
	validated := make([]fieldNoteRequest, len(requests))
	for i, req := range requests {
		normalized, err := normalizeFieldNoteRequest(req)
		if err != nil {
			return nil, fmt.Errorf("note %d: %w", i+1, err)
		}
		validated[i] = normalized
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	base := time.Now().UnixNano()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	created := make([]fieldNote, 0, len(validated))
	for i, req := range validated {
		item := fieldNote{
			ID: fmt.Sprintf("note-%x", base+int64(i)), Title: req.Title, Body: req.Body,
			Status: req.Status, Type: req.Type, WaypointID: req.WaypointID, WorkspaceID: req.WorkspaceID,
			OccurredAt: req.OccurredAt, CreatedAt: now, UpdatedAt: now,
		}
		created = append(created, item)
	}
	items = append(items, created...)
	if err := s.saveLocked(items); err != nil {
		return nil, err
	}
	return created, nil
}

func importFieldNoteBundle(bundle fieldNoteBundle, store *fieldNoteStore) ([]fieldNote, error) {
	if bundle.Format != fieldNoteBundleFormat {
		return nil, errors.New("unsupported field note bundle format")
	}
	if bundle.Version != fieldNoteBundleVersion {
		return nil, fmt.Errorf("unsupported field note bundle version %d", bundle.Version)
	}
	requests := make([]fieldNoteRequest, 0, len(bundle.Notes))
	for _, note := range bundle.Notes {
		requests = append(requests, fieldNoteRequest{
			Title: note.Title, Body: note.Body, Status: note.Status, Type: note.Type,
			WaypointID: note.WaypointID, WorkspaceID: note.WorkspaceID, OccurredAt: note.OccurredAt,
		})
	}
	return store.importMany(requests)
}

func (s *fieldNoteStore) get(id string) (fieldNote, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return fieldNote{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return fieldNote{}, os.ErrNotExist
}

func (s *fieldNoteStore) update(id string, req fieldNoteRequest) (fieldNote, error) {
	req, err := normalizeFieldNoteRequest(req)
	if err != nil {
		return fieldNote{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return fieldNote{}, err
	}
	for i := range items {
		if items[i].ID != id {
			continue
		}
		items[i].Title = req.Title
		items[i].Body = req.Body
		items[i].Status = req.Status
		items[i].Type = req.Type
		items[i].WaypointID = req.WaypointID
		items[i].WorkspaceID = req.WorkspaceID
		items[i].OccurredAt = req.OccurredAt
		items[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := s.saveLocked(items); err != nil {
			return fieldNote{}, err
		}
		return items[i], nil
	}
	return fieldNote{}, os.ErrNotExist
}

func (s *fieldNoteStore) delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID != id {
			continue
		}
		items = append(items[:i], items[i+1:]...)
		return s.saveLocked(items)
	}
	return os.ErrNotExist
}

func (s *fieldNoteStore) loadLocked() ([]fieldNote, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []fieldNote{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []fieldNote
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("read field note store: %w", err)
	}
	return items, nil
}

func (s *fieldNoteStore) saveLocked(items []fieldNote) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
