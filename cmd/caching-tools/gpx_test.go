package main

import (
	"strings"
	"testing"
)

func TestParseGPXWaypoints(t *testing.T) {
	input := `<?xml version="1.0"?><gpx version="1.1" creator="test" xmlns="http://www.topografix.com/GPX/1/1"><wpt lat="59.9139" lon="10.7522"><name>Trailhead</name><cmt>Start here</cmt><type>parking</type></wpt><wpt lat="60.3913" lon="5.3221"><desc>Bergen</desc></wpt></gpx>`
	items, err := parseGPXWaypoints(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("len=%d", len(items))
	}
	if items[0].Name != "Trailhead" || items[0].Type != "parking" || items[0].Comment != "Start here" {
		t.Fatalf("unexpected first waypoint: %+v", items[0])
	}
	if items[1].Name != "GPX waypoint 2" || items[1].Comment != "Bergen" {
		t.Fatalf("unexpected fallback waypoint: %+v", items[1])
	}
}

func TestParseGPXRejectsWrongVersionAndInvalidCoordinate(t *testing.T) {
	wrongVersion := `<gpx version="1.0"><wpt lat="59" lon="10"><name>x</name></wpt></gpx>`
	if _, err := parseGPXWaypoints(strings.NewReader(wrongVersion)); err == nil {
		t.Fatal("expected GPX version error")
	}
	badCoordinate := `<gpx version="1.1"><wpt lat="99" lon="10"><name>x</name></wpt></gpx>`
	if _, err := parseGPXWaypoints(strings.NewReader(badCoordinate)); err == nil {
		t.Fatal("expected coordinate validation error")
	}
}

func TestEncodeGPXWaypoints(t *testing.T) {
	data, err := encodeGPXWaypoints([]waypoint{{Name: "Cache & View", Latitude: 59.9139, Longitude: 10.7522, Type: "geocache", Comment: "A < B"}})
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{`version="1.1"`, `creator="Caching Tools"`, `<name>Cache &amp; View</name>`, `<cmt>A &lt; B</cmt>`, `<type>geocache</type>`} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %q in %s", want, text)
		}
	}
}
