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

type healthResponse struct { Status string `json:"status"`; Time string `json:"time"` }
type coordinateRequest struct { Latitude string `json:"latitude"`; Longitude string `json:"longitude"` }
type navigationRequest struct { From coordinateRequest `json:"from"`; To coordinateRequest `json:"to"` }
type projectionRequest struct { From coordinateRequest `json:"from"`; BearingDeg float64 `json:"bearing_deg"`; DistanceM float64 `json:"distance_m"` }
type errorResponse struct { Error string `json:"error"` }

func main() {
	addr := getenv("CACHING_TOOLS_ADDR", ":8080")
	handler, err := newHandler(); if err != nil { log.Fatal(err) }
	log.Printf("Caching Tools listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil { log.Fatal(err) }
}

func newHandler() (http.Handler, error) {
	staticFS, err := fs.Sub(webFS, "web"); if err != nil { return nil, err }
	dataDir := getenv("CACHING_TOOLS_DATA_DIR", "/data")
	waypoints := newWaypointStore(dataDir); paths := newPathStore(dataDir); workspaces := newMysteryWorkspaceStore(dataDir)
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, http.StatusOK, healthResponse{Status:"ok", Time:time.Now().UTC().Format(time.RFC3339)}) })
	mux.HandleFunc("POST /api/coordinates/convert", handleConvert)
	mux.HandleFunc("POST /api/coordinates/navigation", handleNavigation)
	mux.HandleFunc("POST /api/coordinates/project", handleProjection)
	mux.HandleFunc("POST /api/coordinates/grid", handleGrid)
	mux.HandleFunc("POST /api/coordinates/from-utm", handleFromUTM)
	mux.HandleFunc("POST /api/coordinates/crs", handleCRS)
	mux.HandleFunc("POST /api/coordinates/intersection/bearing-bearing", handleBearingIntersection)
	mux.HandleFunc("POST /api/coordinates/intersection/bearing-distance", handleBearingDistanceIntersection)
	mux.HandleFunc("POST /api/coordinates/intersection/circle-circle", handleCircleIntersection)
	mux.HandleFunc("POST /api/coordinates/final", handleFinalCoordinate)
	mux.HandleFunc("POST /api/puzzle", handlePuzzle)
	mux.HandleFunc("GET /api/mystery-workspaces", func(w http.ResponseWriter,_ *http.Request){ items,err:=workspaces.list(); if err!=nil{writeError(w,500,err);return}; writeJSON(w,200,items) })
	mux.HandleFunc("POST /api/mystery-workspaces", func(w http.ResponseWriter,r *http.Request){ var req mysteryWorkspaceRequest; if err:=decodeJSON(r,&req);err!=nil{writeError(w,400,err);return}; item,err:=workspaces.create(req); if err!=nil{writeError(w,400,err);return}; writeJSON(w,201,item) })
	mux.HandleFunc("GET /api/mystery-workspaces/{id}", func(w http.ResponseWriter,r *http.Request){ item,err:=workspaces.get(r.PathValue("id")); if errors.Is(err,os.ErrNotExist){writeError(w,404,errors.New("workspace not found"));return}; if err!=nil{writeError(w,500,err);return}; writeJSON(w,200,item) })
	mux.HandleFunc("PUT /api/mystery-workspaces/{id}", func(w http.ResponseWriter,r *http.Request){ var req mysteryWorkspaceRequest; if err:=decodeJSON(r,&req);err!=nil{writeError(w,400,err);return}; item,err:=workspaces.update(r.PathValue("id"),req); if errors.Is(err,os.ErrNotExist){writeError(w,404,errors.New("workspace not found"));return}; if err!=nil{writeError(w,400,err);return}; writeJSON(w,200,item) })
	mux.HandleFunc("DELETE /api/mystery-workspaces/{id}", func(w http.ResponseWriter,r *http.Request){ err:=workspaces.delete(r.PathValue("id")); if errors.Is(err,os.ErrNotExist){writeError(w,404,errors.New("workspace not found"));return}; if err!=nil{writeError(w,500,err);return}; w.WriteHeader(204) })
	registerMysteryBundleRoutes(mux, workspaces, waypoints)
	mux.HandleFunc("GET /api/waypoints", func(w http.ResponseWriter, _ *http.Request) { handleWaypointList(w, waypoints) })
	mux.HandleFunc("POST /api/waypoints", func(w http.ResponseWriter, r *http.Request) { handleWaypointCreate(w, r, waypoints) })
	mux.HandleFunc("PUT /api/waypoints/{id}", func(w http.ResponseWriter, r *http.Request) { handleWaypointUpdate(w, r, waypoints) })
	mux.HandleFunc("DELETE /api/waypoints/{id}", func(w http.ResponseWriter, r *http.Request) { handleWaypointDelete(w, r, waypoints) })
	mux.HandleFunc("POST /api/gpx/waypoints/import", func(w http.ResponseWriter, r *http.Request) { handleGPXImport(w, r, waypoints) })
	mux.HandleFunc("GET /api/gpx/waypoints/export", func(w http.ResponseWriter, _ *http.Request) { handleGPXExport(w, waypoints) })
	mux.HandleFunc("POST /api/gpx/inspect", handleGPXInspect)
	mux.HandleFunc("POST /api/gpx/paths/import", func(w http.ResponseWriter, r *http.Request) { handlePathImport(w, r, paths) })
	mux.HandleFunc("GET /api/paths", func(w http.ResponseWriter, _ *http.Request) { handlePathList(w, paths) })
	mux.HandleFunc("GET /api/paths/{id}", func(w http.ResponseWriter, r *http.Request) { handlePathGet(w, r, paths) })
	mux.HandleFunc("POST /api/paths/{id}/edit", func(w http.ResponseWriter, r *http.Request) { var req pathEditRequest; if err:=decodeJSON(r,&req);err!=nil{writeError(w,400,err);return}; item,err:=paths.edit(r.PathValue("id"),req); if errors.Is(err,os.ErrNotExist){writeError(w,404,errors.New("path not found"));return}; if err!=nil{writeError(w,400,err);return}; view,err:=pathView(item); if err!=nil{writeError(w,500,err);return}; writeJSON(w,200,view) })
	mux.HandleFunc("PUT /api/paths/{id}", func(w http.ResponseWriter, r *http.Request) { handlePathRename(w, r, paths) })
	mux.HandleFunc("GET /api/paths/{id}/export", func(w http.ResponseWriter, r *http.Request) { handlePathExport(w, r, paths) })
	mux.HandleFunc("DELETE /api/paths/{id}", func(w http.ResponseWriter, r *http.Request) { handlePathDelete(w, r, paths) })
	mux.Handle("/", http.FileServer(http.FS(staticFS)))
	return mux, nil
}

func handleConvert(w http.ResponseWriter, r *http.Request) { var req coordinateRequest; if err:=decodeJSON(r,&req); err!=nil { writeError(w,400,err); return }; lat,lon,err:=parsePoint(req); if err!=nil { writeError(w,400,err); return }; writeJSON(w,200,point(lat,lon)) }
func handleNavigation(w http.ResponseWriter, r *http.Request) { var req navigationRequest; if err:=decodeJSON(r,&req); err!=nil { writeError(w,400,err); return }; lat1,lon1,err:=parsePoint(req.From); if err!=nil { writeError(w,400,errors.New("from: "+err.Error())); return }; lat2,lon2,err:=parsePoint(req.To); if err!=nil { writeError(w,400,errors.New("to: "+err.Error())); return }; d,b:=distanceAndBearing(lat1,lon1,lat2,lon2); writeJSON(w,200,navigationResponse{From:point(lat1,lon1),To:point(lat2,lon2),DistanceM:d,DistanceKM:d/1000,InitialBearing:b}) }
func handleProjection(w http.ResponseWriter, r *http.Request) { var req projectionRequest; if err:=decodeJSON(r,&req); err!=nil { writeError(w,400,err); return }; lat,lon,err:=parsePoint(req.From); if err!=nil { writeError(w,400,errors.New("from: "+err.Error())); return }; if math.IsNaN(req.BearingDeg)||math.IsInf(req.BearingDeg,0) { writeError(w,400,errors.New("bearing must be finite")); return }; if req.DistanceM<0||math.IsNaN(req.DistanceM)||math.IsInf(req.DistanceM,0) { writeError(w,400,errors.New("distance must be a finite non-negative number")); return }; a,b:=destinationPoint(lat,lon,req.BearingDeg,req.DistanceM); writeJSON(w,200,projectionResponse{From:point(lat,lon),To:point(a,b),DistanceM:req.DistanceM,BearingDeg:math.Mod(req.BearingDeg+360,360)}) }
func handleGrid(w http.ResponseWriter, r *http.Request) { var req coordinateRequest; if err:=decodeJSON(r,&req); err!=nil { writeError(w,400,err); return }; lat,lon,err:=parsePoint(req); if err!=nil { writeError(w,400,err); return }; u,err:=latLonToUTM(lat,lon); if err!=nil { writeError(w,400,err); return }; writeJSON(w,200,gridResponse{WGS84:point(lat,lon),UTM:u}) }
func handleFromUTM(w http.ResponseWriter, r *http.Request) { var req utmRequest; if err:=decodeJSON(r,&req); err!=nil { writeError(w,400,err); return }; h:=strings.ToUpper(strings.TrimSpace(req.Hemisphere)); lat,lon,err:=utmToLatLon(req.Zone,h,req.Easting,req.Northing); if err!=nil { writeError(w,400,err); return }; u,err:=latLonToUTM(lat,lon); if err!=nil { writeError(w,400,err); return }; writeJSON(w,200,gridResponse{WGS84:point(lat,lon),UTM:u}) }
func handleCRS(w http.ResponseWriter, r *http.Request) { var req crsRequest; if err:=decodeJSON(r,&req); err!=nil { writeError(w,400,err); return }; result,err:=convertCRS(req); if err!=nil { writeError(w,400,err); return }; writeJSON(w,200,result) }
func handleFinalCoordinate(w http.ResponseWriter, r *http.Request) { var req finalCoordinateRequest; if err:=decodeJSON(r,&req); err!=nil { writeError(w,400,err); return }; result,err:=solveFinalCoordinate(req); if err!=nil { writeError(w,400,err); return }; writeJSON(w,200,result) }
func handleWaypointList(w http.ResponseWriter, store *waypointStore) { items,err:=store.list(); if err!=nil { writeError(w,500,err); return }; views:=make([]waypointResponse,0,len(items)); for _,item:=range items { views=append(views,waypointView(item)) }; writeJSON(w,200,views) }
func handleWaypointCreate(w http.ResponseWriter, r *http.Request, store *waypointStore) { var req waypointRequest; if err:=decodeJSON(r,&req); err!=nil { writeError(w,400,err); return }; item,err:=store.create(req); if err!=nil { writeError(w,400,err); return }; writeJSON(w,201,waypointView(item)) }
func handleWaypointUpdate(w http.ResponseWriter, r *http.Request, store *waypointStore) { var req waypointRequest; if err:=decodeJSON(r,&req); err!=nil { writeError(w,400,err); return }; item,err:=store.update(r.PathValue("id"),req); if errors.Is(err,os.ErrNotExist) { writeError(w,404,errors.New("waypoint not found")); return }; if err!=nil { writeError(w,400,err); return }; writeJSON(w,200,waypointView(item)) }
func handleWaypointDelete(w http.ResponseWriter, r *http.Request, store *waypointStore) { if err:=store.delete(r.PathValue("id")); errors.Is(err,os.ErrNotExist) { writeError(w,404,errors.New("waypoint not found")); return } else if err!=nil { writeError(w,500,err); return }; w.WriteHeader(204) }
func handleGPXImport(w http.ResponseWriter, r *http.Request, store *waypointStore) { defer r.Body.Close(); requests,err:=parseGPXWaypoints(r.Body); if err!=nil { writeError(w,400,err); return }; items,err:=store.importMany(requests); if err!=nil { writeError(w,500,err); return }; views:=make([]waypointResponse,0,len(items)); for _,item:=range items { views=append(views,waypointView(item)) }; writeJSON(w,201,gpxImportResult{Imported:len(views),Waypoints:views}) }
func handleGPXExport(w http.ResponseWriter, store *waypointStore) { items,err:=store.list(); if err!=nil { writeError(w,500,err); return }; data,err:=encodeGPXWaypoints(items); if err!=nil { writeError(w,500,err); return }; w.Header().Set("Content-Type","application/gpx+xml; charset=utf-8"); w.Header().Set("Content-Disposition",`attachment; filename="caching-tools-waypoints.gpx"`); w.WriteHeader(200); _,_=w.Write(data) }
func handleGPXInspect(w http.ResponseWriter, r *http.Request) { defer r.Body.Close(); result,err:=inspectGPX(r.Body); if err!=nil { writeError(w,400,err); return }; writeJSON(w,200,result) }
func parsePoint(req coordinateRequest) (float64,float64,error) { lat,err:=parseCoordinate(req.Latitude,true); if err!=nil { return 0,0,errors.New("latitude: "+err.Error()) }; lon,err:=parseCoordinate(req.Longitude,false); if err!=nil { return 0,0,errors.New("longitude: "+err.Error()) }; return lat,lon,nil }
func decodeJSON(r *http.Request,dst any) error { defer r.Body.Close(); dec:=json.NewDecoder(r.Body); dec.DisallowUnknownFields(); if err:=dec.Decode(dst); err!=nil { return errors.New("invalid JSON: "+err.Error()) }; return nil }
func writeJSON(w http.ResponseWriter,status int,value any) { w.Header().Set("Content-Type","application/json"); w.WriteHeader(status); _=json.NewEncoder(w).Encode(value) }
func writeError(w http.ResponseWriter,status int,err error) { writeJSON(w,status,errorResponse{Error:err.Error()}) }
func getenv(key,fallback string) string { if value:=os.Getenv(key); value!="" { return value }; return fallback }
