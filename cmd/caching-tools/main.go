package main

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed web/*
var webFS embed.FS

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

type coordinateRequest struct {
	Latitude  string `json:"latitude"`
	Longitude string `json:"longitude"`
}

type navigationRequest struct {
	From coordinateRequest `json:"from"`
	To   coordinateRequest `json:"to"`
}

type projectionRequest struct {
	From       coordinateRequest `json:"from"`
	BearingDeg float64           `json:"bearing_deg"`
	DistanceM  float64           `json:"distance_m"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func main() {
	addr := getenv("CACHING_TOOLS_ADDR", ":8080")
	handler, err := newHandler()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Caching Tools listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal(err)
	}
}

func newHandler() (http.Handler, error) {
	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		return nil, err
	}
	waypoints := newWaypointStore(getenv("CACHING_TOOLS_DATA_DIR", "/data"))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{Status: "ok", Time: time.Now().UTC().Format(time.RFC3339)})
	})
	mux.HandleFunc("POST /api/coordinates/convert", handleConvert)
	mux.HandleFunc("POST /api/coordinates/navigation", handleNavigation)
	mux.HandleFunc("POST /api/coordinates/project", handleProjection)
	mux.HandleFunc("POST /api/coordinates/grid", handleGrid)
	mux.HandleFunc("POST /api/coordinates/from-utm", handleFromUTM)
	mux.HandleFunc("GET /api/waypoints", func(w http.ResponseWriter, _ *http.Request) { handleWaypointList(w, waypoints) })
	mux.HandleFunc("POST /api/waypoints", func(w http.ResponseWriter, r *http.Request) { handleWaypointCreate(w, r, waypoints) })
	mux.HandleFunc("PUT /api/waypoints/{id}", func(w http.ResponseWriter, r *http.Request) { handleWaypointUpdate(w, r, waypoints) })
	mux.HandleFunc("DELETE /api/waypoints/{id}", func(w http.ResponseWriter, r *http.Request) { handleWaypointDelete(w, r, waypoints) })
	mux.Handle("/", http.FileServer(http.FS(staticFS)))
	return mux, nil
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	var req coordinateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	lat, lon, err := parsePoint(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, point(lat, lon))
}

func handleNavigation(w http.ResponseWriter, r *http.Request) {
	var req navigationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	lat1, lon1, err := parsePoint(req.From)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("from: "+err.Error()))
		return
	}
	lat2, lon2, err := parsePoint(req.To)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("to: "+err.Error()))
		return
	}
	distance, bearing := distanceAndBearing(lat1, lon1, lat2, lon2)
	writeJSON(w, http.StatusOK, navigationResponse{
		From: point(lat1, lon1), To: point(lat2, lon2),
		DistanceM: distance, DistanceKM: distance / 1000, InitialBearing: bearing,
	})
}

func handleProjection(w http.ResponseWriter, r *http.Request) {
	var req projectionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	lat, lon, err := parsePoint(req.From)
	if err != nil {
		writeError(w, http.StatusBadRequest, errors.New("from: "+err.Error()))
		return
	}
	if math.IsNaN(req.BearingDeg) || math.IsInf(req.BearingDeg, 0) {
		writeError(w, http.StatusBadRequest, errors.New("bearing must be finite"))
		return
	}
	if req.DistanceM < 0 || math.IsNaN(req.DistanceM) || math.IsInf(req.DistanceM, 0) {
		writeError(w, http.StatusBadRequest, errors.New("distance must be a finite non-negative number"))
		return
	}
	toLat, toLon := destinationPoint(lat, lon, req.BearingDeg, req.DistanceM)
	writeJSON(w, http.StatusOK, projectionResponse{
		From: point(lat, lon), To: point(toLat, toLon), DistanceM: req.DistanceM,
		BearingDeg: math.Mod(req.BearingDeg+360, 360),
	})
}

func handleGrid(w http.ResponseWriter, r *http.Request) {
	var req coordinateRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	lat, lon, err := parsePoint(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	utm, err := latLonToUTM(lat, lon)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, gridResponse{WGS84: point(lat, lon), UTM: utm})
}

func handleFromUTM(w http.ResponseWriter, r *http.Request) {
	var req utmRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	hemisphere := strings.ToUpper(strings.TrimSpace(req.Hemisphere))
	lat, lon, err := utmToLatLon(req.Zone, hemisphere, req.Easting, req.Northing)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	utm, err := latLonToUTM(lat, lon)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, gridResponse{WGS84: point(lat, lon), UTM: utm})
}

func handleWaypointList(w http.ResponseWriter, store *waypointStore) {
	items, err := store.list()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	views := make([]waypointResponse, 0, len(items))
	for _, item := range items {
		views = append(views, waypointView(item))
	}
	writeJSON(w, http.StatusOK, views)
}

func handleWaypointCreate(w http.ResponseWriter, r *http.Request, store *waypointStore) {
	var req waypointRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := store.create(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, waypointView(item))
}

func handleWaypointUpdate(w http.ResponseWriter, r *http.Request, store *waypointStore) {
	var req waypointRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := store.update(r.PathValue("id"), req)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, errors.New("waypoint not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, waypointView(item))
}

func handleWaypointDelete(w http.ResponseWriter, r *http.Request, store *waypointStore) {
	if err := store.delete(r.PathValue("id")); errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, errors.New("waypoint not found"))
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func parsePoint(req coordinateRequest) (float64, float64, error) {
	lat, err := parseCoordinate(req.Latitude, true)
	if err != nil {
		return 0, 0, errors.New("latitude: "+err.Error())
	}
	lon, err := parseCoordinate(req.Longitude, false)
	if err != nil {
		return 0, 0, errors.New("longitude: "+err.Error())
	}
	return lat, lon, nil
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return errors.New("invalid JSON: "+err.Error())
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errorResponse{Error: err.Error()})
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
