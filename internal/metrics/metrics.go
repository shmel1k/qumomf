package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	discoveryInstanceDurations = "instance_durations"
	discoveryClusterDurations  = "cluster_durations"
	shardCriticalLevel         = "critical_level"
	shardState                 = "state"
)

var (
	discoveryInstanceDurationsBuckets = prometheus.ExponentialBuckets(.001, 2.5, 10)
	discoveryClusterDurationsBuckets  = prometheus.ExponentialBuckets(.001, 2.5, 10)
)

var (
	discoveryInstanceDurationsSum = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: "discovery",
		Name:      discoveryInstanceDurations,
		Help:      "Instance discovery latencies in seconds",
		Buckets:   discoveryInstanceDurationsBuckets,
	}, []string{"cluster_name", "hostname"})

	discoveryClusterDurationsSum = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Subsystem: "discovery",
		Name:      discoveryClusterDurations,
		Help:      "Cluster discovery latencies in seconds",
		Buckets:   discoveryClusterDurationsBuckets,
	}, []string{"cluster_name"})

	shardCriticalLevelGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "shard",
		Name:      shardCriticalLevel,
		Help:      "Critical level of the replica set",
	}, []string{"cluster_name", "uuid", "master_uri"})

	shardStateGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "shard",
		Name:      shardState,
		Help:      "The state of each shard in the cluster; it will have one line for each possible state of each shard. A value of 1 means the shard is in the state specified by the state label, a value of 0 means it is not.",
	}, []string{"cluster_name", "uuid", "master_uri", "state"})

	discoveryErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "discovery",
		Name:      "errors",
		Help:      "Errors that happen during discovery process",
	}, []string{"cluster_name", "uri"})
)

func init() {
	discoveryErrors.With(prometheus.Labels{"cluster_name": "", "uri": ""}).Add(0)
	prometheus.MustRegister(
		discoveryInstanceDurationsSum,
		discoveryClusterDurationsSum,
		shardCriticalLevelGauge,
		shardStateGauge,
		discoveryErrors,
	)
}

type Transaction interface {
	Start() Transaction
	End()
}

type timeTransaction struct {
	labels  []string
	summary *prometheus.HistogramVec
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

func SetShardCriticalLevel(clusterName, uuid, masterURI string, level int) {
	shardCriticalLevelGauge.WithLabelValues(clusterName, uuid, masterURI).Set(float64(level))
}

func SetShardState(clusterName, uuid, masterURI, state string, active bool) {
	v := float64(0)
	if active {
		v = 1
	}
	shardStateGauge.With(prometheus.Labels{
		"cluster_name": clusterName,
		"uuid":         uuid,
		"master_uri":   masterURI,
		"state":        state,
	}).Set(v)
}

func RecordDiscoveryError(clusterName, uri string) {
	discoveryErrors.With(prometheus.Labels{
		"cluster_name": clusterName,
		"uri":          uri,
	}).Inc()
}
