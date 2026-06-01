package kori

type RouteInfo struct {
	Method      string
	Pattern     string
	middlewares []Middleware
}

type Option func(*RouteInfo)

func Use(middlewares ...Middleware) Option {
	return func(ri *RouteInfo) {
		ri.middlewares = append(ri.middlewares, middlewares...)
	}
}
