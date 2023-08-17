package backoff

import (
	"flag"

	"k8s.io/client-go/util/retry"
)

// Retry is the recommended retry for a conflict where multiple clients
// are making changes to the same resource.
var Retry = retry.DefaultRetry

// Backoff is the recommended backoff for a conflict where a client
// may be attempting to make an unrelated modification to a resource under
// active management by one or more controllers.
var Backoff = retry.DefaultBackoff

func init() {
	flag.DurationVar(&Retry.Duration, "retry-backoff-duration", Retry.Duration, "retry duration")
	flag.Float64Var(&Retry.Factor, "retry-backoff-factor", Retry.Factor, "retry factor")
	flag.Float64Var(&Retry.Jitter, "retry-backoff-jitter", Retry.Jitter, "retry jitter")
	flag.IntVar(&Retry.Steps, "retry-backoff-steps", Retry.Steps, "retry steps")
	flag.DurationVar(&Retry.Cap, "retry-backoff-cap", Retry.Cap, "retry cap")

	flag.DurationVar(&Backoff.Duration, "backoff-backoff-duration", Backoff.Duration, "backoff duration")
	flag.Float64Var(&Backoff.Factor, "backoff-backoff-factor", Backoff.Factor, "backoff factor")
	flag.Float64Var(&Backoff.Jitter, "backoff-backoff-jitter", Backoff.Jitter, "backoff jitter")
	flag.IntVar(&Backoff.Steps, "backoff-backoff-steps", Backoff.Steps, "backoff steps")
	flag.DurationVar(&Backoff.Cap, "backoff-backoff-cap", Backoff.Cap, "backoff cap")
}
