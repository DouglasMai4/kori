package kori

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Handler is an HTTP handler that returns an error.
// Errors are caught and handled by the configured ErrorHandler.
type Handler func(w http.ResponseWriter, r *http.Request) error

// Middleware is a standard HTTP middleware.
type Middleware = func(http.Handler) http.Handler

// GET registers a handler for the GET method.
//
//	kori.GET(r, "/users", listUsers)
//	kori.GET(r, "/users/{id}", getUser, kori.Use(adminOnly))
func GET(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodGet, pattern, h, opts...)
}

// POST registers a handler for the POST method.
func POST(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodPost, pattern, h, opts...)
}

// PUT registers a handler for the PUT method.
func PUT(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodPut, pattern, h, opts...)
}

// PATCH registers a handler for the PATCH method.
func PATCH(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodPatch, pattern, h, opts...)
}

// DELETE registers a handler for the DELETE method.
func DELETE(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodDelete, pattern, h, opts...)
}

// HEAD registers a handler for the HEAD method.
func HEAD(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodHead, pattern, h, opts...)
}

// OPTIONS registers a handler for the OPTIONS method.
func OPTIONS(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodOptions, pattern, h, opts...)
}

func register(r chi.Router, method, pattern string, h Handler, opts ...Option) {
	if pattern == "" {
		pattern = "/"
	}

	fullPattern := pattern
	if kr, ok := r.(*Router); ok {
		fullPattern = kr.prefix + pattern
		if pattern == "/" && kr.prefix != "" {
			fullPattern = kr.prefix
		}
	}

	ri := &RouteInfo{Method: method, Pattern: fullPattern}

	for _, opt := range opts {
		opt(ri)
	}

	final := wrap(h)

	if len(ri.middlewares) > 0 {
		r.Method(method, pattern, chi.Chain(ri.middlewares...).Handler(final))
		return
	}

	r.Method(method, pattern, final)
}

func wrap(h Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			getErrorHandler()(w, r, err)
		}
	}
}
