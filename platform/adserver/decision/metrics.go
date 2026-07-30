package decision

import "github.com/prometheus/client_golang/prometheus"

var (
	DecisionLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "adserver",
		Name:      "decision_latency_seconds",
		Help:      "Latency of ad decisions in seconds.",
		Buckets:   []float64{0.001, 0.0025, 0.005, 0.01, 0.02, 0.03, 0.05, 0.1},
	})

	DecisionsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "adserver",
		Name:      "decisions_total",
		Help:      "Count of ad decisions by outcome.",
	}, []string{"outcome"})

	CacheErrorsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "adserver",
		Name:      "cache_errors_total",
		Help:      "Count of cache path errors by operation.",
	}, []string{"operation"})

	FreqFailOpenTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "adserver",
		Name:      "freq_fail_open_total",
		Help:      "Count of frequency-cap checks that failed open.",
	})
)

func init() {
	prometheus.MustRegister(
		DecisionLatency,
		DecisionsTotal,
		CacheErrorsTotal,
		FreqFailOpenTotal,
	)
}
