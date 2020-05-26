package qumhttp

import (
	"encoding/json"
	"net/http"
)

func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	})
}

func AboutHandler(version, buildDate string) http.Handler {
	about := struct {
		Version string `json:"version"`
		Build   string `json:"build"`
	}{
		Version: version,
		Build:   buildDate,
	}

	aboutStr, _ := json.Marshal(about)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(aboutStr)
	})
}
