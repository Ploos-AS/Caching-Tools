package main

import (
	"math"
	"strings"
	"testing"
)

func TestLatLonToUTMOslo(t *testing.T) {
	utm, err := latLonToUTM(59.9139, 10.7522)
	if err != nil {
		t.Fatal(err)
	}
	if utm.Zone != 32 || utm.Band != "V" || utm.Hemisphere != "N" {
		t.Fatalf("unexpected zone/band/hemisphere: %+v", utm)
	}
	if math.Abs(utm.Easting-597979.903) > 1 || math.Abs(utm.Northing-6643118.991) > 1 {
		t.Fatalf("unexpected UTM: %.3f %.3f", utm.Easting, utm.Northing)
	}
	if utm.MGRS != "32V NM 97979 43118" {
		t.Fatalf("unexpected MGRS: %s", utm.MGRS)
	}
}

func TestUTMRoundTrip(t *testing.T) {
	utm, err := latLonToUTM(58.1234, 8.0)
	if err != nil {
		t.Fatal(err)
	}
	lat, lon, err := utmToLatLon(utm.Zone, utm.Hemisphere, utm.Easting, utm.Northing)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(lat-58.1234) > 1e-6 || math.Abs(lon-8.0) > 1e-6 {
		t.Fatalf("round trip got %.8f %.8f", lat, lon)
	}
}

func TestNorwayAndSvalbardZones(t *testing.T) {
	if got := utmZone(60, 4); got != 32 {
		t.Fatalf("Norway exception zone=%d want 32", got)
	}
	cases := []struct{ lon float64; zone int }{{5, 31}, {15, 33}, {25, 35}, {35, 37}}
	for _, tc := range cases {
		if got := utmZone(78, tc.lon); got != tc.zone {
			t.Fatalf("Svalbard lon %.1f zone=%d want %d", tc.lon, got, tc.zone)
		}
	}
}

func TestUTMValidation(t *testing.T) {
	if _, err := latLonToUTM(85, 10); err == nil {
		t.Fatal("expected latitude error")
	}
	_, _, err := utmToLatLon(0, "N", 500000, 6500000)
	if err == nil || !strings.Contains(err.Error(), "zone") {
		t.Fatalf("expected zone error, got %v", err)
	}
}
