package bridge

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
)

// FramedMessageCodec is a forward-compatible binary framing codec for future
// high-throughput bridge mode. The payload format is currently JSON, but the
// framing allows swapping to protobuf/flatbuffers later without changing I/O.
type FramedMessageCodec struct{}

func (FramedMessageCodec) ReadFrame(r *bufio.Reader) ([]byte, error) {
	lenBuf := make([]byte, 4)
	if _, err := io.ReadFull(r, lenBuf); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(lenBuf)
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (FramedMessageCodec) WriteFrame(w *bufio.Writer, payload []byte) error {
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(payload)))
	if _, err := w.Write(lenBuf); err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	return w.Flush()
}

func (FramedMessageCodec) EncodeJSON(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (FramedMessageCodec) DecodeJSON(payload []byte, out any) error {
	return json.Unmarshal(payload, out)
}
