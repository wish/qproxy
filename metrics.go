package qproxy

import (
	"github.com/jacksontj/dataman/metrics"
)

type QProxyMetrics struct {
	APILatency *metrics.ObserveArray
	APIHits    *metrics.CounterArray
	APIErrors  *metrics.CounterArray

	Acknowledged *metrics.CounterArray
	Published    *metrics.CounterArray
	Received     *metrics.CounterArray
}

func NewQProxyMetrics(r metrics.Registry) (QProxyMetrics, error) {
	m := QProxyMetrics{}

	m.APILatency, _ = metrics.NewCustomObserveArray(
		metrics.Metric{Name: "api_latency"},
		metrics.NewTDigestCreator([]float64{0.5, 0.9, 0.99, 1.0}),
		[]string{"api", "namespace", "name"},
	)
	if err := r.Register(m.APILatency); err != nil {
		return nil, err
	}

	m.APIHits, _ = metrics.NewCustomCounterArray(
		metrics.Metric{Name: "api_hits"},
		metrics.NewCounter,
		[]string{"api", "namespace", "name"},
	)
	if err := r.Register(m.APIHits); err != nil {
		return nil, err
	}

	m.APIErrors, _ = metrics.NewCustomCounterArray(
		metrics.Metric{Name: "api_errors"},
		metrics.NewCounter,
		[]string{"api", "namespace", "name"},
	)
	if err := r.Register(m.APIHits); err != nil {
		return nil, err
	}

	m.Acknowledged, _ = metrics.NewCustomCounterArray(
		metrics.Metric{Name: "acknowledged"},
		metrics.NewCounter,
		[]string{"namespace", "name"},
	)
	if err := r.Register(m.Acknowledged); err != nil {
		return nil, err
	}

	m.Published, _ = metrics.NewCustomCounterArray(
		metrics.Metric{Name: "published"},
		metrics.NewCounter,
		[]string{"namespace", "name"},
	)
	if err := r.Register(m.Published); err != nil {
		return nil, err
	}

	m.Received, _ = metrics.NewCustomCounterArray(
		metrics.Metric{Name: "received"},
		metrics.NewCounter,
		[]string{"namespace", "name"},
	)
	if err := r.Register(m.Received); err != nil {
		return nil, err
	}

	return m, nil
}
