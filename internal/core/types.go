package core

import (
	"io"
	"time"
)

type Request struct {
	Method         string
	URL            string
	Headers        map[string]string
	Query          map[string]string
	Body           []byte
	BodyStream     io.Reader
	Timeout        time.Duration
	StreamResponse bool
	Priority       int
	TraceID        string
}

type Response struct {
	Status   int
	Headers  map[string]string
	Body     []byte
	Stream   io.ReadCloser
	Duration time.Duration
	TraceID  string
}

type RequestTask struct {
	Request Request
	Result  chan Result
}

type Result struct {
	Response Response
	Err      error
}
