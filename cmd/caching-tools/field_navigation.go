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
	nearestLat, nearestLon, position, distance, err := nearestPointOnStoredPath(lat, lon, item)
	if err != nil {
		return fieldNavigationResponse{}, err
	}
	_, bearing := distanceAndBearing(lat, lon, nearestLat, nearestLon)
	cross := distance
	return fieldNavigationResponse{
		Kind: item.Kind, ID: item.ID, Name: item.Name,
		From: point(lat, lon), Target: point(nearestLat, nearestLon),
		DistanceM: distance, DistanceKM: distance / 1000, BearingDeg: bearing,
		CrossTrackM: &cross, NearestPath: &position,
	}, nil
}

func nearestPointOnStoredPath(lat, lon float64, item storedPath) (float64, float64, fieldPathPosition, float64, error) {
	collections := make([][]gpxPoint, 0)
	if item.Kind == "route" {
		collections = append(collections, item.Route)
	} else if item.Kind == "track" {
		for _, segment := range item.Segments {
			collections = append(collections, segment.Points)
		}
	} else {
		return 0, 0, fieldPathPosition{}, 0, errors.New("unsupported path kind")
	}

	bestDistance := math.Inf(1)
	bestLat, bestLon := 0.0, 0.0
	bestPos := fieldPathPosition{}
	found := false
	for segmentIndex, points := range collections {
		if len(points) == 0 {
			continue
		}
		if len(points) == 1 {
			distance, _ := distanceAndBearing(lat, lon, points[0].Latitude, points[0].Longitude)
			if distance < bestDistance {
				bestDistance, bestLat, bestLon = distance, points[0].Latitude, points[0].Longitude
				bestPos = fieldPathPosition{Segment: segmentIndex, Edge: 0}
				found = true
			}
			continue
		}
		for edge := 0; edge < len(points)-1; edge++ {
			candidateLat, candidateLon := nearestPointOnSegmentLocal(lat, lon, points[edge], points[edge+1])
			distance, _ := distanceAndBearing(lat, lon, candidateLat, candidateLon)
			if distance < bestDistance {
				bestDistance, bestLat, bestLon = distance, candidateLat, candidateLon
				bestPos = fieldPathPosition{Segment: segmentIndex, Edge: edge}
				found = true
			}
		}
	}
	if !found {
		return 0, 0, fieldPathPosition{}, 0, errors.New("path contains no points")
	}
	return bestLat, bestLon, bestPos, bestDistance, nil
}

func nearestPointOnSegmentLocal(lat, lon float64, a, b gpxPoint) (float64, float64) {
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
	t := 0.0
	if denom > 0 {
		t = -(ax*dx + ay*dy) / denom
	}
	if t < 0 { t = 0 }
	if t > 1 { t = 1 }
	x := ax + t*dx
	y := ay + t*dy
	return lat + degrees(y/earthRadiusMeters), lon + degrees(x/(earthRadiusMeters*cosLat))
}
