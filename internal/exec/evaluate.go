package exec

import (
	"bytes"
	"io"
	"net/http"

	"github.com/pkg/errors"
	"github.com/xeipuuv/gojsonschema"
)

func evalStatuscode(actual, expected int) error {
	if actual != expected {
		return errors.Errorf(
			"unmatching status code. expected: %d, actual: %d",
			expected, actual,
		)
	}

	return nil
}

func evalHeaderAtLeast(header http.Header, expected map[string]string) error {
	for key, val := range expected {
		got := header.Get(key)
		if got != val {
			return errors.Errorf(
				"unmatching header value for key: %s. expected: %s, actual: %s",
				key, val, got,
			)
		}
	}

	return nil
}

func evalBodyExact(body io.ReadCloser, expected []byte) error {
	defer body.Close()

	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return errors.Wrap(err, "reading body")
	}

	if !bytes.Equal(bodyBytes, expected) {
		return errors.Errorf(
			"unmatching response body. expected: %s, actual: %s",
			expected, bodyBytes,
		)
	}

	return nil
}

func evalBodyJsonSchema(body io.ReadCloser, schema *gojsonschema.Schema) error {
	defer body.Close()

	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return errors.Wrap(err, "reading body")
	}

	if schema == nil {
		if len(bodyBytes) > 0 {
			return errors.New("schema not provided, but non-empty body was given")
		}
		return nil
	}

	bodyLoader := gojsonschema.NewBytesLoader(bodyBytes)

	result, err := schema.Validate(bodyLoader)
	if err != nil {
		return errors.Wrap(err, "validating body")
	}

	if !result.Valid() {
		errs := make([]string, len(result.Errors()))
		for idx, err := range result.Errors() {
			errs[idx] = err.String()
		}

		return errors.Errorf(
			"failed to validate body with schema. errors: %v", errs,
		)
	}

	return nil
}
