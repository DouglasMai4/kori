package kori

// RouteInfo contains the parsed route metadata.
// Used internally by Option functions to configure routes.
type RouteInfo struct {
	Method      string
	Pattern     string
	middlewares []Middleware
}

// Option configures a route at registration time.
type Option func(*RouteInfo)

// Use attaches middlewares to a specific route.
//
//	kori.GET(r, "/admin", adminHandler, kori.Use(authMiddleware, auditMiddleware))
func Use(middlewares ...Middleware) Option {
	return func(ri *RouteInfo) {
		ri.middlewares = append(ri.middlewares, middlewares...)
	}
}
