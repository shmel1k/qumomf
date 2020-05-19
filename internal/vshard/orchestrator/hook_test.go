package orchestrator

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/shmel1k/qumomf/internal/vshard"
)

type hookerTestSuite struct {
	suite.Suite

	failed   vshard.InstanceIdent
	analysis *ReplicationAnalysis
	recv     *Recovery

	logger zerolog.Logger
}

func (s *hookerTestSuite) SetupTest() {
	s.analysis = mockAnalysis
	s.failed = vshard.InstanceIdent{
		UUID: s.analysis.Set.MasterUUID,
		URI:  "localhost:8080",
	}
	s.recv = NewRecovery(RecoveryScopeSet, s.failed, *s.analysis)
	s.recv.ClusterName = "sandbox"
	s.logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Logger()
}

func TestHooker(t *testing.T) {
	suite.Run(t, &hookerTestSuite{})
}

func (s *hookerTestSuite) Test_ExecuteProcesses() {
	t := s.T()

	env := []string{
		fmt.Sprintf("QUM_FAILURE_TYPE=%s", s.analysis.State),
		fmt.Sprintf("QUM_FAILED_UUID=%s", s.failed.UUID),
		fmt.Sprintf("QUM_FAILED_URI=%s", s.failed.URI),
		fmt.Sprintf("QUM_FAILURE_CLUSTER=%s", s.recv.ClusterName),
		fmt.Sprintf("QUM_FAILURE_REPLICA_SET_UUID=%s", s.analysis.Set.UUID),
		fmt.Sprintf("QUM_COUNT_FOLLOWERS=%d", s.analysis.CountReplicas),
		fmt.Sprintf("QUM_COUNT_WORKING_FOLLOWERS=%d", s.analysis.CountWorkingReplicas),
		fmt.Sprintf("QUM_COUNT_REPLICATING_FOLLOWERS=%d", s.analysis.CountReplicatingReplicas),
		fmt.Sprintf("QUM_COUNT_INCONSISTENT_VSHARD_CONF=%d", s.analysis.CountInconsistentVShardConf),
		fmt.Sprintf("IS_SUCCESSFUL=%t", s.recv.IsSuccessful),
	}

	hooker := NewBashHooker(s.logger)

	filename := genUniqueFilename(os.TempDir(), "qumomf-hook-test")
	require.NotEmpty(t, filename)
	defer func() {
		_ = os.Remove(filename)
	}()

	hooker.AddHook(HookPreFailover, fmt.Sprintf("touch %s", filename))
	hooker.AddHook(HookPreFailover, fmt.Sprintf("echo $(printenv | grep QUM) >> %s", filename))

	hooker.AddHook(HookPostSuccessfulFailover, fmt.Sprintf("rm -f %s", filename))

	err := hooker.ExecuteProcesses(HookPreFailover, s.recv, true)
	require.Nil(t, err)

	f, err := os.Open(filename)
	require.Nil(t, err)
	defer func() { _ = f.Close() }()

	foundEnv := make([]string, 0, len(env))
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for _, e := range env {
			if strings.Contains(line, e) {
				foundEnv = append(foundEnv, e)
			}
		}
	}

	assert.Equal(t, env, foundEnv)

	err = hooker.ExecuteProcesses(HookPostSuccessfulFailover, s.recv, false)
	assert.Nil(t, err)
}

func (s *hookerTestSuite) Test_ExecuteProcesses_Async() {
	t := s.T()

	hooker := NewBashHooker(s.logger)

	start := time.Now()
	hooker.AddHook(HookPreFailover, "&sleep 3")
	err := hooker.ExecuteProcesses(HookPreFailover, s.recv, true)
	end := time.Now()
	assert.Nil(t, err)
	assert.WithinDuration(t, start, end, 1*time.Second)
}

func (s *hookerTestSuite) Test_ExecuteProcesses_CheckArguments() {
	t := s.T()

	s.recv.IsSuccessful = true
	s.recv.Successor = vshard.InstanceIdent{
		UUID: "successor_uuid",
		URI:  "successor_uri",
	}

	args := []string{
		"failureType",
		"failedUUID",
		"failedURI",
		"failureCluster",
		"failureReplicaSetUUID",
		"countFollowers",
		"countWorkingFollowers",
		"countReplicatingFollowers",
		"countInconsistentVShardConf",
		"isSuccessful",
		"successorUUID",
		"successorURI",
	}
	expectedArgs := []string{
		fmt.Sprintf("failureType=%s", s.analysis.State),
		fmt.Sprintf("failedUUID=%s", s.failed.UUID),
		fmt.Sprintf("failedURI=%s", s.failed.URI),
		fmt.Sprintf("failureCluster=%s", s.recv.ClusterName),
		fmt.Sprintf("failureReplicaSetUUID=%s", s.analysis.Set.UUID),
		fmt.Sprintf("countFollowers=%d", s.analysis.CountReplicas),
		fmt.Sprintf("countWorkingFollowers=%d", s.analysis.CountWorkingReplicas),
		fmt.Sprintf("countReplicatingFollowers=%d", s.analysis.CountReplicatingReplicas),
		fmt.Sprintf("countInconsistentVShardConf=%d", s.analysis.CountInconsistentVShardConf),
		fmt.Sprintf("isSuccessful=%t", s.recv.IsSuccessful),
		fmt.Sprintf("successorUUID=%s", s.recv.Successor.UUID),
		fmt.Sprintf("successorURI=%s", s.recv.Successor.URI),
	}

	hooker := NewBashHooker(s.logger)

	filename := genUniqueFilename(os.TempDir(), "qumomf-hook-test")
	require.NotEmpty(t, filename)
	defer func() {
		_ = os.Remove(filename)
	}()

	hooker.AddHook(HookPreFailover, fmt.Sprintf("touch %s", filename))
	for _, arg := range args {
		hooker.AddHook(HookPreFailover, fmt.Sprintf("echo '%s={%s}' >> %s", arg, arg, filename))
	}
	hooker.AddHook(HookPostSuccessfulFailover, fmt.Sprintf("rm -f %s", filename))

	err := hooker.ExecuteProcesses(HookPreFailover, s.recv, true)
	require.Nil(t, err)

	f, err := os.Open(filename)
	require.Nil(t, err)
	defer func() { _ = f.Close() }()

	foundArgs := make([]string, 0, len(expectedArgs))
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		for _, e := range expectedArgs {
			if strings.Contains(line, e) {
				foundArgs = append(foundArgs, e)
			}
		}
	}

	assert.Equal(t, expectedArgs, foundArgs)

	err = hooker.ExecuteProcesses(HookPostSuccessfulFailover, s.recv, false)
	assert.Nil(t, err)
}

func genUniqueFilename(dir, prefix string) string {
	name := ""
	rand := uint32(0)
	for i := 0; i < 1000; i++ {
		name = path.Join(dir, prefix+nextRandom(&rand))
		_, err := os.Stat(name)
		if os.IsExist(err) {
			continue
		}
		break
	}
	return name
}

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextRandom(rand *uint32) string {
	r := *rand
	if r == 0 {
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	*rand = r

	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}
