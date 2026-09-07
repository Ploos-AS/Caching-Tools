package main

import (
	"errors"
	"net/http"
)

func handleBearingIntersection(w http.ResponseWriter, r *http.Request) {
	var req bearingIntersectionRequest
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, err); return }
	lat1, lon1, err := parsePoint(req.A)
	if err != nil { writeError(w, http.StatusBadRequest, errors.New("a: "+err.Error())); return }
	lat2, lon2, err := parsePoint(req.B)
	if err != nil { writeError(w, http.StatusBadRequest, errors.New("b: "+err.Error())); return }
	if !finite(req.BearingA) || !finite(req.BearingB) { writeError(w, http.StatusBadRequest, errors.New("bearings must be finite")); return }
	lat, lon, err := bearingBearingIntersection(lat1, lon1, req.BearingA, lat2, lon2, req.BearingB)
	if err != nil { writeError(w, http.StatusBadRequest, err); return }
	writeJSON(w, http.StatusOK, intersectionResponse{Points: []pointResponse{point(lat, lon)}})
}

func handleBearingDistanceIntersection(w http.ResponseWriter, r *http.Request) {
	var req bearingDistanceIntersectionRequest
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, err); return }
	lat, lon, err := parsePoint(req.From)
	if err != nil { writeError(w, http.StatusBadRequest, errors.New("from: "+err.Error())); return }
	centerLat, centerLon, err := parsePoint(req.Center)
	if err != nil { writeError(w, http.StatusBadRequest, errors.New("center: "+err.Error())); return }
	if !finite(req.BearingDeg) { writeError(w, http.StatusBadRequest, errors.New("bearing must be finite")); return }
	matches, err := bearingDistanceIntersections(lat, lon, req.BearingDeg, centerLat, centerLon, req.DistanceM)
	if err != nil { writeError(w, http.StatusBadRequest, err); return }
	views := make([]pointResponse, 0, len(matches))
	for _, p := range matches { views = append(views, point(p[0], p[1])) }
	writeJSON(w, http.StatusOK, intersectionResponse{Points: views})
}

func handleCircleIntersection(w http.ResponseWriter, r *http.Request) {
	var req circleIntersectionRequest
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, err); return }
	lat1, lon1, err := parsePoint(req.A)
	if err != nil { writeError(w, http.StatusBadRequest, errors.New("a: "+err.Error())); return }
	lat2, lon2, err := parsePoint(req.B)
	if err != nil { writeError(w, http.StatusBadRequest, errors.New("b: "+err.Error())); return }
	matches, err := circleCircleIntersections(lat1, lon1, req.RadiusA, lat2, lon2, req.RadiusB)
	if err != nil { writeError(w, http.StatusBadRequest, err); return }
	views := make([]pointResponse, 0, len(matches))
	for _, p := range matches { views = append(views, point(p[0], p[1])) }
	writeJSON(w, http.StatusOK, intersectionResponse{Points: views})
}
