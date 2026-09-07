package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVisualMapEditorAssets(t *testing.T) {
	h, err := newHandler()
	if err != nil { t.Fatal(err) }

	mapAsset := httptest.NewRecorder()
	h.ServeHTTP(mapAsset, httptest.NewRequest(http.MethodGet, "/map.js", nil))
	if mapAsset.Code != http.StatusOK { t.Fatalf("map.js status=%d", mapAsset.Code) }
	for _, want := range []string{"cachingToolsMap", "clientToMap", "inverse", "caching-tools:map-rendered"} {
		if !strings.Contains(mapAsset.Body.String(), want) { t.Fatalf("missing %q in map.js", want) }
	}

	linkAsset := httptest.NewRecorder()
	h.ServeHTTP(linkAsset, httptest.NewRequest(http.MethodGet, "/map-link.js", nil))
	for _, want := range []string{"/map-editor.js", "data-map-editor"} {
		if !strings.Contains(linkAsset.Body.String(), want) { t.Fatalf("missing %q in map-link.js", want) }
	}

	editor := httptest.NewRecorder()
	h.ServeHTTP(editor, httptest.NewRequest(http.MethodGet, "/map-editor.js", nil))
	if editor.Code != http.StatusOK { t.Fatalf("map-editor.js status=%d", editor.Code) }
	for _, want := range []string{"Visual route/track editor", "set-point", "delete-point", "split-segment", "merge-segments", "pointerdown", "pointermove", "pointerup", "/api/paths/"} {
		if !strings.Contains(editor.Body.String(), want) { t.Fatalf("missing %q in map-editor.js", want) }
	}
}
