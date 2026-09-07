package main

import (
	"math"
	"testing"
)

func TestNearestPointOnRoute(t *testing.T) {
	item := storedPath{Kind:"route", Name:"Line", Route:[]gpxPoint{{Latitude:60,Longitude:10},{Latitude:60,Longitude:11}}}
	lat, lon, pos, distance, err := nearestPointOnStoredPath(60.1, 10.5, item)
	if err != nil { t.Fatal(err) }
	if pos.Segment != 0 || pos.Edge != 0 { t.Fatalf("position=%+v", pos) }
	if math.Abs(lat-60) > 0.002 || math.Abs(lon-10.5) > 0.002 { t.Fatalf("nearest=%f,%f",lat,lon) }
	if distance < 10000 || distance > 12000 { t.Fatalf("distance=%f",distance) }
}

func TestFieldNavigateWaypointAndTrack(t *testing.T) {
	dir := t.TempDir()
	waypoints := newWaypointStore(dir)
	wp, err := waypoints.create(waypointRequest{Name:"Target",Latitude:"60",Longitude:"10.1"})
	if err != nil { t.Fatal(err) }
	paths := newPathStore(dir)
	paths.mu.Lock()
	err = paths.saveLocked([]storedPath{{ID:"path-test",Kind:"track",Name:"Trail",Segments:[]gpxTrackSegment{{Points:[]gpxPoint{{Latitude:60,Longitude:10},{Latitude:60,Longitude:11}}}},CreatedAt:"2026-09-07T00:00:00Z"}})
	paths.mu.Unlock()
	if err != nil { t.Fatal(err) }

	got, err := fieldNavigate(fieldNavigationRequest{From:coordinateRequest{Latitude:"60",Longitude:"10"},WaypointID:wp.ID},waypoints,paths)
	if err != nil { t.Fatal(err) }
	if got.Kind != "waypoint" || got.ID != wp.ID || got.DistanceM <= 0 { t.Fatalf("waypoint result=%+v",got) }
	if got.CrossTrackM != nil { t.Fatalf("waypoint cross track should be nil") }

	got, err = fieldNavigate(fieldNavigationRequest{From:coordinateRequest{Latitude:"60.1",Longitude:"10.5"},PathID:"path-test"},waypoints,paths)
	if err != nil { t.Fatal(err) }
	if got.Kind != "track" || got.CrossTrackM == nil || *got.CrossTrackM < 10000 { t.Fatalf("track result=%+v",got) }
	if got.NearestPath == nil || got.NearestPath.Segment != 0 || got.NearestPath.Edge != 0 { t.Fatalf("nearest=%+v",got.NearestPath) }
}

func TestFieldNavigateRequiresOneTarget(t *testing.T) {
	dir := t.TempDir()
	_, err := fieldNavigate(fieldNavigationRequest{From:coordinateRequest{Latitude:"60",Longitude:"10"}},newWaypointStore(dir),newPathStore(dir))
	if err == nil { t.Fatal("expected target validation error") }
}
