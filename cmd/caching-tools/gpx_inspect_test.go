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

func TestInspectGPXTrackElevationAndTimeStatistics(t *testing.T) {
	input := `<gpx version="1.1"><trk><name>Timed climb</name><trkseg>
<trkpt lat="59.0000" lon="10.0000"><ele>100</ele><time>2026-09-07T08:00:00Z</time></trkpt>
<trkpt lat="59.0010" lon="10.0000"><ele>130</ele><time>2026-09-07T08:01:00Z</time></trkpt>
<trkpt lat="59.0020" lon="10.0000"><ele>120</ele><time>2026-09-07T08:03:00Z</time></trkpt>
</trkseg></trk></gpx>`
	result, err := inspectGPX(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	track := result.Tracks[0]
	if track.ElevationPoints != 3 || track.MinElevationM == nil || track.MaxElevationM == nil {
		t.Fatalf("missing elevation stats: %+v", track)
	}
	if *track.MinElevationM != 100 || *track.MaxElevationM != 130 || track.ElevationGainM != 30 || track.ElevationLossM != 10 {
		t.Fatalf("unexpected elevation stats: %+v", track)
	}
	if track.TimedPoints != 3 || track.DurationS != 180 {
		t.Fatalf("unexpected time stats: %+v", track)
	}
	if track.AverageSpeedKmh == nil || track.MaxSpeedKmh == nil || *track.AverageSpeedKmh <= 0 || *track.MaxSpeedKmh <= *track.AverageSpeedKmh {
		t.Fatalf("unexpected speed stats: %+v", track)
	}
}

func TestInspectGPXTrackStatsDoNotCrossSegmentBoundaries(t *testing.T) {
	input := `<gpx version="1.1"><trk><trkseg>
<trkpt lat="59" lon="10"><ele>100</ele><time>2026-09-07T08:00:00Z</time></trkpt>
<trkpt lat="59.001" lon="10"><ele>110</ele><time>2026-09-07T08:01:00Z</time></trkpt>
</trkseg><trkseg>
<trkpt lat="60" lon="11"><ele>500</ele><time>2026-09-07T09:00:00Z</time></trkpt>
<trkpt lat="60.001" lon="11"><ele>490</ele><time>2026-09-07T09:02:00Z</time></trkpt>
</trkseg></trk></gpx>`
	result, err := inspectGPX(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	track := result.Tracks[0]
	if track.DurationS != 180 {
		t.Fatalf("segment gap incorrectly included in duration: %f", track.DurationS)
	}
	if track.ElevationGainM != 10 || track.ElevationLossM != 10 {
		t.Fatalf("segment boundary incorrectly affected elevation: %+v", track)
	}
}

func TestInspectGPXAllowsPartialTrackMetadata(t *testing.T) {
	input := `<gpx version="1.1"><trk><trkseg>
<trkpt lat="59" lon="10"><ele>0</ele><time>2026-09-07T08:00:00Z</time></trkpt>
<trkpt lat="59.001" lon="10"/>
<trkpt lat="59.002" lon="10"><ele>20</ele><time>2026-09-07T08:02:00Z</time></trkpt>
</trkseg></trk></gpx>`
	result, err := inspectGPX(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	track := result.Tracks[0]
	if track.ElevationPoints != 2 || track.TimedPoints != 2 {
		t.Fatalf("partial metadata counts wrong: %+v", track)
	}
	if track.DurationS != 0 || track.AverageSpeedKmh != nil || track.MaxSpeedKmh != nil {
		t.Fatalf("speed should require adjacent timed points: %+v", track)
	}
	if track.MinElevationM == nil || *track.MinElevationM != 0 {
		t.Fatalf("zero elevation must be preserved: %+v", track)
	}
}

func TestInspectGPXRejectsInvalidPathCoordinate(t *testing.T) {
	input := `<gpx version="1.1"><trk><trkseg><trkpt lat="95" lon="10"/></trkseg></trk></gpx>`
	if _, err := inspectGPX(strings.NewReader(input)); err == nil {
		t.Fatal("expected invalid track coordinate error")
	}
}

func TestInspectGPXRejectsInvalidTrackTime(t *testing.T) {
	input := `<gpx version="1.1"><trk><trkseg><trkpt lat="59" lon="10"><time>not-a-time</time></trkpt></trkseg></trk></gpx>`
	if _, err := inspectGPX(strings.NewReader(input)); err == nil {
		t.Fatal("expected invalid track time error")
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
