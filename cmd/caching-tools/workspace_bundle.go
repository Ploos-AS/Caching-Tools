package main

import (
	"errors"
	"fmt"
	"os"
	"time"
)

const mysteryBundleFormat = "caching-tools-mystery"
const mysteryBundleVersion = 1

type mysteryWorkspaceBundle struct {
	Format     string                   `json:"format"`
	Version    int                      `json:"version"`
	ExportedAt string                   `json:"exported_at"`
	Workspace  mysteryWorkspaceRequest  `json:"workspace"`
	Waypoint   *waypointRequest         `json:"final_waypoint,omitempty"`
}

func exportMysteryWorkspaceBundle(id string, workspaces *mysteryWorkspaceStore, waypoints *waypointStore) (mysteryWorkspaceBundle, error) {
	x, err := workspaces.get(id)
	if err != nil { return mysteryWorkspaceBundle{}, err }
	bundle := mysteryWorkspaceBundle{
		Format: mysteryBundleFormat, Version: mysteryBundleVersion,
		ExportedAt: time.Now().UTC().Format(time.RFC3339Nano),
		Workspace: mysteryWorkspaceRequest{Code:x.Code, Title:x.Title, Notes:x.Notes, Variables:x.Variables, Intermediate:x.Intermediate, LatitudeFormula:x.LatitudeFormula, LongitudeFormula:x.LongitudeFormula},
	}
	if x.FinalWaypointID != "" {
		items, err := waypoints.list()
		if err != nil { return mysteryWorkspaceBundle{}, err }
		for _, wp := range items {
			if wp.ID != x.FinalWaypointID { continue }
			bundle.Waypoint = &waypointRequest{Name:wp.Name, Latitude:fmt.Sprintf("%.10f",wp.Latitude), Longitude:fmt.Sprintf("%.10f",wp.Longitude), Type:wp.Type, Comment:wp.Comment}
			break
		}
	}
	return bundle, nil
}

func importMysteryWorkspaceBundle(bundle mysteryWorkspaceBundle, workspaces *mysteryWorkspaceStore, waypoints *waypointStore) (mysteryWorkspace, error) {
	if bundle.Format != mysteryBundleFormat { return mysteryWorkspace{}, errors.New("unsupported mystery bundle format") }
	if bundle.Version != mysteryBundleVersion { return mysteryWorkspace{}, fmt.Errorf("unsupported mystery bundle version %d", bundle.Version) }
	req, err := normalizeWorkspaceRequest(bundle.Workspace)
	if err != nil { return mysteryWorkspace{}, err }
	if bundle.Waypoint != nil {
		wp, err := waypoints.create(*bundle.Waypoint)
		if err != nil { return mysteryWorkspace{}, fmt.Errorf("import final waypoint: %w", err) }
		req.FinalWaypointID = wp.ID
	}
	x, err := workspaces.create(req)
	if err != nil {
		if req.FinalWaypointID != "" { _ = waypoints.delete(req.FinalWaypointID) }
		return mysteryWorkspace{}, err
	}
	return x, nil
}

func mysteryWorkspaceNotFound(err error) bool { return errors.Is(err, os.ErrNotExist) }
