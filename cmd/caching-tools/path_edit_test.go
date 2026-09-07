package main

import "testing"

func TestRoutePointDeleteAndMove(t *testing.T) {
	s := newPathStore(t.TempDir())
	items, err := s.importGPX(gpxDocument{Routes: []gpxRoute{{Name:"r",Points:[]gpxPoint{{Latitude:1,Longitude:1},{Latitude:2,Longitude:2},{Latitude:3,Longitude:3}}}}})
	if err != nil { t.Fatal(err) }
	id := items[0].ID
	if _,err:=s.edit(id,pathEditRequest{Operation:"move-point",Point:0,Target:2});err!=nil{t.Fatal(err)}
	got,err:=s.get(id);if err!=nil{t.Fatal(err)}
	if got.Route[2].Latitude!=1{t.Fatalf("route=%+v",got.Route)}
	if _,err:=s.edit(id,pathEditRequest{Operation:"delete-point",Point:1});err!=nil{t.Fatal(err)}
	got,_=s.get(id);if len(got.Route)!=2{t.Fatalf("points=%d",len(got.Route))}
}

func TestTrackSplitMergeAndMove(t *testing.T) {
	s:=newPathStore(t.TempDir())
	items,err:=s.importGPX(gpxDocument{Tracks:[]gpxTrack{{Name:"t",Segments:[]gpxTrackSegment{{Points:[]gpxPoint{{Latitude:1,Longitude:1},{Latitude:2,Longitude:2},{Latitude:3,Longitude:3}}}}}}})
	if err!=nil{t.Fatal(err)};id:=items[0].ID
	if _,err:=s.edit(id,pathEditRequest{Operation:"split-segment",Segment:0,Point:1});err!=nil{t.Fatal(err)}
	got,_:=s.get(id);if len(got.Segments)!=2{t.Fatalf("segments=%d",len(got.Segments))}
	if _,err:=s.edit(id,pathEditRequest{Operation:"merge-segments",Segment:0});err!=nil{t.Fatal(err)}
	got,_=s.get(id);if len(got.Segments)!=1||len(got.Segments[0].Points)!=3{t.Fatalf("track=%+v",got.Segments)}
	if _,err:=s.edit(id,pathEditRequest{Operation:"move-point",Segment:0,Point:2,Target:0});err!=nil{t.Fatal(err)}
	got,_=s.get(id);if got.Segments[0].Points[0].Latitude!=3{t.Fatalf("track=%+v",got.Segments)}
}

func TestPathEditRejectsEmptyGeometry(t *testing.T){
	s:=newPathStore(t.TempDir());items,err:=s.importGPX(gpxDocument{Routes:[]gpxRoute{{Points:[]gpxPoint{{Latitude:1,Longitude:1}}}}});if err!=nil{t.Fatal(err)}
	if _,err:=s.edit(items[0].ID,pathEditRequest{Operation:"delete-point",Point:0});err==nil{t.Fatal("expected error")}
}
