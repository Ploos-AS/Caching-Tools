package main

import (
	"errors"
	"math"
	"os"
)

type fieldNavigationRequest struct {
	From       coordinateRequest `json:"from"`
	WaypointID string            `json:"waypoint_id,omitempty"`
	PathID     string            `json:"path_id,omitempty"`
}

type fieldPathPosition struct {
	Segment int `json:"segment"`
	Edge    int `json:"edge"`
}

type fieldPathProgress struct {
	AlongM             float64       `json:"along_m"`
	RemainingM         float64       `json:"remaining_m"`
	TotalM             float64       `json:"total_m"`
	ProgressPercent    float64       `json:"progress_percent"`
	NextPoint          pointResponse `json:"next_point"`
	NextPointDistanceM float64       `json:"next_point_distance_m"`
	ForwardBearingDeg  float64       `json:"forward_bearing_deg"`
}

type fieldNavigationResponse struct {
	Kind              string             `json:"kind"`
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	From              pointResponse      `json:"from"`
	Target            pointResponse      `json:"target"`
	DistanceM         float64            `json:"distance_m"`
	DistanceKM        float64            `json:"distance_km"`
	BearingDeg        float64            `json:"bearing_deg"`
	CrossTrackM       *float64           `json:"cross_track_m,omitempty"`
	NearestPath       *fieldPathPosition `json:"nearest_path,omitempty"`
	Progress          *fieldPathProgress `json:"progress,omitempty"`
}

func fieldNavigate(req fieldNavigationRequest, waypoints *waypointStore, paths *pathStore) (fieldNavigationResponse, error) {
	lat, lon, err := parsePoint(req.From)
	if err != nil {
		return fieldNavigationResponse{}, err
	}
	if (req.WaypointID == "") == (req.PathID == "") {
		return fieldNavigationResponse{}, errors.New("provide exactly one of waypoint_id or path_id")
	}
	if req.WaypointID != "" {
		items, err := waypoints.list()
		if err != nil {
			return fieldNavigationResponse{}, err
		}
		for _, item := range items {
			if item.ID != req.WaypointID {
				continue
			}
			distance, bearing := distanceAndBearing(lat, lon, item.Latitude, item.Longitude)
			return fieldNavigationResponse{
				Kind: "waypoint", ID: item.ID, Name: item.Name,
				From: point(lat, lon), Target: point(item.Latitude, item.Longitude),
				DistanceM: distance, DistanceKM: distance / 1000, BearingDeg: bearing,
			}, nil
		}
		return fieldNavigationResponse{}, os.ErrNotExist
	}

	item, err := paths.get(req.PathID)
	if err != nil {
		return fieldNavigationResponse{}, err
	}
	nearestLat, nearestLon, position, fraction, distance, err := nearestPointOnStoredPathDetailed(lat, lon, item)
	if err != nil {
		return fieldNavigationResponse{}, err
	}
	_, bearing := distanceAndBearing(lat, lon, nearestLat, nearestLon)
	progress, err := pathProgress(item, position, fraction, nearestLat, nearestLon)
	if err != nil {
		return fieldNavigationResponse{}, err
	}
	cross := distance
	return fieldNavigationResponse{
		Kind: item.Kind, ID: item.ID, Name: item.Name,
		From: point(lat, lon), Target: point(nearestLat, nearestLon),
		DistanceM: distance, DistanceKM: distance / 1000, BearingDeg: bearing,
		CrossTrackM: &cross, NearestPath: &position, Progress: &progress,
	}, nil
}

func nearestPointOnStoredPath(lat, lon float64, item storedPath) (float64, float64, fieldPathPosition, float64, error) {
	nearestLat, nearestLon, pos, _, distance, err := nearestPointOnStoredPathDetailed(lat, lon, item)
	return nearestLat, nearestLon, pos, distance, err
}

func nearestPointOnStoredPathDetailed(lat, lon float64, item storedPath) (float64, float64, fieldPathPosition, float64, float64, error) {
	collections := pathCollections(item)
	if collections == nil {
		return 0, 0, fieldPathPosition{}, 0, 0, errors.New("unsupported path kind")
	}

	bestDistance := math.Inf(1)
	bestLat, bestLon, bestFraction := 0.0, 0.0, 0.0
	bestPos := fieldPathPosition{}
	found := false
	for segmentIndex, points := range collections {
		if len(points) == 0 {
			continue
		}
		if len(points) == 1 {
			distance, _ := distanceAndBearing(lat, lon, points[0].Latitude, points[0].Longitude)
			if distance < bestDistance {
				bestDistance, bestLat, bestLon, bestFraction = distance, points[0].Latitude, points[0].Longitude, 0
				bestPos = fieldPathPosition{Segment: segmentIndex, Edge: 0}
				found = true
			}
			continue
		}
		for edge := 0; edge < len(points)-1; edge++ {
			candidateLat, candidateLon, fraction := nearestPointOnSegmentLocalDetailed(lat, lon, points[edge], points[edge+1])
			distance, _ := distanceAndBearing(lat, lon, candidateLat, candidateLon)
			if distance < bestDistance {
				bestDistance, bestLat, bestLon, bestFraction = distance, candidateLat, candidateLon, fraction
				bestPos = fieldPathPosition{Segment: segmentIndex, Edge: edge}
				found = true
			}
		}
	}
	if !found {
		return 0, 0, fieldPathPosition{}, 0, 0, errors.New("path contains no points")
	}
	return bestLat, bestLon, bestPos, bestFraction, bestDistance, nil
}

func pathCollections(item storedPath) [][]gpxPoint {
	if item.Kind == "route" {
		return [][]gpxPoint{item.Route}
	}
	if item.Kind == "track" {
		collections := make([][]gpxPoint, 0, len(item.Segments))
		for _, segment := range item.Segments {
			collections = append(collections, segment.Points)
		}
		return collections
	}
	return nil
}

func pathProgress(item storedPath, position fieldPathPosition, fraction, nearestLat, nearestLon float64) (fieldPathProgress, error) {
	collections := pathCollections(item)
	if collections == nil || position.Segment < 0 || position.Segment >= len(collections) {
		return fieldPathProgress{}, errors.New("invalid path position")
	}
	points := collections[position.Segment]
	if len(points) == 0 {
		return fieldPathProgress{}, errors.New("path contains no points")
	}

	total := 0.0
	along := 0.0
	for segmentIndex, segment := range collections {
		for edge := 0; edge+1 < len(segment); edge++ {
			edgeDistance, _ := distanceAndBearing(segment[edge].Latitude, segment[edge].Longitude, segment[edge+1].Latitude, segment[edge+1].Longitude)
			total += edgeDistance
			if segmentIndex < position.Segment || (segmentIndex == position.Segment && edge < position.Edge) {
				along += edgeDistance
			} else if segmentIndex == position.Segment && edge == position.Edge {
				along += edgeDistance * math.Max(0, math.Min(1, fraction))
			}
		}
	}

	nextIndex := position.Edge + 1
	if len(points) == 1 {
		nextIndex = 0
	}
	if nextIndex >= len(points) {
		nextIndex = len(points) - 1
	}
	next := points[nextIndex]
	nextDistance, _ := distanceAndBearing(nearestLat, nearestLon, next.Latitude, next.Longitude)
	_, forwardBearing := distanceAndBearing(nearestLat, nearestLon, next.Latitude, next.Longitude)
	remaining := math.Max(0, total-along)
	percent := 100.0
	if total > 0 {
		percent = math.Max(0, math.Min(100, along/total*100))
	}
	return fieldPathProgress{
		AlongM: along, RemainingM: remaining, TotalM: total, ProgressPercent: percent,
		NextPoint: point(next.Latitude, next.Longitude), NextPointDistanceM: nextDistance, ForwardBearingDeg: forwardBearing,
	}, nil
}

func nearestPointOnSegmentLocal(lat, lon float64, a, b gpxPoint) (float64, float64) {
	nearestLat, nearestLon, _ := nearestPointOnSegmentLocalDetailed(lat, lon, a, b)
	return nearestLat, nearestLon
}

func nearestPointOnSegmentLocalDetailed(lat, lon float64, a, b gpxPoint) (float64, float64, float64) {
	refLat := radians(lat)
	cosLat := math.Cos(refLat)
	if math.Abs(cosLat) < 1e-12 {
		cosLat = 1e-12
	}
	ax := radians(a.Longitude-lon) * cosLat * earthRadiusMeters
	ay := radians(a.Latitude-lat) * earthRadiusMeters
	bx := radians(b.Longitude-lon) * cosLat * earthRadiusMeters
	by := radians(b.Latitude-lat) * earthRadiusMeters
	dx, dy := bx-ax, by-ay
	denom := dx*dx + dy*dy
	fraction := 0.0
	if denom > 0 {
		fraction = -(ax*dx + ay*dy) / denom
	}
	if fraction < 0 { fraction = 0 }
	if fraction > 1 { fraction = 1 }
	x := ax + fraction*dx
	y := ay + fraction*dy
	return lat + degrees(y/earthRadiusMeters), lon + degrees(x/(earthRadiusMeters*cosLat)), fraction
}
