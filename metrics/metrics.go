package prometheus

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type QProxyMetrics struct {
	APILatency *prometheus.SummaryVec
	APIHits    *prometheus.CounterVec
	APIErrors  *prometheus.CounterVec

	Acknowledged *prometheus.CounterVec
	Published    *prometheus.CounterVec
	Received     *prometheus.CounterVec

	Queued   *prometheus.GaugeVec
	Inflight *prometheus.GaugeVec
}

// Same as DefObjectives from the golang package, but adding 1.0 as an objective
var SummaryVecObjectives = map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001, 1.0: 0.0}

func NewQProxyMetrics() (QProxyMetrics, error) {
	m := QProxyMetrics{}

	m.APILatency = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Namespace:  "qproxy",
			Name:       "api_latency",
			Objectives: SummaryVecObjectives,
			MaxAge:     time.Minute,
		},
		[]string{"api", "namespace", "name"},
	)
	if err := prometheus.Register(m.APILatency); err != nil {
		return m, err
	}

	m.APIHits = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "qproxy",
			Name:      "api_hits",
		},
		[]string{"api", "namespace", "name"},
	)
	if err := prometheus.Register(m.APIHits); err != nil {
		return m, err
	}

	m.APIErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "qproxy",
			Name:      "api_errors",
		},
		[]string{"api", "namespace", "name", "reason"},
	)
	if err := prometheus.Register(m.APIErrors); err != nil {
		return m, err
	}

	m.Acknowledged = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "qproxy",
			Name:      "acknowledged",
		},
		[]string{"namespace", "name"},
	)
	if err := prometheus.Register(m.Acknowledged); err != nil {
		return m, err
	}

	m.Published = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "qproxy",
			Name:      "published",
		},
		[]string{"namespace", "name"},
	)
	if err := prometheus.Register(m.Published); err != nil {
		return m, err
	}

	m.Received = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "qproxy",
			Name:      "received",
		},
		[]string{"namespace", "name"},
	)
	if err := prometheus.Register(m.Received); err != nil {
		return m, err
	}

	m.Queued = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "qproxy",
			Name:      "queued",
		},
		[]string{"namespace", "name", "true_name"},
	)
	if err := prometheus.Register(m.Queued); err != nil {
		return m, err
	}

	m.Inflight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "qproxy",
			Name:      "inflight",
		},
		[]string{"namespace", "name", "true_name" },
	)
	if err := prometheus.Register(m.Inflight); err != nil {
		return m, err
	}

	return m, nil
}
