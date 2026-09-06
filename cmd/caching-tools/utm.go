package main

import (
	"errors"
	"fmt"
	"math"
)

const (
	wgs84A  = 6378137.0
	wgs84F  = 1 / 298.257223563
	utmK0   = 0.9996
)

type utmCoordinate struct {
	Zone       int     `json:"zone"`
	Band       string  `json:"band"`
	Hemisphere string  `json:"hemisphere"`
	Easting    float64 `json:"easting"`
	Northing   float64 `json:"northing"`
	MGRS       string  `json:"mgrs"`
}

type gridResponse struct {
	WGS84 pointResponse `json:"wgs84"`
	UTM   utmCoordinate `json:"utm"`
}

type utmRequest struct {
	Zone       int     `json:"zone"`
	Hemisphere string  `json:"hemisphere"`
	Easting    float64 `json:"easting"`
	Northing   float64 `json:"northing"`
}

func latLonToUTM(lat, lon float64) (utmCoordinate, error) {
	if lat < -80 || lat > 84 {
		return utmCoordinate{}, errors.New("UTM supports latitudes from 80°S to 84°N")
	}
	if lon < -180 || lon > 180 {
		return utmCoordinate{}, errors.New("longitude out of range")
	}

	zone := utmZone(lat, lon)
	band := latitudeBand(lat)
	lon0 := float64((zone-1)*6 - 180 + 3)

	e2 := wgs84F * (2 - wgs84F)
	ep2 := e2 / (1 - e2)
	phi := radians(lat)
	lambda := radians(lon)
	lambda0 := radians(lon0)

	n := wgs84A / math.Sqrt(1-e2*math.Sin(phi)*math.Sin(phi))
	t := math.Tan(phi) * math.Tan(phi)
	c := ep2 * math.Cos(phi) * math.Cos(phi)
	a := math.Cos(phi) * (lambda - lambda0)

	m := wgs84A * ((1-e2/4-3*e2*e2/64-5*e2*e2*e2/256)*phi -
		(3*e2/8+3*e2*e2/32+45*e2*e2*e2/1024)*math.Sin(2*phi) +
		(15*e2*e2/256+45*e2*e2*e2/1024)*math.Sin(4*phi) -
		(35*e2*e2*e2/3072)*math.Sin(6*phi))

	easting := utmK0*n*(a+(1-t+c)*math.Pow(a, 3)/6+(5-18*t+t*t+72*c-58*ep2)*math.Pow(a, 5)/120) + 500000
	northing := utmK0 * (m + n*math.Tan(phi)*(a*a/2+(5-t+9*c+4*c*c)*math.Pow(a, 4)/24+(61-58*t+t*t+600*c-330*ep2)*math.Pow(a, 6)/720))
	hemisphere := "N"
	if lat < 0 {
		northing += 10000000
		hemisphere = "S"
	}

	mgrs := formatMGRS(zone, band, easting, northing)
	return utmCoordinate{Zone: zone, Band: band, Hemisphere: hemisphere, Easting: easting, Northing: northing, MGRS: mgrs}, nil
}

func utmToLatLon(zone int, hemisphere string, easting, northing float64) (float64, float64, error) {
	if zone < 1 || zone > 60 {
		return 0, 0, errors.New("UTM zone must be 1..60")
	}
	if hemisphere != "N" && hemisphere != "S" {
		return 0, 0, errors.New("hemisphere must be N or S")
	}
	if easting < 100000 || easting >= 1000000 {
		return 0, 0, errors.New("UTM easting must be in [100000,1000000)")
	}
	if northing < 0 || northing > 10000000 {
		return 0, 0, errors.New("UTM northing must be in [0,10000000]")
	}

	e2 := wgs84F * (2 - wgs84F)
	ep2 := e2 / (1 - e2)
	x := easting - 500000
	y := northing
	if hemisphere == "S" {
		y -= 10000000
	}

	m := y / utmK0
	mu := m / (wgs84A * (1 - e2/4 - 3*e2*e2/64 - 5*e2*e2*e2/256))
	e1 := (1 - math.Sqrt(1-e2)) / (1 + math.Sqrt(1-e2))
	phi1 := mu +
		(3*e1/2-27*math.Pow(e1, 3)/32)*math.Sin(2*mu) +
		(21*e1*e1/16-55*math.Pow(e1, 4)/32)*math.Sin(4*mu) +
		(151*math.Pow(e1, 3)/96)*math.Sin(6*mu) +
		(1097*math.Pow(e1, 4)/512)*math.Sin(8*mu)

	n1 := wgs84A / math.Sqrt(1-e2*math.Sin(phi1)*math.Sin(phi1))
	r1 := wgs84A * (1 - e2) / math.Pow(1-e2*math.Sin(phi1)*math.Sin(phi1), 1.5)
	t1 := math.Tan(phi1) * math.Tan(phi1)
	c1 := ep2 * math.Cos(phi1) * math.Cos(phi1)
	d := x / (n1 * utmK0)

	lat := phi1 - (n1*math.Tan(phi1)/r1)*(d*d/2-(5+3*t1+10*c1-4*c1*c1-9*ep2)*math.Pow(d, 4)/24+(61+90*t1+298*c1+45*t1*t1-252*ep2-3*c1*c1)*math.Pow(d, 6)/720)
	lon0 := radians(float64((zone-1)*6 - 180 + 3))
	lon := lon0 + (d-(1+2*t1+c1)*math.Pow(d, 3)/6+(5-2*c1+28*t1-3*c1*c1+8*ep2+24*t1*t1)*math.Pow(d, 5)/120)/math.Cos(phi1)

	latDeg, lonDeg := degrees(lat), degrees(lon)
	if latDeg < -80 || latDeg > 84 {
		return 0, 0, errors.New("UTM coordinate resolves outside supported latitude range")
	}
	return latDeg, lonDeg, nil
}

func utmZone(lat, lon float64) int {
	zone := int(math.Floor((lon+180)/6)) + 1
	if lon == 180 {
		zone = 60
	}
	// Norway exception.
	if lat >= 56 && lat < 64 && lon >= 3 && lon < 12 {
		zone = 32
	}
	// Svalbard exceptions.
	if lat >= 72 && lat < 84 {
		switch {
		case lon >= 0 && lon < 9:
			zone = 31
		case lon >= 9 && lon < 21:
			zone = 33
		case lon >= 21 && lon < 33:
			zone = 35
		case lon >= 33 && lon < 42:
			zone = 37
		}
	}
	return zone
}

func latitudeBand(lat float64) string {
	bands := "CDEFGHJKLMNPQRSTUVWXX"
	idx := int(math.Floor((lat + 80) / 8))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(bands) {
		idx = len(bands) - 1
	}
	return string(bands[idx])
}

func formatMGRS(zone int, band string, easting, northing float64) string {
	colSets := []string{"ABCDEFGH", "JKLMNPQR", "STUVWXYZ"}
	rows := "ABCDEFGHJKLMNPQRSTUV"
	colSet := colSets[(zone-1)%3]
	col := int(math.Floor(easting/100000)) - 1
	if col < 0 {
		col = 0
	}
	if col > 7 {
		col = 7
	}
	row := int(math.Floor(northing/100000)) % 20
	if zone%2 == 0 {
		row = (row + 5) % 20
	}
	eRem := int(math.Floor(easting)) % 100000
	nRem := int(math.Floor(northing)) % 100000
	return fmt.Sprintf("%d%s %c%c %05d %05d", zone, band, colSet[col], rows[row], eRem, nRem)
}
