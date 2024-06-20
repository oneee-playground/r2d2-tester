package exec

import (
	"context"

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
			return errors.Wrap(err, "fetching work stream")
		case work, ok = <-stream:
			if !ok {
				return nil
			}
		}

		if err := worker.do(ctx, work); err != nil {
			return err
		}
	}
}

func (e *Executor) testLoad(
	ctx context.Context,
	templates map[uuid.UUID]template, stream <-chan *work.Work, errchan <-chan error,
) error {
	var work *work.Work
	var ok bool

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case work, ok = <-stream:
			if !ok {
				return nil
			}
		}

	}
}
