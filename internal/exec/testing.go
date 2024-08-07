package exec

import (
	"context"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/influxdata/influxdb-client-go/api/write"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/pkg/errors"
)

func (e *Executor) testScenario(
	ctx context.Context, sectionID uuid.UUID,
	templates map[uuid.UUID]template, stream <-chan *work.Work, errchan <-chan error,
) error {
	var work *work.Work
	var ok bool

	worker := &worker{
		target:     e.primaryProcess,
		templates:  templates,
		httpClient: e.HTTPClient,
	}

	defer e.metrics.Flush()

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

		start := time.Now()
		if err := worker.do(ctx, work); err != nil {
			return errors.Wrap(err, "doing work")
		}

		end := time.Now()
		e.metrics.Write(write.NewPoint("response",
			map[string]string{
				"section-id": sectionID.String(),
			},
			map[string]interface{}{
				"latency": end.Sub(start).Nanoseconds(),
			},
			end,
		))
	}
}

func (e *Executor) testLoad(
	ctx context.Context, sectionID uuid.UUID, rpm uint64,
	templates map[uuid.UUID]template, workStream <-chan *work.Work, storageErrchan <-chan error,
) (int, error) {
	defer e.metrics.Flush()

	workerPool := newWorkerPool(
		runtime.GOMAXPROCS(0), e.primaryProcess, templates, e.HTTPClient,
	)

	var requestRate time.Duration
	if rpm <= 60 {
		// We simply set the request rate to 1 second.
		requestRate = time.Second
	} else {
		requestRate = time.Minute / time.Duration(rpm)
	}

	timer := time.NewTimer(requestRate)
	defer timer.Stop()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errchan := make(chan error, 1)
	go func() {
		var err error
		select {
		case <-ctx.Done():
			err = ctx.Err()
		case e := <-storageErrchan:
			err = errors.Wrap(e, "error received from storage")
		case e := <-workerPool.errchan:
			err = errors.Wrap(e, "error received from worker")
		}

		if err != nil {
			errchan <- err
		}
	}()

	var (
		worker      *concurrentWorker
		pendingWork *work.Work
		latestMiss  time.Time
	)

	donechan := workerPool.doneStream
	stream := workStream

	workerStart := make([]time.Time, len(workerPool.workers))

	feedWorker := func() {
		worker.inputStream <- pendingWork

		workerStart[worker.index] = time.Now()

		// Reset current worker and work.
		worker, pendingWork = nil, nil

		// Restore the channels so we can get another work & worker.
		donechan = workerPool.doneStream
		stream = workStream

		if latestMiss.IsZero() {
			timer.Reset(requestRate)
		} else {
			if !timer.Stop() {
				<-timer.C
			}

			remaining := requestRate - time.Since(latestMiss)
			timer.Reset(min(remaining, 10*time.Microsecond))
		}

		latestMiss = time.Time{}
	}

	writeLatency := func(idx int) {
		if start := workerStart[idx]; !start.IsZero() {
			end := time.Now()

			e.metrics.Write(write.NewPoint("response",
				map[string]string{
					"section-id": sectionID.String(),
				},
				map[string]interface{}{
					"latency": end.Sub(start).Nanoseconds(),
				},
				end,
			))
		}
	}

	dueMissed := 0

	for {
		select {
		case err := <-errchan:
			workerPool.close()
			return dueMissed, err
		case t := <-timer.C:
			if pendingWork != nil && worker != nil {
				// Best case. Everything worked normally.
				feedWorker()
				continue
			}

			if donechan == nil && stream == nil {
				workerPool.close()
				// Drain the channel and write metrics.
				for idx := range workerPool.doneStream {
					writeLatency(idx)
				}
				return dueMissed, nil
			}

			if !latestMiss.IsZero() {
				dueMissed++
			}

			latestMiss = t
			timer.Reset(requestRate)
		case workerIdx := <-donechan:
			worker = workerPool.workers[workerIdx]

			writeLatency(worker.index)

			if !latestMiss.IsZero() && pendingWork != nil {
				// Getting free worker was slower.
				feedWorker()
				continue
			}

			// We don't want to receive more than one worker.
			donechan = nil
		case work, ok := <-stream:
			if !ok {
				// No more work should be received.
				// Therefore, we assign nil.
				stream = nil
				continue
			}

			pendingWork = work

			if !latestMiss.IsZero() && worker != nil {
				// Receiving work was slower.
				feedWorker()
				continue
			}

			// We don't want to receive more than one work.
			stream = nil
		}
	}
}
