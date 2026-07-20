package openapi

import (
	"maps"
	"sync"
)

// Config holds the OpenAPI specification metadata.
type Config struct {
	Title       string
	Version     string
	Description string
	Servers     []Server
}

// Spec builds an OpenAPI 3.1 specification from registered routes and types.
type Spec struct {
	mu              sync.RWMutex
	config          Config
	paths           map[string]pathItem
	reg             *registry
	securitySchemes map[string]SecurityScheme
	globalSecurity  []SecurityRequirement
}

// NewSpec creates a new OpenAPI spec builder.
//
//	doc := openapi.NewSpec(openapi.Config{
//	    Title:   "My API",
//	    Version: "1.0.0",
//	})
func NewSpec(cfg Config) *Spec {
	return &Spec{
		config:          cfg,
		paths:           make(map[string]pathItem),
		reg:             newRegistry(),
		securitySchemes: make(map[string]SecurityScheme),
	}
}

// AddSecurityScheme registers a security scheme for use by routes.
func (s *Spec) AddSecurityScheme(name string, scheme SecurityScheme) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.securitySchemes[name] = scheme
}

// SetGlobalSecurity sets security requirements that apply to all routes.
// Individual routes can override this via RouteConfig.Security.
func (s *Spec) SetGlobalSecurity(reqs ...SecurityRequirement) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.globalSecurity = reqs
}

func (s *Spec) build() document {
	s.mu.RLock()
	defer s.mu.RUnlock()

	doc := document{
		OpenAPI: "3.1.0",
		Info: info{
			Title:       s.config.Title,
			Version:     s.config.Version,
			Description: s.config.Description,
		},
		Servers: s.config.Servers,
	}

	if len(s.globalSecurity) > 0 {
		doc.Security = s.globalSecurity
	}

	if len(s.paths) > 0 {
		doc.Paths = make(map[string]pathItem, len(s.paths))
		maps.Copy(doc.Paths, s.paths)
	}

	s.reg.mu.Lock()
	schemas := make(map[string]*Schema, len(s.reg.schemas))
	maps.Copy(schemas, s.reg.schemas)
	s.reg.mu.Unlock()

	schemes := make(map[string]SecurityScheme, len(s.securitySchemes))
	maps.Copy(schemes, s.securitySchemes)

	if len(schemas) > 0 || len(schemes) > 0 {
		doc.Components = &components{}

		if len(schemas) > 0 {
			doc.Components.Schemas = schemas
		}

		if len(schemes) > 0 {
			doc.Components.SecuritySchemes = schemes
		}
	}

	return doc
}
