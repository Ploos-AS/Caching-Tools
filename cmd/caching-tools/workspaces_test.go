package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMysteryWorkspaceCRUDPersistence(t *testing.T) {
	dir:=t.TempDir(); s:=newMysteryWorkspaceStore(dir)
	req:=mysteryWorkspaceRequest{Code:"GC12345",Title:"Forest mystery",Notes:"Check plaque",Variables:map[string]float64{"A":1,"b":2},Intermediate:[]string{"A+B=3"},LatitudeFormula:"N 59 54.[A+B]45",LongitudeFormula:"E 010 45.621"}
	x,err:=s.create(req); if err!=nil{t.Fatal(err)}
	if x.ID=="" || x.Variables["B"]!=2 { t.Fatalf("workspace=%+v",x) }
	if _,err:=os.Stat(filepath.Join(dir,"mystery-workspaces.json"));err!=nil{t.Fatal(err)}
	items,err:=s.list(); if err!=nil||len(items)!=1{t.Fatalf("items=%v err=%v",items,err)}
	req.Title="Updated mystery"; req.FinalWaypointID="wp-final"
	x,err=s.update(x.ID,req); if err!=nil{t.Fatal(err)}
	if x.Title!="Updated mystery" || x.FinalWaypointID!="wp-final"{t.Fatalf("workspace=%+v",x)}
	got,err:=s.get(x.ID); if err!=nil||got.Code!="GC12345"{t.Fatalf("got=%+v err=%v",got,err)}
	if err:=s.delete(x.ID);err!=nil{t.Fatal(err)}
	items,err=s.list(); if err!=nil||len(items)!=0{t.Fatalf("items=%v err=%v",items,err)}
}

func TestMysteryWorkspaceValidation(t *testing.T) {
	s:=newMysteryWorkspaceStore(t.TempDir())
	if _,err:=s.create(mysteryWorkspaceRequest{});err==nil{t.Fatal("expected title error")}
	if _,err:=s.create(mysteryWorkspaceRequest{Title:"x",Variables:map[string]float64{"AA":1}});err==nil{t.Fatal("expected variable error")}
}
