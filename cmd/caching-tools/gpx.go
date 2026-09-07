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
	XMLName xml.Name      `xml:"gpx"`
	Version string        `xml:"version,attr"`
	Creator string        `xml:"creator,attr"`
	Waypoints []gpxWaypoint `xml:"wpt"`
}

type gpxWaypoint struct {
	Latitude  float64 `xml:"lat,attr"`
	Longitude float64 `xml:"lon,attr"`
	Name      string  `xml:"name,omitempty"`
	Comment   string  `xml:"cmt,omitempty"`
	Description string `xml:"desc,omitempty"`
	Symbol    string  `xml:"sym,omitempty"`
	Type      string  `xml:"type,omitempty"`
}

type gpxImportResult struct {
	Imported int                `json:"imported"`
	Waypoints []waypointResponse `json:"waypoints"`
}

func parseGPXWaypoints(r io.Reader) ([]waypointRequest, error) {
	dec := xml.NewDecoder(io.LimitReader(r, 8<<20))
	var doc gpxDocument
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("invalid GPX XML: %w", err)
	}
	if doc.XMLName.Local != "gpx" {
		return nil, errors.New("document root must be gpx")
	}
	if doc.XMLName.Space != "" && doc.XMLName.Space != gpx11Namespace {
		return nil, errors.New("unsupported GPX namespace")
	}
	if strings.TrimSpace(doc.Version) != "1.1" {
		return nil, errors.New("GPX version must be 1.1")
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
			Name: name,
			Latitude: fmt.Sprintf("%.8f", wpt.Latitude),
			Longitude: fmt.Sprintf("%.8f", wpt.Longitude),
			Type: kind,
			Comment: comment,
		}
		if _, _, err := validateWaypointRequest(req); err != nil {
			return nil, fmt.Errorf("waypoint %d: %w", i+1, err)
		}
		out = append(out, req)
	}
	return out, nil
}

func encodeGPXWaypoints(items []waypoint) ([]byte, error) {
	doc := gpxDocument{
		XMLName: xml.Name{Space: gpx11Namespace, Local: "gpx"},
		Version: "1.1",
		Creator: "Caching Tools",
		Waypoints: make([]gpxWaypoint, 0, len(items)),
	}
	for _, item := range items {
		doc.Waypoints = append(doc.Waypoints, gpxWaypoint{
			Latitude: item.Latitude,
			Longitude: item.Longitude,
			Name: item.Name,
			Comment: item.Comment,
			Type: item.Type,
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
