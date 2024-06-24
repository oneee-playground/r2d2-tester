package exec

import (
	"context"
	"runtime"

	"github.com/google/uuid"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/pkg/errors"
)

func (e *Executor) testScenario(
	ctx context.Context,
	templates map[uuid.UUID]template, stream <-chan *work.Work, errchan <-chan error,
) error {
	var work *work.Work
	var ok bool

	worker := &worker{
		target:     e.primaryProcess,
		templates:  templates,
		httpClient: e.HTTPClient,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errchan:
			return errors.Wrap(err, "error received from storage")
		case work, ok = <-stream:
			if !ok {
				return nil
			}
		}

		if err := worker.do(ctx, work); err != nil {
			return errors.Wrap(err, "doing work")
		}
	}
}

func (e *Executor) testLoad(
	ctx context.Context,
	templates map[uuid.UUID]template, stream <-chan *work.Work, storageErrchan <-chan error,
) error {
	var work *work.Work
	var ok bool

	maxProcs := runtime.GOMAXPROCS(0)

	workerPool := newWorkerPool(maxProcs, e.primaryProcess, templates, e.HTTPClient)
	defer workerPool.close()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-storageErrchan:
			return errors.Wrap(err, "error received from storage")
		case err := <-workerPool.errchan:
			return errors.Wrap(err, "error received from worker")
		case work, ok = <-stream:
			if !ok {
				return nil
			}
		}

		if err := workerPool.do(ctx, work); err != nil {
			return errors.Wrap(err, "feeding work to the worker pool")
		}
	}
}
