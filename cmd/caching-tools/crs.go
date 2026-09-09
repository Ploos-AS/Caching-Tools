package main

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type ellipsoid struct {
	A float64
	F float64
}

var (
	wgs84Ellipsoid = ellipsoid{A: 6378137.0, F: 1 / 298.257223563}
	grs80Ellipsoid = ellipsoid{A: 6378137.0, F: 1 / 298.257222101}
)

type crsRequest struct {
	Source     string  `json:"source"`
	Target     string  `json:"target"`
	Latitude   string  `json:"latitude,omitempty"`
	Longitude  string  `json:"longitude,omitempty"`
	Zone       int     `json:"zone,omitempty"`
	Hemisphere string  `json:"hemisphere,omitempty"`
	Easting    float64 `json:"easting,omitempty"`
	Northing   float64 `json:"northing,omitempty"`
}

type crsGrid struct {
	CRS        string  `json:"crs"`
	Zone       int     `json:"zone"`
	Hemisphere string  `json:"hemisphere"`
	Easting    float64 `json:"easting"`
	Northing   float64 `json:"northing"`
}

type crsResponse struct {
	Source      string         `json:"source"`
	Target      string         `json:"target"`
	Point       *pointResponse `json:"point,omitempty"`
	Grid        *crsGrid       `json:"grid,omitempty"`
	Approximate bool           `json:"approximate,omitempty"`
	Note        string         `json:"note,omitempty"`
}

func convertCRS(req crsRequest) (crsResponse, error) {
	source := normalizeCRS(req.Source)
	target := normalizeCRS(req.Target)
	if source == "" || target == "" {
		return crsResponse{}, errors.New("source and target CRS are required")
	}

	sourceZone, sourceIsETRSUTM := etrs89UTMZoneFromCRS(source)
	targetZone, targetIsETRSUTM := etrs89UTMZoneFromCRS(target)

	switch {
	case (source == "EPSG:4326" || source == "EPSG:4258") && (target == "EPSG:4326" || target == "EPSG:4258"):
		lat, lon, err := parsePoint(coordinateRequest{Latitude: req.Latitude, Longitude: req.Longitude})
		if err != nil { return crsResponse{}, err }
		approx := source != target
		note := ""
		if approx { note = "WGS84 and ETRS89 are treated as coincident for this local GPS/hobby conversion; no coordinate epoch or plate-motion transformation is applied." }
		p := point(lat, lon)
		return crsResponse{Source: source, Target: target, Point: &p, Approximate: approx, Note: note}, nil

	case (source == "EPSG:4326" || source == "EPSG:4258") && (target == "ETRS89-UTM" || targetIsETRSUTM):
		lat, lon, err := parsePoint(coordinateRequest{Latitude: req.Latitude, Longitude: req.Longitude})
		if err != nil { return crsResponse{}, err }
		zone, err := resolveETRS89Zone(req.Zone, targetZone, targetIsETRSUTM)
		if err != nil { return crsResponse{}, err }
		e, n, err := latLonToUTMZone(lat, lon, zone, grs80Ellipsoid)
		if err != nil { return crsResponse{}, err }
		gridCRS := etrs89UTMCRS(zone)
		grid := crsGrid{CRS: gridCRS, Zone: zone, Hemisphere: "N", Easting: e, Northing: n}
		return crsResponse{Source: source, Target: gridCRS, Grid: &grid, Approximate: source == "EPSG:4326", Note: etrsApproxNote(source)}, nil

	case (source == "ETRS89-UTM" || sourceIsETRSUTM) && (target == "EPSG:4258" || target == "EPSG:4326"):
		zone, err := resolveETRS89Zone(req.Zone, sourceZone, sourceIsETRSUTM)
		if err != nil { return crsResponse{}, err }
		if strings.ToUpper(strings.TrimSpace(req.Hemisphere)) != "N" && strings.TrimSpace(req.Hemisphere) != "" {
			return crsResponse{}, errors.New("ETRS89 / UTM zones use northern hemisphere coordinates")
		}
		lat, lon, err := utmZoneToLatLon(zone, "N", req.Easting, req.Northing, grs80Ellipsoid)
		if err != nil { return crsResponse{}, err }
		p := point(lat, lon)
		return crsResponse{Source: etrs89UTMCRS(zone), Target: target, Point: &p, Approximate: target == "EPSG:4326", Note: etrsApproxNote(target)}, nil
	default:
		return crsResponse{}, fmt.Errorf("unsupported CRS conversion %q -> %q", req.Source, req.Target)
	}
}

func normalizeCRS(value string) string {
	v := strings.ToUpper(strings.TrimSpace(value))
	switch v {
	case "WGS84", "WGS 84", "EPSG:4326": return "EPSG:4326"
	case "ETRS89", "ETRS 89", "EPSG:4258": return "EPSG:4258"
	case "ETRS89-UTM", "ETRS89 / UTM", "ETRS89 UTM": return "ETRS89-UTM"
	default:
		if zone, ok := etrs89UTMZoneFromCRS(v); ok { return etrs89UTMCRS(zone) }
		return v
	}
}

func etrs89UTMCRS(zone int) string {
	return fmt.Sprintf("EPSG:%d", 25800+zone)
}

func etrs89UTMZoneFromCRS(crs string) (int, bool) {
	v := strings.ToUpper(strings.TrimSpace(crs))
	if !strings.HasPrefix(v, "EPSG:258") || len(v) != len("EPSG:25832") {
		return 0, false
	}
	code, err := strconv.Atoi(strings.TrimPrefix(v, "EPSG:"))
	if err != nil { return 0, false }
	zone := code - 25800
	if zone < 28 || zone > 38 { return 0, false }
	return zone, true
}

func resolveETRS89Zone(requested, explicit int, hasExplicit bool) (int, error) {
	if hasExplicit {
		if requested != 0 && requested != explicit {
			return 0, fmt.Errorf("zone %d conflicts with explicit CRS %s", requested, etrs89UTMCRS(explicit))
		}
		return explicit, nil
	}
	return validateETRS89Zone(requested)
}

func validateETRS89Zone(zone int) (int, error) {
	if zone < 28 || zone > 38 { return 0, errors.New("ETRS89 / UTM zone must be 28..38") }
	return zone, nil
}

func etrsApproxNote(crs string) string {
	if crs == "EPSG:4326" { return "WGS84 and ETRS89 are treated as coincident for this local GPS/hobby conversion; no coordinate epoch or plate-motion transformation is applied." }
	return ""
}

func latLonToUTMZone(lat, lon float64, zone int, ell ellipsoid) (float64, float64, error) {
	if lat < -80 || lat > 84 { return 0, 0, errors.New("UTM supports latitudes from 80°S to 84°N") }
	if lon < -180 || lon > 180 { return 0, 0, errors.New("longitude out of range") }
	if zone < 1 || zone > 60 { return 0, 0, errors.New("UTM zone must be 1..60") }
	lon0 := float64((zone-1)*6 - 180 + 3)
	e2 := ell.F * (2 - ell.F)
	ep2 := e2 / (1 - e2)
	phi, lambda, lambda0 := radians(lat), radians(lon), radians(lon0)
	n := ell.A / math.Sqrt(1-e2*math.Sin(phi)*math.Sin(phi))
	t := math.Tan(phi) * math.Tan(phi)
	c := ep2 * math.Cos(phi) * math.Cos(phi)
	a := math.Cos(phi) * (lambda - lambda0)
	m := ell.A * ((1-e2/4-3*e2*e2/64-5*e2*e2*e2/256)*phi - (3*e2/8+3*e2*e2/32+45*e2*e2*e2/1024)*math.Sin(2*phi) + (15*e2*e2/256+45*e2*e2*e2/1024)*math.Sin(4*phi) - (35*e2*e2*e2/3072)*math.Sin(6*phi))
	easting := utmK0*n*(a+(1-t+c)*math.Pow(a,3)/6+(5-18*t+t*t+72*c-58*ep2)*math.Pow(a,5)/120)+500000
	northing := utmK0*(m+n*math.Tan(phi)*(a*a/2+(5-t+9*c+4*c*c)*math.Pow(a,4)/24+(61-58*t+t*t+600*c-330*ep2)*math.Pow(a,6)/720))
	if lat < 0 { northing += 10000000 }
	return easting, northing, nil
}

func utmZoneToLatLon(zone int, hemisphere string, easting, northing float64, ell ellipsoid) (float64, float64, error) {
	if zone < 1 || zone > 60 { return 0, 0, errors.New("UTM zone must be 1..60") }
	if hemisphere != "N" && hemisphere != "S" { return 0, 0, errors.New("hemisphere must be N or S") }
	if easting < 100000 || easting >= 1000000 { return 0, 0, errors.New("UTM easting must be in [100000,1000000)") }
	if northing < 0 || northing > 10000000 { return 0, 0, errors.New("UTM northing must be in [0,10000000]") }
	e2 := ell.F * (2 - ell.F)
	ep2 := e2 / (1 - e2)
	x, y := easting-500000, northing
	if hemisphere == "S" { y -= 10000000 }
	m := y / utmK0
	mu := m / (ell.A * (1-e2/4-3*e2*e2/64-5*e2*e2*e2/256))
	e1 := (1-math.Sqrt(1-e2))/(1+math.Sqrt(1-e2))
	phi1 := mu+(3*e1/2-27*math.Pow(e1,3)/32)*math.Sin(2*mu)+(21*e1*e1/16-55*math.Pow(e1,4)/32)*math.Sin(4*mu)+(151*math.Pow(e1,3)/96)*math.Sin(6*mu)+(1097*math.Pow(e1,4)/512)*math.Sin(8*mu)
	n1 := ell.A/math.Sqrt(1-e2*math.Sin(phi1)*math.Sin(phi1))
	r1 := ell.A*(1-e2)/math.Pow(1-e2*math.Sin(phi1)*math.Sin(phi1),1.5)
	t1 := math.Tan(phi1)*math.Tan(phi1)
	c1 := ep2*math.Cos(phi1)*math.Cos(phi1)
	d := x/(n1*utmK0)
	lat := phi1-(n1*math.Tan(phi1)/r1)*(d*d/2-(5+3*t1+10*c1-4*c1*c1-9*ep2)*math.Pow(d,4)/24+(61+90*t1+298*c1+45*t1*t1-252*ep2-3*c1*c1)*math.Pow(d,6)/720)
	lon0 := radians(float64((zone-1)*6-180+3))
	lon := lon0+(d-(1+2*t1+c1)*math.Pow(d,3)/6+(5-2*c1+28*t1-3*c1*c1+8*ep2+24*t1*t1)*math.Pow(d,5)/120)/math.Cos(phi1)
	return degrees(lat), degrees(lon), nil
}
