package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestPathEditAPI(t *testing.T) {
	dir:=t.TempDir(); old:=os.Getenv("CACHING_TOOLS_DATA_DIR"); _=os.Setenv("CACHING_TOOLS_DATA_DIR",dir); defer os.Setenv("CACHING_TOOLS_DATA_DIR",old)
	s:=newPathStore(dir); items,err:=s.importGPX(gpxDocument{Routes:[]gpxRoute{{Name:"r",Points:[]gpxPoint{{Latitude:1,Longitude:1},{Latitude:2,Longitude:2}}}}});if err!=nil{t.Fatal(err)}
	h,err:=newHandler();if err!=nil{t.Fatal(err)}
	rec:=httptest.NewRecorder(); req:=httptest.NewRequest(http.MethodPost,"/api/paths/"+items[0].ID+"/edit",strings.NewReader(`{"operation":"move-point","point":0,"target":1}`)); req.Header.Set("Content-Type","application/json"); h.ServeHTTP(rec,req)
	if rec.Code!=http.StatusOK{t.Fatalf("status=%d body=%s",rec.Code,rec.Body.String())}
	if !strings.Contains(rec.Body.String(),`"points":2`){t.Fatalf("body=%s",rec.Body.String())}
}
