package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type mysteryWorkspace struct {
	ID               string             `json:"id"`
	Code             string             `json:"code,omitempty"`
	Title            string             `json:"title"`
	Notes            string             `json:"notes,omitempty"`
	Variables        map[string]float64 `json:"variables,omitempty"`
	Intermediate     []string           `json:"intermediate,omitempty"`
	LatitudeFormula  string             `json:"latitude_formula,omitempty"`
	LongitudeFormula string             `json:"longitude_formula,omitempty"`
	FinalWaypointID  string             `json:"final_waypoint_id,omitempty"`
	CreatedAt        string             `json:"created_at"`
	UpdatedAt        string             `json:"updated_at"`
}

type mysteryWorkspaceRequest struct {
	Code             string             `json:"code,omitempty"`
	Title            string             `json:"title"`
	Notes            string             `json:"notes,omitempty"`
	Variables        map[string]float64 `json:"variables,omitempty"`
	Intermediate     []string           `json:"intermediate,omitempty"`
	LatitudeFormula  string             `json:"latitude_formula,omitempty"`
	LongitudeFormula string             `json:"longitude_formula,omitempty"`
	FinalWaypointID  string             `json:"final_waypoint_id,omitempty"`
}

type mysteryWorkspaceStore struct { mu sync.Mutex; path string }
func newMysteryWorkspaceStore(dataDir string) *mysteryWorkspaceStore { return &mysteryWorkspaceStore{path: filepath.Join(dataDir,"mystery-workspaces.json")} }

func normalizeWorkspaceRequest(req mysteryWorkspaceRequest) (mysteryWorkspaceRequest,error) {
	req.Code=strings.TrimSpace(req.Code); req.Title=strings.TrimSpace(req.Title); req.Notes=strings.TrimSpace(req.Notes); req.LatitudeFormula=strings.TrimSpace(req.LatitudeFormula); req.LongitudeFormula=strings.TrimSpace(req.LongitudeFormula); req.FinalWaypointID=strings.TrimSpace(req.FinalWaypointID)
	if req.Title=="" { return req,errors.New("title is required") }
	clean:=map[string]float64{}
	for k,v:=range req.Variables { key:=strings.ToUpper(strings.TrimSpace(k)); if len(key)!=1 || key[0]<'A' || key[0]>'Z' { return req,fmt.Errorf("variable %q must be A-Z",k) }; clean[key]=v }
	req.Variables=clean
	trimmed:=make([]string,0,len(req.Intermediate)); for _,v:=range req.Intermediate { if s:=strings.TrimSpace(v); s!="" { trimmed=append(trimmed,s) } }; req.Intermediate=trimmed
	return req,nil
}

func (s *mysteryWorkspaceStore) list() ([]mysteryWorkspace,error) { s.mu.Lock(); defer s.mu.Unlock(); items,err:=s.loadLocked(); if err!=nil{return nil,err}; sort.Slice(items,func(i,j int)bool{return items[i].UpdatedAt>items[j].UpdatedAt}); return items,nil }
func (s *mysteryWorkspaceStore) get(id string) (mysteryWorkspace,error) { s.mu.Lock(); defer s.mu.Unlock(); items,err:=s.loadLocked(); if err!=nil{return mysteryWorkspace{},err}; for _,x:=range items { if x.ID==id{return x,nil} }; return mysteryWorkspace{},os.ErrNotExist }
func (s *mysteryWorkspaceStore) create(req mysteryWorkspaceRequest) (mysteryWorkspace,error) { req,err:=normalizeWorkspaceRequest(req); if err!=nil{return mysteryWorkspace{},err}; s.mu.Lock(); defer s.mu.Unlock(); items,err:=s.loadLocked(); if err!=nil{return mysteryWorkspace{},err}; now:=time.Now().UTC().Format(time.RFC3339Nano); x:=mysteryWorkspace{ID:fmt.Sprintf("mystery-%x",time.Now().UnixNano()),Code:req.Code,Title:req.Title,Notes:req.Notes,Variables:req.Variables,Intermediate:req.Intermediate,LatitudeFormula:req.LatitudeFormula,LongitudeFormula:req.LongitudeFormula,FinalWaypointID:req.FinalWaypointID,CreatedAt:now,UpdatedAt:now}; items=append(items,x); if err:=s.saveLocked(items);err!=nil{return mysteryWorkspace{},err}; return x,nil }
func (s *mysteryWorkspaceStore) update(id string, req mysteryWorkspaceRequest) (mysteryWorkspace,error) { req,err:=normalizeWorkspaceRequest(req); if err!=nil{return mysteryWorkspace{},err}; s.mu.Lock(); defer s.mu.Unlock(); items,err:=s.loadLocked(); if err!=nil{return mysteryWorkspace{},err}; for i:=range items { if items[i].ID!=id{continue}; items[i].Code=req.Code; items[i].Title=req.Title; items[i].Notes=req.Notes; items[i].Variables=req.Variables; items[i].Intermediate=req.Intermediate; items[i].LatitudeFormula=req.LatitudeFormula; items[i].LongitudeFormula=req.LongitudeFormula; items[i].FinalWaypointID=req.FinalWaypointID; items[i].UpdatedAt=time.Now().UTC().Format(time.RFC3339Nano); if err:=s.saveLocked(items);err!=nil{return mysteryWorkspace{},err}; return items[i],nil }; return mysteryWorkspace{},os.ErrNotExist }
func (s *mysteryWorkspaceStore) delete(id string) error { s.mu.Lock(); defer s.mu.Unlock(); items,err:=s.loadLocked(); if err!=nil{return err}; for i:=range items { if items[i].ID==id { items=append(items[:i],items[i+1:]...); return s.saveLocked(items) } }; return os.ErrNotExist }
func (s *mysteryWorkspaceStore) loadLocked() ([]mysteryWorkspace,error) { data,err:=os.ReadFile(s.path); if errors.Is(err,os.ErrNotExist){return []mysteryWorkspace{},nil}; if err!=nil{return nil,err}; var items []mysteryWorkspace; if err:=json.Unmarshal(data,&items);err!=nil{return nil,fmt.Errorf("read mystery workspace store: %w",err)}; return items,nil }
func (s *mysteryWorkspaceStore) saveLocked(items []mysteryWorkspace) error { if err:=os.MkdirAll(filepath.Dir(s.path),0o755);err!=nil{return err}; data,err:=json.MarshalIndent(items,"","  "); if err!=nil{return err}; data=append(data,'\n'); tmp:=s.path+".tmp"; if err:=os.WriteFile(tmp,data,0o600);err!=nil{return err}; return os.Rename(tmp,s.path) }
