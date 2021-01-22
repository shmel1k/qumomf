package qumhttp

import (
	"encoding/json"
	"net/http"
)

func HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
	})
}

func AboutHandler(version, commit, buildDate string) http.Handler {
	about := struct {
		Version string `json:"version"`
		Commit  string `json:"commit"`
		Build   string `json:"build"`
	}{
		Version: version,
		Commit:  commit,
		Build:   buildDate,
	}

	aboutStr, _ := json.Marshal(about)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(aboutStr)
	})
}
