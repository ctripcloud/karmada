package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics namespace used by the controller-runtime.
const (
	MetricsNamespace = "controller_runtime"
)

// Metrics rate limit used by the controller-runtime.
const (
	RateLimitSubsystem = "rest_client"
	RateLimitKey       = "rate_limit"
)

var (
	// rate limit metrics.

	CapacityMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: MetricsNamespace,
		Subsystem: RateLimitSubsystem,
		Name:      RateLimitKey,
		Help:      "Rate limit values in config to calculate saturation metrics",
	}, []string{"name"})
)

func init() {
	Registry.MustRegister(CapacityMetric)
}
