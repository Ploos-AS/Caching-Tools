package main

import (
	"math"
	"strings"
	"testing"
)

func TestInspectGPXTracksAndRoutes(t *testing.T) {
	input := `<?xml version="1.0"?><gpx version="1.1" creator="gps-test" xmlns="http://www.topografix.com/GPX/1/1">
<wpt lat="59.9" lon="10.7"><name>WP</name></wpt>
<rte><name>Short route</name><rtept lat="59.9139" lon="10.7522"/><rtept lat="59.9239" lon="10.7522"/></rte>
<trk><name>Morning track</name><trkseg><trkpt lat="59.9139" lon="10.7522"/><trkpt lat="59.9189" lon="10.7522"/></trkseg><trkseg><trkpt lat="60.0" lon="11.0"/><trkpt lat="60.005" lon="11.0"/></trkseg></trk>
</gpx>`
	result, err := inspectGPX(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "1.1" || result.Creator != "gps-test" || result.Waypoints != 1 {
		t.Fatalf("unexpected metadata: %+v", result)
	}
	if len(result.Routes) != 1 || result.Routes[0].Name != "Short route" || result.Routes[0].Points != 2 {
		t.Fatalf("unexpected route summary: %+v", result.Routes)
	}
	if result.Routes[0].DistanceM < 1100 || result.Routes[0].DistanceM > 1120 {
		t.Fatalf("unexpected route distance: %f", result.Routes[0].DistanceM)
	}
	if len(result.Tracks) != 1 || result.Tracks[0].Name != "Morning track" || result.Tracks[0].Segments != 2 || result.Tracks[0].Points != 4 {
		t.Fatalf("unexpected track summary: %+v", result.Tracks)
	}
	// Each segment is about 556 m. Segment boundaries must not be connected.
	if result.Tracks[0].DistanceM < 1100 || result.Tracks[0].DistanceM > 1120 {
		t.Fatalf("unexpected segmented track distance: %f", result.Tracks[0].DistanceM)
	}
	if math.Abs(result.Tracks[0].DistanceKM-result.Tracks[0].DistanceM/1000) > 1e-9 {
		t.Fatal("distance km mismatch")
	}
}

func TestInspectGPXRejectsInvalidPathCoordinate(t *testing.T) {
	input := `<gpx version="1.1"><trk><trkseg><trkpt lat="95" lon="10"/></trkseg></trk></gpx>`
	if _, err := inspectGPX(strings.NewReader(input)); err == nil {
		t.Fatal("expected invalid track coordinate error")
	}
}

func TestInspectGPXUsesFallbackNames(t *testing.T) {
	input := `<gpx version="1.1"><rte><rtept lat="59" lon="10"/></rte><trk><trkseg><trkpt lat="59" lon="10"/></trkseg></trk></gpx>`
	result, err := inspectGPX(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if result.Routes[0].Name != "Route 1" || result.Tracks[0].Name != "Track 1" {
		t.Fatalf("unexpected fallback names: %+v %+v", result.Routes, result.Tracks)
	}
}
