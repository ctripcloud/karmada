package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics capacity used by controller manager such as qps and burst.
const (
	CapacityKey = "controller_manager_capacity"
)

var (
	// capacity metrics.

	CapacityMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: CapacityKey,
		Help: "Capacity values in config to calculate saturation metrics",
	}, []string{"name"})
)

func init() {
	Registry.MustRegister(CapacityMetric)
}
