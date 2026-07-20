package openapi

import "testing"

type listParams struct {
	ID       int      `path:"id" doc:"resource id" example:"7"`
	Page     int      `query:"page" validate:"min=1"`
	Q        string   `query:"q" validate:"required"`
	Tags     []string `query:"tags"`
	Trace    string   `header:"X-Trace"`
	Internal string
}

func TestExtractParams(t *testing.T) {
	reg := newRegistry()
	params := extractParams(&listParams{}, reg)

	byName := map[string]Parameter{}
	for _, p := range params {
		byName[p.Name] = p
	}

	if _, ok := byName["Internal"]; ok {
		t.Error("field without a binding tag must be excluded")
	}
	if len(params) != 5 {
		t.Fatalf("got %d params, want 5: %+v", len(params), params)
	}

	id := byName["id"]
	if id.In != "path" || !id.Required {
		t.Errorf("id = %+v, want in=path required=true", id)
	}
	if id.Description != "resource id" {
		t.Errorf("id.Description = %q", id.Description)
	}
	if id.Example != int64(7) {
		t.Errorf("id.Example = %v (%T), want int64(7)", id.Example, id.Example)
	}

	if !byName["q"].Required {
		t.Error("q with validate:required should be required")
	}
	if byName["page"].Required {
		t.Error("page without required should be optional")
	}
	if byName["X-Trace"].In != "header" {
		t.Errorf("X-Trace in = %q, want header", byName["X-Trace"].In)
	}

	if byName["page"].Schema == nil || byName["page"].Schema.Minimum == nil {
		t.Error("page schema should carry min constraint")
	}
}

type embedParams struct {
	CommonParams
	Extra string `query:"extra"`
}

type CommonParams struct {
	Token string `header:"Authorization" validate:"required"`
}

func TestExtractParams_Embedded(t *testing.T) {
	params := extractParams(&embedParams{}, newRegistry())
	names := map[string]bool{}
	for _, p := range params {
		names[p.Name] = true
	}
	if !names["Authorization"] {
		t.Error("embedded param 'Authorization' not extracted")
	}
	if !names["extra"] {
		t.Error("own param 'extra' not extracted")
	}
}

type unexportedEmbed struct {
	Extra string `query:"extra"`
	commonParams
}

type commonParams struct {
	Token string `header:"Authorization"`
}

func TestExtractParams_UnexportedEmbeddedPromoted(t *testing.T) {
	params := extractParams(&unexportedEmbed{}, newRegistry())
	names := map[string]bool{}
	for _, p := range params {
		names[p.Name] = true
	}
	if !names["Authorization"] {
		t.Error("promoted param 'Authorization' from an unexported embedded type should be extracted")
	}
	if !names["extra"] {
		t.Error("own param 'extra' not extracted")
	}
}

func TestExtractParams_NilAndNonStruct(t *testing.T) {
	if p := extractParams(nil, newRegistry()); p != nil {
		t.Errorf("extractParams(nil) = %v, want nil", p)
	}
	if p := extractParams(42, newRegistry()); p != nil {
		t.Errorf("extractParams(non-struct) = %v, want nil", p)
	}
}
