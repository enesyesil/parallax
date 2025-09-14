package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	ReqTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "lb_requests_total", Help: "Total requests"},
		[]string{"status"},
	)
	Latency = prometheus.NewHistogram(
		prometheus.HistogramOpts{Name: "lb_latency_seconds", Help: "End-to-end latency"},
	)
)

func Collectors() []prometheus.Collector {
	return []prometheus.Collector{ReqTotal, Latency}
}
