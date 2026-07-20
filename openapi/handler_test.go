package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func specWithRoute() *Spec {
	s := NewSpec(Config{Title: "Docs API", Version: "1.0.0"})
	s.buildOperation("GET", "/ping", RouteConfig{Summary: "Ping"})
	s.paths["/ping"] = pathItem{"get": s.buildOperation("GET", "/ping", RouteConfig{Summary: "Ping"})}
	return s
}

func TestJSONHandler(t *testing.T) {
	s := specWithRoute()
	rec := httptest.NewRecorder()
	s.JSONHandler()(rec, httptest.NewRequest("GET", "/openapi.json", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}

	var doc map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Errorf("openapi = %v, want 3.1.0", doc["openapi"])
	}
	info, _ := doc["info"].(map[string]any)
	if info == nil || info["title"] != "Docs API" {
		t.Errorf("info = %v", doc["info"])
	}
	if _, ok := doc["paths"]; !ok {
		t.Error("paths missing from JSON spec")
	}
}

func TestYAMLHandler(t *testing.T) {
	s := specWithRoute()
	rec := httptest.NewRecorder()
	s.YAMLHandler()(rec, httptest.NewRequest("GET", "/openapi.yaml", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/yaml; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}

	var doc map[string]any
	if err := yaml.Unmarshal(rec.Body.Bytes(), &doc); err != nil {
		t.Fatalf("response is not valid YAML: %v", err)
	}
	if doc["openapi"] != "3.1.0" {
		t.Errorf("openapi = %v, want 3.1.0", doc["openapi"])
	}
}

func TestScalarHandler(t *testing.T) {
	s := NewSpec(Config{Title: "My Docs", Version: "1"})
	rec := httptest.NewRecorder()
	s.ScalarHandler("/openapi.json")(rec, httptest.NewRequest("GET", "/docs", nil))

	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "My Docs") {
		t.Error("HTML should contain the spec title")
	}
	if !strings.Contains(body, "/openapi.json") {
		t.Error("HTML should reference the spec URL")
	}
	if !strings.Contains(body, "@scalar/api-reference") {
		t.Error("HTML should load the Scalar script")
	}
}

func TestScalarHandler_Options(t *testing.T) {
	s := NewSpec(Config{Title: "T", Version: "1"})
	rec := httptest.NewRecorder()
	s.ScalarHandler("/spec.json", ScalarOptions{
		Theme:     "moon",
		DarkMode:  true,
		CustomCSS: ".scalar { color: red; }",
	})(rec, httptest.NewRequest("GET", "/docs", nil))

	body := rec.Body.String()
	if !strings.Contains(body, "moon") {
		t.Error("theme option not rendered")
	}
	if !strings.Contains(body, ".scalar { color: red; }") {
		t.Error("custom CSS not rendered")
	}
}

func TestBuildScalarHTML_DefaultTheme(t *testing.T) {
	html := buildScalarHTML("T", "/s.json", ScalarOptions{})
	if !strings.Contains(html, "default") {
		t.Error("default theme should be applied when none specified")
	}

	if strings.Contains(html, `data-configuration='{`) && strings.Contains(html, "'}'") {
		t.Error("unescaped single quote leaked into data attribute")
	}
}
