package main

import (
	"errors"
	"net/http"
	"os"
)

func handlePathImport(w http.ResponseWriter, r *http.Request, store *pathStore) {
	defer r.Body.Close()
	doc, err := decodeGPX(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	items, err := store.importGPX(doc)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	views := make([]pathResponse, 0, len(items))
	for _, item := range items {
		view, err := pathView(item)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		views = append(views, view)
	}
	writeJSON(w, http.StatusCreated, pathImportResult{Imported: len(views), Paths: views})
}

func handlePathList(w http.ResponseWriter, store *pathStore) {
	items, err := store.list()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	views := make([]pathResponse, 0, len(items))
	for _, item := range items {
		view, err := pathView(item)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		views = append(views, view)
	}
	writeJSON(w, http.StatusOK, views)
}

func handlePathGet(w http.ResponseWriter, r *http.Request, store *pathStore) {
	item, err := store.get(r.PathValue("id"))
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, errors.New("path not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func handlePathRename(w http.ResponseWriter, r *http.Request, store *pathStore) {
	var req pathRenameRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	item, err := store.rename(r.PathValue("id"), req.Name)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, errors.New("path not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	view, err := pathView(item)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, view)
}

func handlePathExport(w http.ResponseWriter, r *http.Request, store *pathStore) {
	item, err := store.get(r.PathValue("id"))
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, errors.New("path not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	data, err := encodeStoredPathGPX(item)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "application/gpx+xml; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+item.ID+`.gpx"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func handlePathDelete(w http.ResponseWriter, r *http.Request, store *pathStore) {
	if err := store.delete(r.PathValue("id")); errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, errors.New("path not found"))
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
