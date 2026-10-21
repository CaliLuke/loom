package selvedge

import (
	"github.com/CaliLuke/loom/execution"
	sv "github.com/CaliLuke/selvedge"
	"github.com/CaliLuke/selvedge/retry"
)

// invocationActivityOptions leaves zero fields unset so recorded definition defaults survive.
func invocationActivityOptions(options execution.ActivityOptions) sv.ActivityOptions {
	result := sv.ActivityOptions{Queue: options.Queue, StartToCloseTimeout: options.StartToCloseTimeout, ScheduleToStartTimeout: options.ScheduleToStartTimeout, HeartbeatTimeout: options.HeartbeatTimeout}
	p := options.RetryPolicy
	if p.MaxAttempts != 0 || p.InitialInterval != 0 || p.BackoffCoefficient != 0 {
		result.RetryPolicy = &retry.Policy{MaxAttempts: p.MaxAttempts, InitialInterval: p.InitialInterval, Multiplier: p.BackoffCoefficient}
	}
	return result
}
