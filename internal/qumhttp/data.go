package qumhttp

import (
	"net/http"

	"github.com/shmel1k/qumomf/internal/vshard"
)

type response struct {
	statusCode int
	data       []byte
	err        error
}

func newOKResponse(data []byte) response {
	return response{
		statusCode: http.StatusOK,
		data:       data,
	}
}

func newBadRequestResponse(msg string) response {
	return response{
		statusCode: http.StatusBadRequest,
		data:       []byte(msg),
	}
}

func newInternalErrResponse(msg string, err error) response {
	return response{
		statusCode: http.StatusInternalServerError,
		data:       []byte(msg),
		err:        err,
	}
}

type params struct {
	clusterName  string
	shardUUID    vshard.ReplicaSetUUID
	instanceUUID vshard.InstanceUUID
}

func parseParams(vars map[string]string) params {
	return params{
		clusterName:  vars[paramClusterName],
		shardUUID:    vshard.ReplicaSetUUID(vars[paramShardUUID]),
		instanceUUID: vshard.InstanceUUID(vars[paramInstanceUUID]),
	}
}
