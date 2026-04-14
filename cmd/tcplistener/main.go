package main

import (
	"fmt"
	"net"
	"sort"

	request "httpfromtcp/internal/request"
)

func main() {
	const port = 42069
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		fmt.Println("Connection accepted")

		parsedRequest, err := request.RequestFromReader(conn)
		if err != nil {
			fmt.Println("Error parsing request:", err)
			conn.Close()
			continue
		}

		fmt.Println("Request line:")
		fmt.Printf("- Method: %s\n", parsedRequest.RequestLine.Method)
		fmt.Printf("- Target: %s\n", parsedRequest.RequestLine.RequestTarget)
		fmt.Printf("- Version: %s\n", parsedRequest.RequestLine.HttpVersion)
		fmt.Println("Headers:")

		headerKeys := make([]string, 0, len(parsedRequest.Headers))
		for key := range parsedRequest.Headers {
			headerKeys = append(headerKeys, key)
		}
		sort.Strings(headerKeys)
		for _, key := range headerKeys {
			fmt.Printf("- %s: %s\n", key, parsedRequest.Headers[key])
		}
		fmt.Println("Body:")
		fmt.Println(string(parsedRequest.Body))

		conn.Close()
		fmt.Println("Connection closed")
	}
}
