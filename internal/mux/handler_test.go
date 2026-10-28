package mux

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Shared helper to register a test route that echoes its pattern
func register(h *MyHandler, pattern string) {
	h.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(pattern))
	})
}

// Shared helper to execute a request and extract the response
func executeRequest(h http.Handler, path string) (int, string, string) {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	res := w.Result()
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	location := res.Header.Get("Location")

	return res.StatusCode, string(body), location
}

// ---------------------------------------------------------
// Tests for standard ServeMux compliance (Prefix, Redirects)
// ---------------------------------------------------------

func TestServeMuxCompliance(t *testing.T) {
	t.Run("Non-slash pattern is strictly exact", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/items")

		tests := []struct {
			path           string
			expectedStatus int
			expectedBody   string
		}{
			{
				path:           "/items",
				expectedStatus: http.StatusOK,
				expectedBody:   "/items",
			},
			{
				path:           "/items/",
				expectedStatus: http.StatusNotFound, // Trailing slash is NOT matched by /items
			},
			{
				path:           "/items/123",
				expectedStatus: http.StatusNotFound, // Child paths are NOT matched by /items
			},
		}

		for _, tt := range tests {
			status, body, _ := executeRequest(h, tt.path)
			if status != tt.expectedStatus {
				t.Errorf("Path %q: expected status %d, got %d", tt.path, tt.expectedStatus, status)
			}
			if tt.expectedStatus == http.StatusOK && body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
		}
	})

	t.Run("Trailing slash pattern acts as a subtree prefix", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/tree/")

		tests := []struct {
			path             string
			expectedStatus   int
			expectedBody     string
			expectedLocation string
		}{
			{
				path:           "/tree/",
				expectedStatus: http.StatusOK,
				expectedBody:   "/tree/",
			},
			{
				path:           "/tree/sub",
				expectedStatus: http.StatusOK,
				expectedBody:   "/tree/",
			},
			{
				path:           "/tree/sub/deep",
				expectedStatus: http.StatusOK,
				expectedBody:   "/tree/",
			},
			{
				path:             "/tree",
				expectedStatus:   http.StatusMovedPermanently, // 301 redirect to /tree/
				expectedLocation: "/tree/",
			},
		}

		for _, tt := range tests {
			status, body, location := executeRequest(h, tt.path)
			if status != tt.expectedStatus {
				t.Errorf("Path %q: expected status %d, got %d", tt.path, tt.expectedStatus, status)
			}
			if tt.expectedStatus == http.StatusOK && body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
			if tt.expectedStatus == http.StatusMovedPermanently && location != tt.expectedLocation {
				t.Errorf("Path %q: expected Location %q, got %q", tt.path, tt.expectedLocation, location)
			}
		}
	})

	t.Run("Exact non-slash suppresses 301 redirect", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/api")
		register(h, "/api/")

		tests := []struct {
			path           string
			expectedStatus int
			expectedBody   string
		}{
			{
				path:           "/api",
				expectedStatus: http.StatusOK,
				expectedBody:   "/api", // Handled directly, NO 301 redirect
			},
			{
				path:           "/api/",
				expectedStatus: http.StatusOK,
				expectedBody:   "/api/",
			},
			{
				path:           "/api/users",
				expectedStatus: http.StatusOK,
				expectedBody:   "/api/", // Matched by /api/
			},
		}

		for _, tt := range tests {
			status, body, _ := executeRequest(h, tt.path)
			if status != tt.expectedStatus {
				t.Errorf("Path %q: expected status %d, got %d", tt.path, tt.expectedStatus, status)
			}
			if body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
		}
	})

	t.Run("Longest prefix wins among overlapping subtrees", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/files/")
		register(h, "/files/images/")

		tests := []struct {
			path         string
			expectedBody string
		}{
			{
				path:         "/files/doc.pdf",
				expectedBody: "/files/",
			},
			{
				path:         "/files/images/pic.png",
				expectedBody: "/files/images/", // More specific prefix wins
			},
		}

		for _, tt := range tests {
			status, body, _ := executeRequest(h, tt.path)
			if status != http.StatusOK {
				t.Errorf("Path %q: expected status 200, got %d", tt.path, status)
			}
			if body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
		}
	})

	t.Run("Root slash acts as a catch-all for unmatched routes", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/")
		register(h, "/specific")

		tests := []struct {
			path         string
			expectedBody string
		}{
			{
				path:         "/specific",
				expectedBody: "/specific",
			},
			{
				path:         "/",
				expectedBody: "/",
			},
			{
				path:         "/unregistered/nested/path",
				expectedBody: "/", // Falls back to root /
			},
		}

		for _, tt := range tests {
			status, body, _ := executeRequest(h, tt.path)
			if status != http.StatusOK {
				t.Errorf("Path %q: expected status 200, got %d", tt.path, status)
			}
			if body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
		}
	})
}

// ---------------------------------------------------------
// Tests for Go 1.22+ end-of-path pattern matching ({$})
// ---------------------------------------------------------

func TestEndOfPathDollarPatterns(t *testing.T) {
	t.Run("Case 1: Root /{$} alone matches only exact root /", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/{$}")

		tests := []struct {
			path           string
			expectedStatus int
			expectedBody   string
		}{
			{
				path:           "/",
				expectedStatus: http.StatusOK,
				expectedBody:   "/{$}",
			},
			{
				path:           "/foo",
				expectedStatus: http.StatusNotFound,
			},
			{
				path:           "/foo/",
				expectedStatus: http.StatusNotFound,
			},
		}

		for _, tt := range tests {
			status, body, _ := executeRequest(h, tt.path)
			if status != tt.expectedStatus {
				t.Errorf("Path %q: expected status %d, got %d", tt.path, tt.expectedStatus, status)
			}
			if tt.expectedStatus == http.StatusOK && body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
		}
	})

	t.Run("Case 2: Subpath .../{$} alone matches only exact directory with trailing slash", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/mypath/{$}")

		tests := []struct {
			path           string
			expectedStatus int
			expectedBody   string
		}{
			{
				path:           "/mypath/",
				expectedStatus: http.StatusOK,
				expectedBody:   "/mypath/{$}",
			},
			{
				path:           "/mypath/sub",
				expectedStatus: http.StatusNotFound, // Not a subtree match
			},
			{
				path:           "/mypath/sub/deep",
				expectedStatus: http.StatusNotFound,
			},
			{
				path:           "/mypath",
				expectedStatus: http.StatusNotFound, // Exact non-slash was not registered
			},
		}

		for _, tt := range tests {
			status, body, _ := executeRequest(h, tt.path)
			if status != tt.expectedStatus {
				t.Errorf("Path %q: expected status %d, got %d", tt.path, tt.expectedStatus, status)
			}
			if tt.expectedStatus == http.StatusOK && body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
		}
	})

	t.Run("Case 3: Coexistence of /{$} and fallback /", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/{$}") // Exact root
		register(h, "/")    // Catch-all subtree

		tests := []struct {
			path         string
			expectedBody string
		}{
			{
				path:         "/",
				expectedBody: "/{$}", // More specific exact root wins
			},
			{
				path:         "/unregistered",
				expectedBody: "/", // Falls back to root catch-all
			},
			{
				path:         "/unregistered/nested/",
				expectedBody: "/", // Falls back to root catch-all
			},
		}

		for _, tt := range tests {
			status, body, _ := executeRequest(h, tt.path)
			if status != http.StatusOK {
				t.Errorf("Path %q: expected status 200, got %d", tt.path, status)
			}
			if body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
		}
	})

	t.Run("Case 4: Coexistence of /mypath/ and /mypath/{$}", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/mypath/{$}") // More specific: only /mypath/
		register(h, "/mypath/")    // Subtree: /mypath/*

		tests := []struct {
			path             string
			expectedStatus   int
			expectedBody     string
			expectedLocation string
		}{
			{
				path:           "/mypath/",
				expectedStatus: http.StatusOK,
				expectedBody:   "/mypath/{$}", // Exact match wins over prefix
			},
			{
				path:           "/mypath/sub",
				expectedStatus: http.StatusOK,
				expectedBody:   "/mypath/", // Handled by subtree prefix
			},
			{
				path:           "/mypath/a/b/c",
				expectedStatus: http.StatusOK,
				expectedBody:   "/mypath/", // Handled by subtree prefix
			},
			{
				path:             "/mypath",
				expectedStatus:   http.StatusMovedPermanently, // Triggered by /mypath/ subtree
				expectedLocation: "/mypath/",
			},
		}

		for _, tt := range tests {
			status, body, location := executeRequest(h, tt.path)
			if status != tt.expectedStatus {
				t.Errorf("Path %q: expected status %d, got %d", tt.path, tt.expectedStatus, status)
			}
			if tt.expectedStatus == http.StatusOK && body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
			if tt.expectedStatus == http.StatusMovedPermanently && location != tt.expectedLocation {
				t.Errorf("Path %q: expected redirect %q, got %q", tt.path, tt.expectedLocation, location)
			}
		}
	})

	t.Run("Case 5: Multi-level exact paths with /{$}", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/api/v1/users/{$}")
		register(h, "/api/v1/{$}")

		tests := []struct {
			path           string
			expectedStatus int
			expectedBody   string
		}{
			{
				path:           "/api/v1/users/",
				expectedStatus: http.StatusOK,
				expectedBody:   "/api/v1/users/{$}",
			},
			{
				path:           "/api/v1/",
				expectedStatus: http.StatusOK,
				expectedBody:   "/api/v1/{$}",
			},
			{
				path:           "/api/v1/users/123",
				expectedStatus: http.StatusNotFound, // Not a wildcard
			},
			{
				path:           "/api/v1/posts/",
				expectedStatus: http.StatusNotFound,
			},
		}

		for _, tt := range tests {
			status, body, _ := executeRequest(h, tt.path)
			if status != tt.expectedStatus {
				t.Errorf("Path %q: expected status %d, got %d", tt.path, tt.expectedStatus, status)
			}
			if tt.expectedStatus == http.StatusOK && body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
		}
	})

	t.Run("Case 6: Explicit non-slash and slash with /{$} side-by-side", func(t *testing.T) {
		h := NewMyHandler()
		register(h, "/users")
		register(h, "/users/{$}")

		tests := []struct {
			path           string
			expectedStatus int
			expectedBody   string
		}{
			{
				path:           "/users",
				expectedStatus: http.StatusOK,
				expectedBody:   "/users",
			},
			{
				path:           "/users/",
				expectedStatus: http.StatusOK,
				expectedBody:   "/users/{$}",
			},
			{
				path:           "/users/all",
				expectedStatus: http.StatusNotFound,
			},
		}

		for _, tt := range tests {
			status, body, _ := executeRequest(h, tt.path)
			if status != tt.expectedStatus {
				t.Errorf("Path %q: expected status %d, got %d", tt.path, tt.expectedStatus, status)
			}
			if tt.expectedStatus == http.StatusOK && body != tt.expectedBody {
				t.Errorf("Path %q: expected body %q, got %q", tt.path, tt.expectedBody, body)
			}
		}
	})

	//t.Run("Case 7: Invalid {$} patterns must panic at registration time", func(t *testing.T) {
	//	invalidPatterns := []string{
	//		"/{$}/",       // {$} not at end
	//		"/{$}/extra",  // {$} not at end
	//		"/api/{$}/v1", // {$} in middle of path
	//		"/{$foo}",     // Malformed wildcard
	//		"/{$...}",     // Cannot be followed by ellipsis
	//	}
	//
	//	for _, pattern := range invalidPatterns {
	//		t.Run(pattern, func(t *testing.T) {
	//			defer func() {
	//				r := recover()
	//				if r == nil {
	//					t.Errorf("Pattern %q should have panicked at registration, but it succeeded", pattern)
	//				}
	//			}()
	//
	//			h := NewMyHandler()
	//			register(h, pattern)
	//		})
	//	}
	//})
}
