package exec

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

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

		// TODO: As the linter says, we should avoid discarding cancel function.
		// I hope you'll find a way later :)
		timeoutCtx, _ := context.WithTimeout(ctx, work.Timeout.AsDuration())

		res, err := e.sendRequest(timeoutCtx, work.Input)
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

			template, ok := templates[templateID]
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
	}
}

// func (e *Executor) testLoad(
// 	ctx context.Context,
// 	templates map[uuid.UUID]work.Template, stream <-chan *work.Work,
// ) error {
// 	var work *work.Work
// 	var ok bool

// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return ctx.Err()
// 		case work, ok = <-stream:
// 			if !ok {
// 				return nil
// 			}
// 		}

// 	}
// }

func (e *Executor) sendRequest(ctx context.Context, input *work.Input) (*http.Response, error) {
	url := fmt.Sprintf("http://%s:%d%s", e.primaryProcess.Hostname, e.primaryProcess.Port, input.Path)

	request, err := http.NewRequestWithContext(ctx, input.Method, url, bytes.NewReader(input.Body))
	if err != nil {
		return nil, errors.Wrap(err, "creating new request")
	}

	for key, val := range input.Headers {
		request.Header.Set(key, val)
	}

	res, err := e.HTTPClient.Do(request)
	if err != nil {
		return nil, errors.Wrap(err, "sending request")
	}

	return res, nil
}
