package main

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

const earthRadiusMeters = 6371008.8

var numberRE = regexp.MustCompile(`[+-]?(?:\d+(?:\.\d*)?|\.\d+)`)

type coordinateFormats struct {
	DD  string `json:"dd"`
	DMM string `json:"dmm"`
	DMS string `json:"dms"`
}

type pointResponse struct {
	Latitude  float64           `json:"latitude"`
	Longitude float64           `json:"longitude"`
	Lat       coordinateFormats `json:"lat"`
	Lon       coordinateFormats `json:"lon"`
}

type navigationResponse struct {
	From           pointResponse `json:"from"`
	To             pointResponse `json:"to"`
	DistanceM      float64       `json:"distance_m"`
	DistanceKM     float64       `json:"distance_km"`
	InitialBearing float64       `json:"initial_bearing_deg"`
}

type projectionResponse struct {
	From       pointResponse `json:"from"`
	To         pointResponse `json:"to"`
	DistanceM  float64       `json:"distance_m"`
	BearingDeg float64       `json:"bearing_deg"`
}

func parseCoordinate(input string, latitude bool) (float64, error) {
	s := strings.TrimSpace(strings.ToUpper(input))
	if s == "" {
		return 0, errors.New("coordinate is empty")
	}

	hemi := byte(0)
	for _, h := range []byte{'N', 'S', 'E', 'W'} {
		if strings.ContainsRune(s, rune(h)) {
			if hemi != 0 {
				return 0, errors.New("coordinate contains multiple hemisphere letters")
			}
			hemi = h
		}
	}
	if latitude && (hemi == 'E' || hemi == 'W') {
		return 0, errors.New("latitude must use N/S hemisphere")
	}
	if !latitude && (hemi == 'N' || hemi == 'S') {
		return 0, errors.New("longitude must use E/W hemisphere")
	}

	raw := numberRE.FindAllString(s, -1)
	if len(raw) < 1 || len(raw) > 3 {
		return 0, errors.New("coordinate must contain degrees, optionally minutes and seconds")
	}
	vals := make([]float64, len(raw))
	for i, token := range raw {
		v, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid number %q", token)
		}
		vals[i] = v
	}

	if len(vals) > 1 && vals[0] < 0 {
		return 0, errors.New("use hemisphere or a negative decimal degree value, not both with minutes/seconds")
	}
	if len(vals) > 1 && (vals[1] < 0 || vals[1] >= 60) {
		return 0, errors.New("minutes must be in [0,60)")
	}
	if len(vals) > 2 && (vals[2] < 0 || vals[2] >= 60) {
		return 0, errors.New("seconds must be in [0,60)")
	}

	sign := 1.0
	deg := vals[0]
	if deg < 0 {
		sign = -1
		deg = -deg
	}
	if hemi == 'S' || hemi == 'W' {
		if sign < 0 {
			return 0, errors.New("negative sign conflicts with hemisphere")
		}
		sign = -1
	}
	if hemi == 'N' || hemi == 'E' {
		if sign < 0 {
			return 0, errors.New("negative sign conflicts with hemisphere")
		}
	}

	value := deg
	if len(vals) > 1 {
		value += vals[1] / 60
	}
	if len(vals) > 2 {
		value += vals[2] / 3600
	}
	value *= sign

	limit := 180.0
	kind := "longitude"
	if latitude {
		limit = 90
		kind = "latitude"
	}
	if value < -limit || value > limit {
		return 0, fmt.Errorf("%s out of range", kind)
	}
	return value, nil
}

func formatCoordinate(value float64, latitude bool) coordinateFormats {
	positive, negative := "E", "W"
	width := 3
	if latitude {
		positive, negative = "N", "S"
		width = 2
	}
	hemi := positive
	if value < 0 {
		hemi = negative
	}
	abs := math.Abs(value)
	deg := int(math.Floor(abs))
	minutesFull := (abs - float64(deg)) * 60
	min := int(math.Floor(minutesFull))
	sec := (minutesFull - float64(min)) * 60

	return coordinateFormats{
		DD:  fmt.Sprintf("%s %.*f", hemi, 6, abs),
		DMM: fmt.Sprintf("%s %0*d° %.3f'", hemi, width, deg, minutesFull),
		DMS: fmt.Sprintf("%s %0*d° %02d' %.2f\"", hemi, width, deg, min, sec),
	}
}

func point(lat, lon float64) pointResponse {
	return pointResponse{Latitude: lat, Longitude: lon, Lat: formatCoordinate(lat, true), Lon: formatCoordinate(lon, false)}
}

func distanceAndBearing(lat1, lon1, lat2, lon2 float64) (float64, float64) {
	phi1, phi2 := radians(lat1), radians(lat2)
	dPhi := radians(lat2 - lat1)
	dLambda := radians(lon2 - lon1)

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) + math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	distance := earthRadiusMeters * c

	y := math.Sin(dLambda) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(dLambda)
	bearing := math.Mod(degrees(math.Atan2(y, x))+360, 360)
	return distance, bearing
}

func destinationPoint(lat, lon, bearingDeg, distanceM float64) (float64, float64) {
	phi1 := radians(lat)
	lambda1 := radians(lon)
	theta := radians(math.Mod(bearingDeg+360, 360))
	delta := distanceM / earthRadiusMeters

	phi2 := math.Asin(math.Sin(phi1)*math.Cos(delta) + math.Cos(phi1)*math.Sin(delta)*math.Cos(theta))
	lambda2 := lambda1 + math.Atan2(
		math.Sin(theta)*math.Sin(delta)*math.Cos(phi1),
		math.Cos(delta)-math.Sin(phi1)*math.Sin(phi2),
	)
	lambda2 = math.Mod(lambda2+3*math.Pi, 2*math.Pi) - math.Pi
	return degrees(phi2), degrees(lambda2)
}

func radians(v float64) float64 { return v * math.Pi / 180 }
func degrees(v float64) float64 { return v * 180 / math.Pi }
