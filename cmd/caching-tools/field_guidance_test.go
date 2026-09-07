package main

import "testing"

func TestNavigationThresholdDefaultsAndValidation(t *testing.T) {
	off, arrival, err := navigationThresholds(fieldNavigationRequest{})
	if err != nil { t.Fatal(err) }
	if off != defaultOffRouteThresholdM || arrival != defaultArrivalRadiusM {
		t.Fatalf("defaults off=%f arrival=%f", off, arrival)
	}
	zero := 0.0
	if _, _, err := navigationThresholds(fieldNavigationRequest{ArrivalRadiusM:&zero}); err == nil {
		t.Fatal("expected arrival radius validation error")
	}
}

func TestWaypointArrivalGuidance(t *testing.T) {
	g := waypointGuidance(10, 50, 20)
	if g.Status != "arrived" || !g.Arrived || g.OffRoute { t.Fatalf("guidance=%+v", g) }
	g = waypointGuidance(100, 50, 20)
	if g.Status != "go-to" || g.Arrived { t.Fatalf("guidance=%+v", g) }
}

func TestPathDeviationAndArrivalGuidance(t *testing.T) {
	progress := fieldPathProgress{RemainingM:1000, NextPointDistanceM:300}
	g := pathGuidance(80, progress, 50, 20)
	if g.Status != "off-route" || !g.OffRoute || g.Arrived { t.Fatalf("guidance=%+v", g) }

	g = pathGuidance(10, progress, 50, 20)
	if g.Status != "on-route" || g.OffRoute { t.Fatalf("guidance=%+v", g) }

	progress.NextPointDistanceM = 10
	g = pathGuidance(10, progress, 50, 20)
	if g.Status != "next-point-arrival" || g.Arrived { t.Fatalf("guidance=%+v", g) }

	progress.RemainingM = 10
	g = pathGuidance(100, progress, 50, 20)
	if g.Status != "arrived" || !g.Arrived || g.OffRoute { t.Fatalf("guidance=%+v", g) }
}
