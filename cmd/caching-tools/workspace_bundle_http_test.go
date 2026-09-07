package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMysteryWorkspaceBundleHTTP(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR", t.TempDir())
	h, err := newHandler(); if err != nil { t.Fatal(err) }
	create := httptest.NewRecorder()
	h.ServeHTTP(create, httptest.NewRequest(http.MethodPost, "/api/mystery-workspaces", strings.NewReader(`{"code":"GCX","title":"Bundle case","variables":{"A":7}}`)))
	if create.Code != http.StatusCreated { t.Fatalf("create=%d %s", create.Code, create.Body.String()) }
	var x mysteryWorkspace; if err := json.Unmarshal(create.Body.Bytes(), &x); err != nil { t.Fatal(err) }
	export := httptest.NewRecorder()
	h.ServeHTTP(export, httptest.NewRequest(http.MethodGet, "/api/mystery-workspaces/"+x.ID+"/export", nil))
	if export.Code != http.StatusOK { t.Fatalf("export=%d %s", export.Code, export.Body.String()) }
	if !strings.Contains(export.Header().Get("Content-Disposition"), "mystery-workspace.json") { t.Fatalf("disposition=%q", export.Header().Get("Content-Disposition")) }
	if !strings.Contains(export.Body.String(), `"format":"caching-tools-mystery"`) { t.Fatalf("body=%s", export.Body.String()) }
	imp := httptest.NewRecorder()
	h.ServeHTTP(imp, httptest.NewRequest(http.MethodPost, "/api/mystery-workspaces/import", strings.NewReader(export.Body.String())))
	if imp.Code != http.StatusCreated { t.Fatalf("import=%d %s", imp.Code, imp.Body.String()) }
	var y mysteryWorkspace; if err := json.Unmarshal(imp.Body.Bytes(), &y); err != nil { t.Fatal(err) }
	if y.ID == x.ID || y.Title != x.Title { t.Fatalf("imported=%+v", y) }
}
