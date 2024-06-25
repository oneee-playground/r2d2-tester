package exec

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/pkg/errors"
)

type worker struct {
	target    *process
	templates map[uuid.UUID]template

	httpClient *http.Client
}

func (w *worker) do(ctx context.Context, work *work.Work) error {
	ctx, cancel := context.WithTimeout(ctx, work.Timeout.AsDuration())
	defer cancel()

	res, err := w.sendRequest(ctx, work.Input)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return errors.New("deadline exceeded while waiting response")
		}
		return errors.Wrap(err, "sending request")
	}

	useTemplate := len(work.TemplateId) > 0
	if useTemplate {
		// Need to use template to evaluate.
		templateID, err := uuid.FromBytes(work.TemplateId)
		if err != nil {
			return errors.Wrap(err, "making uuid out of template id")
		}

		template, ok := w.templates[templateID]
		if !ok {
			return errors.New("template not found")
		}

		schema, ok := template.schemaTable[res.StatusCode]
		if !ok {
			return errors.Errorf("untemplated status code: %d", res.StatusCode)
		}

		if err := evalHeaderAtLeast(res.Header, schema.headers); err != nil {
			return err
		}
		if err := evalBodyJsonSchema(res.Body, schema.jsonSchema); err != nil {
			return err
		}
	} else {
		// Expecting exact value.
		expected := work.ExpectedValue

		if err := evalStatuscode(res.StatusCode, int(expected.Status)); err != nil {
			return err
		}
		if err := evalHeaderAtLeast(res.Header, expected.Headers); err != nil {
			return err
		}
		if err := evalBodyExact(res.Body, expected.Body); err != nil {
			return err
		}
	}

	return nil
}

func (w *worker) sendRequest(ctx context.Context, input *work.Input) (*http.Response, error) {
	url := fmt.Sprintf("http://%s:%d%s", w.target.Hostname, w.target.Port, input.Path)

	request, err := http.NewRequestWithContext(ctx, input.Method, url, bytes.NewReader(input.Body))
	if err != nil {
		return nil, errors.Wrap(err, "creating new request")
	}

	for key, val := range input.Headers {
		request.Header.Set(key, val)
	}

	res, err := w.httpClient.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, "sending request")
	}

	return res, nil
}

type concurrentWorker struct {
	underlying *worker

	index       int
	inputStream chan *work.Work
}

func (cw *concurrentWorker) run(
	ctx context.Context, wg *sync.WaitGroup,
	doneStream chan<- int, errchan chan<- error,
) {
	defer wg.Done()
	defer close(cw.inputStream)

	var work *work.Work

	for {
		select {
		case <-ctx.Done():
			return
		case work = <-cw.inputStream:
		}

		if err := cw.underlying.do(ctx, work); err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			errchan <- err
			return
		}

		doneStream <- cw.index
	}
}

type workerPool struct {
	workers []*concurrentWorker
	wg      *sync.WaitGroup

	doneStream chan int
	errchan    chan error

	closeFunc func()
}

func newWorkerPool(
	count int,
	target *process, templates map[uuid.UUID]template, httpClient *http.Client,
) *workerPool {
	pool := &workerPool{
		workers:    make([]*concurrentWorker, count),
		wg:         new(sync.WaitGroup),
		doneStream: make(chan int, count),
		errchan:    make(chan error, count),
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool.closeFunc = cancel

	for i := 0; i < count; i++ {
		pool.wg.Add(1)

		cw := &concurrentWorker{
			index: i,
			underlying: &worker{
				target:     target,
				templates:  templates,
				httpClient: httpClient,
			},
			inputStream: make(chan *work.Work),
		}

		go cw.run(ctx, pool.wg, pool.doneStream, pool.errchan)

		pool.workers[i] = cw
		pool.doneStream <- i
	}

	return pool
}

func (wp *workerPool) close() {
	wp.closeFunc()
	wp.wg.Wait()
}
