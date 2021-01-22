package storage

import "github.com/shmel1k/qumomf/internal/vshard"

type ClusterSnapshotResp struct {
	Name     string
	Snapshot vshard.Snapshot
}
