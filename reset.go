package main

import "net/http"

// POST /admin/reset handler : db 리셋
func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	if cfg.platform != "dev" {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Reset is only allowed in dev environment."))
		return
	}

	err := cfg.db.Reset()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't reset database", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Database reset to initial state"))
}
