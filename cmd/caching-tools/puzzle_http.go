package main

import (
	"errors"
	"net/http"
	"strings"
)

type puzzleOperationRequest struct {
	Operation string `json:"operation"`
	Text string `json:"text"`
	Shift int `json:"shift,omitempty"`
	Alphabet string `json:"alphabet,omitempty"`
	Mapping string `json:"mapping,omitempty"`
}

func handlePuzzle(w http.ResponseWriter, r *http.Request) {
	var req puzzleOperationRequest
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, err); return }
	switch strings.ToLower(strings.TrimSpace(req.Operation)) {
	case "a1z26":
		writeJSON(w, http.StatusOK, a1z26(req.Text))
	case "caesar", "rot":
		writeJSON(w, http.StatusOK, puzzleResponse{Result: caesar(req.Text, req.Shift)})
	case "digitsum", "checksum":
		sum, root := digitChecksum(req.Text)
		writeJSON(w, http.StatusOK, puzzleResponse{DigitSum: sum, Root: root})
	case "substitution", "substitute":
		result, err := substitute(req.Text, req.Alphabet, req.Mapping)
		if err != nil { writeError(w, http.StatusBadRequest, err); return }
		writeJSON(w, http.StatusOK, puzzleResponse{Result: result})
	default:
		writeError(w, http.StatusBadRequest, errors.New("unknown puzzle operation"))
	}
}
