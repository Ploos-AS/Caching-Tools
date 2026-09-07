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
	for _,want:=range []string{`id="mystery-workspace-form"`,`id="mystery-workspace-list"`,`src="/workspaces.js"`,`Caching Tools M1.16`} { if !strings.Contains(page.Body.String(),want){t.Fatalf("missing %q",want)} }
	asset:=httptest.NewRecorder(); h.ServeHTTP(asset,httptest.NewRequest(http.MethodGet,"/workspaces.js",nil))
	if asset.Code!=200{t.Fatalf("status=%d",asset.Code)}
	for _,want:=range []string{"/api/mystery-workspaces","parseWorkspaceVariables","final_waypoint_id","loadMysteryWorkspaces"} { if !strings.Contains(asset.Body.String(),want){t.Fatalf("missing %q",want)} }
}
