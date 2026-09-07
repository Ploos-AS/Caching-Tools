package main

import (
	"errors"
	"net/http"
	"os"
)

func registerMysteryBundleRoutes(mux *http.ServeMux, workspaces *mysteryWorkspaceStore, waypoints *waypointStore) {
	mux.HandleFunc("GET /api/mystery-workspaces/{id}/export", func(w http.ResponseWriter, r *http.Request) {
		bundle, err := exportMysteryWorkspaceBundle(r.PathValue("id"), workspaces, waypoints)
		if errors.Is(err, os.ErrNotExist) { writeError(w, http.StatusNotFound, errors.New("workspace not found")); return }
		if err != nil { writeError(w, http.StatusInternalServerError, err); return }
		w.Header().Set("Content-Disposition", `attachment; filename="mystery-workspace.json"`)
		writeJSON(w, http.StatusOK, bundle)
	})
	mux.HandleFunc("POST /api/mystery-workspaces/import", func(w http.ResponseWriter, r *http.Request) {
		var bundle mysteryWorkspaceBundle
		if err := decodeJSON(r, &bundle); err != nil { writeError(w, http.StatusBadRequest, err); return }
		x, err := importMysteryWorkspaceBundle(bundle, workspaces, waypoints)
		if err != nil { writeError(w, http.StatusBadRequest, err); return }
		writeJSON(w, http.StatusCreated, x)
	})
}
