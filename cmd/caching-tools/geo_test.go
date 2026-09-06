package main

import (
	"math"
	"testing"
)

func TestParseCoordinate(t *testing.T) {
	cases := []struct {
		in   string
		lat  bool
		want float64
	}{
		{"58.1234", true, 58.1234},
		{"N 58 7.404", true, 58.1234},
		{"58° 7' 24.24\" N", true, 58.1234},
		{"W 8 30 0", false, -8.5},
		{"-8.5", false, -8.5},
	}
	for _, tc := range cases {
		got, err := parseCoordinate(tc.in, tc.lat)
		if err != nil {
			t.Fatalf("parseCoordinate(%q): %v", tc.in, err)
		}
		if math.Abs(got-tc.want) > 1e-8 {
			t.Fatalf("parseCoordinate(%q)=%f want %f", tc.in, got, tc.want)
		}
	}
}

func TestParseCoordinateRejectsInvalidValues(t *testing.T) {
	bad := []struct {
		in  string
		lat bool
	}{
		{"N 91", true},
		{"E 181", false},
		{"N 58 60", true},
		{"58 10 E", true},
		{"N S 58", true},
	}
	for _, tc := range bad {
		if _, err := parseCoordinate(tc.in, tc.lat); err == nil {
			t.Fatalf("parseCoordinate(%q) unexpectedly succeeded", tc.in)
		}
	}
}

func TestDistanceAndBearing(t *testing.T) {
	distance, bearing := distanceAndBearing(59.9139, 10.7522, 60.3913, 5.3221)
	if math.Abs(distance/1000-304.7) > 2.0 {
		t.Fatalf("distance %.1f km outside tolerance", distance/1000)
	}
	if math.Abs(bearing-281.7) > 2.0 {
		t.Fatalf("bearing %.1f outside tolerance", bearing)
	}
}

func TestDestinationPointRoundTrip(t *testing.T) {
	const lat1, lon1 = 59.9139, 10.7522
	const distanceM = 12345.0
	const bearing = 42.0
	lat2, lon2 := destinationPoint(lat1, lon1, bearing, distanceM)
	gotDistance, gotBearing := distanceAndBearing(lat1, lon1, lat2, lon2)
	if math.Abs(gotDistance-distanceM) > 0.01 {
		t.Fatalf("distance %.4f want %.4f", gotDistance, distanceM)
	}
	if math.Abs(gotBearing-bearing) > 0.0001 {
		t.Fatalf("bearing %.6f want %.6f", gotBearing, bearing)
	}
}

func TestDestinationPointNormalizesLongitude(t *testing.T) {
	_, lon := destinationPoint(0, 179.9, 90, 30000)
	if lon < -180 || lon > 180 {
		t.Fatalf("longitude not normalized: %f", lon)
	}
}

func TestFormatCoordinate(t *testing.T) {
	f := formatCoordinate(58.1234, true)
	if f.DMM != "N 58° 7.404'" {
		t.Fatalf("unexpected DMM: %s", f.DMM)
	}
	if f.DMS != "N 58° 07' 24.24\"" {
		t.Fatalf("unexpected DMS: %s", f.DMS)
	}
}
