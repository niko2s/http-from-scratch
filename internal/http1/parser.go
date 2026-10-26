package http1

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// debug
func printAll(reader *bufio.Reader) {
	for {
		l, _, _ := reader.ReadLine()
		if len(l) == 0 {
			break
		}

		fmt.Printf("%s\n", string(l))
	}
	fmt.Printf("\n\n---- printAll end ----\n\n")
}

func ParseRequest(reader *bufio.Reader) (*http.Request, error) {

	//debug for comparison
	//printAll(reader)

	req := &http.Request{Header: http.Header{}}

	requestLine, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	requestLineArr := strings.Split(requestLine, " ")
	if len(requestLineArr) != 3 {
		return nil, fmt.Errorf("invalid request line")
	}

	method := requestLineArr[0]
	target := requestLineArr[1]
	httpVer := requestLineArr[2]

	fmt.Printf("method: %s, target: %s, httpVer: %s\n", method, target, httpVer)

	req.Method = method
	req.URL, err = url.ParseRequestURI(target)
	if err != nil {
		return nil, fmt.Errorf("failed parsing request url")
	}

	//parse header k:v
	for {
		line, prefix, err := reader.ReadLine()
		header := string(line)

		if prefix {
			fmt.Printf("isPrefix %t\n", prefix)
		}
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		if len(line) == 0 {
			fmt.Println("done")
			break
		}

		kv := strings.Split(header, ": ")
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid header")
		}

		req.Header.Add(kv[0], kv[1])

	}

	//fmt.Printf("printing headers:\n %s", req.Header)

	return req, nil
}
