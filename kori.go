package kori

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler func(w http.ResponseWriter, r *http.Request) error

type Middleware = func(http.Handler) http.Handler

func GET(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodGet, pattern, h, opts...)
}

func POST(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodPost, pattern, h, opts...)
}

func PUT(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodPut, pattern, h, opts...)
}

func PATCH(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodPatch, pattern, h, opts...)
}

func DELETE(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodDelete, pattern, h, opts...)
}

func HEAD(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodHead, pattern, h, opts...)
}

func OPTIONS(r chi.Router, pattern string, h Handler, opts ...Option) {
	register(r, http.MethodOptions, pattern, h, opts...)
}

func register(r chi.Router, method, pattern string, h Handler, opts ...Option) {
	fullPattern := pattern
	if kr, ok := r.(*Router); ok {
		fullPattern = kr.prefix + pattern
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
