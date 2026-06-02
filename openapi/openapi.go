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
	mu     sync.RWMutex
	config Config
	paths  map[string]pathItem
	reg    *registry
}

func NewSpec(cfg Config) *Spec {
	return &Spec{
		config: cfg,
		paths:  make(map[string]pathItem),
		reg:    newRegistry(),
	}
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

	if len(schemas) > 0 {
		doc.Components = &components{Schemas: schemas}
	}

	return doc
}
