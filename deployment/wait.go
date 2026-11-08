package deployment

import (
	"context"
	"fmt"
	"time"
)

// WaitOptions bounds task observation. Zero values use a one-second polling
// interval and ten-minute timeout. An earlier context deadline takes precedence.
type WaitOptions struct {
	Interval time.Duration
	Timeout  time.Duration
}

// FailureError reports a task that failed or was cancelled on the server.
// Task retains the state and detailed message; Error omits server message text.
type FailureError struct {
	Task Task
}

func (e *FailureError) Error() string {
	return fmt.Sprintf("deployment: task %d ended in state %s", e.Task.ID, e.Task.State)
}

// StateError means the task's state cannot establish progress or completion.
// It includes BAM's UNKNOWN state and values introduced by other releases.
type StateError struct {
	Task Task
}

func (e *StateError) Error() string {
	return fmt.Sprintf("deployment: task %d has an unrecognized state", e.Task.ID)
}

// Wait observes a task until it completes, fails, or the wait expires. It never
// resubmits a deployment or cancels one on the server. Completed tasks with
// warnings return successfully with StateCompletedWithWarnings preserved.
// On polling errors or cancellation, the last successfully read task is returned.
func (t Tasks) Wait(ctx context.Context, id int64, options WaitOptions) (Task, error) {
	if options.Interval < 0 || options.Timeout < 0 {
		return Task{}, fmt.Errorf("deployment: wait durations cannot be negative")
	}
	if options.Interval == 0 {
		options.Interval = time.Second
	}
	if options.Timeout == 0 {
		options.Timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, options.Timeout)
	defer cancel()

	var last Task
	for {
		task, err := t.Get(ctx, id)
		if err != nil {
			return last, err
		}
		last = task
		switch task.State {
		case StateCompleted, StateCompletedWithWarnings:
			return task, nil
		case StateFailed, StateCancelled:
			return task, &FailureError{Task: task}
		case StatePending, StateQueued, StateRunning, StateCancelling:
		default:
			return task, &StateError{Task: task}
		}

		timer := time.NewTimer(options.Interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return last, ctx.Err()
		case <-timer.C:
		}
	}
}
