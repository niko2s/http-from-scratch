package http1

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
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
			} else if errors.Is(err, ErrUriTooLong) {
				resp.WriteHeader(http.StatusRequestURITooLong)
			} else if errors.Is(err, ErrTooManyHeaders) {
				resp.WriteHeader(http.StatusRequestHeaderFieldsTooLarge)
			} else {
				resp.WriteHeader(http.StatusBadRequest)
			}
			resp.Write(nil)
			return nil
		}

		closeConn := shouldClose(req)

		if closeConn {
			resp.Header().Set("Connection", "close")
		} else if req.ProtoMinor == 0 {
			resp.Header().Set("Connection", "keep-alive")
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

		if closeConn {
			return nil
		}
	}
}

func shouldClose(req *http.Request) bool {
	conn := req.Header.Get("Connection")
	minor := req.ProtoMinor
	if minor == 0 {
		return !strings.EqualFold(conn, "keep-alive")
	}
	return strings.EqualFold(conn, "close")
}
