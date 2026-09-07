package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPathEditorUIAsset(t *testing.T) {
	h,err:=newHandler(); if err!=nil{t.Fatal(err)}
	rec:=httptest.NewRecorder(); h.ServeHTTP(rec,httptest.NewRequest(http.MethodGet,"/path-editor.js",nil))
	if rec.Code!=http.StatusOK{t.Fatalf("status=%d",rec.Code)}
	for _,want:=range []string{"Edit geometry","delete-point","move-point","split-segment","merge-segments","/edit","refreshLocalMap"} { if !strings.Contains(rec.Body.String(),want){t.Fatalf("missing %q",want)} }
}
