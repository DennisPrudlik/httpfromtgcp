package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"strings"
)

func getLinesChannel(f io.ReadCloser) <-chan string {
	lines := make(chan string)
	go func() {
		defer close(lines)
		defer f.Close()

		buffer := make([]byte, 8)
		var currentLine string

		for {
			n, err := f.Read(buffer)
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Println("Error reading file:", err)
				return
			}
			parts := strings.Split(string(buffer[:n]), "\n")
			for i := 0; i < len(parts)-1; i++ {
				lines <- currentLine + parts[i]
				currentLine = ""
			}
			if len(parts) > 0 {
				currentLine += parts[len(parts)-1]
			}
		}

		if currentLine != "" {
			lines <- currentLine
		}
	}()
	return lines
}

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

		f, err := os.Create("/tmp/tcp.txt")
		if err != nil {
			fmt.Println("Error creating file:", err)
			continue
		}

		for line := range getLinesChannel(conn) {
			fmt.Printf("read: %s\n", line)
			fmt.Fprintln(f, line)
		}
		f.Close()
		fmt.Println("Connection closed")
	}
}
