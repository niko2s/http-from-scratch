package main

import (
	"httpServer/internal/http1"
	"net/http"
)

const addr = ":8080"

func main() {
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		body := []byte("hello")

		w.Write(body)
	})

	if err := http1.ListenAndServe(addr, nil); err != nil {
		panic(err)
	}
}
