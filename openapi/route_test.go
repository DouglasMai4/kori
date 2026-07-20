package openapi

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/douglasmai4/kori"
	"github.com/go-chi/chi/v5"
)

func TestBodyAllowed(t *testing.T) {
	allowed := []string{"POST", "PUT", "PATCH", "post"}
	denied := []string{"GET", "HEAD", "DELETE", "OPTIONS", "TRACE", "get"}
	for _, m := range allowed {
		if !bodyAllowed(m) {
			t.Errorf("bodyAllowed(%q) = false, want true", m)
		}
	}
	for _, m := range denied {
		if bodyAllowed(m) {
			t.Errorf("bodyAllowed(%q) = true, want false", m)
		}
	}
}

type createReq struct {
	Name string `json:"name" validate:"required"`
}

type createResp struct {
	ID string `json:"id"`
}

func TestBuildOperation_DefaultOperationID(t *testing.T) {
	s := NewSpec(Config{})
	op := s.buildOperation("GET", "/users/{id}", RouteConfig{})
	if op.OperationID != "get-user" {
		t.Errorf("OperationID = %q, want get-user (auto-generated)", op.OperationID)
	}
}

func TestBuildOperation_ExplicitOperationID(t *testing.T) {
	s := NewSpec(Config{})
	op := s.buildOperation("GET", "/users", RouteConfig{OperationID: "myOp"})
	if op.OperationID != "myOp" {
		t.Errorf("OperationID = %q, want myOp", op.OperationID)
	}
}

func TestBuildOperation_BodyOnlyForBodyMethods(t *testing.T) {
	s := NewSpec(Config{})

	post := s.buildOperation("POST", "/users", RouteConfig{Body: &createReq{}})
	if post.RequestBody == nil {
		t.Fatal("POST with Body should produce a RequestBody")
	}
	if !post.RequestBody.Required {
		t.Error("RequestBody.Required should be true")
	}
	if _, ok := post.RequestBody.Content["application/json"]; !ok {
		t.Error("RequestBody should carry application/json content")
	}

	get := s.buildOperation("GET", "/users", RouteConfig{Body: &createReq{}})
	if get.RequestBody != nil {
		t.Error("GET must not have a RequestBody")
	}
}

func TestBuildOperation_DefaultResponse(t *testing.T) {
	s := NewSpec(Config{})
	op := s.buildOperation("GET", "/x", RouteConfig{})
	if _, ok := op.Responses["default"]; !ok {
		t.Error("expected a 'default' response when none configured")
	}
}

func TestBuildOperation_ResponsesWithSchema(t *testing.T) {
	s := NewSpec(Config{})
	op := s.buildOperation("POST", "/users", RouteConfig{
		Responses: map[int]any{
			201: &createResp{},
			204: nil,
		},
	})

	r201, ok := op.Responses["201"]
	if !ok {
		t.Fatal("missing 201 response")
	}
	if r201.Description != http.StatusText(201) {
		t.Errorf("201 description = %q, want %q", r201.Description, http.StatusText(201))
	}
	if _, ok := r201.Content["application/json"]; !ok {
		t.Error("201 should carry a JSON schema")
	}

	r204, ok := op.Responses["204"]
	if !ok {
		t.Fatal("missing 204 response")
	}
	if r204.Content != nil {
		t.Error("204 with nil body should have no content")
	}
}

func TestBuildOperation_Security(t *testing.T) {
	s := NewSpec(Config{})

	noSec := s.buildOperation("GET", "/public", RouteConfig{NoSecurity: true})
	if noSec.Security == nil || len(noSec.Security) != 0 {
		t.Errorf("Security = %v, want non-nil empty slice", noSec.Security)
	}

	req := Require("bearer")
	withSec := s.buildOperation("GET", "/private", RouteConfig{Security: []SecurityRequirement{req}})
	if len(withSec.Security) != 1 {
		t.Fatalf("Security len = %d, want 1", len(withSec.Security))
	}

	none := s.buildOperation("GET", "/x", RouteConfig{})
	if none.Security != nil {
		t.Errorf("Security = %v, want nil", none.Security)
	}
}

func TestBuildOperation_Metadata(t *testing.T) {
	s := NewSpec(Config{})
	op := s.buildOperation("GET", "/x", RouteConfig{
		Summary:     "Sum",
		Description: "Desc",
		Tags:        []string{"a", "b"},
		Deprecated:  true,
	})
	if op.Summary != "Sum" || op.Description != "Desc" || !op.Deprecated {
		t.Errorf("metadata not carried: %+v", op)
	}
	if len(op.Tags) != 2 {
		t.Errorf("Tags = %v", op.Tags)
	}
}

func TestRoute_Integration(t *testing.T) {
	doc := NewSpec(Config{Title: "T", Version: "1"})
	r := chi.NewRouter()

	kori.GET(r, "/todos", func(w http.ResponseWriter, req *http.Request) error {
		return kori.NoContent(w)
	}, doc.Route(RouteConfig{Summary: "List"}))

	built := doc.build()
	item, ok := built.Paths["/todos"]
	if !ok {
		t.Fatalf("paths = %v, want /todos registered", built.Paths)
	}
	if item["get"] == nil {
		t.Fatal("get operation not registered for /todos")
	}
	if item["get"].Summary != "List" {
		t.Errorf("summary = %q, want List", item["get"].Summary)
	}

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest("GET", "/todos", nil))
	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
}

func TestRoute_Integration_GroupPrefix(t *testing.T) {
	doc := NewSpec(Config{Title: "T", Version: "1"})
	r := chi.NewRouter()

	api := kori.Group(r, "/api")
	kori.GET(api, "/health", func(w http.ResponseWriter, req *http.Request) error {
		return kori.NoContent(w)
	}, doc.Route(RouteConfig{}))

	built := doc.build()
	if _, ok := built.Paths["/api/health"]; !ok {
		t.Errorf("paths = %v, want /api/health (prefix-aware)", built.Paths)
	}
}
