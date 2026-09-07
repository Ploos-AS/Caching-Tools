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
	if !strings.Contains(page.Body.String(),`src="/field-navigation.js"`) { t.Fatal("missing field navigation asset reference") }
	asset:=httptest.NewRecorder(); h.ServeHTTP(asset,httptest.NewRequest(http.MethodGet,"/field-navigation.js",nil))
	if asset.Code!=http.StatusOK{t.Fatalf("asset status=%d",asset.Code)}
	for _,want:=range []string{
		"Field navigation","/api/navigation/field","/api/waypoints","/api/paths","cross_track_m","progress_percent","remaining_m","next_point_distance_m","forward_bearing_deg",
		"off-route-threshold","arrival-radius","guidance.status","off_route_threshold_m","arrival_radius_m","navigator.geolocation","caching-tools:map-select",
		"Start live navigation","Stop live navigation","watchPosition","clearWatch","processLivePosition","livePendingPosition","liveRequestInFlight","Session breadcrumbs","field-breadcrumb-clear","liveBreadcrumbs","never written to <code>/data</code>","pagehide","Caching Tools M1.25",
	} { if !strings.Contains(asset.Body.String(),want){t.Fatalf("missing %q",want)} }
}
