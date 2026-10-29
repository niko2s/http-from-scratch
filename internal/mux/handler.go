package mux

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type MyHandler struct {
	routes      map[string]func(http.ResponseWriter, *http.Request)
	exactRoutes map[string]func(http.ResponseWriter, *http.Request)
}

func NewMyHandler() *MyHandler {
	return &MyHandler{
		routes:      make(map[string]func(http.ResponseWriter, *http.Request)),
		exactRoutes: make(map[string]func(http.ResponseWriter, *http.Request)),
	}
}

func (h MyHandler) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	if strings.HasSuffix(pattern, "/{$}") {
		p := strings.TrimSuffix(pattern, "{$}")
		h.exactRoutes[p] = handler
	} else {
		h.routes[pattern] = handler
	}
}

func (h MyHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	start := time.Now()
	defer func() {
		log.Printf("%s %s %s (%s)", request.Method, request.URL.Path, request.RemoteAddr, time.Since(start))
	}()
	
	pattern := request.URL.Path
	fn := h.searchFnByPattern(pattern)
	if fn == nil {
		http.NotFound(writer, request)
		return
	}
	fn(writer, request)
}

func (h MyHandler) searchFnByPattern(pattern string) func(http.ResponseWriter, *http.Request) {

	// exact match
	fn := h.exactRoutes[pattern]
	if fn != nil {
		return fn
	}

	fn = h.routes[pattern]
	if fn != nil {
		return fn
	}

	// if without / check with slash
	if !strings.HasSuffix(pattern, "/") {
		patternTree := pattern + "/"
		fn = h.routes[patternTree]
		if fn != nil {
			return func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, patternTree, http.StatusMovedPermanently)
			}
		}
	}

	p := pattern
	for {
		p, _, _ = strings.CutLast(p, "/")

		fn = h.routes[p+"/"]
		if fn != nil {
			return fn
		}

		if p == "" {
			return nil
		}
	}
}
