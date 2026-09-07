package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestMysteryWorkspaceAPI(t *testing.T) {
	t.Setenv("CACHING_TOOLS_DATA_DIR",t.TempDir())
	h,err:=newHandler(); if err!=nil{t.Fatal(err)}
	create:=httptest.NewRecorder(); h.ServeHTTP(create,httptest.NewRequest(http.MethodPost,"/api/mystery-workspaces",strings.NewReader(`{"code":"GC123","title":"Mystery","variables":{"A":1},"latitude_formula":"N 59 54.[A]45","longitude_formula":"E 010 45.621"}`)))
	if create.Code!=http.StatusCreated{t.Fatalf("status=%d body=%s",create.Code,create.Body.String())}
	if !strings.Contains(create.Body.String(),`"code":"GC123"`){t.Fatal(create.Body.String())}
	list:=httptest.NewRecorder(); h.ServeHTTP(list,httptest.NewRequest(http.MethodGet,"/api/mystery-workspaces",nil)); if list.Code!=200||!strings.Contains(list.Body.String(),`"title":"Mystery"`){t.Fatalf("status=%d body=%s",list.Code,list.Body.String())}
	_ = os.Unsetenv("CACHING_TOOLS_DATA_DIR")
}
