package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"httpfromtcp/internal/headers"
	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
	"httpfromtcp/internal/server"
)

const port = 42069

func handler(w *response.Writer, req *request.Request) {
	if strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin/") {
		upstreamPath := strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin")
		upstreamURL := "https://httpbin.org" + upstreamPath

		upstreamResp, err := http.Get(upstreamURL)
		if err != nil {
			body := []byte("upstream request failed")
			if err := w.WriteStatusLine(response.StatusInternalServerError); err != nil {
				return
			}
			headers := response.GetDefaultHeaders(len(body))
			headers.Set("Content-Type", "text/plain")
			if err := w.WriteHeaders(headers); err != nil {
				return
			}
			_, _ = w.WriteBody(body)
			return
		}
		defer upstreamResp.Body.Close()

		if err := w.WriteStatusLine(response.StatusCode(upstreamResp.StatusCode)); err != nil {
			return
		}
		hdrs := response.GetDefaultHeaders(0)
		hdrs.Delete("Content-Length")
		hdrs.Set("Transfer-Encoding", "chunked")
		hdrs.Set("Trailer", "X-Content-SHA256, X-Content-Length")
		if contentType := upstreamResp.Header.Get("Content-Type"); contentType != "" {
			hdrs.Set("Content-Type", contentType)
		}
		if err := w.WriteHeaders(hdrs); err != nil {
			return
		}

		var fullBody []byte
		buf := make([]byte, 1024)
		for {
			n, readErr := upstreamResp.Body.Read(buf)
			if n > 0 {
				fullBody = append(fullBody, buf[:n]...)
				if _, err := w.WriteChunkedBody(buf[:n]); err != nil {
					return
				}
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				return
			}
		}
		if _, err := w.WriteChunkedBodyDone(); err != nil {
			return
		}

		hash := sha256.Sum256(fullBody)
		trailers := headers.NewHeaders()
		trailers.Set("X-Content-SHA256", fmt.Sprintf("%x", hash))
		trailers.Set("X-Content-Length", fmt.Sprintf("%d", len(fullBody)))
		_ = w.WriteTrailers(trailers)
		return
	}

	statusCode := response.StatusOK
	body := []byte(`<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
    <p>Your request was an absolute banger.</p>
  </body>
</html>
`)

	switch req.RequestLine.RequestTarget {
	case "/video":
		videoData, err := os.ReadFile("assets/vim.mp4")
		if err != nil {
			body := []byte("video not found")
			if err := w.WriteStatusLine(response.StatusInternalServerError); err != nil {
				return
			}
			hdrs := response.GetDefaultHeaders(len(body))
			if err := w.WriteHeaders(hdrs); err != nil {
				return
			}
			_, _ = w.WriteBody(body)
			return
		}
		if err := w.WriteStatusLine(response.StatusOK); err != nil {
			return
		}
		hdrs := response.GetDefaultHeaders(len(videoData))
		hdrs.Set("Content-Type", "video/mp4")
		if err := w.WriteHeaders(hdrs); err != nil {
			return
		}
		_, _ = w.WriteBody(videoData)
		return
	case "/yourproblem":
		statusCode = response.StatusBadRequest
		body = []byte(`<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
    <p>Your request honestly kinda sucked.</p>
  </body>
</html>
`)
	case "/myproblem":
		statusCode = response.StatusInternalServerError
		body = []byte(`<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
    <p>Okay, you know what? This one is on me.</p>
  </body>
</html>
`)
	}

	if err := w.WriteStatusLine(statusCode); err != nil {
		return
	}
	headers := response.GetDefaultHeaders(len(body))
	headers.Set("Content-Type", "text/html")
	if err := w.WriteHeaders(headers); err != nil {
		return
	}
	_, _ = w.WriteBody(body)
}

func main() {
	s, err := server.Serve(port, handler)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer s.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}
