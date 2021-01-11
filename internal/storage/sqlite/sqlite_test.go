package sqlite

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/shmel1k/qumomf/internal/storage"
	"github.com/shmel1k/qumomf/internal/vshard"
	"github.com/shmel1k/qumomf/internal/vshard/orchestrator"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	tFileName    = "tFileName.db"
	tClusterName = "testCluster"
	tSnapshot    = vshard.Snapshot{
		Created:     123,
		Routers:     []vshard.Router{},
		ReplicaSets: []vshard.ReplicaSet{},
	}
	tRecovery = orchestrator.Recovery{
		Type:        "test type",
		ClusterName: tClusterName,
	}
)

var (
	dummyContext = context.Background()
)

type storageSuite struct {
	suite.Suite
	db storage.Storage
}

func TestStorage(t *testing.T) {
	suite.Run(t, &storageSuite{
		Suite: suite.Suite{},
	})
}

func (s *storageSuite) BeforeTest(_, _ string) {
	t := s.T()

	db, err := NewSQLiteStorage(Config{
		FileName:       tFileName,
		ConnectTimeout: 3 * time.Second,
		QueryTimeout:   3 * time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, db)

	s.db = db
}

func (s *storageSuite) AfterTest(_, _ string) {
	err := os.Remove(tFileName)
	require.NoError(s.T(), err)
}

func (s *storageSuite) TestEmptyResult() {
	t := s.T()
	_, err := s.db.GetClusterLastSnapshot(dummyContext, tClusterName)
	require.Equal(t, ErrEmptyResult, err)
}

func (s *storageSuite) TestSaveSnapshot() {
	t := s.T()
	err := s.db.SaveSnapshot(dummyContext, tClusterName, tSnapshot)
	require.NoError(t, err)

	snap, err := s.db.GetClusterLastSnapshot(dummyContext, tClusterName)
	require.NoError(t, err)
	require.Equal(t, tSnapshot, snap)
}

func (s *storageSuite) TestSaveRecovery() {
	t := s.T()
	err := s.db.SaveRecovery(dummyContext, tRecovery)
	require.NoError(t, err)

	results, err := s.db.GetRecoveries(dummyContext, tClusterName)
	require.NoError(t, err)
	require.Equal(t, []orchestrator.Recovery{tRecovery}, results)
}
