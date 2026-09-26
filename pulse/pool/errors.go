package pool

import "errors"

// ErrRequeue indicates that a worker failed to process a job's start or stop operation
// and requests the job to be requeued for another attempt.
var ErrRequeue = errors.New("requeue")

// ErrOwnershipLost indicates that a worker cannot confirm an active local run
// at the requested epoch. It includes a paused or stopped worker.
var ErrOwnershipLost = errors.New("job ownership lost")
