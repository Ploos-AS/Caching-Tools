package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPuzzleHelperUIAssets(t *testing.T) {
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	page := httptest.NewRecorder(); h.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK { t.Fatalf("status=%d", page.Code) }
	for _, want := range []string{`id="puzzle-a1z26"`,`id="puzzle-caesar"`,`id="puzzle-rot47"`,`id="puzzle-morse"`,`value="morse-encode"`,`id="puzzle-bacon"`,`value="bacon-encode"`,`id="puzzle-base"`,`id="puzzle-keypad"`,`id="puzzle-digits"`,`id="puzzle-substitution"`,`src="/puzzle.js"`,`Caching Tools M1.23`} { if !strings.Contains(page.Body.String(), want) { t.Fatalf("missing %q", want) } }
	asset := httptest.NewRecorder(); h.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/puzzle.js", nil)); if asset.Code != http.StatusOK { t.Fatalf("puzzle.js status=%d", asset.Code) }
	for _, want := range []string{"/api/puzzle","a1z26","caesar","#puzzle-rot47","#puzzle-morse","#puzzle-bacon","from_base","#puzzle-keypad","digitsum","substitution"} { if !strings.Contains(asset.Body.String(), want) { t.Fatalf("missing %q in puzzle.js", want) } }
}
