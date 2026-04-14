package request

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"httpfromtcp/internal/headers"
)

type Request struct {
	RequestLine RequestLine
	Headers     headers.Headers
	Body        []byte
	state       requestState
}

type requestState int

const (
	requestStateInitialized requestState = iota
	requestStateParsingHeaders
	requestStateParsingBody
	requestStateDone
)

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := &Request{
		Headers: headers.NewHeaders(),
		state:   requestStateInitialized,
	}
	buffer := make([]byte, 8)
	bytesInBuffer := 0

	for request.state != requestStateDone {
		if bytesInBuffer == len(buffer) {
			grownBuffer := make([]byte, len(buffer)*2)
			copy(grownBuffer, buffer[:bytesInBuffer])
			buffer = grownBuffer
		}

		bytesRead, readErr := reader.Read(buffer[bytesInBuffer:])
		if bytesRead > 0 {
			bytesInBuffer += bytesRead

			bytesParsed, parseErr := request.parse(buffer[:bytesInBuffer])
			if parseErr != nil {
				return nil, parseErr
			}
			if bytesParsed > 0 {
				copy(buffer, buffer[bytesParsed:bytesInBuffer])
				bytesInBuffer -= bytesParsed
			}
		}

		if readErr == io.EOF {
			if request.state != requestStateDone {
				return nil, fmt.Errorf("invalid request: incomplete request")
			}
			break
		}
		if readErr != nil {
			return nil, readErr
		}

		if bytesRead == 0 {
			return nil, fmt.Errorf("invalid request: no data read")
		}
	}

	return request, nil
}

func (request *Request) parse(data []byte) (int, error) {
	totalBytesParsed := 0
	for request.state != requestStateDone {
		bytesParsed, err := request.parseSingle(data[totalBytesParsed:])
		if err != nil {
			return 0, err
		}
		if bytesParsed == 0 {
			break
		}
		totalBytesParsed += bytesParsed
	}

	return totalBytesParsed, nil
}

func (request *Request) parseSingle(data []byte) (int, error) {
	if request.state == requestStateDone {
		return 0, nil
	}

	switch request.state {
	case requestStateInitialized:
		requestLine, bytesConsumed, err := parseRequestLine(data)
		if err != nil {
			return 0, err
		}
		if bytesConsumed == 0 {
			return 0, nil
		}

		request.RequestLine = requestLine
		request.state = requestStateParsingHeaders
		return bytesConsumed, nil
	case requestStateParsingHeaders:
		bytesConsumed, done, err := request.Headers.Parse(data)
		if err != nil {
			return 0, err
		}
		if bytesConsumed == 0 {
			return 0, nil
		}
		if done {
			request.state = requestStateParsingBody
		}
		return bytesConsumed, nil
	case requestStateParsingBody:
		contentLengthStr := request.Headers.Get("content-length")
		if contentLengthStr == "" {
			request.state = requestStateDone
			return 0, nil
		}
		contentLength := 0
		_, err := fmt.Sscanf(contentLengthStr, "%d", &contentLength)
		if err != nil {
			return 0, fmt.Errorf("invalid Content-Length: %s", contentLengthStr)
		}
		request.Body = append(request.Body, data...)
		if len(request.Body) > contentLength {
			return 0, fmt.Errorf("body exceeds Content-Length")
		}
		if len(request.Body) == contentLength {
			request.state = requestStateDone
		}
		return len(data), nil
	default:
		return 0, fmt.Errorf("invalid parser state")
	}
}

func parseRequestLine(data []byte) (RequestLine, int, error) {
	requestLineEnd := bytes.Index(data, []byte("\r\n"))
	if requestLineEnd == -1 {
		return RequestLine{}, 0, nil
	}

	requestLine := string(data[:requestLineEnd])
	parts := strings.Split(requestLine, " ")
	if len(parts) != 3 {
		return RequestLine{}, 0, fmt.Errorf("invalid request line: expected 3 parts")
	}

	method := parts[0]
	requestTarget := parts[1]
	httpVersionPart := parts[2]

	if method == "" {
		return RequestLine{}, 0, fmt.Errorf("invalid request line: empty method")
	}
	for _, character := range method {
		if character < 'A' || character > 'Z' {
			return RequestLine{}, 0, fmt.Errorf("invalid request line: invalid method")
		}
	}

	versionParts := strings.Split(httpVersionPart, "/")
	if len(versionParts) != 2 || versionParts[0] != "HTTP" {
		return RequestLine{}, 0, fmt.Errorf("invalid request line: invalid http version format")
	}
	if versionParts[1] != "1.1" {
		return RequestLine{}, 0, fmt.Errorf("invalid request line: unsupported http version")
	}

	return RequestLine{
		Method:        method,
		RequestTarget: requestTarget,
		HttpVersion:   versionParts[1],
	}, requestLineEnd + 2, nil
}
