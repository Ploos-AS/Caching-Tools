package main

import "testing"

func TestSetPointPreservesPointMetadata(t *testing.T) {
	ele := 123.4
	item := storedPath{
		Kind: "track",
		Segments: []gpxTrackSegment{{Points: []gpxPoint{{Latitude: 59, Longitude: 10, Elevation: &ele, Time: "2026-09-07T12:00:00Z"}, {Latitude: 59.1, Longitude: 10.1}}}},
	}
	if err := editStoredPath(&item, "set-point", pathEditRequest{Segment: 0, Point: 0, Latitude: 60.0, Longitude: 11.0}); err != nil {
		t.Fatal(err)
	}
	got := item.Segments[0].Points[0]
	if got.Latitude != 60 || got.Longitude != 11 {
		t.Fatalf("coordinates=%v,%v", got.Latitude, got.Longitude)
	}
	if got.Elevation == nil || *got.Elevation != ele || got.Time != "2026-09-07T12:00:00Z" {
		t.Fatalf("metadata not preserved: %+v", got)
	}
}

func TestSetPointRejectsInvalidCoordinates(t *testing.T) {
	item := storedPath{Kind: "route", Route: []gpxPoint{{Latitude: 59, Longitude: 10}}}
	if err := editStoredPath(&item, "set-point", pathEditRequest{Point: 0, Latitude: 91, Longitude: 10}); err == nil {
		t.Fatal("expected invalid latitude error")
	}
	if item.Route[0].Latitude != 59 || item.Route[0].Longitude != 10 {
		t.Fatalf("route mutated after failed edit: %+v", item.Route[0])
	}
}
