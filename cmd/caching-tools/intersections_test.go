package main

import (
	"math"
	"testing"
)

func TestBearingBearingIntersection(t *testing.T) {
	lat, lon, err := bearingBearingIntersection(0, 0, 45, 0, 1, 315)
	if err != nil { t.Fatal(err) }
	if math.Abs(lat-0.5) > 0.02 || math.Abs(lon-0.5) > 0.02 { t.Fatalf("got %.6f, %.6f", lat, lon) }
}

func TestBearingDistanceIntersection(t *testing.T) {
	centerLat, centerLon := destinationPoint(0, 0, 90, 10000)
	pts, err := bearingDistanceIntersections(0, 0, 90, centerLat, centerLon, 5000)
	if err != nil { t.Fatal(err) }
	if len(pts) != 2 { t.Fatalf("expected 2 intersections, got %d", len(pts)) }
	for _, p := range pts {
		d, _ := distanceAndBearing(p[0], p[1], centerLat, centerLon)
		if math.Abs(d-5000) > 2 { t.Fatalf("distance %.3f", d) }
	}
}

func TestCircleCircleIntersection(t *testing.T) {
	lat2, lon2 := destinationPoint(0, 0, 90, 10000)
	pts, err := circleCircleIntersections(0, 0, 7000, lat2, lon2, 7000)
	if err != nil { t.Fatal(err) }
	if len(pts) != 2 { t.Fatalf("expected 2 intersections, got %d", len(pts)) }
	for _, p := range pts {
		d1, _ := distanceAndBearing(0, 0, p[0], p[1])
		d2, _ := distanceAndBearing(lat2, lon2, p[0], p[1])
		if math.Abs(d1-7000) > 2 || math.Abs(d2-7000) > 2 { t.Fatalf("distances %.3f %.3f", d1, d2) }
	}
}

func TestCircleCircleRejectsSeparateCircles(t *testing.T) {
	lat2, lon2 := destinationPoint(0, 0, 90, 20000)
	if _, err := circleCircleIntersections(0, 0, 1000, lat2, lon2, 1000); err == nil { t.Fatal("expected error") }
}
