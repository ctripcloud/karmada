package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

/*
Label refers to the reconcile result.
*/
const (
	LabelError        = "error"
	LabelRequeueAfter = "requeue_after"
	LabelRequeue      = "requeue"
	LabelSuccess      = "success"
	LabelDrop         = "drop"
)

var (

	// ReconcileTotal is a prometheus counter metrics which holds the total
	// number of reconciliations per controller. It has two labels. controller label refers
	// to the controller name and result label refers to the reconcile result i.e
	// success, error, requeue, requeue_after, drop.
	ReconcileTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "async_worker_reconcile_total",
		Help: "Total number of reconciliations per controller",
	}, []string{"controller", "result"})

	// ReconcileErrors is a prometheus counter metrics which holds the total
	// number of errors from the Reconciler.
	ReconcileErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "async_worker_reconcile_errors_total",
		Help: "Total number of reconciliation errors per controller",
	}, []string{"controller"})

	// ReconcileTime is a prometheus metric which keeps track of the duration
	// of reconciliations
	ReconcileTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "async_worker_reconcile_time_seconds",
		Help: "Length of time per reconciliation per controller",
	}, []string{"controller"})
)

// ObserveReconcileTime observe ReconcileTime
func ObserveReconcileTime(controller string, reconcile time.Duration) {
	ReconcileTime.WithLabelValues(controller).Observe(reconcile.Seconds())
}

func init() {
	metrics.Registry.MustRegister(
		ReconcileTotal,
		ReconcileErrors,
		ReconcileTime,
	)
}
