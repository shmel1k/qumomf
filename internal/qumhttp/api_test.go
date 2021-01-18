package qumhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/shmel1k/qumomf/internal/vshard/orchestrator"

	"github.com/shmel1k/qumomf/internal/api"
	"github.com/shmel1k/qumomf/internal/storage/sqlite"
	"github.com/shmel1k/qumomf/internal/vshard"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	tDBFileName   string                = "test.db"
	tClusterName  string                = "test_cluster"
	tShardUUID    vshard.ReplicaSetUUID = "7c652540-2d9c-4eb1-8473-a41ec7ab3554"
	tInstanceUUID vshard.InstanceUUID   = "11a6a15d-1ddd-4d10-af53-d489774b6ad6"
	tInstanceURI  string                = "test_inst"
)
var (
	dummyLogger  = zerolog.New(nil)
	dummyContext = context.Background()
)

type apiSuite struct {
	suite.Suite
	handler APIHandler

	router *mux.Router
}

func (a *apiSuite) SetupSuite() {
	t := a.Suite.T()

	db, err := sqlite.New(sqlite.Config{
		FileName:       tDBFileName,
		ConnectTimeout: time.Second,
		QueryTimeout:   time.Second,
	})
	require.NoError(t, err)

	err = db.SaveSnapshot(dummyContext, tClusterName, tSnapshot)
	require.NoError(t, err)

	err = db.SaveRecovery(dummyContext, tRecovery)
	require.NoError(t, err)

	a.handler = NewHandler(dummyLogger, api.NewService(db))

	router := mux.NewRouter()
	RegisterAPIHandlers(router, a.handler)

	a.router = router
}

func (a *apiSuite) TearDownSuite() {
	err := os.Remove(tDBFileName)
	require.NoError(a.T(), err)
}

func TestAPI(t *testing.T) {
	suite.Run(t, &apiSuite{
		Suite: suite.Suite{},
	})
}

func (a *apiSuite) TestGetClustersList() {
	t := a.T()
	r := httptest.NewRequest(http.MethodGet, "/api/v0/snapshots", nil)
	w := httptest.NewRecorder()

	a.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, a.jsonMarshal([]api.ClusterInfo{{
		Name:         tClusterName,
		ShardsCount:  len(tSnapshot.ReplicaSets),
		RoutersCount: len(tSnapshot.Routers),
	}}), w.Body.String())
}

func (a *apiSuite) TestClusterSnapshot() {
	t := a.T()
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v0/snapshots/%s", tClusterName), nil)
	w := httptest.NewRecorder()

	a.router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, a.jsonMarshal(tSnapshot), w.Body.String())
}

func (a *apiSuite) TestShardSnapshot() {
	shard := tSnapshot.ReplicaSets[0]

	t := a.T()
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v0/snapshots/%s/%s", tClusterName, shard.UUID), nil)
	w := httptest.NewRecorder()

	a.router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, a.jsonMarshal(shard), w.Body.String())
}

func (a *apiSuite) TestInstanceSnapshot() {
	shard := tSnapshot.ReplicaSets[0]
	inst := shard.Instances[0]

	t := a.T()
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v0/snapshots/%s/%s/%s", tClusterName, shard.UUID, inst.UUID), nil)
	w := httptest.NewRecorder()

	a.router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, a.jsonMarshal(inst), w.Body.String())
}

func (a *apiSuite) TestGetRecoveries() {
	t := a.T()
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v0/recoveries/%s/%s", tClusterName, tShardUUID), nil)
	w := httptest.NewRecorder()

	a.router.ServeHTTP(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, a.jsonMarshal([]orchestrator.Recovery{tRecovery}), w.Body.String())
}

func (a *apiSuite) TestGetAlerts() {
	t := a.T()
	r := httptest.NewRequest(http.MethodGet, "/api/v0/alerts", nil)
	w := httptest.NewRecorder()

	a.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, a.jsonMarshal([]api.AlertInfo{{
		ClusterName: tClusterName,
		ShardUUID:   tShardUUID,
		InstanceURI: tInstanceURI,
		Alerts:      []vshard.Alert{tAlert},
	}}), w.Body.String())
}

func (a *apiSuite) TestClusterAlerts() {
	t := a.T()
	r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v0/alerts/%s", tClusterName), nil)
	w := httptest.NewRecorder()

	a.router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, a.jsonMarshal([]api.AlertInfo{{
		ClusterName: tClusterName,
		ShardUUID:   tShardUUID,
		InstanceURI: tInstanceURI,
		Alerts:      []vshard.Alert{tAlert},
	}}), w.Body.String())
}

func (a *apiSuite) jsonMarshal(v interface{}) string {
	data, err := json.Marshal(v)
	require.NoError(a.T(), err)

	return string(data)
}

var (
	tSnapshot = vshard.Snapshot{
		Created: 117236231,
		Routers: []vshard.Router{
			{
				URI:  "",
				Info: vshard.RouterInfo{},
			},
		},
		ReplicaSets: []vshard.ReplicaSet{tReplicaSet},
	}

	tReplicaSet = vshard.ReplicaSet{
		UUID:       tShardUUID,
		MasterUUID: "ca2f08e1-bc2b-421a-b7de-e6f1fc4e6cdc",
		MasterURI:  "localhost:2021",
		Instances:  []vshard.Instance{tInstance},
	}

	tInstance = vshard.Instance{
		UUID: tInstanceUUID,
		ID:   1,
		URI:  "test_inst",
		StorageInfo: vshard.StorageInfo{
			Status:      0,
			Replication: vshard.Replication{},
			Bucket:      vshard.InstanceBucket{},
			Alerts:      []vshard.Alert{tAlert},
		},
	}

	tAlert = vshard.Alert{
		Type:        vshard.AlertUnreachableMaster,
		Description: "unreachable master alert",
	}

	tRecovery = orchestrator.Recovery{
		Type:  "test_recovery",
		Scope: "test_scope",
		AnalysisEntry: orchestrator.ReplicationAnalysis{
			CountReplicas:               3,
			CountWorkingReplicas:        2,
			CountReplicatingReplicas:    1,
			CountInconsistentVShardConf: 0,
		},
		ClusterName:  tClusterName,
		SetUUID:      tShardUUID,
		Failed:       vshard.InstanceIdent{},
		Successor:    vshard.InstanceIdent{},
		IsSuccessful: true,
		EndTimestamp: time.Now().Unix(),
	}
)
