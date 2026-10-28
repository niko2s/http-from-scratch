package http1

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const maxRequestLineLength = 8192
const maxAmountHeaders = 100 //amount of max headers, but would be better if actual size, or limit size of fields too
const maxBodySize = 10 << 20 // 10 MiB

var (
	ErrVersionNotSupported = errors.New("http version not supported")
	ErrBadRequest          = errors.New("bad request")
	ErrUriTooLong          = errors.New("request uri is too long")
	ErrTooManyHeaders      = errors.New("too many headers")
)

func ParseRequest(reader *bufio.Reader) (*http.Request, error) {
	req := &http.Request{Header: http.Header{}}

	requestLine, err := readLine(reader)
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

	major, minor, err := parseHttpVer(httpVer)
	if err != nil {
		return nil, err
	}

	req.Method = method
	req.URL, err = url.ParseRequestURI(target)
	if err != nil {
		return nil, fmt.Errorf("failed parsing request url")
	}
	req.ProtoMajor = major
	req.ProtoMinor = minor

	var amountHeaders int
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

		if amountHeaders >= maxAmountHeaders {
			return nil, ErrTooManyHeaders
		}

		kv := strings.SplitN(header, ":", 2)
		if len(kv) != 2 || !isToken(kv[0]) || strings.ContainsAny(kv[1], "\t\r") {
			return nil, fmt.Errorf("%w: invalid header: %q", ErrBadRequest, header)
		}

		req.Header.Add(kv[0], strings.TrimSpace(kv[1]))
		amountHeaders += 1

	}

	req.Body = http.NoBody

	// read the whole body here, so a broken body becomes a 400 before the handler runs
	body, err := readBody(reader, req.Header)
	if err != nil {
		return nil, err
	}
	if len(body) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	return req, nil
}

// Transfer-Encoding chunked wins over Content-Length, no header means no body
func readBody(reader *bufio.Reader, header http.Header) ([]byte, error) {
	if strings.EqualFold(header.Get("Transfer-Encoding"), "chunked") {
		return readChunkedBody(reader)
	}

	clValues := header.Values("Content-Length")
	if len(clValues) == 0 {
		return nil, nil
	}
	for _, v := range clValues {
		if v != clValues[0] {
			return nil, fmt.Errorf("%w: conflicting content-length: %v", ErrBadRequest, clValues)
		}
	}

	cl, err := strconv.ParseInt(clValues[0], 10, 64)
	if err != nil || cl < 0 || cl > maxBodySize {
		return nil, fmt.Errorf("%w: invalid content-length: %q", ErrBadRequest, clValues[0])
	}

	body := make([]byte, cl)
	if _, err := io.ReadFull(reader, body); err != nil {
		return nil, fmt.Errorf("%w: body shorter than content-length", ErrBadRequest)
	}
	return body, nil
}

// chunk: <size in hex>[;ext]\r\n<data>\r\n ... 0\r\n[trailers]\r\n
func readChunkedBody(reader *bufio.Reader) ([]byte, error) {
	var body []byte
	for {
		sizeLine, err := readLine(reader)
		if err != nil {
			return nil, err
		}

		// ignore chunk extensions after ';'
		sizeStr, _, _ := strings.Cut(sizeLine, ";")
		size, err := strconv.ParseInt(strings.TrimSpace(sizeStr), 16, 64)
		if err != nil || size < 0 || size > maxBodySize-int64(len(body)) {
			return nil, fmt.Errorf("%w: invalid chunk size: %q", ErrBadRequest, sizeLine)
		}
		if size == 0 {
			break
		}

		chunk := make([]byte, size+2) // data + \r\n
		if _, err := io.ReadFull(reader, chunk); err != nil {
			return nil, fmt.Errorf("%w: chunk shorter than size", ErrBadRequest)
		}
		if string(chunk[size:]) != "\r\n" {
			return nil, fmt.Errorf("%w: chunk not terminated by CRLF", ErrBadRequest)
		}
		body = append(body, chunk[:size]...)
	}

	// skip trailers until empty line
	for {
		line, err := readLine(reader)
		if err != nil {
			return nil, err
		}
		if line == "" {
			break
		}
	}
	return body, nil
}

func parseHttpVer(httpVer string) (int, int, error) {
	verArr := strings.Split(strings.TrimPrefix(httpVer, "HTTP/"), ".")
	if !strings.HasPrefix(httpVer, "HTTP/") || len(verArr) != 2 {
		return 0, 0, fmt.Errorf("%w: http version: %s", ErrBadRequest, httpVer)
	}

	major, err := strconv.Atoi(verArr[0])
	if err != nil {
		return 0, 0, fmt.Errorf("%w: http version: %s, major not a number", ErrBadRequest, httpVer)
	}
	minor, err := strconv.Atoi(verArr[1])
	if err != nil {
		return 0, 0, fmt.Errorf("%w: http version: %s, minor not a number", ErrBadRequest, httpVer)
	}

	if major != 1 {
		return 0, 0, fmt.Errorf("%w: major: %d not supported", ErrVersionNotSupported, major)
	}
	if minor != 0 && minor != 1 {
		return 0, 0, fmt.Errorf("%w: minor: %d not supported", ErrVersionNotSupported, minor)
	}
	return major, minor, nil
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("%w: failed reading line", ErrBadRequest)
	}
	res := strings.TrimSuffix(line, "\r\n")

	if len(res) > maxRequestLineLength {
		return "", fmt.Errorf("%w", ErrUriTooLong)
	}

	if strings.ContainsRune(res, '\r') {
		return "", fmt.Errorf("%w: %s", ErrBadRequest, res)
	}

	return res, nil
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
