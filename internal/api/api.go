package api

import (
	"context"
	"errors"

	"github.com/shmel1k/qumomf/internal/storage"
	"github.com/shmel1k/qumomf/internal/storage/sqlite"
	"github.com/shmel1k/qumomf/internal/vshard"
	"github.com/shmel1k/qumomf/internal/vshard/orchestrator"
)

var (
	ErrEmptyResult = errors.New("empty result")
)

type Service interface {
	GetClustersList(context.Context) ([]ClusterInfo, error)
	GetCluster(context.Context, string) (vshard.Snapshot, error)
	GetShard(context.Context, string, vshard.ReplicaSetUUID) (vshard.ReplicaSet, error)
	GetInstance(context.Context, string, vshard.ReplicaSetUUID, vshard.InstanceUUID) (vshard.Instance, error)
	GetRecoveries(context.Context, string, vshard.ReplicaSetUUID) ([]orchestrator.Recovery, error)
	Alerts(context.Context) ([]AlertInfo, error)
	ClusterAlerts(context.Context, string) ([]AlertInfo, error)
}

func NewService(db storage.Storage) Service {
	return &service{
		db: db,
	}
}

type service struct {
	db storage.Storage
}

func (s *service) GetClustersList(ctx context.Context) ([]ClusterInfo, error) {
	clustersList, err := s.db.GetClusters(ctx)
	if err != nil {
		return nil, err
	}

	resp := make([]ClusterInfo, 0, len(clustersList))
	for _, cluster := range clustersList {
		resp = append(resp, ClusterInfo{
			Name:         cluster.Name,
			ShardsCount:  len(cluster.Snapshot.ReplicaSets),
			RoutersCount: len(cluster.Snapshot.Routers),
		})
	}

	return resp, nil
}

func (s *service) GetCluster(ctx context.Context, clusterName string) (vshard.Snapshot, error) {
	snap, err := s.db.GetClusterSnapshot(ctx, clusterName)
	if err == sqlite.ErrEmptyResult {
		return vshard.Snapshot{}, ErrEmptyResult
	}

	return snap, err
}

func (s *service) GetShard(ctx context.Context, clusterName string, shardUUID vshard.ReplicaSetUUID) (vshard.ReplicaSet, error) {
	snap, err := s.db.GetClusterSnapshot(ctx, clusterName)
	if err != nil {
		if err == sqlite.ErrEmptyResult {
			return vshard.ReplicaSet{}, ErrEmptyResult
		}
		return vshard.ReplicaSet{}, err
	}

	shard, err := snap.ReplicaSet(shardUUID)
	if err != nil {
		if err == vshard.ErrReplicaSetNotFound {
			return vshard.ReplicaSet{}, ErrEmptyResult
		}

		return vshard.ReplicaSet{}, err
	}

	return shard, nil
}

func (s *service) GetInstance(ctx context.Context, clusterName string, shardUUID vshard.ReplicaSetUUID, instanceUUID vshard.InstanceUUID) (vshard.Instance, error) {
	shard, err := s.GetShard(ctx, clusterName, shardUUID)
	if err != nil {
		return vshard.Instance{}, err
	}

	for i := range shard.Instances {
		if shard.Instances[i].UUID == instanceUUID {
			return shard.Instances[i], nil
		}
	}

	return vshard.Instance{}, ErrEmptyResult
}

func (s *service) GetRecoveries(ctx context.Context, clusterName string, shardUUID vshard.ReplicaSetUUID) ([]orchestrator.Recovery, error) {
	recoveries, err := s.db.GetRecoveries(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	resp := make([]orchestrator.Recovery, 0, len(recoveries))
	for i := range recoveries {
		if recoveries[i].SetUUID == shardUUID {
			resp = append(resp, recoveries[i])
		}
	}

	return resp, nil
}

func (s *service) Alerts(ctx context.Context) ([]AlertInfo, error) {
	clusters, err := s.db.GetClusters(ctx)
	if err != nil {
		return nil, err
	}

	resp := make([]AlertInfo, 0)
	for i := range clusters {
		shard := clusters[i].Snapshot.ReplicaSets
		for j := range shard {
			instances := shard[j].Instances
			for k := range instances {
				alerts := instances[k].StorageInfo.Alerts
				resp = append(resp, AlertInfo{
					ClusterName: clusters[i].Name,
					ShardUUID:   shard[j].UUID,
					InstanceURI: instances[k].URI,
					Alerts:      alerts,
				})
			}
		}
	}

	return resp, nil
}

func (s *service) ClusterAlerts(ctx context.Context, clusterName string) ([]AlertInfo, error) {
	cluster, err := s.GetCluster(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	resp := make([]AlertInfo, 0)
	shards := cluster.ReplicaSets
	for i := range shards {
		instances := shards[i].Instances
		for j := range instances {
			alerts := instances[j].StorageInfo.Alerts
			resp = append(resp, AlertInfo{
				ClusterName: clusterName,
				ShardUUID:   shards[i].UUID,
				InstanceURI: instances[j].URI,
				Alerts:      alerts,
			})
		}
	}

	return resp, nil
}
