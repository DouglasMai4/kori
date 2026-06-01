package kori

import "github.com/go-chi/chi/v5"

func Group(r chi.Router, prefix string, middlewares ...Middleware) chi.Router {
	var sub chi.Router

	r.Route(prefix, func(sr chi.Router) {
		for _, m := range middlewares {
			sr.Use(m)
		}
		sub = sr
	})

	return sub
}
