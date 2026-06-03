package openapi

import (
	"sync"
)

type Config struct {
	Title       string
	Version     string
	Description string
	Servers     []Server
}

type Spec struct {
	mu              sync.RWMutex
	config          Config
	paths           map[string]pathItem
	reg             *registry
	securitySchemes map[string]SecurityScheme
	globalSecurity  []SecurityRequirement
}

func NewSpec(cfg Config) *Spec {
	return &Spec{
		config:          cfg,
		paths:           make(map[string]pathItem),
		reg:             newRegistry(),
		securitySchemes: make(map[string]SecurityScheme),
	}
}

func (s *Spec) AddSecurityScheme(name string, scheme SecurityScheme) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.securitySchemes[name] = scheme
}

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
		for pattern, item := range s.paths {
			doc.Paths[pattern] = item
		}
	}

	s.reg.mu.Lock()
	schemas := make(map[string]*Schema, len(s.reg.schemas))
	for k, v := range s.reg.schemas {
		schemas[k] = v
	}
	s.reg.mu.Unlock()

	schemes := make(map[string]SecurityScheme, len(s.securitySchemes))
	for k, v := range s.securitySchemes {
		schemes[k] = v
	}

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
