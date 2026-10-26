package http1

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
)

func ListenAndServe(addr string, handler http.Handler) error {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	if handler == nil {
		handler = http.DefaultServeMux
	}

	for {
		listener, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("accept: %w", err)
		}
		err = HandleConnection(listener, handler)
		if err != nil {
			return fmt.Errorf("handle: %w", err)
		}

		//panic("ListenAndServe: Not implemented")
		/*
				Functions that might be useful:
				* listener.Accept()
			  * HandleConnection()
		*/
	}
}

func HandleConnection(conn net.Conn, mux http.Handler) error {
	fmt.Println("handling connection")
	reader := bufio.NewReader(conn)

	req, err := ParseRequest(reader)
	if err != nil {
		return err
	}

	resp := NewResponse(conn)
	mux.ServeHTTP(resp, req)

	return nil
}
