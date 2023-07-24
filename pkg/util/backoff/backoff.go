package backoff

import (
	"flag"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// Retry default value equal to retry.DefaultRetry
var Retry = wait.Backoff{}

func init() {
	flag.DurationVar(&Retry.Duration, "retry-backoff-duration", 10*time.Millisecond, "retry backoff duration")
	flag.Float64Var(&Retry.Factor, "retry-backoff-factor", 1.0, "retry backoff factor")
	flag.Float64Var(&Retry.Jitter, "retry-backoff-jitter", 0.1, "retry backoff jitter")
	flag.IntVar(&Retry.Steps, "retry-backoff-steps", 5, "retry backoff steps")
	flag.DurationVar(&Retry.Cap, "retry-backoff-cap", 0, "retry backoff cap")
}
