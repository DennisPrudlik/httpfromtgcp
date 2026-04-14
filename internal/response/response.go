package response

import (
	"fmt"
	"io"

	"httpfromtcp/internal/headers"
)

type Headers = headers.Headers

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

type writerState int

const (
	writerStateStatusLine writerState = iota
	writerStateHeaders
	writerStateBody
	writerStateTrailers
)

type Writer struct {
	writer io.Writer
	state  writerState
}

func NewWriter(writer io.Writer) *Writer {
	return &Writer{writer: writer, state: writerStateStatusLine}
}

func (w *Writer) WriteStatusLine(statusCode StatusCode) error {
	if w.state != writerStateStatusLine {
		return fmt.Errorf("status line must be written first")
	}
	if err := WriteStatusLine(w.writer, statusCode); err != nil {
		return err
	}
	w.state = writerStateHeaders
	return nil
}

func (w *Writer) WriteHeaders(headers Headers) error {
	if w.state != writerStateHeaders {
		return fmt.Errorf("headers must be written after status line and before body")
	}
	if err := WriteHeaders(w.writer, headers); err != nil {
		return err
	}
	w.state = writerStateBody
	return nil
}

func (w *Writer) WriteBody(p []byte) (int, error) {
	if w.state != writerStateBody {
		return 0, fmt.Errorf("body must be written after headers")
	}
	return w.writer.Write(p)
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if w.state != writerStateBody {
		return 0, fmt.Errorf("body must be written after headers")
	}
	chunkHeader := fmt.Sprintf("%x\r\n", len(p))
	if _, err := w.writer.Write([]byte(chunkHeader)); err != nil {
		return 0, err
	}
	n, err := w.writer.Write(p)
	if err != nil {
		return n, err
	}
	if _, err := w.writer.Write([]byte("\r\n")); err != nil {
		return n, err
	}
	return n, nil
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	if w.state != writerStateBody {
		return 0, fmt.Errorf("body must be written after headers")
	}
	w.state = writerStateTrailers
	return w.writer.Write([]byte("0\r\n"))
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	if w.state != writerStateTrailers {
		return fmt.Errorf("trailers must be written after chunked body is done")
	}
	for key, val := range h {
		_, err := fmt.Fprintf(w.writer, "%s: %s\r\n", key, val)
		if err != nil {
			return err
		}
	}
	_, err := w.writer.Write([]byte("\r\n"))
	return err
}

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	var reason string
	switch statusCode {
	case StatusOK:
		reason = "OK"
	case StatusBadRequest:
		reason = "Bad Request"
	case StatusInternalServerError:
		reason = "Internal Server Error"
	}
	var line string
	if reason != "" {
		line = fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, reason)
	} else {
		line = fmt.Sprintf("HTTP/1.1 %d \r\n", statusCode)
	}
	_, err := w.Write([]byte(line))
	return err
}

func GetDefaultHeaders(contentLen int) headers.Headers {
	h := headers.NewHeaders()
	h["content-length"] = fmt.Sprintf("%d", contentLen)
	h["connection"] = "close"
	h["content-type"] = "text/plain"
	return h
}

func WriteHeaders(w io.Writer, hdrs headers.Headers) error {
	for key, val := range hdrs {
		_, err := fmt.Fprintf(w, "%s: %s\r\n", key, val)
		if err != nil {
			return err
		}
	}
	_, err := w.Write([]byte("\r\n"))
	return err
}
