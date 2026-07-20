package openapi

import (
	"testing"
)

func TestNewSpec_InitializesMaps(t *testing.T) {
	s := NewSpec(Config{Title: "API", Version: "1.0.0", Description: "d"})
	if s.paths == nil || s.reg == nil || s.securitySchemes == nil {
		t.Fatal("NewSpec left an internal map/registry nil")
	}
}

func TestBuild_InfoAndOpenAPIVersion(t *testing.T) {
	s := NewSpec(Config{
		Title:       "My API",
		Version:     "2.1.0",
		Description: "desc",
		Servers:     []Server{{URL: "https://api.example.com", Description: "prod"}},
	})
	doc := s.build()

	if doc.OpenAPI != "3.1.0" {
		t.Errorf("OpenAPI = %q, want 3.1.0", doc.OpenAPI)
	}
	if doc.Info.Title != "My API" || doc.Info.Version != "2.1.0" || doc.Info.Description != "desc" {
		t.Errorf("Info = %+v", doc.Info)
	}
	if len(doc.Servers) != 1 || doc.Servers[0].URL != "https://api.example.com" {
		t.Errorf("Servers = %+v", doc.Servers)
	}
}

func TestBuild_EmptySpecOmitsOptionalSections(t *testing.T) {
	doc := NewSpec(Config{Title: "T", Version: "1"}).build()
	if doc.Paths != nil {
		t.Error("Paths should be nil when no routes registered")
	}
	if doc.Components != nil {
		t.Error("Components should be nil when nothing registered")
	}
	if doc.Security != nil {
		t.Error("Security should be nil when no global security set")
	}
}

func TestAddSecurityScheme_And_Build(t *testing.T) {
	s := NewSpec(Config{Title: "T", Version: "1"})
	s.AddSecurityScheme("bearerAuth", BearerAuth("JWT"))

	doc := s.build()
	if doc.Components == nil || doc.Components.SecuritySchemes == nil {
		t.Fatal("security scheme not present in components")
	}
	scheme, ok := doc.Components.SecuritySchemes["bearerAuth"]
	if !ok {
		t.Fatal("bearerAuth scheme missing")
	}
	if scheme.Type != "http" || scheme.Scheme != "bearer" || scheme.BearerFormat != "JWT" {
		t.Errorf("scheme = %+v", scheme)
	}
}

func TestSetGlobalSecurity(t *testing.T) {
	s := NewSpec(Config{Title: "T", Version: "1"})
	s.SetGlobalSecurity(Require("bearerAuth"))

	doc := s.build()
	if len(doc.Security) != 1 {
		t.Fatalf("global Security len = %d, want 1", len(doc.Security))
	}
	if _, ok := doc.Security[0]["bearerAuth"]; !ok {
		t.Errorf("global Security = %v, want bearerAuth", doc.Security[0])
	}
}

func TestBuild_ConcurrentAccessIsSafe(t *testing.T) {
	s := NewSpec(Config{Title: "T", Version: "1"})
	done := make(chan struct{})
	go func() {
		for range 100 {
			s.AddSecurityScheme("k", BasicAuth())
		}
		close(done)
	}()
	for range 100 {
		_ = s.build()
	}
	<-done
}
