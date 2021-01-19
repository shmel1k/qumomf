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
	ClustersList(context.Context) ([]ClusterInfo, error)
	ClusterSnapshot(context.Context, string) (vshard.Snapshot, error)
	ReplicaSet(context.Context, string, vshard.ReplicaSetUUID) (vshard.ReplicaSet, error)
	Instance(context.Context, string, vshard.ReplicaSetUUID, vshard.InstanceUUID) (vshard.Instance, error)
	Recoveries(context.Context, string, vshard.ReplicaSetUUID) ([]orchestrator.Recovery, error)
	Alerts(context.Context) (AlertsResponse, error)
	ClusterAlerts(context.Context, string) (AlertsResponse, error)
}

func NewService(db storage.Storage) Service {
	return &service{
		db: db,
	}
}

type service struct {
	db storage.Storage
}

func (s *service) ClustersList(ctx context.Context) ([]ClusterInfo, error) {
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

func (s *service) ClusterSnapshot(ctx context.Context, clusterName string) (vshard.Snapshot, error) {
	snap, err := s.db.GetClusterSnapshot(ctx, clusterName)
	if err == sqlite.ErrEmptyResult {
		return vshard.Snapshot{}, ErrEmptyResult
	}

	return snap, err
}

func (s *service) ReplicaSet(ctx context.Context, clusterName string, replicaSetUUID vshard.ReplicaSetUUID) (vshard.ReplicaSet, error) {
	snap, err := s.db.GetClusterSnapshot(ctx, clusterName)
	if err != nil {
		if err == sqlite.ErrEmptyResult {
			return vshard.ReplicaSet{}, ErrEmptyResult
		}
		return vshard.ReplicaSet{}, err
	}

	replicaSet, err := snap.ReplicaSet(replicaSetUUID)
	if err != nil {
		if err == vshard.ErrReplicaSetNotFound {
			return vshard.ReplicaSet{}, ErrEmptyResult
		}

		return vshard.ReplicaSet{}, err
	}

	return replicaSet, nil
}

func (s *service) Instance(ctx context.Context, clusterName string, replicaSetUUID vshard.ReplicaSetUUID, instanceUUID vshard.InstanceUUID) (vshard.Instance, error) {
	replicaSet, err := s.ReplicaSet(ctx, clusterName, replicaSetUUID)
	if err != nil {
		return vshard.Instance{}, err
	}

	for i := range replicaSet.Instances {
		if replicaSet.Instances[i].UUID == instanceUUID {
			return replicaSet.Instances[i], nil
		}
	}

	return vshard.Instance{}, ErrEmptyResult
}

func (s *service) Recoveries(ctx context.Context, clusterName string, replicaSetUUID vshard.ReplicaSetUUID) ([]orchestrator.Recovery, error) {
	recoveries, err := s.db.GetRecoveries(ctx, clusterName)
	if err != nil {
		return nil, err
	}

	resp := make([]orchestrator.Recovery, 0, len(recoveries))
	for i := range recoveries {
		if recoveries[i].SetUUID == replicaSetUUID {
			resp = append(resp, recoveries[i])
		}
	}

	return resp, nil
}

func (s *service) Alerts(ctx context.Context) (AlertsResponse, error) {
	clusters, err := s.db.GetClusters(ctx)
	if err != nil {
		return AlertsResponse{}, err
	}

	instanceAlertsList := make([]InstanceAlerts, 0)
	routerAlertsList := make([]RoutersAlerts, 0)
	for i := range clusters {
		routerAlertsList = append(routerAlertsList, routersAlerts(clusters[i].Snapshot.Routers)...)
		instanceAlertsList = append(instanceAlertsList, instanceAlerts(clusters[i].Name, clusters[i].Snapshot.ReplicaSets)...)
	}

	return AlertsResponse{
		InstancesAlerts: instanceAlertsList,
		RoutersAlerts:   routerAlertsList,
	}, nil
}

func (s *service) ClusterAlerts(ctx context.Context, clusterName string) (AlertsResponse, error) {
	cluster, err := s.ClusterSnapshot(ctx, clusterName)
	if err != nil {
		return AlertsResponse{}, err
	}

	return AlertsResponse{
		InstancesAlerts: instanceAlerts(clusterName, cluster.ReplicaSets),
		RoutersAlerts:   routersAlerts(cluster.Routers),
	}, nil
}

func routersAlerts(routers []vshard.Router) []RoutersAlerts {
	result := make([]RoutersAlerts, 0)
	for i := range routers {
		if len(routers[i].Info.Alerts) > 0 {
			result = append(result, RoutersAlerts{
				URI:    routers[i].URI,
				Alerts: routers[i].Info.Alerts,
			})
		}
	}

	return result
}

func instanceAlerts(clusterName string, replicaSets []vshard.ReplicaSet) []InstanceAlerts {
	resp := make([]InstanceAlerts, 0)

	for i := range replicaSets {
		instances := replicaSets[i].Instances
		for j := range instances {
			alerts := instances[j].StorageInfo.Alerts
			if len(alerts) != 0 {
				resp = append(resp, InstanceAlerts{
					ClusterName: clusterName,
					ShardUUID:   replicaSets[i].UUID,
					InstanceURI: instances[j].URI,
					Alerts:      alerts,
				})
			}
		}
	}

	return resp
}
