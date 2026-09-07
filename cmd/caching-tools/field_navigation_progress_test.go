package main

import (
	"math"
	"testing"
)

func TestPathProgressRouteMidpoint(t *testing.T) {
	item := storedPath{Kind:"route", Route:[]gpxPoint{
		{Latitude:0, Longitude:0},
		{Latitude:0, Longitude:0.01},
		{Latitude:0, Longitude:0.02},
	}}
	lat, lon, pos, fraction, _, err := nearestPointOnStoredPathDetailed(0.001, 0.005, item)
	if err != nil { t.Fatal(err) }
	if pos.Segment != 0 || pos.Edge != 0 { t.Fatalf("position=%+v", pos) }
	if math.Abs(fraction-0.5) > 0.02 { t.Fatalf("fraction=%f", fraction) }
	p, err := pathProgress(item, pos, fraction, lat, lon)
	if err != nil { t.Fatal(err) }
	if p.ProgressPercent < 24 || p.ProgressPercent > 26 { t.Fatalf("progress=%f", p.ProgressPercent) }
	if math.Abs(p.TotalM-(2*p.NextPointDistanceM+2*p.AlongM-p.TotalM)) > p.TotalM { t.Fatalf("unexpected progress values: %+v", p) }
	if p.RemainingM <= p.AlongM { t.Fatalf("remaining=%f along=%f", p.RemainingM, p.AlongM) }
	if p.NextPointDistanceM < 500 || p.NextPointDistanceM > 600 { t.Fatalf("next distance=%f", p.NextPointDistanceM) }
	if p.ForwardBearingDeg < 89 || p.ForwardBearingDeg > 91 { t.Fatalf("bearing=%f", p.ForwardBearingDeg) }
}

func TestPathProgressTrackDoesNotBridgeSegments(t *testing.T) {
	item := storedPath{Kind:"track", Segments:[]gpxTrackSegment{
		{Points:[]gpxPoint{{Latitude:0,Longitude:0},{Latitude:0,Longitude:0.01}}},
		{Points:[]gpxPoint{{Latitude:1,Longitude:1},{Latitude:1,Longitude:1.01}}},
	}}
	lat, lon, pos, fraction, _, err := nearestPointOnStoredPathDetailed(1.0, 1.005, item)
	if err != nil { t.Fatal(err) }
	if pos.Segment != 1 || pos.Edge != 0 { t.Fatalf("position=%+v", pos) }
	p, err := pathProgress(item, pos, fraction, lat, lon)
	if err != nil { t.Fatal(err) }
	first, _ := distanceAndBearing(0,0,0,0.01)
	second, _ := distanceAndBearing(1,1,1,1.01)
	if math.Abs(p.TotalM-(first+second)) > 1 { t.Fatalf("total=%f want=%f", p.TotalM, first+second) }
	if p.ProgressPercent < 74 || p.ProgressPercent > 76 { t.Fatalf("progress=%f", p.ProgressPercent) }
}

func TestPathProgressSinglePoint(t *testing.T) {
	item := storedPath{Kind:"route", Route:[]gpxPoint{{Latitude:59, Longitude:10}}}
	p, err := pathProgress(item, fieldPathPosition{Segment:0, Edge:0}, 0, 59, 10)
	if err != nil { t.Fatal(err) }
	if p.TotalM != 0 || p.RemainingM != 0 || p.ProgressPercent != 100 { t.Fatalf("progress=%+v", p) }
	if p.NextPointDistanceM != 0 { t.Fatalf("next distance=%f", p.NextPointDistanceM) }
}
