package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMysteryWorkspaceUIAssets(t *testing.T) {
	h,err:=newHandler(); if err!=nil{t.Fatal(err)}
	page:=httptest.NewRecorder(); h.ServeHTTP(page,httptest.NewRequest(http.MethodGet,"/",nil))
	if page.Code!=200{t.Fatalf("status=%d",page.Code)}
	for _,want:=range []string{`id="mystery-workspace-form"`,`id="mystery-workspace-list"`,`src="/workspaces.js"`} { if !strings.Contains(page.Body.String(),want){t.Fatalf("missing %q",want)} }
	asset:=httptest.NewRecorder(); h.ServeHTTP(asset,httptest.NewRequest(http.MethodGet,"/workspaces.js",nil))
	if asset.Code!=200{t.Fatalf("status=%d",asset.Code)}
	for _,want:=range []string{"/api/mystery-workspaces","parseWorkspaceVariables","final_waypoint_id","loadMysteryWorkspaces","Use in solver","caching-tools:puzzle-result","addPuzzleIntermediate","setPuzzleVariable","linkFinalWaypointToActiveWorkspace","Export JSON","/export","/api/mystery-workspaces/import","mystery-workspace-import-file","importWorkspaceFile","Caching Tools M1.18"} { if !strings.Contains(asset.Body.String(),want){t.Fatalf("missing %q",want)} }

	formula:=httptest.NewRecorder(); h.ServeHTTP(formula,httptest.NewRequest(http.MethodGet,"/formula.js",nil))
	for _,want:=range []string{"loadFinalWorkspace","getActiveMysteryWorkspace","linkFinalWaypointToActiveWorkspace","saved.id"} { if !strings.Contains(formula.Body.String(),want){t.Fatalf("missing %q in formula.js",want)} }

	puzzle:=httptest.NewRecorder(); h.ServeHTTP(puzzle,httptest.NewRequest(http.MethodGet,"/puzzle.js",nil))
	for _,want:=range []string{"caching-tools:puzzle-result","emitPuzzleResult"} { if !strings.Contains(puzzle.Body.String(),want){t.Fatalf("missing %q in puzzle.js",want)} }
}
