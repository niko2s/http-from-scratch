package main

import (
	"httpServer/internal/http1"
	"httpServer/internal/mux"
	"net/http"

	"github.com/VictoriaMetrics/metrics"
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

	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, true)
	})

	if err := http1.ListenAndServe(addr, mux); err != nil {
		panic(err)
	}
}
