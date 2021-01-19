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

	"github.com/shmel1k/qumomf/internal/api"
	"github.com/shmel1k/qumomf/internal/storage/sqlite"
	"github.com/shmel1k/qumomf/internal/vshard"
	"github.com/shmel1k/qumomf/internal/vshard/orchestrator"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	tDBFileName                                 = "test.db"
	tClusterName                                = "test_cluster"
	tNotFoundCluster                            = "not_found_cluster"
	tShardUUID            vshard.ReplicaSetUUID = "7c652540-2d9c-4eb1-8473-a41ec7ab3554"
	tNotFoundShardUUID    vshard.ReplicaSetUUID = "2e353da3-0170-497b-b502-c94a1c1ed251"
	tInstanceUUID         vshard.InstanceUUID   = "11a6a15d-1ddd-4d10-af53-d489774b6ad6"
	tNotFoundInstanceUUID vshard.InstanceUUID   = "cd44ae9e-3655-4e6e-89c8-716c9c2bee8a"
	tInstanceURI                                = "test_inst"
	tRouterURI                                  = "test_router_uri"
)
var (
	dummyLogger  = zerolog.New(nil)
	dummyContext = context.Background()
)

var (
	tSnapshot = vshard.Snapshot{
		Created:     117236231,
		Routers:     []vshard.Router{tRouter},
		ReplicaSets: []vshard.ReplicaSet{tReplicaSet},
	}

	tRouter = vshard.Router{
		URI: tRouterURI,
		Info: vshard.RouterInfo{
			Alerts: []vshard.Alert{tAlert},
		},
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

type testCase struct {
	name             string
	clusterName      string
	shardUUID        vshard.ReplicaSetUUID
	instanceUUID     vshard.InstanceUUID
	expectedCode     int
	expectedResponse string
}

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
	for _, tt := range []testCase{
		{
			name:             "Success_case",
			clusterName:      tClusterName,
			expectedCode:     http.StatusOK,
			expectedResponse: a.jsonMarshal(tSnapshot),
		},
		{
			name:             "Not_found_cluster",
			clusterName:      tNotFoundCluster,
			expectedCode:     http.StatusBadRequest,
			expectedResponse: "cluster snapshot not found",
		},
	} {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v0/snapshots/%s", tc.clusterName), nil)
			w := httptest.NewRecorder()

			a.router.ServeHTTP(w, r)
			assert.Equal(t, tc.expectedCode, w.Code)
			assert.Equal(t, tc.expectedResponse, w.Body.String())
		})
	}
}

func (a *apiSuite) TestShardSnapshot() {
	t := a.T()
	for _, tt := range []testCase{
		{
			name:             "Success_case",
			clusterName:      tClusterName,
			shardUUID:        tShardUUID,
			expectedCode:     http.StatusOK,
			expectedResponse: a.jsonMarshal(tSnapshot.ReplicaSets[0]),
		},
		{
			name:             "Not_found_cluster",
			clusterName:      tNotFoundCluster,
			shardUUID:        tShardUUID,
			expectedCode:     http.StatusBadRequest,
			expectedResponse: "cluster or shard snapshots not found",
		},
		{
			name:             "Not_found_shard",
			clusterName:      tClusterName,
			shardUUID:        tNotFoundShardUUID,
			expectedCode:     http.StatusBadRequest,
			expectedResponse: "cluster or shard snapshots not found",
		},
	} {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v0/snapshots/%s/%s", tc.clusterName, tc.shardUUID), nil)
			w := httptest.NewRecorder()

			a.router.ServeHTTP(w, r)
			assert.Equal(t, tc.expectedCode, w.Code)
			assert.Equal(t, tc.expectedResponse, w.Body.String())
		})
	}
}

func (a *apiSuite) TestInstanceSnapshot() {
	t := a.T()

	for _, tt := range []testCase{
		{
			name:             "Success_case",
			clusterName:      tClusterName,
			shardUUID:        tShardUUID,
			instanceUUID:     tInstanceUUID,
			expectedCode:     http.StatusOK,
			expectedResponse: a.jsonMarshal(tInstance),
		},
		{
			name:             "Not_found_cluster",
			clusterName:      tNotFoundCluster,
			shardUUID:        tShardUUID,
			instanceUUID:     tInstanceUUID,
			expectedCode:     http.StatusBadRequest,
			expectedResponse: "cluster, shard or instance snapshots not found",
		},
		{
			name:             "Not_found_shard",
			clusterName:      tClusterName,
			shardUUID:        tNotFoundShardUUID,
			instanceUUID:     tInstanceUUID,
			expectedCode:     http.StatusBadRequest,
			expectedResponse: "cluster, shard or instance snapshots not found",
		},
		{
			name:             "Not_found_instance",
			clusterName:      tClusterName,
			shardUUID:        tShardUUID,
			instanceUUID:     tNotFoundInstanceUUID,
			expectedCode:     http.StatusBadRequest,
			expectedResponse: "cluster, shard or instance snapshots not found",
		},
	} {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v0/snapshots/%s/%s/%s", tc.clusterName, tc.shardUUID, tc.instanceUUID), nil)
			w := httptest.NewRecorder()

			a.router.ServeHTTP(w, r)
			assert.Equal(t, tc.expectedCode, w.Code)
			assert.Equal(t, tc.expectedResponse, w.Body.String())
		})
	}
}

func (a *apiSuite) TestGetRecoveries() {
	t := a.T()

	for _, tt := range []testCase{
		{
			name:             "Success_case",
			clusterName:      tClusterName,
			shardUUID:        tShardUUID,
			expectedCode:     http.StatusOK,
			expectedResponse: a.jsonMarshal([]orchestrator.Recovery{tRecovery}),
		},
		{
			name:             "Not_found_cluster_Expected_empty_result",
			clusterName:      tNotFoundCluster,
			shardUUID:        tShardUUID,
			expectedCode:     http.StatusOK,
			expectedResponse: a.jsonMarshal([]orchestrator.Recovery{}),
		},
		{
			name:             "Not_found_shard_Expected_empty_result",
			clusterName:      tClusterName,
			shardUUID:        tNotFoundShardUUID,
			expectedCode:     http.StatusOK,
			expectedResponse: a.jsonMarshal([]orchestrator.Recovery{}),
		},
	} {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v0/recoveries/%s/%s", tc.clusterName, tc.shardUUID), nil)
			w := httptest.NewRecorder()

			a.router.ServeHTTP(w, r)
			assert.Equal(t, tc.expectedCode, w.Code)
			assert.Equal(t, tc.expectedResponse, w.Body.String())
		})
	}
}

func (a *apiSuite) TestGetAlerts() {
	t := a.T()
	for _, tt := range []testCase{
		{
			name:         "Success_case",
			expectedCode: http.StatusOK,
			expectedResponse: a.jsonMarshal(api.AlertsResponse{
				InstancesAlerts: []api.InstanceAlerts{{
					ClusterName: tClusterName,
					ShardUUID:   tShardUUID,
					InstanceURI: tInstanceURI,
					Alerts:      []vshard.Alert{tAlert},
				}},
				RoutersAlerts: []api.RoutersAlerts{
					{
						URI:    tRouterURI,
						Alerts: []vshard.Alert{tAlert},
					},
				},
			}),
		},
	} {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/api/v0/alerts", nil)
			w := httptest.NewRecorder()

			a.router.ServeHTTP(w, r)

			assert.Equal(t, http.StatusOK, w.Code)
			assert.Equal(t, tc.expectedResponse, w.Body.String())
		})
	}
}

func (a *apiSuite) TestClusterAlerts() {
	t := a.T()
	for _, tt := range []testCase{
		{
			name:        "Success_case",
			clusterName: tClusterName,
			expectedResponse: a.jsonMarshal(api.AlertsResponse{
				InstancesAlerts: []api.InstanceAlerts{{
					ClusterName: tClusterName,
					ShardUUID:   tShardUUID,
					InstanceURI: tInstanceURI,
					Alerts:      []vshard.Alert{tAlert},
				}},
				RoutersAlerts: []api.RoutersAlerts{
					{
						URI:    tRouterURI,
						Alerts: []vshard.Alert{tAlert},
					},
				},
			}),
			expectedCode: http.StatusOK,
		},
		{
			name:             "Not_found_cluster_Expected_empty_result",
			clusterName:      tNotFoundCluster,
			expectedResponse: "cluster not found",
			expectedCode:     http.StatusBadRequest,
		},
	} {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v0/alerts/%s", tc.clusterName), nil)
			w := httptest.NewRecorder()

			a.router.ServeHTTP(w, r)

			assert.Equal(t, tc.expectedCode, w.Code)
			assert.Equal(t, tc.expectedResponse, w.Body.String())
		})
	}
}

func (a *apiSuite) jsonMarshal(v interface{}) string {
	data, err := json.Marshal(v)
	require.NoError(a.T(), err)

	return string(data)
}
