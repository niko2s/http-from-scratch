package http1

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

type Response struct {
	conn net.Conn
	h    http.Header
}

func NewResponse(conn net.Conn) *Response {
	return &Response{conn: conn, h: http.Header{}}
}

func (r *Response) Header() http.Header {
	return r.h
}

func (r *Response) Write(data []byte) (n int, err error) {

	nn, err := r.conn.Write(data)

	if err != nil {
		return 0, fmt.Errorf("error writing response: %s", err)
	}
	err = r.conn.Close()
	if err != nil {
		return 0, err
	}
	return nn, nil

}

func (r *Response) WriteHeader(statusCode int) {

	var header strings.Builder

	statusLine := fmt.Sprintf("HTTP/1.1 %d %s\r\n", statusCode, http.StatusText(statusCode))
	header.WriteString(statusLine)
	for k, values := range r.h {
		for _, v := range values {
			header.WriteString(fmt.Sprintf("%s: %v\r\n", k, v))
		}
	}
	header.WriteString("\r\n")

	r.conn.Write([]byte(header.String()))
}
