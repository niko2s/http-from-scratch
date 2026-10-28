package http1

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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

	for {
		resp := NewResponse(conn)

		req, err := ParseRequest(reader)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			if errors.Is(err, ErrVersionNotSupported) {
				resp.WriteHeader(http.StatusHTTPVersionNotSupported)
			} else {
				resp.WriteHeader(http.StatusBadRequest)
			}
			resp.Write(nil)
			return nil
		}

		mux.ServeHTTP(resp, req)

		// handler wrote nothing, still send a response
		if !resp.wroteBody {
			resp.Write(nil)
		}

		if req.Body != nil {
			io.Copy(io.Discard, req.Body)
			req.Body.Close()
		}
	}
}
