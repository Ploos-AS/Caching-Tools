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
		{`{"operation":"rot47","text":"Hello!"}`, `"result":"w6==@P"`},
		{`{"operation":"morse-encode","text":"SOS"}`, `"result":"... --- ..."`},
		{`{"operation":"morse-decode","text":"... --- ..."}`, `"result":"SOS"`},
		{`{"operation":"bacon-encode","text":"AB"}`, `"result":"AAAAA AAAAB"`},
		{`{"operation":"bacon-decode","text":"AAAAA AAAAB"}`, `"result":"AB"`},
		{`{"operation":"base","text":"FF","from_base":16,"to_base":10}`, `"result":"255"`},
		{`{"operation":"keypad","text":"ABC"}`, `"sum":6`},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/puzzle", strings.NewReader(tc.body)))
		if rec.Code != http.StatusOK { t.Fatalf("status=%d body=%s request=%s", rec.Code, rec.Body.String(), tc.body) }
		if !strings.Contains(rec.Body.String(), tc.want) { t.Fatalf("want=%s body=%s", tc.want, rec.Body.String()) }
	}
}

func TestPuzzleAPIRejectsBadExtendedInput(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	for _, body := range []string{
		`{"operation":"morse-decode","text":"... --..--"}`,
		`{"operation":"bacon-decode","text":"AAAA"}`,
		`{"operation":"base","text":"2","from_base":2,"to_base":10}`,
	} {
		rec := httptest.NewRecorder(); h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/puzzle", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest { t.Fatalf("status=%d body=%s request=%s", rec.Code, rec.Body.String(), body) }
	}
}
