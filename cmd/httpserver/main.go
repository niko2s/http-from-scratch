package main

import (
	"httpServer/internal/http1"
	"net/http"
	"strconv"
)

const addr = ":8080"

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := []byte("this is the response from server\n")
		w.Header().Add("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(200)
		w.Write(body)
	})

	if err := http1.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}
}
