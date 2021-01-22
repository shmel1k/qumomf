package qumhttp

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func RegisterDebugHandlers(r *mux.Router, version, commit, buildDate string) {
	r.Handle("/debug/metrics", promhttp.Handler()).Methods(http.MethodGet)
	r.Handle("/debug/health", HealthHandler()).Methods(http.MethodGet)
	r.Handle("/debug/about", AboutHandler(version, commit, buildDate)).Methods(http.MethodGet)
}

func RegisterAPIHandlers(r *mux.Router, h APIHandler) {
	r.HandleFunc("/api/v0/snapshots", h.ClusterList).Methods(http.MethodGet)
	r.HandleFunc("/api/v0/snapshots/{cluster_name}", h.ClusterSnapshot).Methods(http.MethodGet)
	r.HandleFunc("/api/v0/snapshots/{cluster_name}/{shard_uuid}", h.ShardSnapshot).Methods(http.MethodGet)
	r.HandleFunc("/api/v0/snapshots/{cluster_name}/{shard_uuid}/{instance_uuid}", h.InstanceSnapshot).Methods(http.MethodGet)

	r.HandleFunc("/api/v0/recoveries/{cluster_name}/{shard_uuid}", h.ShardRecoveries).Methods(http.MethodGet)

	r.HandleFunc("/api/v0/alerts", h.Alerts).Methods(http.MethodGet)
	r.HandleFunc("/api/v0/alerts/{cluster_name}", h.ClusterAlerts).Methods(http.MethodGet)
}
