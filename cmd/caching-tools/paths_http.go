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
