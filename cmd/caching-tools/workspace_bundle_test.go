package main

import "testing"

func TestMysteryWorkspaceBundleRoundTrip(t *testing.T) {
	dir := t.TempDir()
	workspaces := newMysteryWorkspaceStore(dir)
	waypoints := newWaypointStore(dir)
	wp, err := waypoints.create(waypointRequest{Name:"Final", Latitude:"59.9", Longitude:"10.7", Type:"final", Comment:"answer"})
	if err != nil { t.Fatal(err) }
	x, err := workspaces.create(mysteryWorkspaceRequest{Code:"GC123", Title:"Mystery", Variables:map[string]float64{"A":1}, Intermediate:[]string{"A=1"}, LatitudeFormula:"N 59 54.123", LongitudeFormula:"E 010 45.456", FinalWaypointID:wp.ID})
	if err != nil { t.Fatal(err) }
	bundle, err := exportMysteryWorkspaceBundle(x.ID, workspaces, waypoints)
	if err != nil { t.Fatal(err) }
	if bundle.Format != mysteryBundleFormat || bundle.Version != 1 || bundle.Waypoint == nil { t.Fatalf("bundle=%+v", bundle) }
	if bundle.Workspace.FinalWaypointID != "" { t.Fatal("export must not retain local waypoint id") }
	imported, err := importMysteryWorkspaceBundle(bundle, workspaces, waypoints)
	if err != nil { t.Fatal(err) }
	if imported.ID == x.ID || imported.FinalWaypointID == "" || imported.FinalWaypointID == wp.ID { t.Fatalf("imported=%+v", imported) }
	items, err := waypoints.list(); if err != nil { t.Fatal(err) }
	if len(items) != 2 { t.Fatalf("waypoints=%d", len(items)) }
}

func TestMysteryWorkspaceBundleWithoutWaypoint(t *testing.T) {
	dir := t.TempDir(); workspaces := newMysteryWorkspaceStore(dir); waypoints := newWaypointStore(dir)
	x, err := workspaces.create(mysteryWorkspaceRequest{Title:"No final"}); if err != nil { t.Fatal(err) }
	bundle, err := exportMysteryWorkspaceBundle(x.ID, workspaces, waypoints); if err != nil { t.Fatal(err) }
	if bundle.Waypoint != nil { t.Fatalf("waypoint=%+v", bundle.Waypoint) }
	imported, err := importMysteryWorkspaceBundle(bundle, workspaces, waypoints); if err != nil { t.Fatal(err) }
	if imported.FinalWaypointID != "" { t.Fatalf("id=%q", imported.FinalWaypointID) }
}

func TestMysteryWorkspaceBundleRejectsVersion(t *testing.T) {
	_, err := importMysteryWorkspaceBundle(mysteryWorkspaceBundle{Format:mysteryBundleFormat, Version:99, Workspace:mysteryWorkspaceRequest{Title:"x"}}, newMysteryWorkspaceStore(t.TempDir()), newWaypointStore(t.TempDir()))
	if err == nil { t.Fatal("expected version error") }
}
