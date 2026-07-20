package openapi

import "testing"

func TestGenerateOperationID(t *testing.T) {
	cases := []struct {
		method  string
		pattern string
		want    string
	}{
		// GET on a collection -> list
		{"GET", "/users", "list-users"},
		// GET ending on a param -> get + singularized collection
		{"GET", "/users/{id}", "get-user"},
		{"GET", "/categories/{id}", "get-category"},
		// POST -> create + singularized
		{"POST", "/users", "create-user"},
		// PUT -> replace
		{"PUT", "/users/{id}", "replace-user"},
		// PATCH -> update
		{"PATCH", "/users/{id}", "update-user"},
		// DELETE -> delete
		{"DELETE", "/users/{id}", "delete-user"},
		// nested resources
		{"GET", "/users/{id}/posts", "list-users-posts"},
		{"GET", "/users/{id}/posts/{postId}", "get-users-post"},
		// version segments are dropped
		{"GET", "/v1/users", "list-users"},
		{"GET", "/v2/users/{id}", "get-user"},
		// HEAD / OPTIONS keep their verb, no singularize
		{"HEAD", "/users", "head-users"},
		{"OPTIONS", "/users", "options-users"},
		// no non-param segments -> falls back to the lowercased method
		// (no resource name to build a verb phrase from).
		{"GET", "/", "get"},
		{"POST", "/", "post"},
		// trailing slash tolerated
		{"GET", "/users/", "list-users"},
	}

	for _, tc := range cases {
		t.Run(tc.method+" "+tc.pattern, func(t *testing.T) {
			got := generateOperationID(tc.method, tc.pattern)
			if got != tc.want {
				t.Errorf("generateOperationID(%q, %q) = %q, want %q", tc.method, tc.pattern, got, tc.want)
			}
		})
	}
}

func TestSingularize(t *testing.T) {
	cases := map[string]string{
		"categories": "category", // ies -> y
		"boxes":      "box",      // xes -> x
		"buses":      "bus",      // ses -> s
		"batches":    "batch",    // ches -> ch
		"dishes":     "dish",     // shes -> sh
		"users":      "user",     // s -> ""
		"class":      "class",    // ss unchanged
		"data":       "data",     // no suffix rule
		// Known limitation: the "zes" rule only strips "es", so a doubled
		// consonant is left behind (e.g. "quizzes" -> "quizz", not "quiz").
		// Asserted here to lock the current behavior across versions.
		"quizzes": "quizz",
	}
	for in, want := range cases {
		if got := singularize(in); got != want {
			t.Errorf("singularize(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsVersionSegment(t *testing.T) {
	yes := []string{"v1", "v2", "v10", "v0"}
	no := []string{"v", "version", "1", "v1a", "va", ""}
	for _, s := range yes {
		if !isVersionSegment(s) {
			t.Errorf("isVersionSegment(%q) = false, want true", s)
		}
	}
	for _, s := range no {
		if isVersionSegment(s) {
			t.Errorf("isVersionSegment(%q) = true, want false", s)
		}
	}
}

func TestNonParamSegments(t *testing.T) {
	got := nonParamSegments("/v1/users/{id}/posts")
	want := []string{"users", "posts"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("segment %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestEndsOnParam(t *testing.T) {
	if !endsOnParam("/users/{id}") {
		t.Error("/users/{id} should end on param")
	}
	if !endsOnParam("/users/{id}/") {
		t.Error("trailing slash should be ignored")
	}
	if endsOnParam("/users") {
		t.Error("/users should not end on param")
	}
	if endsOnParam("/") {
		t.Error("/ should not end on param")
	}
}
