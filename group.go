package kori

import "github.com/go-chi/chi/v5"

// Router wraps chi.Router with a prefix for nested route groups.
type Router struct {
	chi.Router
	prefix string
}

// Group creates a sub-router with the given prefix and middlewares.
//
//	users := kori.Group(r, "/users", authMiddleware)
//	kori.GET(users, "", listUsers)
//	kori.GET(users, "/{id}", getUser)
func Group(r chi.Router, prefix string, middlewares ...Middleware) *Router {
	fullPrefix := prefix
	if kr, ok := r.(*Router); ok {
		fullPrefix = kr.prefix + prefix
	}

	var sub chi.Router

	r.Route(prefix, func(sr chi.Router) {
		for _, m := range middlewares {
			sr.Use(m)
		}
		sub = sr
	})

	return &Router{Router: sub, prefix: fullPrefix}
}
