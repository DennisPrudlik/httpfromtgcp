package request

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type chunkReader struct {
	data            string
	numBytesPerRead int
	pos             int
}

func (chunkedReader *chunkReader) Read(target []byte) (int, error) {
	if chunkedReader.pos >= len(chunkedReader.data) {
		return 0, io.EOF
	}

	endIndex := chunkedReader.pos + chunkedReader.numBytesPerRead
	if endIndex > len(chunkedReader.data) {
		endIndex = len(chunkedReader.data)
	}

	n := copy(target, chunkedReader.data[chunkedReader.pos:endIndex])
	chunkedReader.pos += n
	return n, nil
}

func TestRequestFromReader(t *testing.T) {
	testCases := []struct {
		name           string
		input          string
		expectError    bool
		expectedMethod string
		expectedTarget string
		expectedVer    string
	}{
		{
			name: "Good Request line",
			input: "GET / HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n",
			expectedMethod: "GET",
			expectedTarget: "/",
			expectedVer:    "1.1",
		},
		{
			name: "Good Request line with path",
			input: "GET /coffee HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n",
			expectedMethod: "GET",
			expectedTarget: "/coffee",
			expectedVer:    "1.1",
		},
		{
			name: "Good POST Request with path",
			input: "POST /brew HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n",
			expectedMethod: "POST",
			expectedTarget: "/brew",
			expectedVer:    "1.1",
		},
		{
			name: "Invalid number of parts in request line",
			input: "GET /\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n",
			expectError: true,
		},
		{
			name: "Invalid method (out of order) Request line",
			input: "GeT / HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n",
			expectError: true,
		},
		{
			name: "Invalid version in Request line",
			input: "GET / HTTP/1.0\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n",
			expectError: true,
		},
		{
			name: "Invalid method lowercase",
			input: "Get / HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n",
			expectError: true,
		},
		{
			name: "Missing CRLF request line terminator",
			input: "GET / HTTP/1.1\n" +
				"Host: localhost:42069\n\n",
			expectError: true,
		},
		{
			name: "Invalid version literal format",
			input: "GET / HTTPS/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n",
			expectError: true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			for chunkSize := 1; chunkSize <= len(testCase.input); chunkSize++ {
				reader := &chunkReader{
					data:            testCase.input,
					numBytesPerRead: chunkSize,
				}

				request, err := RequestFromReader(reader)
				if testCase.expectError {
					require.Error(t, err, "chunk size %d", chunkSize)
					continue
				}

				require.NoError(t, err, "chunk size %d", chunkSize)
				require.NotNil(t, request)
				assert.Equal(t, testCase.expectedMethod, request.RequestLine.Method)
				assert.Equal(t, testCase.expectedTarget, request.RequestLine.RequestTarget)
				assert.Equal(t, testCase.expectedVer, request.RequestLine.HttpVersion)
			}
		})
	}
}

func TestRequestFromReaderHeaders(t *testing.T) {
	testCases := []struct {
		name            string
		input           string
		expectError     bool
		expectedHeaders map[string]string
		numBytesPerRead int
	}{
		{
			name: "Standard Headers",
			input: "GET / HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"User-Agent: curl/7.81.0\r\n" +
				"Accept: */*\r\n" +
				"\r\n",
			numBytesPerRead: 3,
			expectedHeaders: map[string]string{
				"host":       "localhost:42069",
				"user-agent": "curl/7.81.0",
				"accept":     "*/*",
			},
		},
		{
			name: "Empty Headers",
			input: "GET / HTTP/1.1\r\n" +
				"\r\n",
			numBytesPerRead: 2,
			expectedHeaders: map[string]string{},
		},
		{
			name: "Malformed Header",
			input: "GET / HTTP/1.1\r\n" +
				"Host localhost:42069\r\n" +
				"\r\n",
			numBytesPerRead: 3,
			expectError:     true,
		},
		{
			name: "Duplicate Headers",
			input: "GET / HTTP/1.1\r\n" +
				"Set-Person: lane-loves-go\r\n" +
				"Set-Person: prime-loves-zig\r\n" +
				"Set-Person: tj-loves-ocaml\r\n" +
				"\r\n",
			numBytesPerRead: 4,
			expectedHeaders: map[string]string{
				"set-person": "lane-loves-go, prime-loves-zig, tj-loves-ocaml",
			},
		},
		{
			name: "Case Insensitive Headers",
			input: "GET / HTTP/1.1\r\n" +
				"hOsT: localhost:42069\r\n" +
				"HOST: localhost:42070\r\n" +
				"\r\n",
			numBytesPerRead: 1,
			expectedHeaders: map[string]string{
				"host": "localhost:42069, localhost:42070",
			},
		},
		{
			name: "Missing End of Headers",
			input: "GET / HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n",
			numBytesPerRead: 3,
			expectError:     true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reader := &chunkReader{
				data:            testCase.input,
				numBytesPerRead: testCase.numBytesPerRead,
			}

			request, err := RequestFromReader(reader)
			if testCase.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, request)
			for key, expectedValue := range testCase.expectedHeaders {
				assert.Equal(t, expectedValue, request.Headers[key])
			}
		})
	}
}

func TestRequestFromReaderBody(t *testing.T) {
	testCases := []struct {
		name            string
		input           string
		numBytesPerRead int
		expectError     bool
		expectedBody    string
	}{
		{
			name: "Standard Body",
			input: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"Content-Length: 13\r\n" +
				"\r\n" +
				"hello world!\n",
			numBytesPerRead: 3,
			expectedBody:    "hello world!\n",
		},
		{
			name: "Empty Body, 0 reported content length",
			input: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"Content-Length: 0\r\n" +
				"\r\n",
			numBytesPerRead: 3,
			expectedBody:    "",
		},
		{
			name: "Empty Body, no reported content length",
			input: "GET / HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n",
			numBytesPerRead: 3,
			expectedBody:    "",
		},
		{
			name: "Body shorter than reported content length",
			input: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"Content-Length: 20\r\n" +
				"\r\n" +
				"partial content",
			numBytesPerRead: 3,
			expectError:     true,
		},
		{
			name: "No Content-Length but Body Exists",
			input: "POST /submit HTTP/1.1\r\n" +
				"Host: localhost:42069\r\n" +
				"\r\n" +
				"this body will be ignored",
			numBytesPerRead: 3,
			expectedBody:    "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			reader := &chunkReader{
				data:            testCase.input,
				numBytesPerRead: testCase.numBytesPerRead,
			}

			request, err := RequestFromReader(reader)
			if testCase.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, request)
			assert.Equal(t, testCase.expectedBody, string(request.Body))
		})
	}
}
