package main

import (
	"bytes"
	"encoding/xml"
	"fmt"
)

func encodeStoredPathGPX(item storedPath) ([]byte, error) {
	doc := gpxDocument{
		XMLName: xml.Name{Space: gpx11Namespace, Local: "gpx"},
		Version: "1.1",
		Creator: "Caching Tools",
	}

	switch item.Kind {
	case "route":
		doc.Routes = []gpxRoute{{Name: item.Name, Points: item.Route}}
	case "track":
		doc.Tracks = []gpxTrack{{Name: item.Name, Segments: item.Segments}}
	default:
		return nil, fmt.Errorf("unsupported stored path kind %q", item.Kind)
	}

	var buf bytes.Buffer
	buf.WriteString(xml.Header)
	enc := xml.NewEncoder(&buf)
	enc.Indent("", "  ")
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}
