package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConvertAPI(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/convert", strings.NewReader(`{"latitude":"N 58 7.404","longitude":"E 8 0 0"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"latitude":58.1234`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestNavigationAPIRejectsInvalidCoordinate(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/navigation", strings.NewReader(`{"from":{"latitude":"N 99","longitude":"E 10"},"to":{"latitude":"N 60","longitude":"E 5"}}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestProjectionAPI(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/project", strings.NewReader(`{"from":{"latitude":"59.9139","longitude":"10.7522"},"bearing_deg":42,"distance_m":12345}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"distance_m":12345`) || !strings.Contains(rec.Body.String(), `"bearing_deg":42`) {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestProjectionAPIRejectsNegativeDistance(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/project", strings.NewReader(`{"from":{"latitude":"59.9","longitude":"10.7"},"bearing_deg":42,"distance_m":-1}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestGridAPI(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/grid", strings.NewReader(`{"latitude":"59.9139","longitude":"10.7522"}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"zone":32`) || !strings.Contains(body, `"mgrs":"32V NM 97979 43118"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestFromUTMAPI(t *testing.T) {
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/coordinates/from-utm", strings.NewReader(`{"zone":32,"hemisphere":"n","easting":597979.903,"northing":6643118.991}`))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"dd":"N 59.913900"`) || !strings.Contains(body, `"dd":"E 10.752200"`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestWaypointCRUDAPI(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler()
	if err != nil {
		t.Fatal(err)
	}

	create := httptest.NewRequest(http.MethodPost, "/api/waypoints", strings.NewReader(`{"name":"Trailhead","latitude":"N 59 54.834","longitude":"E 10 45.132","type":"parking","comment":"Start here"}`))
	created := httptest.NewRecorder()
	h.ServeHTTP(created, create)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	var item waypointResponse
	if err := json.Unmarshal(created.Body.Bytes(), &item); err != nil {
		t.Fatal(err)
	}
	if item.ID == "" || item.Name != "Trailhead" || item.Type != "parking" {
		t.Fatalf("unexpected created waypoint: %+v", item)
	}

	list := httptest.NewRecorder()
	h.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/waypoints", nil))
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), `"name":"Trailhead"`) {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}

	updateBody := `{"name":"Trailhead updated","latitude":"59.9139","longitude":"10.7522","type":"trailhead","comment":"Updated"}`
	updated := httptest.NewRecorder()
	h.ServeHTTP(updated, httptest.NewRequest(http.MethodPut, "/api/waypoints/"+item.ID, strings.NewReader(updateBody)))
	if updated.Code != http.StatusOK || !strings.Contains(updated.Body.String(), `"name":"Trailhead updated"`) {
		t.Fatalf("update status=%d body=%s", updated.Code, updated.Body.String())
	}

	deleted := httptest.NewRecorder()
	h.ServeHTTP(deleted, httptest.NewRequest(http.MethodDelete, "/api/waypoints/"+item.ID, nil))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}

	empty := httptest.NewRecorder()
	h.ServeHTTP(empty, httptest.NewRequest(http.MethodGet, "/api/waypoints", nil))
	if empty.Code != http.StatusOK || strings.TrimSpace(empty.Body.String()) != "[]" {
		t.Fatalf("post-delete list status=%d body=%s", empty.Code, empty.Body.String())
	}
}
