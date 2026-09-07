package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

const gpx11Namespace = "http://www.topografix.com/GPX/1/1"

type gpxDocument struct {
	XMLName   xml.Name      `xml:"gpx"`
	Version   string        `xml:"version,attr"`
	Creator   string        `xml:"creator,attr"`
	Waypoints []gpxWaypoint `xml:"wpt"`
	Routes    []gpxRoute    `xml:"rte"`
	Tracks    []gpxTrack    `xml:"trk"`
}

type gpxWaypoint struct {
	Latitude    float64 `xml:"lat,attr"`
	Longitude   float64 `xml:"lon,attr"`
	Name        string  `xml:"name,omitempty"`
	Comment     string  `xml:"cmt,omitempty"`
	Description string  `xml:"desc,omitempty"`
	Symbol      string  `xml:"sym,omitempty"`
	Type        string  `xml:"type,omitempty"`
}

type gpxRoute struct {
	Name   string     `xml:"name,omitempty"`
	Points []gpxPoint `xml:"rtept"`
}

type gpxTrack struct {
	Name     string            `xml:"name,omitempty"`
	Segments []gpxTrackSegment `xml:"trkseg"`
}

type gpxTrackSegment struct {
	Points []gpxPoint `xml:"trkpt"`
}

type gpxPoint struct {
	Latitude  float64 `xml:"lat,attr"`
	Longitude float64 `xml:"lon,attr"`
	Elevation float64 `xml:"ele,omitempty"`
	Time      string  `xml:"time,omitempty"`
}

type gpxImportResult struct {
	Imported  int                `json:"imported"`
	Waypoints []waypointResponse `json:"waypoints"`
}

type gpxPathSummary struct {
	Name       string  `json:"name"`
	Points     int     `json:"points"`
	Segments   int     `json:"segments,omitempty"`
	DistanceM  float64 `json:"distance_m"`
	DistanceKM float64 `json:"distance_km"`
}

type gpxInspection struct {
	Version   string           `json:"version"`
	Creator   string           `json:"creator,omitempty"`
	Waypoints int              `json:"waypoints"`
	Routes    []gpxPathSummary `json:"routes"`
	Tracks    []gpxPathSummary `json:"tracks"`
}

func decodeGPX(r io.Reader) (gpxDocument, error) {
	dec := xml.NewDecoder(io.LimitReader(r, 8<<20))
	var doc gpxDocument
	if err := dec.Decode(&doc); err != nil {
		return gpxDocument{}, fmt.Errorf("invalid GPX XML: %w", err)
	}
	if doc.XMLName.Local != "gpx" {
		return gpxDocument{}, errors.New("document root must be gpx")
	}
	if doc.XMLName.Space != "" && doc.XMLName.Space != gpx11Namespace {
		return gpxDocument{}, errors.New("unsupported GPX namespace")
	}
	if strings.TrimSpace(doc.Version) != "1.1" {
		return gpxDocument{}, errors.New("GPX version must be 1.1")
	}
	return doc, nil
}

func parseGPXWaypoints(r io.Reader) ([]waypointRequest, error) {
	doc, err := decodeGPX(r)
	if err != nil {
		return nil, err
	}
	if len(doc.Waypoints) == 0 {
		return nil, errors.New("GPX contains no waypoints")
	}

	out := make([]waypointRequest, 0, len(doc.Waypoints))
	for i, wpt := range doc.Waypoints {
		name := strings.TrimSpace(wpt.Name)
		if name == "" {
			name = fmt.Sprintf("GPX waypoint %d", i+1)
		}
		kind := strings.TrimSpace(wpt.Type)
		if kind == "" {
			kind = strings.TrimSpace(wpt.Symbol)
		}
		comment := strings.TrimSpace(wpt.Comment)
		if comment == "" {
			comment = strings.TrimSpace(wpt.Description)
		}
		req := waypointRequest{
			Name:      name,
			Latitude:  fmt.Sprintf("%.8f", wpt.Latitude),
			Longitude: fmt.Sprintf("%.8f", wpt.Longitude),
			Type:      kind,
			Comment:   comment,
		}
		if _, _, err := validateWaypointRequest(req); err != nil {
			return nil, fmt.Errorf("waypoint %d: %w", i+1, err)
		}
		out = append(out, req)
	}
	return out, nil
}

func inspectGPX(r io.Reader) (gpxInspection, error) {
	doc, err := decodeGPX(r)
	if err != nil {
		return gpxInspection{}, err
	}
	result := gpxInspection{
		Version:   doc.Version,
		Creator:   strings.TrimSpace(doc.Creator),
		Waypoints: len(doc.Waypoints),
		Routes:    make([]gpxPathSummary, 0, len(doc.Routes)),
		Tracks:    make([]gpxPathSummary, 0, len(doc.Tracks)),
	}
	for i, route := range doc.Routes {
		if err := validateGPXPoints(route.Points); err != nil {
			return gpxInspection{}, fmt.Errorf("route %d: %w", i+1, err)
		}
		name := strings.TrimSpace(route.Name)
		if name == "" {
			name = fmt.Sprintf("Route %d", i+1)
		}
		distance := pathDistance(route.Points)
		result.Routes = append(result.Routes, gpxPathSummary{Name: name, Points: len(route.Points), DistanceM: distance, DistanceKM: distance / 1000})
	}
	for i, track := range doc.Tracks {
		name := strings.TrimSpace(track.Name)
		if name == "" {
			name = fmt.Sprintf("Track %d", i+1)
		}
		totalPoints := 0
		totalDistance := 0.0
		for j, segment := range track.Segments {
			if err := validateGPXPoints(segment.Points); err != nil {
				return gpxInspection{}, fmt.Errorf("track %d segment %d: %w", i+1, j+1, err)
			}
			totalPoints += len(segment.Points)
			totalDistance += pathDistance(segment.Points)
		}
		result.Tracks = append(result.Tracks, gpxPathSummary{Name: name, Points: totalPoints, Segments: len(track.Segments), DistanceM: totalDistance, DistanceKM: totalDistance / 1000})
	}
	return result, nil
}

func validateGPXPoints(points []gpxPoint) error {
	for i, p := range points {
		if p.Latitude < -90 || p.Latitude > 90 || p.Longitude < -180 || p.Longitude > 180 {
			return fmt.Errorf("point %d has invalid coordinates", i+1)
		}
	}
	return nil
}

func pathDistance(points []gpxPoint) float64 {
	var total float64
	for i := 1; i < len(points); i++ {
		distance, _ := distanceAndBearing(points[i-1].Latitude, points[i-1].Longitude, points[i].Latitude, points[i].Longitude)
		total += distance
	}
	return total
}

func encodeGPXWaypoints(items []waypoint) ([]byte, error) {
	doc := gpxDocument{
		XMLName:   xml.Name{Space: gpx11Namespace, Local: "gpx"},
		Version:   "1.1",
		Creator:   "Caching Tools",
		Waypoints: make([]gpxWaypoint, 0, len(items)),
	}
	for _, item := range items {
		doc.Waypoints = append(doc.Waypoints, gpxWaypoint{
			Latitude:  item.Latitude,
			Longitude: item.Longitude,
			Name:      item.Name,
			Comment:   item.Comment,
			Type:      item.Type,
		})
	}
	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}
