package proto

import (
	"bytes"
	"testing"

	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestData(cnt int) (*bytes.Buffer, *work.Input, error) {
	buf := bytes.NewBuffer(nil)

	data := work.Input{
		Method: "POST",
		Path:   "/foo",
		Body:   []byte("bar"),
	}

	for i := 0; i < cnt; i++ {
		b, err := MarshalWithSize(&data)
		if err != nil {
			return nil, nil, err
		}

		buf.Write(b)
	}

	return buf, &data, nil
}

func TestDecode(t *testing.T) {
	cnt := 10

	testdata, expected, err := createTestData(cnt)
	require.NoError(t, err)

	dec := NewDecoder(testdata)

	dst := new(work.Input)
	for i := 0; i < cnt; i++ {
		err := dec.Decode(dst)
		if !assert.NoError(t, err) {
			return
		}

		assert.Equal(t, expected.Method, dst.Method)
		assert.Equal(t, expected.Path, dst.Path)
		assert.Equal(t, expected.Headers, dst.Headers)
		assert.Equal(t, expected.Body, dst.Body)
	}
}
