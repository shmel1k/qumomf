package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var (
	tFileName    = "tFileName.db"
	tClusterName = "testCluster"
	tData        = []byte(`test data`)
	tSaveRequest = SaveRequest{
		ClusterName: tClusterName,
		CreatedAt:   time.Now().Unix(),
		Data:        tData,
	}
)

var (
	dummyContext = context.Background()
)

type storageSuite struct {
	suite.Suite
	relStorage Storage
}

func TestStorage(t *testing.T) {
	suite.Run(t, &storageSuite{
		Suite: suite.Suite{},
	})
}

func (s *storageSuite) BeforeTest(_, _ string) {
	t := s.T()

	relStorage, err := NewStorage(Config{
		FileName:       tFileName,
		ConnectTimeout: 3 * time.Second,
		QueryTimeout:   3 * time.Second,
	})
	require.NoError(t, err)
	require.NotNil(t, relStorage)

	s.relStorage = relStorage
}

func (s *storageSuite) AfterTest(_, _ string) {
	err := os.Remove(tFileName)
	require.NoError(s.T(), err)
}

func (s *storageSuite) TestEmptyResult() {
	t := s.T()
	_, err := s.relStorage.GetClusterLastSnapshot(dummyContext, tClusterName)
	require.Equal(t, ErrEmptyResult, err)
}

func (s *storageSuite) TestSaveSnapshot() {
	t := s.T()
	err := s.relStorage.SaveSnapshot(dummyContext, tSaveRequest)
	require.NoError(t, err)

	data, err := s.relStorage.GetClusterLastSnapshot(dummyContext, tClusterName)
	require.NoError(t, err)
	require.Equal(t, tData, data)
}

func (s *storageSuite) TestSaveRecovery() {
	t := s.T()
	err := s.relStorage.SaveRecovery(dummyContext, tSaveRequest)
	require.NoError(t, err)

	results, err := s.relStorage.GetRecoveries(dummyContext, tClusterName)
	require.NoError(t, err)
	require.Equal(t, [][]byte{tData}, results)
}
