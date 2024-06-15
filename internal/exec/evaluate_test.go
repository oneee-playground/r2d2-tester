package exec

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xeipuuv/gojsonschema"
)

func TestEvalStatus(t *testing.T) {
	t.Run("same value", func(t *testing.T) {
		assert.NoError(t, evalStatuscode(200, 200))
	})
	t.Run("different value", func(t *testing.T) {
		assert.Error(t, evalStatuscode(200, 0))
	})
}

func TestEvalHeaderExact(t *testing.T) {
	testcases := []struct {
		desc    string
		input   http.Header
		expect  map[string]string
		wantErr bool
	}{
		{
			desc: "exact",
			input: http.Header{
				"Foo": []string{"bar"},
			},
			expect: map[string]string{
				"Foo": "bar",
			},
			wantErr: false,
		},
		{
			desc:  "missing key 'Foo'",
			input: http.Header{},
			expect: map[string]string{
				"Foo": "bar",
			},
			wantErr: true,
		},
		{
			desc: "unexpected key 'Baz'",
			input: http.Header{
				"Foo": []string{"bar"},
				"Baz": []string{"quz"},
			},
			expect: map[string]string{
				"Foo": "bar",
			},
			wantErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			err := evalHeaderExact(tc.input, tc.expect)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEvalBodyExact(t *testing.T) {
	testcases := []struct {
		desc    string
		body    io.ReadCloser
		expect  []byte
		wantErr bool
	}{
		{
			desc:    "exact",
			body:    io.NopCloser(strings.NewReader("example")),
			expect:  []byte("example"),
			wantErr: false,
		},
		{
			desc:    "unmatch (wrong chars)",
			body:    io.NopCloser(strings.NewReader("example")),
			expect:  bytes.Repeat([]byte{'v'}, len("example")),
			wantErr: true,
		},
		{
			desc:    "unmatch (wrong length)",
			body:    io.NopCloser(strings.NewReader("example")),
			expect:  []byte("exam"),
			wantErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			err := evalBodyExact(tc.body, tc.expect)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEvalBodyJsonSchema(t *testing.T) {
	schemaString := `
{
  "title": "Product",
  "description": "A product from Acme's catalog",
  "type": "object",
  "properties": {
    "productId": {
      "description": "The unique identifier for a product",
      "type": "integer"
    },
    "productName": {
      "description": "Name of the product",
      "type": "string"
    },
    "price": {
      "description": "The price of the product",
      "type": "number",
      "exclusiveMinimum": 0
    },
    "tags": {
      "description": "Tags for the product",
      "type": "array",
      "items": {
        "type": "string"
      },
      "minItems": 1,
      "uniqueItems": true
    },
    "dimensions": {
      "type": "object",
      "properties": {
        "length": {
          "type": "number"
        }
      },
      "required": [ "length" ]
    }
  },
  "required": [ "productId", "productName", "price", "dimensions" ]
}
`

	loader := gojsonschema.NewStringLoader(schemaString)
	schema, err := gojsonschema.NewSchema(loader)
	require.NoError(t, err)

	testcases := []struct {
		desc    string
		body    string
		wantErr bool
	}{
		{
			desc: "fits (with tags)",
			body: `
			{
				"productId": 1,
				"productName": "foo",
				"price": 1.0,
				"tags": [
					"bar"
				],
				"dimensions": {
					"length": 0.0
				}
			}
			`,
			wantErr: false,
		},
		{
			desc: "fits (without tags)",
			body: `
			{
				"productId": 1,
				"productName": "foo",
				"price": 1.0,
				"dimensions": {
					"length": 0.0
				}
			}
			`,
			wantErr: false,
		},
		{
			desc: "no product id",
			body: `
			{
				"productName": "foo",
				"price": 1.0,
				"tags": [
					"bar"
				],
				"dimensions": {
					"length": 0.0
				}
			}
			`,
			wantErr: true,
		},
		{
			desc: "product id is string",
			body: `
			{
				"productId": "1",
				"productName": "foo",
				"price": 1.0,
				"tags": [
					"bar"
				],
				"dimensions": {
					"length": 0.0
				}
			}
			`,
			wantErr: true,
		},
		{
			desc: "price is below minimum",
			body: `
			{
				"productId": 1,
				"productName": "foo",
				"price": 0,
				"tags": [
					"bar"
				],
				"dimensions": {
					"length": 0.0
				}
			}
			`,
			wantErr: true,
		},
		{
			desc: "malformed json",
			body: `
			{
				"productId": 1,
				"productName": "foo",
				"price": 1.0,
				"tags": [
					"bar"
				],
				"dimensions": {
			`,
			wantErr: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			bodyReader := io.NopCloser(strings.NewReader(tc.body))

			err := evalBodyJsonSchema(bodyReader, schema)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
