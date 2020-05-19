package orchestrator

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

type HookType string

const (
	HookPreFailover              HookType = "PreFailover"
	HookPostSuccessfulFailover   HookType = "PostSuccessfulFailover"
	HookPostUnsuccessfulFailover HookType = "PostUnsuccessfulFailover"
)

const (
	ShellBash = "bash"
)

type Hooker struct {
	processesShellCommand string
	processes             map[HookType][]string
	timeout               time.Duration
	timeoutAsync          time.Duration
	logger                zerolog.Logger
}

func NewHooker(shell string, logger zerolog.Logger) *Hooker {
	return &Hooker{
		processesShellCommand: shell,
		processes:             make(map[HookType][]string),
		timeout:               2 * time.Second,
		timeoutAsync:          10 * time.Minute,
		logger:                logger,
	}
}

func NewBashHooker(logger zerolog.Logger) *Hooker {
	return NewHooker(ShellBash, logger)
}

// SetTimeout sets timeout for basic hook.
func (h *Hooker) SetTimeout(t time.Duration) {
	h.timeout = t
}

// SetTimeoutAsync sets timeout for async hook.
func (h *Hooker) SetTimeoutAsync(t time.Duration) {
	h.timeoutAsync = t
}

func (h *Hooker) AddHook(t HookType, commands ...string) {
	hooks, ok := h.processes[t]
	if !ok {
		hooks = make([]string, 0, len(commands))
	}
	hooks = append(hooks, commands...)
	h.processes[t] = hooks
}

// ExecuteProcesses executes a list of processes.
func (h *Hooker) ExecuteProcesses(t HookType, recv *Recovery, failOnError bool) (err error) {
	processes := h.processes[t]
	if len(processes) == 0 {
		h.logger.Info().Msgf("No %s hooks to run", t)
		return nil
	}

	h.logger.Info().Msgf("Running %d %s hooks", len(processes), t)
	for i, process := range processes {
		command, async := prepareCommand(process, recv)
		env := applyEnvironmentVariables(recv)

		fullDescription := fmt.Sprintf("%s hook %d of %d", t, i+1, len(processes))
		if async {
			fullDescription = fmt.Sprintf("%s (async)", fullDescription)
		}
		if async {
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), h.timeoutAsync)
				// Ignore errors, it is async process.
				_ = h.executeProcess(ctx, command, env, fullDescription)
				cancel()
			}()
		} else {
			ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
			cmdErr := h.executeProcess(ctx, command, env, fullDescription)
			cancel()

			if cmdErr != nil {
				if failOnError {
					h.logger.Warn().Msgf("Not running further %s hooks", t)
					return cmdErr
				}
				if err == nil {
					// Keep first error encountered.
					err = cmdErr
				}
			}
		}
	}
	h.logger.Info().Msgf("Done running %s hooks", t)

	return err
}

func (h *Hooker) executeProcess(ctx context.Context, command string, env []string, fullDescription string) error {
	// Log the command to be run and record how long it takes as this may be useful.
	h.logger.Info().Msgf("Running %s: %s", fullDescription, command)
	start := time.Now()

	cmd := exec.CommandContext(ctx, h.processesShellCommand, "-c", command) //nolint:gosec
	cmd.Env = env

	err := cmd.Run()
	if err == nil {
		h.logger.Info().Msgf("Completed %s in %v", fullDescription, time.Since(start))
	} else {
		h.logger.Error().Msgf("Execution of %s failed in %v with error: %v", fullDescription, time.Since(start), err)
	}

	return err
}

// prepareCommand replaces agreed-upon placeholders with recovery data.
func prepareCommand(command string, recv *Recovery) (result string, async bool) {
	command = strings.TrimSpace(command)
	if strings.HasPrefix(command, "&") {
		command = strings.TrimLeft(command, "&")
		async = true
	}

	analysis := recv.AnalysisEntry

	command = strings.Replace(command, "{failureType}", recv.Type, -1)
	command = strings.Replace(command, "{failedUUID}", string(recv.Failed.UUID), -1)
	command = strings.Replace(command, "{failedURI}", recv.Failed.URI, -1)
	command = strings.Replace(command, "{failureCluster}", recv.ClusterName, -1)
	command = strings.Replace(command, "{failureReplicaSetUUID}", string(recv.SetUUID), -1)
	command = strings.Replace(command, "{countFollowers}", strconv.Itoa(analysis.CountReplicas), -1)
	command = strings.Replace(command, "{countWorkingFollowers}", strconv.Itoa(analysis.CountWorkingReplicas), -1)
	command = strings.Replace(command, "{countReplicatingFollowers}", strconv.Itoa(analysis.CountReplicatingReplicas), -1)
	command = strings.Replace(command, "{countInconsistentVShardConf}", strconv.Itoa(analysis.CountInconsistentVShardConf), -1)
	command = strings.Replace(command, "{isSuccessful}", fmt.Sprint(recv.IsSuccessful), -1)

	if recv.IsSuccessful {
		command = strings.Replace(command, "{successorUUID}", string(recv.Successor.UUID), -1)
		command = strings.Replace(command, "{successorURI}", recv.Successor.URI, -1)
	}

	return command, async
}

// applyEnvironmentVariables sets the relevant environment variables for a recovery.
//nolint:gocritic
func applyEnvironmentVariables(recv *Recovery) []string {
	env := os.Environ()

	env = append(env, fmt.Sprintf("QUM_FAILURE_TYPE=%s", recv.Type))
	env = append(env, fmt.Sprintf("QUM_FAILED_UUID=%s", string(recv.Failed.UUID)))
	env = append(env, fmt.Sprintf("QUM_FAILED_URI=%s", recv.Failed.URI))
	env = append(env, fmt.Sprintf("QUM_FAILURE_CLUSTER=%s", recv.ClusterName))
	env = append(env, fmt.Sprintf("QUM_FAILURE_REPLICA_SET_UUID=%s", recv.SetUUID))
	env = append(env, fmt.Sprintf("QUM_COUNT_FOLLOWERS=%d", recv.AnalysisEntry.CountReplicas))
	env = append(env, fmt.Sprintf("QUM_COUNT_WORKING_FOLLOWERS=%d", recv.AnalysisEntry.CountWorkingReplicas))
	env = append(env, fmt.Sprintf("QUM_COUNT_REPLICATING_FOLLOWERS=%d", recv.AnalysisEntry.CountReplicatingReplicas))
	env = append(env, fmt.Sprintf("QUM_COUNT_INCONSISTENT_VSHARD_CONF=%d", recv.AnalysisEntry.CountInconsistentVShardConf))
	env = append(env, fmt.Sprintf("QUM_IS_SUCCESSFUL=%t", recv.IsSuccessful))

	if recv.IsSuccessful {
		env = append(env, fmt.Sprintf("QUM_SUCCESSOR_UUID=%s", recv.Successor.UUID))
		env = append(env, fmt.Sprintf("QUM_SUCCESSOR_URI=%s", recv.Successor.URI))
	}

	return env
}
