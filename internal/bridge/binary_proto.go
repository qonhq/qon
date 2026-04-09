package bridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"time"

	"github.com/qonhq/qon/internal/core"
)

const (
	protocolVersion byte = 1

	msgKindRequest  byte = 1
	msgKindResponse byte = 2
)

type BinaryRequest struct {
	Method    string
	URL       string
	Headers   map[string]string
	Query     map[string]string
	Body      []byte
	TimeoutMS int64
	Priority  int
	TraceID   string
	AccessKey string
}

type BinaryResponse struct {
	Status     int
	Headers    map[string]string
	Body       []byte
	DurationMS int64
	TraceID    string
	ErrorKind  string
	ErrorMsg   string
}

type FramedMessageCodec struct{}

func (FramedMessageCodec) ReadFrame(r *bufio.Reader) ([]byte, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, err
	}
	size := binary.BigEndian.Uint32(lenBuf[:])
	if size == 0 {
		return nil, errors.New("empty frame")
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func (FramedMessageCodec) WriteFrame(w *bufio.Writer, payload []byte) error {
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(payload)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	if _, err := w.Write(payload); err != nil {
		return err
	}
	return w.Flush()
}

func (FramedMessageCodec) EncodeRequest(req BinaryRequest) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(256 + len(req.Body))

	buf.WriteByte(protocolVersion)
	buf.WriteByte(msgKindRequest)
	writeString(&buf, req.Method)
	writeString(&buf, req.URL)
	writeMap(&buf, req.Headers)
	writeMap(&buf, req.Query)
	writeBytes(&buf, req.Body)
	writeI64(&buf, req.TimeoutMS)
	writeI32(&buf, int32(req.Priority))
	writeString(&buf, req.TraceID)
	writeString(&buf, req.AccessKey)

	return buf.Bytes(), nil
}

func (FramedMessageCodec) DecodeRequest(payload []byte) (BinaryRequest, error) {
	r := bytes.NewReader(payload)

	version, err := r.ReadByte()
	if err != nil {
		return BinaryRequest{}, err
	}
	if version != protocolVersion {
		return BinaryRequest{}, errors.New("unsupported protocol version")
	}
	kind, err := r.ReadByte()
	if err != nil {
		return BinaryRequest{}, err
	}
	if kind != msgKindRequest {
		return BinaryRequest{}, errors.New("invalid message kind for request")
	}

	method, err := readString(r)
	if err != nil {
		return BinaryRequest{}, err
	}
	url, err := readString(r)
	if err != nil {
		return BinaryRequest{}, err
	}
	headers, err := readMap(r)
	if err != nil {
		return BinaryRequest{}, err
	}
	query, err := readMap(r)
	if err != nil {
		return BinaryRequest{}, err
	}
	body, err := readBytes(r)
	if err != nil {
		return BinaryRequest{}, err
	}
	timeoutMS, err := readI64(r)
	if err != nil {
		return BinaryRequest{}, err
	}
	priority, err := readI32(r)
	if err != nil {
		return BinaryRequest{}, err
	}
	traceID, err := readString(r)
	if err != nil {
		return BinaryRequest{}, err
	}
	accessKey, err := readString(r)
	if err != nil {
		return BinaryRequest{}, err
	}

	if r.Len() != 0 {
		return BinaryRequest{}, errors.New("request has trailing bytes")
	}

	return BinaryRequest{
		Method:    method,
		URL:       url,
		Headers:   headers,
		Query:     query,
		Body:      body,
		TimeoutMS: timeoutMS,
		Priority:  int(priority),
		TraceID:   traceID,
		AccessKey: accessKey,
	}, nil
}

func (FramedMessageCodec) EncodeResponse(resp BinaryResponse) ([]byte, error) {
	var buf bytes.Buffer
	buf.Grow(256 + len(resp.Body))

	buf.WriteByte(protocolVersion)
	buf.WriteByte(msgKindResponse)
	writeI32(&buf, int32(resp.Status))
	writeMap(&buf, resp.Headers)
	writeBytes(&buf, resp.Body)
	writeI64(&buf, resp.DurationMS)
	writeString(&buf, resp.TraceID)
	writeString(&buf, resp.ErrorKind)
	writeString(&buf, resp.ErrorMsg)

	return buf.Bytes(), nil
}

func (FramedMessageCodec) DecodeResponse(payload []byte) (BinaryResponse, error) {
	r := bytes.NewReader(payload)

	version, err := r.ReadByte()
	if err != nil {
		return BinaryResponse{}, err
	}
	if version != protocolVersion {
		return BinaryResponse{}, errors.New("unsupported protocol version")
	}
	kind, err := r.ReadByte()
	if err != nil {
		return BinaryResponse{}, err
	}
	if kind != msgKindResponse {
		return BinaryResponse{}, errors.New("invalid message kind for response")
	}

	status, err := readI32(r)
	if err != nil {
		return BinaryResponse{}, err
	}
	headers, err := readMap(r)
	if err != nil {
		return BinaryResponse{}, err
	}
	body, err := readBytes(r)
	if err != nil {
		return BinaryResponse{}, err
	}
	durationMS, err := readI64(r)
	if err != nil {
		return BinaryResponse{}, err
	}
	traceID, err := readString(r)
	if err != nil {
		return BinaryResponse{}, err
	}
	errKind, err := readString(r)
	if err != nil {
		return BinaryResponse{}, err
	}
	errMsg, err := readString(r)
	if err != nil {
		return BinaryResponse{}, err
	}

	if r.Len() != 0 {
		return BinaryResponse{}, errors.New("response has trailing bytes")
	}

	return BinaryResponse{
		Status:     int(status),
		Headers:    headers,
		Body:       body,
		DurationMS: durationMS,
		TraceID:    traceID,
		ErrorKind:  errKind,
		ErrorMsg:   errMsg,
	}, nil
}

func RunBinaryStdio(client *core.Client, input io.Reader, output io.Writer) error {
	reader := bufio.NewReader(input)
	writer := bufio.NewWriter(output)
	codec := FramedMessageCodec{}

	for {
		frame, err := codec.ReadFrame(reader)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
				return nil
			}
			return err
		}

		req, err := codec.DecodeRequest(frame)
		if err != nil {
			if writeErr := writeBinaryError(writer, codec, string(core.ErrorInvalidRequest), err.Error()); writeErr != nil {
				return writeErr
			}
			continue
		}

		resp, execErr := client.Execute(
			context.Background(),
			core.Request{
				Method:   req.Method,
				URL:      req.URL,
				Headers:  req.Headers,
				Query:    req.Query,
				Body:     req.Body,
				Timeout:  time.Duration(req.TimeoutMS) * time.Millisecond,
				Priority: req.Priority,
				TraceID:  req.TraceID,
			},
			req.AccessKey,
		)

		if execErr != nil {
			kind := string(core.ErrorNetwork)
			if qe, ok := execErr.(*core.QonError); ok {
				kind = string(qe.Kind)
			}
			if writeErr := writeBinaryError(writer, codec, kind, execErr.Error()); writeErr != nil {
				return writeErr
			}
			continue
		}

		wireResp, err := codec.EncodeResponse(BinaryResponse{
			Status:     resp.Status,
			Headers:    resp.Headers,
			Body:       resp.Body,
			DurationMS: resp.Duration.Milliseconds(),
			TraceID:    resp.TraceID,
		})
		if err != nil {
			return err
		}
		if err := codec.WriteFrame(writer, wireResp); err != nil {
			return err
		}
	}
}

func writeBinaryError(writer *bufio.Writer, codec FramedMessageCodec, kind string, message string) error {
	payload, err := codec.EncodeResponse(BinaryResponse{
		ErrorKind: kind,
		ErrorMsg:  message,
	})
	if err != nil {
		return err
	}
	return codec.WriteFrame(writer, payload)
}

func writeU32(buf *bytes.Buffer, v uint32) {
	var out [4]byte
	binary.BigEndian.PutUint32(out[:], v)
	buf.Write(out[:])
}

func writeI32(buf *bytes.Buffer, v int32) {
	writeU32(buf, uint32(v))
}

func writeI64(buf *bytes.Buffer, v int64) {
	var out [8]byte
	binary.BigEndian.PutUint64(out[:], uint64(v))
	buf.Write(out[:])
}

func writeString(buf *bytes.Buffer, s string) {
	writeU32(buf, uint32(len(s)))
	buf.WriteString(s)
}

func writeBytes(buf *bytes.Buffer, b []byte) {
	writeU32(buf, uint32(len(b)))
	if len(b) > 0 {
		buf.Write(b)
	}
}

func writeMap(buf *bytes.Buffer, m map[string]string) {
	if len(m) == 0 {
		writeU32(buf, 0)
		return
	}
	writeU32(buf, uint32(len(m)))
	for k, v := range m {
		writeString(buf, k)
		writeString(buf, v)
	}
}

func readU32(r *bytes.Reader) (uint32, error) {
	var out [4]byte
	if _, err := io.ReadFull(r, out[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(out[:]), nil
}

func readI32(r *bytes.Reader) (int32, error) {
	v, err := readU32(r)
	if err != nil {
		return 0, err
	}
	return int32(v), nil
}

func readI64(r *bytes.Reader) (int64, error) {
	var out [8]byte
	if _, err := io.ReadFull(r, out[:]); err != nil {
		return 0, err
	}
	return int64(binary.BigEndian.Uint64(out[:])), nil
}

func readString(r *bytes.Reader) (string, error) {
	n, err := readU32(r)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", nil
	}
	if uint64(n) > uint64(r.Len()) {
		return "", io.ErrUnexpectedEOF
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return "", err
	}
	return string(b), nil
}

func readBytes(r *bytes.Reader) ([]byte, error) {
	n, err := readU32(r)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, nil
	}
	if uint64(n) > uint64(r.Len()) {
		return nil, io.ErrUnexpectedEOF
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, err
	}
	return b, nil
}

func readMap(r *bytes.Reader) (map[string]string, error) {
	count, err := readU32(r)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, nil
	}
	out := make(map[string]string, count)
	for i := uint32(0); i < count; i++ {
		k, err := readString(r)
		if err != nil {
			return nil, err
		}
		v, err := readString(r)
		if err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, nil
}
