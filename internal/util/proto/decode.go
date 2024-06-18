package proto

import (
	"encoding/binary"
	"io"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

const decoderBufSize = 4096

type Decoder struct {
	src    io.Reader
	lenbuf []byte
	buf    []byte
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		src:    r,
		lenbuf: make([]byte, 4),
		buf:    make([]byte, decoderBufSize),
	}
}

// Decode decodes bytes read from source info proto.Message.
// It is caller's responsibility to handle EOF.
func (d *Decoder) Decode(m proto.Message) error {
	if _, err := io.ReadFull(d.src, d.lenbuf); err != nil {
		return errors.Wrap(err, "reading length")
	}

	size := binary.LittleEndian.Uint32(d.lenbuf)

	if size > decoderBufSize {
		return io.ErrShortBuffer
	}

	_, err := io.ReadFull(d.src, d.buf[:size])
	if err != nil {
		return errors.Wrap(err, "reading message")
	}

	if err := proto.Unmarshal(d.buf[:size], m); err != nil {
		return errors.Wrap(err, "unmarshaling message")
	}

	return nil
}
