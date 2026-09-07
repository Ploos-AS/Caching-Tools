package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPuzzleAPI(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	cases := []struct{ body, want string }{
		{`{"operation":"a1z26","text":"ABC"}`, `"sum":6`},
		{`{"operation":"caesar","text":"ABC","shift":13}`, `"result":"NOP"`},
		{`{"operation":"digitsum","text":"12345"}`, `"digital_root":6`},
		{`{"operation":"substitution","text":"ABC","alphabet":"ABC","mapping":"XYZ"}`, `"result":"XYZ"`},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/puzzle", strings.NewReader(tc.body)))
		if rec.Code != http.StatusOK { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
		if !strings.Contains(rec.Body.String(), tc.want) { t.Fatalf("want=%s body=%s", tc.want, rec.Body.String()) }
	}
}
