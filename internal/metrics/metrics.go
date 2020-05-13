package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	discoveryInstanceDurations = "instance_durations"
	discoveryInstanceFailures  = "instance_failures"
	discoveryClusterDurations  = "cluster_durations"
	discoveryClusterFailures   = "cluster_failures"
	shardCriticalLevel         = "critical_level"
	totalRecoveryAttempts      = "attempt_count"
)

var (
	discoveryInstanceDurationsSum = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Subsystem:  "discovery",
		Name:       discoveryInstanceDurations,
		Help:       "Instance discovery latencies in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"hostname"})

	discoveryInstanceFailuresCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "discovery",
		Name:      discoveryInstanceFailures,
		Help:      "Total number of failed instance discoveries",
	}, []string{"hostname"})

	discoveryClusterDurationsSum = prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Subsystem:  "discovery",
		Name:       discoveryClusterDurations,
		Help:       "Cluster discovery latencies in seconds",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	}, []string{"name"})

	discoveryClusterFailuresCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "discovery",
		Name:      discoveryClusterFailures,
		Help:      "Total number of failed cluster discoveries",
	}, []string{"name"})

	shardCriticalLevelGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Subsystem: "shard",
		Name:      shardCriticalLevel,
		Help:      "Critical level of the replica set",
	}, []string{"uuid"})

	recoveryAttemptsCnt = prometheus.NewCounterVec(prometheus.CounterOpts{
		Subsystem: "recovery",
		Name:      totalRecoveryAttempts,
		Help:      "Total number of recovery attempts",
	}, []string{"uuid", "reason", "success"})
)

func init() {
	prometheus.MustRegister(discoveryInstanceDurationsSum)
	prometheus.MustRegister(discoveryInstanceFailuresCnt)
	prometheus.MustRegister(discoveryClusterDurationsSum)
	prometheus.MustRegister(discoveryClusterFailuresCnt)
	prometheus.MustRegister(shardCriticalLevelGauge)
	prometheus.MustRegister(recoveryAttemptsCnt)
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

func StartInstanceDiscovery(hostname string) Transaction {
	txn := &timeTransaction{
		summary: discoveryInstanceDurationsSum,
		labels:  []string{hostname},
	}
	return txn.Start()
}

func NewFailedInstanceDiscoveryAttempt(hostname string) {
	discoveryInstanceFailuresCnt.WithLabelValues(hostname).Inc()
}

func StartClusterDiscovery(name string) Transaction {
	txn := &timeTransaction{
		summary: discoveryClusterDurationsSum,
		labels:  []string{name},
	}
	return txn.Start()
}

func NewFailedClusterDiscoveryAttempt(name string) {
	discoveryClusterFailuresCnt.WithLabelValues(name).Inc()
}

func SetShardCriticalLevel(uuid string, level int) {
	shardCriticalLevelGauge.WithLabelValues(uuid).Set(float64(level))
}

func NewRecoveryAttempt(uuid, reason string, success bool) {
	successValue := "0"
	if success {
		successValue = "1"
	}
	recoveryAttemptsCnt.With(prometheus.Labels{
		"uuid":    uuid,
		"reason":  reason,
		"success": successValue,
	}).Inc()
}
