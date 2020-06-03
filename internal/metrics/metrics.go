package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	discoveryInstanceDurations = "instance_durations"
	discoveryClusterDurations  = "cluster_durations"
	shardCriticalLevel         = "critical_level"
	shardState                 = "state"
)

var (
	discoveryInstanceDurationsSum = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Subsystem:  "discovery",
		Name:       discoveryInstanceDurations,
		Help:       "Instance discovery latencies in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"cluster_name", "hostname"})

	discoveryClusterDurationsSum = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Subsystem:  "discovery",
		Name:       discoveryClusterDurations,
		Help:       "Cluster discovery latencies in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"cluster_name"})

	shardCriticalLevelGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "shard",
		Name:      shardCriticalLevel,
		Help:      "Critical level of the replica set",
	}, []string{"cluster_name", "uuid"})

	shardStateGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "shard",
		Name:      shardState,
		Help:      "The state of each shard in the cluster; it will have one line for each possible state of each shard. A value of 1 means the shard is in the state specified by the state label, a value of 0 means it is not.",
	}, []string{"cluster_name", "uuid", "state"})
)

func init() {
	prometheus.MustRegister(discoveryInstanceDurationsSum)
	prometheus.MustRegister(discoveryClusterDurationsSum)
	prometheus.MustRegister(shardCriticalLevelGauge)
	prometheus.MustRegister(shardStateGauge)
}

type Transaction interface {
	Start() Transaction
	End()
}

type timeTransaction struct {
	labels  []string
	summary *prometheus.SummaryVec
	timer   *prometheus.Timer
}

func (txn *timeTransaction) Start() Transaction {
	txn.timer = prometheus.NewTimer(txn.summary.WithLabelValues(txn.labels...))
	return txn
}

func (txn *timeTransaction) End() {
	txn.timer.ObserveDuration()
}

func StartInstanceDiscovery(clusterName, hostname string) Transaction {
	txn := &timeTransaction{
		summary: discoveryInstanceDurationsSum,
		labels:  []string{clusterName, hostname},
	}
	return txn.Start()
}

func StartClusterDiscovery(clusterName string) Transaction {
	txn := &timeTransaction{
		summary: discoveryClusterDurationsSum,
		labels:  []string{clusterName},
	}
	return txn.Start()
}

func SetShardCriticalLevel(clusterName, uuid string, level int) {
	shardCriticalLevelGauge.WithLabelValues(clusterName, uuid).Set(float64(level))
}

func SetShardState(clusterName, uuid, state string, active bool) {
	v := float64(0)
	if active {
		v = 1
	}
	shardStateGauge.With(prometheus.Labels{
		"cluster_name": clusterName,
		"uuid":         uuid,
		"state":        state,
	}).Set(v)
}
