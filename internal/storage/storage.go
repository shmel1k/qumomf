package storage

import (
	"context"

	"github.com/shmel1k/qumomf/internal/vshard"
	"github.com/shmel1k/qumomf/internal/vshard/orchestrator"
)

type Storage interface {
	SaveSnapshot(context.Context, string, vshard.Snapshot) error
	SaveRecovery(context.Context, orchestrator.Recovery) error
	GetClusterLastSnapshot(context.Context, string) (vshard.Snapshot, error)
	GetRecoveries(context.Context, string) ([]orchestrator.Recovery, error)
}
