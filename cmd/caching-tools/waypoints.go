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

type waypoint struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Type      string  `json:"type,omitempty"`
	Comment   string  `json:"comment,omitempty"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

type waypointRequest struct {
	Name      string `json:"name"`
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
	Type      string `json:"type,omitempty"`
	Comment   string `json:"comment,omitempty"`
}

type waypointResponse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Point     pointResponse     `json:"point"`
	Type      string            `json:"type,omitempty"`
	Comment   string            `json:"comment,omitempty"`
	CreatedAt string            `json:"created_at"`
	UpdatedAt string            `json:"updated_at"`
}

type waypointStore struct {
	mu   sync.Mutex
	path string
}

func newWaypointStore(dataDir string) *waypointStore {
	return &waypointStore{path: filepath.Join(dataDir, "waypoints.json")}
}

func (s *waypointStore) list() ([]waypoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (s *waypointStore) create(req waypointRequest) (waypoint, error) {
	lat, lon, err := validateWaypointRequest(req)
	if err != nil {
		return waypoint{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return waypoint{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	item := waypoint{
		ID:        fmt.Sprintf("wp-%x", time.Now().UnixNano()),
		Name:      strings.TrimSpace(req.Name),
		Latitude:  lat,
		Longitude: lon,
		Type:      strings.TrimSpace(req.Type),
		Comment:   strings.TrimSpace(req.Comment),
		CreatedAt: now,
		UpdatedAt: now,
	}
	items = append(items, item)
	if err := s.saveLocked(items); err != nil {
		return waypoint{}, err
	}
	return item, nil
}

func (s *waypointStore) importMany(requests []waypointRequest) ([]waypoint, error) {
	validated := make([]waypoint, 0, len(requests))
	base := time.Now().UnixNano()
	for i, req := range requests {
		lat, lon, err := validateWaypointRequest(req)
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC().Format(time.RFC3339Nano)
		validated = append(validated, waypoint{
			ID:        fmt.Sprintf("wp-%x", base+int64(i)),
			Name:      strings.TrimSpace(req.Name),
			Latitude:  lat,
			Longitude: lon,
			Type:      strings.TrimSpace(req.Type),
			Comment:   strings.TrimSpace(req.Comment),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return nil, err
	}
	items = append(items, validated...)
	if err := s.saveLocked(items); err != nil {
		return nil, err
	}
	return validated, nil
}

func (s *waypointStore) update(id string, req waypointRequest) (waypoint, error) {
	lat, lon, err := validateWaypointRequest(req)
	if err != nil {
		return waypoint{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return waypoint{}, err
	}
	for i := range items {
		if items[i].ID != id {
			continue
		}
		items[i].Name = strings.TrimSpace(req.Name)
		items[i].Latitude = lat
		items[i].Longitude = lon
		items[i].Type = strings.TrimSpace(req.Type)
		items[i].Comment = strings.TrimSpace(req.Comment)
		items[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
		if err := s.saveLocked(items); err != nil {
			return waypoint{}, err
		}
		return items[i], nil
	}
	return waypoint{}, os.ErrNotExist
}

func (s *waypointStore) delete(id string) error {
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

func validateWaypointRequest(req waypointRequest) (float64, float64, error) {
	if strings.TrimSpace(req.Name) == "" {
		return 0, 0, errors.New("name is required")
	}
	lat, lon, err := parsePoint(coordinateRequest{Latitude: req.Latitude, Longitude: req.Longitude})
	if err != nil {
		return 0, 0, err
	}
	return lat, lon, nil
}

func (s *waypointStore) loadLocked() ([]waypoint, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return []waypoint{}, nil
	}
	if err != nil {
		return nil, err
	}
	var items []waypoint
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("read waypoint store: %w", err)
	}
	return items, nil
}

func (s *waypointStore) saveLocked(items []waypoint) error {
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

func waypointView(item waypoint) waypointResponse {
	return waypointResponse{
		ID: item.ID, Name: item.Name, Point: point(item.Latitude, item.Longitude),
		Type: item.Type, Comment: item.Comment, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt,
	}
}
