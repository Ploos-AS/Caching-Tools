package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type pathEditRequest struct {
	Operation string  `json:"operation"`
	Segment   int     `json:"segment,omitempty"`
	Point     int     `json:"point,omitempty"`
	Target    int     `json:"target,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
}

func (s *pathStore) edit(id string, req pathEditRequest) (storedPath, error) {
	op := strings.ToLower(strings.TrimSpace(req.Operation))
	if op == "" {
		return storedPath{}, errors.New("operation is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	items, err := s.loadLocked()
	if err != nil {
		return storedPath{}, err
	}
	for i := range items {
		if items[i].ID != id {
			continue
		}
		item := items[i]
		if err := editStoredPath(&item, op, req); err != nil {
			return storedPath{}, err
		}
		if err := validateStoredPathGeometry(item); err != nil {
			return storedPath{}, err
		}
		items[i] = item
		if err := s.saveLocked(items); err != nil {
			return storedPath{}, err
		}
		return item, nil
	}
	return storedPath{}, os.ErrNotExist
}

func editStoredPath(item *storedPath, op string, req pathEditRequest) error {
	switch item.Kind {
	case "route":
		return editRoute(item, op, req)
	case "track":
		return editTrack(item, op, req)
	default:
		return fmt.Errorf("unsupported stored path kind %q", item.Kind)
	}
}

func validatePointCoordinates(lat, lon float64) error {
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return errors.New("point has invalid coordinates")
	}
	return nil
}

func editRoute(item *storedPath, op string, req pathEditRequest) error {
	switch op {
	case "delete-point":
		if len(item.Route) <= 1 {
			return errors.New("route must keep at least one point")
		}
		if req.Point < 0 || req.Point >= len(item.Route) {
			return errors.New("point index out of range")
		}
		item.Route = append(item.Route[:req.Point], item.Route[req.Point+1:]...)
		return nil
	case "move-point":
		var err error
		item.Route, err = moveGPXPoint(item.Route, req.Point, req.Target)
		return err
	case "set-point":
		if req.Point < 0 || req.Point >= len(item.Route) {
			return errors.New("point index out of range")
		}
		if err := validatePointCoordinates(req.Latitude, req.Longitude); err != nil {
			return err
		}
		item.Route[req.Point].Latitude = req.Latitude
		item.Route[req.Point].Longitude = req.Longitude
		return nil
	default:
		return fmt.Errorf("operation %q is not valid for routes", op)
	}
}

func editTrack(item *storedPath, op string, req pathEditRequest) error {
	if len(item.Segments) == 0 {
		return errors.New("track has no segments")
	}
	if req.Segment < 0 || req.Segment >= len(item.Segments) {
		return errors.New("segment index out of range")
	}
	switch op {
	case "delete-point":
		points := item.Segments[req.Segment].Points
		if len(points) <= 1 {
			return errors.New("track segment must keep at least one point")
		}
		if req.Point < 0 || req.Point >= len(points) {
			return errors.New("point index out of range")
		}
		item.Segments[req.Segment].Points = append(points[:req.Point], points[req.Point+1:]...)
		return nil
	case "move-point":
		points, err := moveGPXPoint(item.Segments[req.Segment].Points, req.Point, req.Target)
		if err != nil {
			return err
		}
		item.Segments[req.Segment].Points = points
		return nil
	case "set-point":
		points := item.Segments[req.Segment].Points
		if req.Point < 0 || req.Point >= len(points) {
			return errors.New("point index out of range")
		}
		if err := validatePointCoordinates(req.Latitude, req.Longitude); err != nil {
			return err
		}
		item.Segments[req.Segment].Points[req.Point].Latitude = req.Latitude
		item.Segments[req.Segment].Points[req.Point].Longitude = req.Longitude
		return nil
	case "split-segment":
		points := item.Segments[req.Segment].Points
		if req.Point <= 0 || req.Point >= len(points) {
			return errors.New("split point must be between existing points")
		}
		left := gpxTrackSegment{Points: append([]gpxPoint(nil), points[:req.Point]...)}
		right := gpxTrackSegment{Points: append([]gpxPoint(nil), points[req.Point:]...)}
		next := make([]gpxTrackSegment, 0, len(item.Segments)+1)
		next = append(next, item.Segments[:req.Segment]...)
		next = append(next, left, right)
		next = append(next, item.Segments[req.Segment+1:]...)
		item.Segments = next
		return nil
	case "merge-segments":
		if req.Segment+1 >= len(item.Segments) {
			return errors.New("segment has no following segment to merge")
		}
		merged := gpxTrackSegment{Points: append(append([]gpxPoint(nil), item.Segments[req.Segment].Points...), item.Segments[req.Segment+1].Points...)}
		next := make([]gpxTrackSegment, 0, len(item.Segments)-1)
		next = append(next, item.Segments[:req.Segment]...)
		next = append(next, merged)
		next = append(next, item.Segments[req.Segment+2:]...)
		item.Segments = next
		return nil
	default:
		return fmt.Errorf("unknown path edit operation %q", op)
	}
}

func moveGPXPoint(points []gpxPoint, from, to int) ([]gpxPoint, error) {
	if from < 0 || from >= len(points) || to < 0 || to >= len(points) {
		return nil, errors.New("point index out of range")
	}
	if from == to {
		return points, nil
	}
	out := append([]gpxPoint(nil), points...)
	point := out[from]
	out = append(out[:from], out[from+1:]...)
	out = append(out, gpxPoint{})
	copy(out[to+1:], out[to:])
	out[to] = point
	return out, nil
}

func validateStoredPathGeometry(item storedPath) error {
	switch item.Kind {
	case "route":
		if len(item.Route) == 0 {
			return errors.New("route must contain at least one point")
		}
		return validateGPXPoints(item.Route)
	case "track":
		if len(item.Segments) == 0 {
			return errors.New("track must contain at least one segment")
		}
		for i, segment := range item.Segments {
			if len(segment.Points) == 0 {
				return fmt.Errorf("segment %d must contain at least one point", i+1)
			}
			if err := validateGPXPoints(segment.Points); err != nil {
				return fmt.Errorf("segment %d: %w", i+1, err)
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported stored path kind %q", item.Kind)
	}
}
