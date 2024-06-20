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
