package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFinalCoordinateAPI(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	rec := httptest.NewRecorder()
	body := `{"variables":{"A":1,"B":2,"C":4,"D":5,"E":6},"latitude_formula":"N 59 54.[A+B][C][D]","longitude_formula":"E 010 45.[E][B][A]"}`
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/coordinates/final", strings.NewReader(body)))
	if rec.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
	for _, want := range []string{`"latitude_expanded":"N 59 54.345"`, `"longitude_expanded":"E 010 45.621"`, `"latitude":59.90575`} {
		if !strings.Contains(rec.Body.String(), want) { t.Fatalf("missing %s in %s", want, rec.Body.String()) }
	}
}

func TestFinalCoordinateAPIRejectsUnknownVariable(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	rec := httptest.NewRecorder()
	body := `{"variables":{},"latitude_formula":"N 59 54.[A]","longitude_formula":"E 010 45.000"}`
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/coordinates/final", strings.NewReader(body)))
	if rec.Code != http.StatusBadRequest { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
	if !strings.Contains(rec.Body.String(), "variable A is not defined") { t.Fatalf("body=%s", rec.Body.String()) }
}
