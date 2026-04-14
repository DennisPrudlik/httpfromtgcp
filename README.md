# HTTP from TCP

A learning project that builds an HTTP server directly on top of raw TCP in Go.

This repo includes:
- A low-level HTTP request parser (request line, headers, body)
- A TCP-based HTTP server and handler system
- A response writer with status/header/body state enforcement
- Chunked transfer encoding and HTTP trailers
- A small proxy route and binary video serving example

## Project Structure

- `cmd/httpserver/main.go` — main HTTP server entrypoint and routes
- `cmd/tcplistener/main.go` — raw TCP listener used in earlier parser exercises
- `internal/request/` — request parsing logic
- `internal/headers/` — header parsing and utilities
- `internal/response/` — HTTP response writing helpers and `Writer`
- `internal/server/` — TCP server, connection handling, handler dispatch

## Requirements

- Go 1.25+

## Run

```bash
cd /Users/dennisprudlik/Desktop/httpfromtcp
go run ./cmd/httpserver
```

Server listens on `http://localhost:42069`.

## Test

```bash
cd /Users/dennisprudlik/Desktop/httpfromtcp
go test ./...
```

## Build

```bash
cd /Users/dennisprudlik/Desktop/httpfromtcp
go build ./...
```

## Routes

- `GET /` — default HTML response
- `GET /yourproblem` — returns `400 Bad Request`
- `GET /myproblem` — returns `500 Internal Server Error`
- `GET /httpbin/*` — proxies to `https://httpbin.org/*` with chunked streaming + trailers
- `GET /video` — serves `assets/vim.mp4` as `video/mp4`

## Video Asset Note

The `assets/` directory is gitignored. If `assets/vim.mp4` is missing, download it:

```bash
cd /Users/dennisprudlik/Desktop/httpfromtcp
mkdir -p assets
curl -o assets/vim.mp4 https://storage.googleapis.com/qvault-webapp-dynamic-assets/lesson_videos/vim-vs-neovim-prime.mp4
```
