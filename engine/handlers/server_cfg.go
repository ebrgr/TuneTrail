package handlers

import (
	"net/http"

	"github.com/ebrgr/TuneTrail/internal/cfg"
	"github.com/ebrgr/TuneTrail/internal/utils"
)

type ServerConfig struct {
	WebUI        string `json:"web_ui"`
	DefaultTheme string `json:"default_theme"`
}

func GetCfgHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSON(w, http.StatusOK, ServerConfig{DefaultTheme: cfg.DefaultTheme(), WebUI: cfg.WebUI()})
	}
}
