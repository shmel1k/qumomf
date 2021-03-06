package qumhttp

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"

	"github.com/shmel1k/qumomf/internal/api"
)

const (
	paramClusterName  = "cluster_name"
	paramShardUUID    = "shard_uuid"
	paramInstanceUUID = "instance_uuid"
)

const (
	msgMarshallingError = "failed to marshal data"
	msgInvalidParams    = "one or more parameters are invalid"
)

type APIHandler interface {
	ClusterList(http.ResponseWriter, *http.Request)
	ClusterSnapshot(http.ResponseWriter, *http.Request)
	ShardSnapshot(http.ResponseWriter, *http.Request)
	InstanceSnapshot(http.ResponseWriter, *http.Request)
	ShardRecoveries(http.ResponseWriter, *http.Request)
	Alerts(http.ResponseWriter, *http.Request)
	ClusterAlerts(http.ResponseWriter, *http.Request)
}

type apiHandler struct {
	apiSrv api.Service
	logger zerolog.Logger
}

func NewHandler(logger zerolog.Logger, apiSrv api.Service) APIHandler {
	return &apiHandler{
		logger: logger,
		apiSrv: apiSrv,
	}
}

func (a *apiHandler) ClusterList(w http.ResponseWriter, r *http.Request) {
	resp, err := a.apiSrv.ClustersList(r.Context())
	if err != nil {
		a.writeResponse(w, newInternalErrResponse("failed to get cluster list", err))
		return
	}

	data, err := json.Marshal(resp)
	if err != nil {
		a.writeResponse(w, newInternalErrResponse(msgMarshallingError, err))
		return
	}

	a.writeResponse(w, newOKResponse(data))
}

// nolint: dupl
func (a *apiHandler) ClusterSnapshot(w http.ResponseWriter, r *http.Request) {
	reqParams := parseParams(mux.Vars(r))
	if reqParams.clusterName == "" {
		a.writeResponse(w, newBadRequestResponse(msgInvalidParams))
		return
	}

	snap, err := a.apiSrv.ClusterSnapshot(r.Context(), reqParams.clusterName)
	if err != nil {
		if isNotFoundTypeErr(err) {
			a.writeResponse(w, newBadRequestResponse(parseNotFoundTypeErr(err)))
			return
		}
		a.writeResponse(w, newInternalErrResponse("failed get cluster snapshot", err))
		return
	}

	data, err := json.Marshal(snap)
	if err != nil {
		a.writeResponse(w, newInternalErrResponse(msgMarshallingError, err))
		return
	}

	a.writeResponse(w, newOKResponse(data))
}

func (a *apiHandler) ShardSnapshot(w http.ResponseWriter, r *http.Request) {
	reqParams := parseParams(mux.Vars(r))
	if reqParams.clusterName == "" || reqParams.shardUUID == "" {
		a.writeResponse(w, newBadRequestResponse(msgInvalidParams))
		return
	}

	shard, err := a.apiSrv.ReplicaSet(r.Context(), reqParams.clusterName, reqParams.shardUUID)
	if err != nil {
		if isNotFoundTypeErr(err) {
			a.writeResponse(w, newBadRequestResponse(parseNotFoundTypeErr(err)))
			return
		}
		a.writeResponse(w, newInternalErrResponse("failed get shard snapshot", err))
		return
	}

	data, err := json.Marshal(shard)
	if err != nil {
		a.writeResponse(w, newInternalErrResponse(msgMarshallingError, err))
		return
	}

	a.writeResponse(w, newOKResponse(data))
}

func (a *apiHandler) InstanceSnapshot(w http.ResponseWriter, r *http.Request) {
	reqParams := parseParams(mux.Vars(r))
	if reqParams.clusterName == "" || reqParams.shardUUID == "" || reqParams.instanceUUID == "" {
		a.writeResponse(w, newBadRequestResponse(msgInvalidParams))
		return
	}

	inst, err := a.apiSrv.Instance(r.Context(), reqParams.clusterName, reqParams.shardUUID, reqParams.instanceUUID)
	if err != nil {
		if isNotFoundTypeErr(err) {
			a.writeResponse(w, newBadRequestResponse(parseNotFoundTypeErr(err)))
			return
		}
		a.writeResponse(w, newInternalErrResponse("failed get instance snapshot", err))

		return
	}

	data, err := json.Marshal(inst)
	if err != nil {
		a.writeResponse(w, newInternalErrResponse(msgMarshallingError, err))
		return
	}

	a.writeResponse(w, newOKResponse(data))
}

func (a *apiHandler) ShardRecoveries(w http.ResponseWriter, r *http.Request) {
	reqParams := parseParams(mux.Vars(r))
	if reqParams.clusterName == "" || reqParams.shardUUID == "" {
		a.writeResponse(w, newBadRequestResponse(msgInvalidParams))
		return
	}

	recoveries, err := a.apiSrv.Recoveries(r.Context(), reqParams.clusterName, reqParams.shardUUID)
	if err != nil {
		a.writeResponse(w, newInternalErrResponse("failed get shard recovery", err))
		return
	}

	data, err := json.Marshal(recoveries)
	if err != nil {
		a.writeResponse(w, newInternalErrResponse(msgMarshallingError, err))
		return
	}

	a.writeResponse(w, newOKResponse(data))
}

func (a *apiHandler) Alerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := a.apiSrv.Alerts(r.Context())
	if err != nil {
		a.writeResponse(w, newInternalErrResponse("failed get alerts list", err))
		return
	}

	data, err := json.Marshal(alerts)
	if err != nil {
		a.writeResponse(w, newInternalErrResponse(msgMarshallingError, err))
		return
	}

	a.writeResponse(w, newOKResponse(data))
}

// nolint: dupl
func (a *apiHandler) ClusterAlerts(w http.ResponseWriter, r *http.Request) {
	reqParams := parseParams(mux.Vars(r))
	if reqParams.clusterName == "" {
		a.writeResponse(w, newBadRequestResponse(msgInvalidParams))
		return
	}

	alerts, err := a.apiSrv.ClusterAlerts(r.Context(), reqParams.clusterName)
	if err != nil {
		if isNotFoundTypeErr(err) {
			a.writeResponse(w, newBadRequestResponse(parseNotFoundTypeErr(err)))
			return
		}
		a.writeResponse(w, newInternalErrResponse("failed get cluster alerts", err))
		return
	}

	data, err := json.Marshal(alerts)
	if err != nil {
		a.writeResponse(w, newInternalErrResponse(msgMarshallingError, err))
		return
	}

	a.writeResponse(w, newOKResponse(data))
}

func isNotFoundTypeErr(err error) bool {
	return err == api.ErrClusterNotFound || err == api.ErrReplicaSetNotFound || err == api.ErrInstanceNotFound
}

func parseNotFoundTypeErr(err error) string {
	switch err {
	case api.ErrClusterNotFound:
		return "cluster snapshot not found"
	case api.ErrReplicaSetNotFound:
		return "shard snapshot not found"
	case api.ErrInstanceNotFound:
		return "instance snapshot not found"
	}

	return "cluster not found"
}

func (a *apiHandler) writeResponse(w http.ResponseWriter, resp response) {
	if resp.err != nil {
		a.logger.Err(resp.err).Msg(string(resp.data))
	}

	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.statusCode)

	_, err := w.Write(resp.data)
	if err != nil {
		a.logger.Err(err).Msg("failed to write response")
	}
}
