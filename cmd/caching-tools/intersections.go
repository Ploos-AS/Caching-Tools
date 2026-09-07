package main

import (
	"errors"
	"math"
)

type bearingIntersectionRequest struct {
	A       coordinateRequest `json:"a"`
	BearingA float64           `json:"bearing_a_deg"`
	B       coordinateRequest `json:"b"`
	BearingB float64           `json:"bearing_b_deg"`
}

type bearingDistanceIntersectionRequest struct {
	From       coordinateRequest `json:"from"`
	BearingDeg float64           `json:"bearing_deg"`
	Center     coordinateRequest `json:"center"`
	DistanceM  float64           `json:"distance_m"`
}

type circleIntersectionRequest struct {
	A       coordinateRequest `json:"a"`
	RadiusA float64           `json:"radius_a_m"`
	B       coordinateRequest `json:"b"`
	RadiusB float64           `json:"radius_b_m"`
}

type intersectionResponse struct {
	Points []pointResponse `json:"points"`
}

func bearingBearingIntersection(lat1, lon1, bearing1, lat2, lon2, bearing2 float64) (float64, float64, error) {
	phi1, phi2 := radians(lat1), radians(lat2)
	lambda1, lambda2 := radians(lon1), radians(lon2)
	theta13, theta23 := radians(normBearing(bearing1)), radians(normBearing(bearing2))
	dPhi := phi2 - phi1
	dLambda := lambda2 - lambda1

	delta12 := 2 * math.Asin(math.Sqrt(math.Sin(dPhi/2)*math.Sin(dPhi/2)+math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)))
	if delta12 < 1e-12 {
		return 0, 0, errors.New("bearing origins must be different")
	}

	thetaA := math.Acos(clamp((math.Sin(phi2)-math.Sin(phi1)*math.Cos(delta12))/(math.Sin(delta12)*math.Cos(phi1)), -1, 1))
	thetaB := math.Acos(clamp((math.Sin(phi1)-math.Sin(phi2)*math.Cos(delta12))/(math.Sin(delta12)*math.Cos(phi2)), -1, 1))
	var theta12, theta21 float64
	if math.Sin(lambda2-lambda1) > 0 {
		theta12, theta21 = thetaA, 2*math.Pi-thetaB
	} else {
		theta12, theta21 = 2*math.Pi-thetaA, thetaB
	}

	alpha1 := signedAngle(theta13 - theta12)
	alpha2 := signedAngle(theta21 - theta23)
	if math.Abs(math.Sin(alpha1)) < 1e-12 && math.Abs(math.Sin(alpha2)) < 1e-12 {
		return 0, 0, errors.New("bearings are collinear")
	}
	if math.Sin(alpha1)*math.Sin(alpha2) < 0 {
		return 0, 0, errors.New("bearing rays diverge")
	}
	alpha3 := math.Acos(clamp(-math.Cos(alpha1)*math.Cos(alpha2)+math.Sin(alpha1)*math.Sin(alpha2)*math.Cos(delta12), -1, 1))
	delta13 := math.Atan2(math.Sin(delta12)*math.Sin(alpha1)*math.Sin(alpha2), math.Cos(alpha2)+math.Cos(alpha1)*math.Cos(alpha3))
	if delta13 < 0 {
		return 0, 0, errors.New("intersection lies behind a bearing origin")
	}
	lat, lon := destinationPoint(lat1, lon1, bearing1, delta13*earthRadiusMeters)
	return lat, lon, nil
}

func bearingDistanceIntersections(lat, lon, bearing, centerLat, centerLon, radiusM float64) ([][2]float64, error) {
	if radiusM < 0 || !finite(radiusM) {
		return nil, errors.New("distance must be finite and non-negative")
	}
	d, centerBearing := distanceAndBearing(lat, lon, centerLat, centerLon)
	delta := d / earthRadiusMeters
	rho := radiusM / earthRadiusMeters
	beta := radians(normBearing(bearing) - centerBearing)
	a := math.Cos(delta)
	b := math.Sin(delta) * math.Cos(beta)
	c := math.Cos(rho)
	r := math.Hypot(a, b)
	if r < 1e-15 || math.Abs(c) > r+1e-12 {
		return nil, errors.New("bearing ray does not intersect distance circle")
	}
	phi := math.Atan2(b, a)
	gamma := math.Acos(clamp(c/r, -1, 1))
	candidates := []float64{phi - gamma, phi + gamma}
	out := make([][2]float64, 0, 2)
	for _, x := range candidates {
		for x < 0 { x += 2 * math.Pi }
		if x > math.Pi { continue }
		pLat, pLon := destinationPoint(lat, lon, bearing, x*earthRadiusMeters)
		if !nearPoint(out, pLat, pLon) { out = append(out, [2]float64{pLat, pLon}) }
	}
	if len(out) == 0 { return nil, errors.New("intersections lie behind bearing origin") }
	return out, nil
}

func circleCircleIntersections(lat1, lon1, r1, lat2, lon2, r2 float64) ([][2]float64, error) {
	if r1 < 0 || r2 < 0 || !finite(r1) || !finite(r2) {
		return nil, errors.New("radii must be finite and non-negative")
	}
	d, bearing12 := distanceAndBearing(lat1, lon1, lat2, lon2)
	delta := d / earthRadiusMeters
	rho1, rho2 := r1/earthRadiusMeters, r2/earthRadiusMeters
	if delta < 1e-12 {
		if math.Abs(rho1-rho2) < 1e-12 { return nil, errors.New("coincident circles have infinitely many intersections") }
		return nil, errors.New("concentric circles do not intersect")
	}
	denom := math.Sin(rho1) * math.Sin(delta)
	if math.Abs(denom) < 1e-15 { return nil, errors.New("circle geometry is degenerate") }
	cosAlpha := (math.Cos(rho2) - math.Cos(rho1)*math.Cos(delta)) / denom
	if cosAlpha < -1-1e-12 || cosAlpha > 1+1e-12 { return nil, errors.New("circles do not intersect") }
	alpha := math.Acos(clamp(cosAlpha, -1, 1))
	p1Lat, p1Lon := destinationPoint(lat1, lon1, bearing12+degrees(alpha), r1)
	out := [][2]float64{{p1Lat, p1Lon}}
	if alpha > 1e-10 {
		p2Lat, p2Lon := destinationPoint(lat1, lon1, bearing12-degrees(alpha), r1)
		out = append(out, [2]float64{p2Lat, p2Lon})
	}
	return out, nil
}

func normBearing(v float64) float64 { return math.Mod(v+360, 360) }
func signedAngle(v float64) float64 { return math.Atan2(math.Sin(v), math.Cos(v)) }
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
func nearPoint(points [][2]float64, lat, lon float64) bool {
	for _, p := range points { if math.Abs(p[0]-lat) < 1e-9 && math.Abs(p[1]-lon) < 1e-9 { return true } }
	return false
}
