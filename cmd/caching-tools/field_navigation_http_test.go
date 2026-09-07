package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFieldNavigationAPI(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CACHING_TOOLS_DATA_DIR", dir)
	waypoints := newWaypointStore(dir)
	wp, err := waypoints.create(waypointRequest{Name:"Target",Latitude:"60",Longitude:"10.1"})
	if err != nil { t.Fatal(err) }
	paths := []storedPath{{ID:"path-api",Kind:"route",Name:"Route",Route:[]gpxPoint{{Latitude:60,Longitude:10},{Latitude:60,Longitude:11}},CreatedAt:"2026-09-07T00:00:00Z"}}
	data, _ := json.Marshal(paths)
	if err := os.WriteFile(filepath.Join(dir,"paths.json"),data,0o600); err != nil { t.Fatal(err) }

	h, err := newHandler(); if err != nil { t.Fatal(err) }
	body := `{"from":{"latitude":"60","longitude":"10"},"waypoint_id":"`+wp.ID+`"}`
	rr := httptest.NewRecorder(); h.ServeHTTP(rr,httptest.NewRequest(http.MethodPost,"/api/navigation/field",strings.NewReader(body)))
	if rr.Code != http.StatusOK { t.Fatalf("status=%d body=%s",rr.Code,rr.Body.String()) }
	if !strings.Contains(rr.Body.String(),`"kind":"waypoint"`) || !strings.Contains(rr.Body.String(),wp.ID) { t.Fatalf("body=%s",rr.Body.String()) }

	body = `{"from":{"latitude":"60.1","longitude":"10.5"},"path_id":"path-api"}`
	rr = httptest.NewRecorder(); h.ServeHTTP(rr,httptest.NewRequest(http.MethodPost,"/api/navigation/field",strings.NewReader(body)))
	if rr.Code != http.StatusOK { t.Fatalf("status=%d body=%s",rr.Code,rr.Body.String()) }
	for _, want := range []string{`"kind":"route"`,`"cross_track_m"`,`"nearest_path"`} { if !strings.Contains(rr.Body.String(),want) { t.Fatalf("missing %s in %s",want,rr.Body.String()) } }
}

func TestFieldNavigationAPIBadTarget(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR",t.TempDir())
	h,err:=newHandler(); if err!=nil{t.Fatal(err)}
	rr:=httptest.NewRecorder(); h.ServeHTTP(rr,httptest.NewRequest(http.MethodPost,"/api/navigation/field",strings.NewReader(`{"from":{"latitude":"60","longitude":"10"}}`)))
	if rr.Code!=http.StatusBadRequest { t.Fatalf("status=%d body=%s",rr.Code,rr.Body.String()) }
}
