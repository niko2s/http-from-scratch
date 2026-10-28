package main

import (
	"httpServer/internal/http1"
	"httpServer/internal/mux"
	"net/http"
)

const addr = ":8080"

func main() {

	mux := mux.NewMyHandler()
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		body := []byte("hello")
		w.Write(body)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body := []byte("default")
		w.Write(body)
	})

	if err := http1.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}
