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

type storedPath struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Route     []gpxPoint        `json:"route,omitempty"`
	Segments  []gpxTrackSegment `json:"segments,omitempty"`
	CreatedAt string            `json:"created_at"`
}

type pathResponse struct {
	ID        string         `json:"id"`
	Kind      string         `json:"kind"`
	CreatedAt string         `json:"created_at"`
	Summary   gpxPathSummary `json:"summary"`
}

type pathImportResult struct {
	Imported int            `json:"imported"`
	Paths    []pathResponse `json:"paths"`
}

type pathStore struct {
	mu   sync.Mutex
	path string
}

func newPathStore(dataDir string) *pathStore {
	return &pathStore{path: filepath.Join(dataDir, "paths.json")}
}

func (s *pathStore) list() ([]storedPath, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Name == items[j].Name {
			return items[i].CreatedAt < items[j].CreatedAt
		}
		return items[i].Name < items[j].Name
	})
	return items, nil
}

func (s *pathStore) get(id string) (storedPath, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return storedPath{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return storedPath{}, os.ErrNotExist
}

func (s *pathStore) delete(id string) error {
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

func (s *pathStore) importGPX(doc gpxDocument) ([]storedPath, error) {
	pending := make([]storedPath, 0, len(doc.Routes)+len(doc.Tracks))
	now := time.Now().UTC()

	for i, route := range doc.Routes {
		if err := validateGPXPoints(route.Points); err != nil {
			return nil, fmt.Errorf("route %d: %w", i+1, err)
		}
		name := strings.TrimSpace(route.Name)
		if name == "" {
			name = fmt.Sprintf("Route %d", i+1)
		}
		pending = append(pending, storedPath{
			ID:        fmt.Sprintf("path-%x-r%d", now.UnixNano(), i),
			Kind:      "route",
			Name:      name,
			Route:     route.Points,
			CreatedAt: now.Format(time.RFC3339Nano),
		})
	}

	for i, track := range doc.Tracks {
		for j, segment := range track.Segments {
			if err := validateGPXPoints(segment.Points); err != nil {
				return nil, fmt.Errorf("track %d segment %d: %w", i+1, j+1, err)
			}
		}
		name := strings.TrimSpace(track.Name)
		if name == "" {
			name = fmt.Sprintf("Track %d", i+1)
		}
		pending = append(pending, storedPath{
			ID:        fmt.Sprintf("path-%x-t%d", now.UnixNano(), i),
			Kind:      "track",
			Name:      name,
			Segments:  track.Segments,
			CreatedAt: now.Format(time.RFC3339Nano),
		})
	}

	if len(pending) == 0 {
		return nil, errors.New("GPX contains no routes or tracks")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items = append(items, pending...)
	if err := s.saveLocked(items); err != nil {
		return nil, err
	}
	return pending, nil
}

func pathView(item storedPath) (pathResponse, error) {
	var summary gpxPathSummary
	switch item.Kind {
	case "route":
		distance := pathDistance(item.Route)
		summary = gpxPathSummary{Name: item.Name, Points: len(item.Route), DistanceM: distance, DistanceKM: distance / 1000}
	case "track":
		var err error
		summary, err = summarizeTrack(item.Segments)
		if err != nil {
			return pathResponse{}, err
		}
		summary.Name = item.Name
	default:
		return pathResponse{}, fmt.Errorf("unsupported stored path kind %q", item.Kind)
	}
	return pathResponse{ID: item.ID, Kind: item.Kind, CreatedAt: item.CreatedAt, Summary: summary}, nil
}

func (s *pathStore) loadLocked() ([]storedPath, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []storedPath{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []storedPath
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("read path store: %w", err)
	}
	return items, nil
}

func (s *pathStore) saveLocked(items []storedPath) error {
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
