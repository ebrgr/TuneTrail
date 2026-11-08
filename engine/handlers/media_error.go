package handlers

import (
	"errors"
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

func writeMediaReadError(w http.ResponseWriter, err error, kind string) {
	if errors.Is(err, db.ErrNotFound) {
		utils.WriteError(w, kind+" with specified id could not be found", http.StatusNotFound)
		return
	}
	utils.WriteError(w, "unable to load "+kind, http.StatusInternalServerError)
}
