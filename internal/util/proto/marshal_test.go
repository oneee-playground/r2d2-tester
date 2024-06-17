package proto

import (
	"encoding/binary"
	"testing"

	"github.com/oneee-playground/r2d2-tester/internal/work"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestMarshalWithSize(t *testing.T) {
	data := work.Input{
		Method:  "GET",
		Path:    "/foo",
		Headers: nil,
		Body:    []byte("bar"),
	}

	b, err := MarshalWithSize(&data)
	require.NoError(t, err)

	if !assert.Greater(t, len(b), 4) {
		return
	}

	size := binary.LittleEndian.Uint32(b[:4])
	if !assert.Len(t, b, int(size)+4) {
		return
	}

	var dst work.Input
	err = proto.Unmarshal(b[4:], &dst)
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, data.Method, dst.Method)
	assert.Equal(t, data.Path, dst.Path)
	assert.Equal(t, data.Headers, dst.Headers)
	assert.Equal(t, data.Body, dst.Body)
}
