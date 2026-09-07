package main

import (
	"math"
	"testing"
)

func TestCRSWGS84ToETRS89UTMAndBack(t *testing.T) {
	forward, err := convertCRS(crsRequest{Source:"EPSG:4326", Target:"ETRS89-UTM", Latitude:"59.9139", Longitude:"10.7522", Zone:32})
	if err != nil { t.Fatal(err) }
	if forward.Grid == nil { t.Fatal("missing grid") }
	if forward.Grid.CRS != "EPSG:25832" || forward.Grid.Zone != 32 || forward.Grid.Hemisphere != "N" { t.Fatalf("grid=%+v", forward.Grid) }
	if !forward.Approximate { t.Fatal("expected WGS84->ETRS89 approximation flag") }
	if math.Abs(forward.Grid.Easting-597979.9) > 2 || math.Abs(forward.Grid.Northing-6643118.9) > 2 { t.Fatalf("grid=%+v", forward.Grid) }

	back, err := convertCRS(crsRequest{Source:"ETRS89-UTM", Target:"EPSG:4326", Zone:32, Easting:forward.Grid.Easting, Northing:forward.Grid.Northing})
	if err != nil { t.Fatal(err) }
	if back.Point == nil { t.Fatal("missing point") }
	if math.Abs(back.Point.Latitude-59.9139) > 1e-5 || math.Abs(back.Point.Longitude-10.7522) > 1e-5 { t.Fatalf("point=%+v", back.Point) }
	if !back.Approximate { t.Fatal("expected ETRS89->WGS84 approximation flag") }
}

func TestCRSETRS89GeographicToGridIsNotApproximate(t *testing.T) {
	result, err := convertCRS(crsRequest{Source:"EPSG:4258", Target:"ETRS89-UTM", Latitude:"59.9139", Longitude:"10.7522", Zone:32})
	if err != nil { t.Fatal(err) }
	if result.Approximate { t.Fatal("ETRS89 geographic to ETRS89 UTM should not be marked approximate") }
	if result.Grid == nil || result.Grid.CRS != "EPSG:25832" { t.Fatalf("result=%+v", result) }
}

func TestCRSGeographicIdentityApproximation(t *testing.T) {
	result, err := convertCRS(crsRequest{Source:"WGS84", Target:"ETRS89", Latitude:"58", Longitude:"8"})
	if err != nil { t.Fatal(err) }
	if !result.Approximate || result.Note == "" { t.Fatalf("result=%+v", result) }
	if result.Point == nil || result.Point.Latitude != 58 || result.Point.Longitude != 8 { t.Fatalf("point=%+v", result.Point) }
}

func TestCRSValidation(t *testing.T) {
	if _, err := convertCRS(crsRequest{Source:"EPSG:4326", Target:"ETRS89-UTM", Latitude:"59", Longitude:"10", Zone:27}); err == nil { t.Fatal("expected zone validation error") }
	if _, err := convertCRS(crsRequest{Source:"foo", Target:"bar"}); err == nil { t.Fatal("expected unsupported CRS error") }
}
