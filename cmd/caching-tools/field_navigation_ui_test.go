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
		"Start live navigation","Stop live navigation","watchPosition","clearWatch","processLivePosition","livePendingPosition","liveRequestInFlight","Session breadcrumbs","field-breadcrumb-clear","liveBreadcrumbs","never written to <code>/data</code>","pagehide",
		"Export session GPX","field-breadcrumb-export","breadcrumbGPX","exportBreadcrumbGPX","escapeXML","<gpx version=\"1.1\"","<trkseg>","<trkpt lat=","application/gpx+xml","new Blob","URL.createObjectURL","URL.revokeObjectURL","caching-tools-session-","No breadcrumb points to export",
		"Session statistics","field-session-stats","sessionStatistics","breadcrumbDistanceMeters","renderSessionStatistics","Average speed","Maximum accepted segment speed","Speed samples above 360 km/h","Caching Tools M1.27",
	} { if !strings.Contains(asset.Body.String(),want){t.Fatalf("missing %q",want)} }

	quality:=httptest.NewRecorder(); h.ServeHTTP(quality,httptest.NewRequest(http.MethodGet,"/field-quality.js",nil))
	if quality.Code!=http.StatusOK{t.Fatalf("quality asset status=%d",quality.Code)}
	for _,want:=range []string{
		"Minimum breadcrumb distance","Maximum GPS accuracy","Simplify GPX export","field-min-distance","field-max-accuracy","field-export-simplify","field-simplify-tolerance",
		"addBreadcrumbFiltered","qualityRejectedAccuracy","qualityRejectedDistance","simplifyBreadcrumbs","pointSegmentDistanceMeters","breadcrumbGPXWithQuality","GPX simplification","Caching Tools M1.28",
	} { if !strings.Contains(quality.Body.String(),want){t.Fatalf("missing %q from field-quality.js",want)} }

	link:=httptest.NewRecorder(); h.ServeHTTP(link,httptest.NewRequest(http.MethodGet,"/map-link.js",nil))
	if link.Code!=http.StatusOK{t.Fatalf("map-link status=%d",link.Code)}
	for _,want:=range []string{"/field-quality.js","data-field-quality","/map-editor.js"} { if !strings.Contains(link.Body.String(),want){t.Fatalf("missing %q from map-link.js",want)} }
}
