package server

import (
	"fmt"
	"log"
	"net"
	"sync/atomic"

	"httpfromtcp/internal/request"
	"httpfromtcp/internal/response"
)

type Handler func(w *response.Writer, req *request.Request)

type Server struct {
	listener net.Listener
	closed   atomic.Bool
	handler  Handler
}

func Serve(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, fmt.Errorf("error starting server: %w", err)
	}
	s := &Server{listener: listener, handler: handler}
	go s.listen()
	return s, nil
}

func (s *Server) Close() error {
	s.closed.Store(true)
	return s.listener.Close()
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			log.Printf("Error accepting connection: %v", err)
			continue
		}
		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()
	writer := response.NewWriter(conn)

	req, err := request.RequestFromReader(conn)
	if err != nil {
		body := []byte(err.Error())
		if writeErr := writer.WriteStatusLine(response.StatusBadRequest); writeErr != nil {
			log.Printf("Error writing status line: %v", writeErr)
			return
		}
		headers := response.GetDefaultHeaders(len(body))
		if writeErr := writer.WriteHeaders(headers); writeErr != nil {
			log.Printf("Error writing headers: %v", writeErr)
			return
		}
		if _, writeErr := writer.WriteBody(body); writeErr != nil {
			log.Printf("Error writing body: %v", writeErr)
		}
		return
	}

	s.handler(writer, req)
}
