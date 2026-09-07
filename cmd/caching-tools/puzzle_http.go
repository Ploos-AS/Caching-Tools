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
	FromBase int `json:"from_base,omitempty"`
	ToBase int `json:"to_base,omitempty"`
}

func handlePuzzle(w http.ResponseWriter, r *http.Request) {
	var req puzzleOperationRequest
	if err := decodeJSON(r, &req); err != nil { writeError(w, http.StatusBadRequest, err); return }
	switch strings.ToLower(strings.TrimSpace(req.Operation)) {
	case "a1z26": writeJSON(w, http.StatusOK, a1z26(req.Text))
	case "caesar", "rot": writeJSON(w, http.StatusOK, puzzleResponse{Result: caesar(req.Text, req.Shift)})
	case "rot47": writeJSON(w, http.StatusOK, puzzleResponse{Result: rot47(req.Text)})
	case "digitsum", "checksum": sum, root := digitChecksum(req.Text); writeJSON(w, http.StatusOK, puzzleResponse{DigitSum: sum, Root: root})
	case "substitution", "substitute": result, err := substitute(req.Text, req.Alphabet, req.Mapping); if err != nil { writeError(w, http.StatusBadRequest, err); return }; writeJSON(w, http.StatusOK, puzzleResponse{Result: result})
	case "morse-encode": result, err := morse(req.Text, false); if err != nil { writeError(w, http.StatusBadRequest, err); return }; writeJSON(w, http.StatusOK, puzzleResponse{Result: result})
	case "morse-decode": result, err := morse(req.Text, true); if err != nil { writeError(w, http.StatusBadRequest, err); return }; writeJSON(w, http.StatusOK, puzzleResponse{Result: result})
	case "bacon-encode": result, err := bacon(req.Text, false); if err != nil { writeError(w, http.StatusBadRequest, err); return }; writeJSON(w, http.StatusOK, puzzleResponse{Result: result})
	case "bacon-decode": result, err := bacon(req.Text, true); if err != nil { writeError(w, http.StatusBadRequest, err); return }; writeJSON(w, http.StatusOK, puzzleResponse{Result: result})
	case "base": result, err := convertBase(req.Text, req.FromBase, req.ToBase); if err != nil { writeError(w, http.StatusBadRequest, err); return }; writeJSON(w, http.StatusOK, puzzleResponse{Result: result})
	case "keypad", "phone-keypad": writeJSON(w, http.StatusOK, phoneKeypad(req.Text))
	default: writeError(w, http.StatusBadRequest, errors.New("unknown puzzle operation"))
	}
}
