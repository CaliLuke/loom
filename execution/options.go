package execution

import (
	"context"
	"time"
)

type (
	// Future represents a pending activity result. It belongs to the workflow
	// execution that scheduled it and must not be shared across workflows.
	// Waiting and cancellation follow the selected backend's rules.
	Future[T any] interface {
		// Get blocks until the activity completes and returns the typed result.
		// Once completed, repeated calls return the recorded result.
		Get(ctx context.Context) (T, error)

		// IsReady returns true if the activity has completed (success or failure) and Get()
		// will not block. This allows workflows to poll or implement custom waiting strategies.
		IsReady() bool
	}

	// ActivityOptions configures retry and timeouts for an activity.
	ActivityOptions struct {
		// Queue overrides the activity queue. Empty uses registration or backend defaults.
		Queue string
		// RetryPolicy controls retry behavior for this activity. If zero-valued, the
		// engine uses its default retry policy.
		RetryPolicy RetryPolicy
		// ScheduleToStartTimeout bounds how long the activity may wait in the task
		// queue before a worker starts the attempt. Zero means leave queue-wait
		// unspecified here and let the engine adapter apply its own defaults.
		ScheduleToStartTimeout time.Duration
		// StartToCloseTimeout bounds one activity attempt once a worker has started
		// executing it. Zero means use the engine default.
		StartToCloseTimeout time.Duration
		// HeartbeatTimeout bounds the maximum gap between heartbeats emitted by the
		// running activity. Zero leaves the backend default unchanged.
		HeartbeatTimeout time.Duration
	}

	// RetryPolicy defines retry semantics shared by workflows and activities.
	// Zero-valued fields mean the engine uses its defaults.
	RetryPolicy struct {
		// MaxAttempts caps total attempts, including the first. Zero uses backend defaults.
		MaxAttempts int
		// InitialInterval is the delay before the first retry. Zero means use engine default.
		InitialInterval time.Duration
		// BackoffCoefficient multiplies the delay after each retry. Zero uses backend
		// defaults; the backend defines the valid range of explicit coefficients.
		BackoffCoefficient float64
	}
)
