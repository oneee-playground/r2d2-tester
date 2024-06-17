package proto

import (
	"encoding/binary"

	"google.golang.org/protobuf/proto"
)

// MarshalWithSize marshals protobuf message into bytes.
// It adds message's (32bit) size in front of the message.
func MarshalWithSize(m proto.Message) ([]byte, error) {
	b, err := proto.Marshal(m)
	if err != nil {
		return nil, err
	}

	size := uint32(len(b))
	sizebuf := make([]byte, 4)

	binary.LittleEndian.PutUint32(sizebuf, size)

	return append(sizebuf, b...), nil
}
