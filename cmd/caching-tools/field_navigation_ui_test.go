package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFieldNavigationUIAssets(t *testing.T) {
	h,err:=newHandler(); if err!=nil{t.Fatal(err)}
	page:=httptest.NewRecorder(); h.ServeHTTP(page,httptest.NewRequest(http.MethodGet,"/",nil))
	if page.Code!=http.StatusOK{t.Fatalf("status=%d",page.Code)}
	for _,want:=range []string{`src="/field-navigation.js"`,`Caching Tools M1.22`} { if !strings.Contains(page.Body.String(),want){t.Fatalf("missing %q",want)} }
	asset:=httptest.NewRecorder(); h.ServeHTTP(asset,httptest.NewRequest(http.MethodGet,"/field-navigation.js",nil))
	if asset.Code!=http.StatusOK{t.Fatalf("asset status=%d",asset.Code)}
	for _,want:=range []string{"Field navigation","/api/navigation/field","/api/waypoints","/api/paths","cross_track_m","navigator.geolocation","caching-tools:map-select","Caching Tools M1.22"} { if !strings.Contains(asset.Body.String(),want){t.Fatalf("missing %q",want)} }
}
