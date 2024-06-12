package job

import (
	"context"

	"github.com/pkg/errors"
)

var NoErrEmptyJobs = errors.New("jobs are empty")

type Poller interface {
	Poll(ctx context.Context) (id string, job Job, err error)
	MarkAsDone(ctx context.Context, id string) (err error)
}
