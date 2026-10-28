package http1

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var (
	ErrVersionNotSupported = errors.New("http version not supported")
	ErrBadRequest          = errors.New("bad request")
)

func ParseRequest(reader *bufio.Reader) (*http.Request, error) {
	req := &http.Request{Header: http.Header{}}

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	requestLine = strings.TrimSuffix(requestLine, "\r\n")
	requestLineArr := strings.Split(requestLine, " ")
	if len(requestLineArr) != 3 {
		return nil, fmt.Errorf("invalid request line")
	}

	method := requestLineArr[0]
	target := requestLineArr[1]
	httpVer := requestLineArr[2]

	if !isToken(method) {
		return nil, fmt.Errorf("%w: invalid method: %q", ErrBadRequest, method)
	}

	verArr := strings.Split(strings.TrimPrefix(httpVer, "HTTP/"), ".")
	if !strings.HasPrefix(httpVer, "HTTP/") || len(verArr) != 2 {
		return nil, fmt.Errorf("%w: http version: %s", ErrBadRequest, httpVer)
	}

	major, err := strconv.Atoi(verArr[0])
	if err != nil {
		return nil, fmt.Errorf("%w: http version: %s, major not a number", ErrBadRequest, httpVer)
	}
	minor, err := strconv.Atoi(verArr[1])
	if err != nil {
		return nil, fmt.Errorf("%w: http version: %s, minor not a number", ErrBadRequest, httpVer)
	}

	if major != 1 {
		return nil, fmt.Errorf("%w: major: %d not supported", ErrVersionNotSupported, major)
	}
	if minor != 0 && minor != 1 {
		return nil, fmt.Errorf("%w: minor: %d not supported", ErrVersionNotSupported, minor)
	}

	fmt.Printf("method: %s, target: %s, httpVer: %s\n", method, target, httpVer)

	req.Method = method
	req.URL, err = url.ParseRequestURI(target)
	if err != nil {
		return nil, fmt.Errorf("failed parsing request url")
	}
	req.ProtoMajor = 1
	req.ProtoMinor = 1

	//parse header k:v
	for {
		line, prefix, err := reader.ReadLine()
		header := string(line)

		if prefix {
			fmt.Printf("isPrefix %t\n", prefix)
		}
		if err != nil {
			return nil, fmt.Errorf("%w: reading header: %s", ErrBadRequest, err)
		}
		if len(line) == 0 {
			break
		}

		kv := strings.SplitN(header, ":", 2)
		if len(kv) != 2 || !isToken(kv[0]) || strings.ContainsAny(kv[1], "\t\r") {
			return nil, fmt.Errorf("%w: invalid header: %q", ErrBadRequest, header)
		}

		req.Header.Add(kv[0], strings.TrimSpace(kv[1]))

	}

	req.Body = http.NoBody

	if method == http.MethodPost {
		clStr := req.Header.Get("Content-Length")
		cl, _ := strconv.ParseInt(clStr, 10, 64)
		if cl > 0 {
			req.Body = io.NopCloser(io.LimitReader(reader, cl))
		}
		fmt.Println("POST")
	}

	return req, nil
}

// only letters, digits, '-' and '_'
func isToken(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		isLetter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		if !isLetter && !isDigit && c != '-' && c != '_' {
			return false
		}
	}
	return true
}
